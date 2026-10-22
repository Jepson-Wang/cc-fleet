//go:build !windows

package codexproxy

import (
	"os"
	"path/filepath"
	"testing"
)

// profileTargetsPort must match the profile's base_url port exactly: a substring
// hit (another port sharing the digits, or the same port on a remote host) would
// count an unrelated claude as this daemon's worker and keep it alive.
func TestProfileTargetsPort(t *testing.T) {
	dir := t.TempDir()
	profile := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	withURL := func(name, url string) string {
		return profile(name, `{"apiKeyHelper":"x","env":{"ANTHROPIC_BASE_URL":"`+url+`"}}`)
	}
	exact := withURL("exact.json", "http://127.0.0.1:17222/")
	localhost := withURL("localhost.json", "http://localhost:17222")
	remote := withURL("remote.json", "https://gateway.example:1722/anthropic")
	direct := withURL("direct.json", "https://api.deepseek.com/anthropic")
	noEnv := profile("noenv.json", `{"apiKeyHelper":"x"}`)
	broken := profile("broken.json", `{"env": {`)

	cases := []struct {
		name string
		argv []string
		port int
		want bool
	}{
		{"exact loopback port", []string{"claude", "--settings", exact}, 17222, true},
		{"--settings= form", []string{"claude", "--settings=" + exact}, 17222, true},
		{"localhost host", []string{"claude", "--settings", localhost}, 17222, true},
		{"other port sharing the digits", []string{"claude", "--settings", exact}, 1722, false},
		{"same port on a remote host", []string{"claude", "--settings", remote}, 1722, false},
		{"direct provider, no port", []string{"claude", "--settings", direct}, 17222, false},
		{"no --settings", []string{"/bin/bash", "-l"}, 17222, false},
		{"--settings without value", []string{"claude", "--settings"}, 17222, false},
		{"missing profile", []string{"claude", "--settings", filepath.Join(dir, "gone.json")}, 17222, false},
		{"corrupt profile", []string{"claude", "--settings", broken}, 17222, false},
		{"profile without env", []string{"claude", "--settings", noEnv}, 17222, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := profileTargetsPort(c.argv, c.port); got != c.want {
				t.Fatalf("profileTargetsPort(%q, %d) = %v, want %v", c.argv, c.port, got, c.want)
			}
		})
	}
}
