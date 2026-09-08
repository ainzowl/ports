package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"unicode/utf16"
)

// hideConsole prevents spawned processes from flashing a terminal window.
func hideConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}

type Port struct {
	Port  int    `json:"port"`
	Proto string `json:"proto"`
	Addr  string `json:"addr"`
}

type PortProcess struct {
	Key    string `json:"key"`
	PID    int    `json:"pid"`
	Name   string `json:"name"`
	Source string `json:"source"`
	Distro string `json:"distro,omitempty"`
	Path   string `json:"path,omitempty"`
	Ports  []Port `json:"ports"`
}

func runOut(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	hideConsole(cmd)
	return cmd.Output()
}

func runCombined(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	hideConsole(cmd)
	return cmd.CombinedOutput()
}

func wslRun(ctx context.Context, distro, script string) ([]byte, error) {
	return runOut(ctx, "wsl.exe", "-d", distro, "-u", "root", "--", "sh", "-c", script)
}

func listAll(ctx context.Context) []PortProcess {
	rows := windowsPortProcesses(ctx)
	for _, d := range wslRunningDistros(ctx) {
		rows = append(rows, wslPortProcesses(ctx, d)...)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Source != rows[j].Source {
			return rows[i].Source < rows[j].Source
		}
		return strings.ToLower(rows[i].Name) < strings.ToLower(rows[j].Name)
	})
	return rows
}

var netstatLine = regexp.MustCompile(`^\s*(TCP|UDP)\s+(\S+)\s+(\S+)(?:\s+(\S+))?\s+(\d+)\s*$`)

var wslInfraNames = map[string]bool{
	"wsl":        true,
	"wslhost":    true,
	"wslrelay":   true,
	"wslservice": true,
}

func windowsPortProcesses(ctx context.Context) []PortProcess {
	names := map[int]string{}
	if out, err := runOut(ctx, "tasklist", "/fo", "csv", "/nh"); err == nil {
		r := csv.NewReader(bytes.NewReader(out))
		r.LazyQuotes = true
		if recs, err := r.ReadAll(); err == nil {
			for _, rec := range recs {
				if len(rec) < 2 {
					continue
				}
				pid, err := strconv.Atoi(strings.TrimSpace(rec[1]))
				if err != nil || pid <= 0 {
					continue
				}
				names[pid] = rec[0]
			}
		}
	}

	groups := map[int]*PortProcess{}
	if out, err := runOut(ctx, "netstat", "-ano"); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			m := netstatLine.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			proto := strings.ToLower(m[1])
			if proto == "tcp" && !strings.HasPrefix(strings.ToUpper(m[4]), "LISTEN") {
				continue
			}
			host, portStr, err := net.SplitHostPort(m[2])
			if err != nil {
				continue
			}
			port, err := strconv.Atoi(portStr)
			if err != nil {
				continue
			}
			pid, _ := strconv.Atoi(m[5])
			if pid <= 0 {
				continue
			}
			name := names[pid]
			base := strings.TrimSuffix(name, ".exe")
			if wslInfraNames[strings.ToLower(base)] {
				continue
			}
			if name == "" {
				name = fmt.Sprintf("PID %d", pid)
			}
			g := groups[pid]
			if g == nil {
				g = &PortProcess{Key: fmt.Sprintf("windows||%d", pid), PID: pid, Name: name, Source: "windows"}
				groups[pid] = g
			}
			g.Ports = append(g.Ports, Port{Port: port, Proto: proto, Addr: host})
		}
	}

	for pid, g := range groups {
		g.Path = windowsExePath(pid)
	}
	return finalize(groups)
}
func decodeWslText(b []byte) string {
	b = bytes.TrimPrefix(b, []byte{0xFF, 0xFE})
	b = bytes.TrimPrefix(b, []byte{0xFE, 0xFF})
	hasNulls := false
	for _, c := range b {
		if c == 0 {
			hasNulls = true
			break
		}
	}
	if !hasNulls {
		return strings.ReplaceAll(string(b), "\x00", "")
	}
	u16 := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		u16 = append(u16, uint16(b[i])|uint16(b[i+1])<<8)
	}
	return strings.ReplaceAll(string(utf16.Decode(u16)), "\x00", "")
}

func wslRunningDistros(ctx context.Context) []string {
	out, err := runCombined(ctx, "wsl.exe", "-l", "--running")
	if err != nil && len(out) == 0 {
		return nil
	}
	var ds []string
	for _, ln := range strings.Split(decodeWslText(out), "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		low := strings.ToLower(ln)
		if strings.Contains(ln, ":") || strings.Contains(low, "no running") || strings.Contains(low, "copyright") {
			continue
		}
		ln = strings.TrimSuffix(ln, " (Default)")
		if ln != "" {
			ds = append(ds, ln)
		}
	}
	return ds
}

var ssUserRE = regexp.MustCompile(`users:\(\("([^"]*)",pid=(\d+)`)
var netstatPidProgRE = regexp.MustCompile(`^(\d+)/(.+)$`)

func wslPortProcesses(ctx context.Context, distro string) []PortProcess {
	groups := map[int]*PortProcess{}

	if out, err := wslRun(ctx, distro, "ss -tulpn 2>/dev/null || /usr/sbin/ss -tulpn 2>/dev/null || /bin/ss -tulpn 2>/dev/null"); err == nil || len(out) > 0 {
		for _, line := range strings.Split(string(out), "\n") {
			f := strings.Fields(line)
			if len(f) < 6 {
				continue
			}
			proto := strings.ToLower(f[0])
			if proto != "tcp" && proto != "udp" {
				continue
			}
			state := strings.ToUpper(f[1])
			if state != "LISTEN" && state != "UNCONN" {
				continue
			}
			m := ssUserRE.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			host, portStr, err := splitAddrPort(f[4])
			if err != nil {
				continue
			}
			port, err := strconv.Atoi(portStr)
			if err != nil {
				continue
			}
			pid, _ := strconv.Atoi(m[2])
			if pid <= 0 {
				continue
			}
			addPort(groups, distro, pid, m[1], host, port, proto)
		}
	}

	if len(groups) == 0 {
		if out, err := wslRun(ctx, distro, "netstat -tulpn 2>/dev/null || /bin/netstat -tulpn 2>/dev/null || /usr/bin/netstat -tulpn 2>/dev/null"); err == nil || len(out) > 0 {
			for _, line := range strings.Split(string(out), "\n") {
				f := strings.Fields(line)
				if len(f) < 6 {
					continue
				}
				proto := strings.ToLower(f[0])
				if proto != "tcp" && proto != "udp" {
					continue
				}
				host, portStr, err := splitAddrPort(f[3])
				if err != nil {
					continue
				}
				port, err := strconv.Atoi(portStr)
				if err != nil {
					continue
				}
				pidProg := f[len(f)-1]
				if proto == "tcp" {
					if len(f) < 7 || !strings.EqualFold(f[5], "LISTEN") {
						continue
					}
					pidProg = f[6]
				} else if len(f) < 6 {
					continue
				}
				pm := netstatPidProgRE.FindStringSubmatch(pidProg)
				if pm == nil {
					continue
				}
				pid, _ := strconv.Atoi(pm[1])
				if pid <= 0 {
					continue
				}
				addPort(groups, distro, pid, pm[2], host, port, proto)
			}
		}
	}

	paths := wslExePaths(ctx, distro, groupPIDs(groups))
	for pid, g := range groups {
		g.Path = paths[pid]
	}
	return finalize(groups)
}

// groupPIDs collects the PIDs that need path lookups.
func groupPIDs(groups map[int]*PortProcess) []int {
	pids := make([]int, 0, len(groups))
	for pid := range groups {
		pids = append(pids, pid)
	}
	sort.Ints(pids)
	return pids
}

// wslExePaths resolves /proc/<pid>/exe for many PIDs. Work is chunked
// because wsl.exe silently fails on long command lines (>~16 PIDs).
func wslExePaths(ctx context.Context, distro string, pids []int) map[int]string {
	paths := map[int]string{}
	const chunk = 10
	for start := 0; start < len(pids); start += chunk {
		end := start + chunk
		if end > len(pids) {
			end = len(pids)
		}
		batch := pids[start:end]

		var b strings.Builder
		for _, pid := range batch {
			// Plain `readlink` (no -f): also resolves snap binaries whose
			// fully-qualified target does not exist.
			fmt.Fprintf(&b, "echo %d $(readlink /proc/%d/exe 2>/dev/null); ", pid, pid)
		}
		out, err := wslRun(ctx, distro, b.String())
		if err != nil && len(out) == 0 {
			continue
		}
		for _, ln := range strings.Split(decodeWslText(out), "\n") {
			ln = strings.TrimSpace(ln)
			if ln == "" {
				continue
			}
			parts := strings.SplitN(ln, " ", 2)
			pid, err := strconv.Atoi(parts[0])
			if err != nil {
				continue
			}
			if len(parts) == 2 && parts[1] != "" {
				paths[pid] = parts[1]
			}
		}
	}
	return paths
}

func addPort(groups map[int]*PortProcess, distro string, pid int, name, host string, port int, proto string) {
	g := groups[pid]
	if g == nil {
		display := name
		if display == "" {
			display = fmt.Sprintf("PID %d", pid)
		}
		g = &PortProcess{Key: fmt.Sprintf("wsl|%s|%d", distro, pid), PID: pid, Name: display, Source: "wsl", Distro: distro}
		groups[pid] = g
	}
	g.Ports = append(g.Ports, Port{Port: port, Proto: proto, Addr: host})
}

func splitAddrPort(s string) (string, string, error) {
	if h, p, err := net.SplitHostPort(s); err == nil {
		return h, p, nil
	}
	i := strings.LastIndex(s, ":")
	if i < 0 {
		return "", "", fmt.Errorf("no port in %q", s)
	}
	return strings.Trim(s[:i], "[]"), s[i+1:], nil
}

func isWildcardAddr(addr string) bool {
	switch addr {
	case "0.0.0.0", "::", "[::]", "*", "[::]:*":
		return true
	}
	return false
}

func dedupePorts(ports []Port) []Port {
	res := make([]Port, 0, len(ports))
	seen := map[string]int{}
	for _, p := range ports {
		k := p.Proto + ":" + strconv.Itoa(p.Port)
		if i, ok := seen[k]; ok {
			if isWildcardAddr(p.Addr) && !isWildcardAddr(res[i].Addr) {
				res[i].Addr = p.Addr
			}
			continue
		}
		seen[k] = len(res)
		res = append(res, p)
	}
	sort.Slice(res, func(i, j int) bool {
		if res[i].Port != res[j].Port {
			return res[i].Port < res[j].Port
		}
		return res[i].Proto < res[j].Proto
	})
	return res
}

func finalize(groups map[int]*PortProcess) []PortProcess {
	out := make([]PortProcess, 0, len(groups))
	for _, g := range groups {
		g.Ports = dedupePorts(g.Ports)
		out = append(out, *g)
	}
	return out
}

func killProcess(ctx context.Context, source, distro string, pid int) error {
	if pid <= 1 {
		return fmt.Errorf("refusing to kill PID %d", pid)
	}
	switch source {
	case "windows":
		out, err := runCombined(ctx, "taskkill", "/F", "/T", "/PID", strconv.Itoa(pid))
		if err != nil {
			msg := strings.TrimSpace(string(out))
			if msg == "" {
				msg = err.Error()
			}
			return fmt.Errorf("%s", msg)
		}
	case "wsl":
		if distro == "" {
			return fmt.Errorf("missing WSL distro")
		}
		out, err := runCombined(ctx, "wsl.exe", "-d", distro, "-u", "root", "--", "sh", "-c",
			"kill -9 "+strconv.Itoa(pid))
		if err != nil {
			msg := strings.TrimSpace(decodeWslText(out))
			if msg == "" {
				msg = err.Error()
			}
			return fmt.Errorf("%s", msg)
		}
	default:
		return fmt.Errorf("unknown source %q", source)
	}
	return nil
}
