package main

import "testing"

func TestIsNewerVersion(t *testing.T) {
	cases := []struct {
		current, latest string
		want            bool
	}{
		{"v0.6.0", "v0.6.1", true},
		{"v0.6.0", "v0.7.0", true},
		{"v0.6.0", "v1.0.0", true},
		{"0.6.0", "0.6.0", false},
		{"v0.6.0", "v0.6.0", false},
		{"v0.6.1", "v0.6.0", false},
		{"v1.0.0", "v0.9.9", false},
		{"v0.6.0", "v0.6.1-rc1", true},
		{"dev", "v0.6.0", true}, // dev parses to 0.0.0
	}
	for _, c := range cases {
		if got := isNewerVersion(c.current, c.latest); got != c.want {
			t.Errorf("isNewerVersion(%q, %q) = %v; want %v", c.current, c.latest, got, c.want)
		}
	}
}
