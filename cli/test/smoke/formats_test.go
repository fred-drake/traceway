//go:build smoke

package smoke

import (
	"strings"
	"testing"
)

// TestSmokeOutputFormats exercises --output table and --output yaml on every
// list-shaped command. Only JSON shape is covered elsewhere; this test exists
// to catch regressions in the table/yaml renderers (which have no JSON
// schema to assert against, only "exit 0 and stdout is non-empty").
func TestSmokeOutputFormats(t *testing.T) {
	url, user, pass, proj := requireEnv(t)
	freshXDG(t)
	if _, _, code := runCLI(t, pass+"\n", "login", "--url", url, "--username", user, "--password-stdin"); code != 0 {
		t.Fatal("login failed")
	}
	if _, _, code := runCLI(t, "", "projects", "use", proj); code != 0 {
		t.Fatalf("projects use %s failed", proj)
	}

	cases := []struct {
		name string
		args []string
	}{
		{"projects-table", []string{"projects", "list", "--output", "table"}},
		{"projects-yaml", []string{"projects", "list", "--output", "yaml"}},
		{"profiles-table", []string{"profiles", "list", "--output", "table"}},
		{"profiles-yaml", []string{"profiles", "list", "--output", "yaml"}},
		{"exceptions-table", []string{"exceptions", "list", "--since", "24h", "--output", "table"}},
		{"exceptions-yaml", []string{"exceptions", "list", "--since", "24h", "--output", "yaml"}},
		{"endpoints-table", []string{"endpoints", "list", "--since", "24h", "--output", "table"}},
		{"endpoints-yaml", []string{"endpoints", "list", "--since", "24h", "--output", "yaml"}},
		{"logs-table", []string{"logs", "query", "--since", "1h", "--page-size", "5", "--output", "table"}},
		{"logs-yaml", []string{"logs", "query", "--since", "1h", "--page-size", "5", "--output", "yaml"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, code := runCLI(t, "", tc.args...)
			if code != 0 {
				t.Fatalf("exit %d\nargs: %v\nstderr: %s", code, tc.args, stderr)
			}
			if strings.TrimSpace(stdout) == "" {
				t.Errorf("empty stdout\nargs: %v", tc.args)
			}
		})
	}
}
