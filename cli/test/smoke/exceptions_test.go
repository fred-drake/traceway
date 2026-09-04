//go:build smoke

package smoke

import (
	"encoding/json"
	"strings"
	"testing"
)

// exceptionsLogin runs the login + projects-use prelude shared by the
// exceptions subtests.
func exceptionsLogin(t *testing.T) {
	t.Helper()
	url, user, pass, proj := requireEnv(t)
	freshXDG(t)
	if _, _, code := runCLI(t, pass+"\n", "login", "--url", url, "--username", user, "--password-stdin"); code != 0 {
		t.Fatal("login failed")
	}
	if _, _, code := runCLI(t, "", "projects", "use", proj); code != 0 {
		t.Fatalf("projects use %s failed", proj)
	}
}

func TestSmokeExceptionsList(t *testing.T) {
	exceptionsLogin(t)

	stdout, stderr, code := runCLI(t, "", "exceptions", "list", "--since", "24h", "--output", "json")
	if code != 0 {
		t.Fatalf("exceptions list exit %d\nstderr: %s", code, stderr)
	}
	var resp struct {
		Data       []map[string]any `json:"data"`
		Pagination map[string]any   `json:"pagination"`
	}
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		t.Fatalf("exceptions list stdout not JSON: %v\n%s", err, stdout)
	}
	if resp.Pagination == nil {
		t.Errorf("exceptions list response missing pagination wrapper: %s", stdout)
	}
}

func TestSmokeExceptionsListBadSearchType(t *testing.T) {
	exceptionsLogin(t)
	_, stderr, code := runCLI(t, "",
		"exceptions", "list", "--since", "1h",
		"--search-type", "bogus", "--output", "json")
	if code != 2 {
		t.Fatalf("exceptions list --search-type bogus exit = %d, want 2\nstderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, "usage_error") {
		t.Errorf("expected usage_error envelope, got: %s", stderr)
	}
}

func TestSmokeExceptionsListBadDuration(t *testing.T) {
	exceptionsLogin(t)
	// 7D (capital D) is not parseable — should trigger invalid_time_range.
	_, stderr, code := runCLI(t, "",
		"exceptions", "list", "--since", "7D", "--output", "json")
	if code != 2 {
		t.Fatalf("exceptions list --since 7D exit = %d, want 2\nstderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, "invalid_time_range") {
		t.Errorf("expected invalid_time_range envelope, got: %s", stderr)
	}
}

func TestSmokeExceptionsListFromWithoutTo(t *testing.T) {
	exceptionsLogin(t)
	_, stderr, code := runCLI(t, "",
		"exceptions", "list",
		"--from", "2026-01-01T00:00:00Z", "--output", "json")
	if code != 2 {
		t.Fatalf("exceptions list --from (no --to) exit = %d, want 2\nstderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, "invalid_time_range") {
		t.Errorf("expected invalid_time_range envelope, got: %s", stderr)
	}
}

func TestSmokeExceptionsListSinceAndAbsolute(t *testing.T) {
	exceptionsLogin(t)
	_, stderr, code := runCLI(t, "",
		"exceptions", "list", "--since", "1h",
		"--from", "2026-01-01T00:00:00Z",
		"--to", "2026-01-02T00:00:00Z",
		"--output", "json")
	if code != 2 {
		t.Fatalf("exceptions list (mixing --since + --from/--to) exit = %d, want 2\nstderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, "invalid_time_range") {
		t.Errorf("expected invalid_time_range envelope, got: %s", stderr)
	}
}

func TestSmokeExceptionsListFarFuture(t *testing.T) {
	exceptionsLogin(t)
	stdout, stderr, code := runCLI(t, "",
		"exceptions", "list",
		"--from", "2099-01-01T00:00:00Z",
		"--to", "2099-01-02T00:00:00Z",
		"--output", "json")
	if code != 0 {
		t.Fatalf("exceptions list (far future) exit %d\nstderr: %s", code, stderr)
	}
	// Server returns data:null for empty result sets; assert clean JSON shape
	// and that data is null OR an empty slice.
	var resp struct {
		Data       []map[string]any `json:"data"`
		Pagination map[string]any   `json:"pagination"`
	}
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		t.Fatalf("exceptions list far-future stdout not JSON: %v\n%s", err, stdout)
	}
	if len(resp.Data) != 0 {
		t.Errorf("far-future window returned %d rows, expected 0: %s", len(resp.Data), stdout)
	}
	if resp.Pagination == nil {
		t.Errorf("far-future response missing pagination wrapper: %s", stdout)
	}
}

func TestSmokeExceptionsShowBogusHash(t *testing.T) {
	exceptionsLogin(t)
	_, stderr, code := runCLI(t, "",
		"exceptions", "show",
		"0000000000000000000000000000000000000000000000000000000000000000",
		"--output", "json")
	if code != 5 {
		t.Fatalf("exceptions show <zeros> exit = %d, want 5\nstderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, "not_found") {
		t.Errorf("expected not_found envelope, got: %s", stderr)
	}
}
