//go:build smoke

package smoke

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestSmokeFlagCoverage exercises positive-path flag combinations documented
// by the traceway-cli usage skill that have no other smoke coverage. They
// share one login + one projects-use to keep wall-clock cost down; each
// subtest is otherwise independent.
func TestSmokeFlagCoverage(t *testing.T) {
	url, user, pass, proj := requireEnv(t)
	freshXDG(t)
	if _, _, code := runCLI(t, pass+"\n", "login", "--url", url, "--username", user, "--password-stdin"); code != 0 {
		t.Fatal("login failed")
	}
	if _, _, code := runCLI(t, "", "projects", "use", proj); code != 0 {
		t.Fatalf("projects use %s failed", proj)
	}

	t.Run("exceptions-since-Nd", func(t *testing.T) {
		// 7d must parse as 168h (the 839e873 fix). Smoke already covers 7D
		// (capital) as the negative case; this is the missing positive.
		stdout, stderr, code := runCLI(t, "",
			"exceptions", "list", "--since", "7d", "--page-size", "1", "--output", "json")
		if code != 0 {
			t.Fatalf("exceptions list --since 7d exit %d\nstderr: %s", code, stderr)
		}
		var resp struct {
			Pagination map[string]any `json:"pagination"`
		}
		if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
			t.Fatalf("stdout not JSON: %v\n%s", err, stdout)
		}
		if resp.Pagination == nil {
			t.Errorf("response missing pagination wrapper: %s", stdout)
		}
	})

	t.Run("exceptions-search-type-text", func(t *testing.T) {
		_, stderr, code := runCLI(t, "",
			"exceptions", "list", "--since", "24h",
			"--search", "Exception", "--search-type", "text",
			"--page-size", "1", "--output", "json")
		if code != 0 {
			t.Fatalf("exceptions list --search-type text exit %d\nstderr: %s", code, stderr)
		}
	})

	t.Run("exceptions-include-archived", func(t *testing.T) {
		_, stderr, code := runCLI(t, "",
			"exceptions", "list", "--since", "24h",
			"--include-archived", "--page-size", "1", "--output", "json")
		if code != 0 {
			t.Fatalf("exceptions list --include-archived exit %d\nstderr: %s", code, stderr)
		}
	})

	t.Run("endpoints-order-by-valid", func(t *testing.T) {
		// Cycle through all documented valid --order-by choices to confirm
		// the enum validator accepts them (complements the bogus negative).
		for _, choice := range []string{"impact", "count", "p95", "lastSeen"} {
			_, stderr, code := runCLI(t, "",
				"endpoints", "list", "--since", "24h",
				"--order-by", choice, "--page-size", "1", "--output", "json")
			if code != 0 {
				t.Errorf("endpoints list --order-by %s exit %d\nstderr: %s", choice, code, stderr)
			}
		}
	})

	t.Run("logs-min-severity-and-service", func(t *testing.T) {
		// --min-severity is numeric (OTel severity). The doc warns against
		// the wrong-shaped --severity error form; this pins the real flag.
		// --service with a name that almost certainly doesn't exist still
		// exits 0 (empty filter result).
		stdout, stderr, code := runCLI(t, "",
			"logs", "query", "--since", "1h",
			"--min-severity", "17", "--service", "nonexistent-service-xyz",
			"--page-size", "1", "--output", "json")
		if code != 0 {
			t.Fatalf("logs query --min-severity 17 --service ... exit %d\nstderr: %s", code, stderr)
		}
		var resp struct {
			Pagination map[string]any `json:"pagination"`
		}
		if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
			t.Fatalf("stdout not JSON: %v\n%s", err, stdout)
		}
		if resp.Pagination == nil {
			t.Errorf("response missing pagination wrapper: %s", stdout)
		}
	})

	t.Run("projects-fields-projection", func(t *testing.T) {
		// --fields id should yield objects with id only (no name) when
		// applied to projects list (which is a bare array, not a wrapper).
		stdout, stderr, code := runCLI(t, "",
			"projects", "list", "--fields", "id", "--output", "json")
		if code != 0 {
			t.Fatalf("projects list --fields id exit %d\nstderr: %s", code, stderr)
		}
		var arr []map[string]any
		if err := json.Unmarshal([]byte(stdout), &arr); err != nil {
			t.Fatalf("stdout not a JSON array: %v\n%s", err, stdout)
		}
		if len(arr) == 0 {
			t.Fatalf("--fields id returned 0 projects: %s", stdout)
		}
		for i, p := range arr {
			if _, ok := p["id"]; !ok {
				t.Errorf("project[%d] missing 'id': %v", i, p)
			}
			if _, ok := p["name"]; ok {
				t.Errorf("project[%d] still has 'name' after --fields id: %v", i, p)
			}
		}
	})

	t.Run("default-output-when-piped", func(t *testing.T) {
		// runCLI captures stdout via a pipe (non-TTY). The CLI's auto-mode
		// resolution should pick JSON, not table. Run without --output and
		// assert the stdout parses as JSON.
		stdout, stderr, code := runCLI(t, "", "projects", "list")
		if code != 0 {
			t.Fatalf("projects list (default --output) exit %d\nstderr: %s", code, stderr)
		}
		var arr []map[string]any
		if err := json.Unmarshal([]byte(stdout), &arr); err != nil {
			t.Fatalf("piped default-output stdout not JSON: %v\n%s",
				err, strings.SplitN(stdout, "\n", 4))
		}
		if len(arr) == 0 {
			t.Fatalf("projects list returned 0 projects")
		}
	})
}
