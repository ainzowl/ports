package main

import "testing"

func TestParseDockerPorts(t *testing.T) {
	cases := []struct {
		in   string
		want []int
	}{
		{"0.0.0.0:8080->80/tcp, [::]:8080->80/tcp", []int{8080}},
		{"127.0.0.1:5432->5432/tcp", []int{5432}},
		{"0.0.0.0:5000-5002->5000-5002/udp", []int{5000, 5001, 5002}},
		{"", nil},
		{"8443/tcp", nil}, // exposed but not published
	}
	for _, c := range cases {
		got := parseDockerPorts(c.in)
		var ports []int
		for _, p := range got {
			ports = append(ports, p.Port)
		}
		if len(ports) != len(c.want) {
			t.Fatalf("parseDockerPorts(%q) = %v, want %v", c.in, ports, c.want)
		}
		for i := range ports {
			if ports[i] != c.want[i] {
				t.Fatalf("parseDockerPorts(%q) = %v, want %v", c.in, ports, c.want)
			}
		}
	}
}

func TestParseDockerPs(t *testing.T) {
	line := `{"ID":"a1b2c3d4e5f6","Names":"my-api","Image":"nginx:alpine","State":"running","Labels":"com.docker.compose.project=web","Ports":"0.0.0.0:8080->80/tcp, [::]:8080->80/tcp"}`
	cs := parseDockerPs([]byte(line))
	if len(cs) != 1 {
		t.Fatalf("expected 1 container, got %d", len(cs))
	}
	c := cs[0]
	if c.Name != "my-api" || c.Image != "nginx:alpine" || c.Compose != "web" {
		t.Fatalf("unexpected container %+v", c)
	}
	if len(c.Ports) != 1 || c.Ports[0].Port != 8080 {
		t.Fatalf("unexpected ports %+v", c.Ports)
	}
}
