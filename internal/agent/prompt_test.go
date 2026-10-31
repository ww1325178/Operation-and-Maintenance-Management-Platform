package agent

import "testing"

func TestBuildServerURL(t *testing.T) {
	cases := map[string]string{
		"192.168.1.100":                     "ws://192.168.1.100:8080/ws/agent",
		"192.168.1.100:9000":                "ws://192.168.1.100:9000/ws/agent",
		"ws://192.168.1.100:8080/ws/agent":  "ws://192.168.1.100:8080/ws/agent",
		"ws://1.2.3.4:8080":                 "ws://1.2.3.4:8080/ws/agent",
		" 192.168.1.100 ":                   "ws://192.168.1.100:8080/ws/agent",
	}
	for in, want := range cases {
		if got := BuildServerURL(in); got != want {
			t.Errorf("BuildServerURL(%q) = %q, want %q", in, got, want)
		}
	}
}
