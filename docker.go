package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// dockerContainer is one running container as reported by `docker ps`,
// including the host ports it publishes.
type dockerContainer struct {
	ID      string
	Name    string
	Image   string
	State   string
	Compose string
	Ports   []Port
}

// Processes that exist only to forward traffic into containers or WSL. When
// one of these holds a published container port, the row is shown as the
// container itself instead of the proxy binary. Rows for these processes that
// match no container are dropped (infrastructure noise), except for Docker
// Desktop's backend which is kept when unmatched.
var dockerProxyNames = map[string]bool{
	"com.docker.backend":  true,
	"com.docker.build":    true,
	"vpnkit":              true,
	"vpnkit-bridge":       true,
	"com.docker.dev-envs": true,
}

// Proxy processes that are pure infrastructure: matched ports become
// containers, unmatched ports are hidden entirely.
var dockerInfraNames = map[string]bool{
	"wslrelay":     true,
	"wslhost":      true,
	"wslservice":   true,
	"dockerd":      true,
	"docker-proxy": true,
}

// Matches one mapping inside the `docker ps` Ports column, e.g.
//
//	0.0.0.0:8080->80/tcp   [::]:9090->9090/tcp   127.0.0.1:5000-5002->5000-5002/udp
var dockerPortRE = regexp.MustCompile(`^(?:(?:\[([^\]]+)\]|([^:\s]+)):)?(\d+)(?:-(\d+))?->(\d+)(?:-(\d+))?/(tcp|udp)$`)

// dockerContainers lists running containers via the docker CLI. Docker is
// commonly installed inside WSL rather than on Windows, so the Windows CLI is
// tried first and each running distro after that. Returns nil when no docker
// engine is reachable.
func dockerContainers(ctx context.Context) []dockerContainer {
	out, err := runOut(ctx, "docker", "ps", "--format", "{{json .}}")
	if err == nil {
		if cs := parseDockerPs(out); len(cs) > 0 {
			return cs
		}
	}
	for _, d := range wslRunningDistros(ctx) {
		out, err := wslRun(ctx, d, "docker ps --format '{{json .}}' 2>/dev/null")
		if err != nil && len(out) == 0 {
			continue
		}
		if cs := parseDockerPs(out, decodeWslText(out)); len(cs) > 0 {
			return cs
		}
	}
	return nil
}

// parseDockerPs parses `docker ps --format {{json .}}` output (one JSON
// object per line). extra is an optional decoded variant of out (WSL output
// arrives UTF-16 encoded).
func parseDockerPs(out []byte, extra ...string) []dockerContainer {
	text := string(out)
	if len(extra) > 0 {
		text = extra[0]
	}
	var cs []dockerContainer
	for _, ln := range strings.Split(text, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" || !strings.HasPrefix(ln, "{") {
			continue
		}
		var raw struct {
			ID     string `json:"ID"`
			Names  string `json:"Names"`
			Image  string `json:"Image"`
			State  string `json:"State"`
			Labels string `json:"Labels"`
			Ports  string `json:"Ports"`
		}
		if json.Unmarshal([]byte(ln), &raw) != nil || raw.ID == "" || raw.Names == "" {
			continue
		}
		c := dockerContainer{
			ID:    raw.ID,
			Name:  raw.Names,
			Image: raw.Image,
			State: raw.State,
		}
		for _, l := range strings.Split(raw.Labels, ",") {
			if kv := strings.SplitN(l, "=", 2); len(kv) == 2 && kv[0] == "com.docker.compose.project" {
				c.Compose = kv[1]
			}
		}
		c.Ports = parseDockerPorts(raw.Ports)
		cs = append(cs, c)
	}
	return cs
}

// parseDockerPorts expands the comma-separated Ports column into published
// host ports. Ranges are expanded up to a sane cap so a mapping like
// 0.0.0.0:1000-2000 cannot explode the list. Dual-stack mappings (0.0.0.0 and
// [::] for the same port) collapse to one entry.
func parseDockerPorts(s string) []Port {
	var ports []Port
	seen := map[string]int{}
	add := func(p Port) {
		k := p.Proto + ":" + strconv.Itoa(p.Port)
		if i, ok := seen[k]; ok {
			if isWildcardAddr(p.Addr) && !isWildcardAddr(ports[i].Addr) {
				ports[i].Addr = p.Addr
			}
			return
		}
		seen[k] = len(ports)
		ports = append(ports, p)
	}
	for _, piece := range strings.Split(s, ", ") {
		m := dockerPortRE.FindStringSubmatch(strings.TrimSpace(piece))
		if m == nil {
			continue
		}
		proto := strings.ToLower(m[7])
		hostLo, _ := strconv.Atoi(m[3])
		hostHi, _ := strconv.Atoi(orDefault(m[4], m[3]))
		ctrLo, _ := strconv.Atoi(m[5])
		ctrHi, _ := strconv.Atoi(orDefault(m[6], m[5]))
		if hostLo == 0 {
			continue
		}
		if hostHi-hostLo > 64 {
			hostHi = hostLo + 64
			ctrHi = ctrLo + 64
		}
		addr := m[1]
		if addr == "" {
			addr = m[2]
		}
		for h, c := hostLo, ctrLo; h <= hostHi && c <= ctrHi; h, c = h+1, c+1 {
			// Only the host side is observable via netstat; the container
			// port is kept so the UI can show the full mapping later.
			_ = c
			add(Port{Port: h, Proto: proto, Addr: addr})
		}
	}
	return ports
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// mergeDockerContainers replaces rows held by Docker/WSL port-forwarding
// processes with one row per running container, keyed by container ID.
// dockerProxyNames rows that match no container are kept (Docker Desktop's own
// endpoints); dockerInfraNames rows that match no container are dropped
// (pure relay noise).
func mergeDockerContainers(ctx context.Context, rows []PortProcess) []PortProcess {
	containers := dockerContainers(ctx)
	if len(containers) == 0 {
		return rows
	}

	byPort := map[string]*dockerContainer{}
	for i := range containers {
		c := &containers[i]
		for _, p := range c.Ports {
			byPort[p.Proto+":"+strconv.Itoa(p.Port)] = c
		}
	}

	var out []PortProcess
	cgroups := map[string]*PortProcess{}
	for _, r := range rows {
		lower := strings.ToLower(r.Name)
		base := strings.TrimSuffix(lower, ".exe")
		isProxy := dockerProxyNames[base]
		isInfra := dockerInfraNames[base]
		if !isProxy && !isInfra {
			out = append(out, r)
			continue
		}
		matched := false
		for _, p := range r.Ports {
			c, ok := byPort[p.Proto+":"+strconv.Itoa(p.Port)]
			if !ok {
				continue
			}
			matched = true
			g := cgroups[c.ID]
			if g == nil {
				g = &PortProcess{
					Key:         "docker|" + c.ID,
					PID:         r.PID,
					Name:        c.Name,
					Source:      "docker",
					Distro:      c.Compose,
					Image:       c.Image,
					State:       c.State,
					ContainerID: c.ID,
				}
				cgroups[c.ID] = g
			}
			g.Ports = append(g.Ports, Port{Port: p.Port, Proto: p.Proto, Addr: p.Addr})
		}
		if !matched && isProxy {
			out = append(out, r)
		}
	}

	for _, g := range cgroups {
		g.Ports = dedupePorts(g.Ports)
		out = append(out, *g)
	}
	return out
}

// dockerStop stops a container gracefully via the docker CLI. Tries the
// Windows CLI first, then each running WSL distro (Docker is commonly
// installed inside WSL, where the Windows PATH has no docker executable).
func dockerStop(ctx context.Context, id string) error {
	if out, err := runCombined(ctx, "docker", "stop", id); err == nil {
		return nil
	} else if !isExecNotFound(err) {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", msg)
	}
	for _, d := range wslRunningDistros(ctx) {
		out, err := wslRun(ctx, d, "docker stop "+shellQuote(id)+" 2>/dev/null")
		if err == nil {
			return nil
		}
		if isExecNotFound(err) {
			continue
		}
		msg := strings.TrimSpace(decodeWslText(out))
		if msg != "" && !strings.Contains(msg, "not found") {
			return fmt.Errorf("%s", msg)
		}
	}
	return fmt.Errorf("docker cli not found on windows or in any running wsl distro")
}

// isExecNotFound reports whether err is exec's "executable not found" error.
func isExecNotFound(err error) bool {
	var ee *exec.Error
	return errors.As(err, &ee)
}

// shellQuote single-quotes a value for safe use inside sh -c.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
