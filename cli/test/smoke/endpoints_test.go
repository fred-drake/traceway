//go:build smoke

package smoke

import (
	"encoding/json"
	"strings"
	"testing"
)

func endpointsLogin(t *testing.T) {
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

func TestSmokeEndpointsList(t *testing.T) {
	endpointsLogin(t)

	stdout, stderr, code := runCLI(t, "", "endpoints", "list", "--since", "24h", "--output", "json")
	if code != 0 {
		t.Fatalf("endpoints list exit %d\nstderr: %s", code, stderr)
	}
	var resp map[string]any
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		t.Fatalf("endpoints list stdout not JSON: %v\n%s", err, stdout)
	}
	if _, ok := resp["data"]; !ok {
		t.Errorf("endpoints list response missing 'data' field: %s", stdout)
	}
}

func TestSmokeEndpointsListBadOrderBy(t *testing.T) {
	endpointsLogin(t)
	_, stderr, code := runCLI(t, "",
		"endpoints", "list", "--since", "1h",
		"--order-by", "bogus", "--output", "json")
	if code != 2 {
		t.Fatalf("endpoints list --order-by bogus exit = %d, want 2\nstderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, "usage_error") {
		t.Errorf("expected usage_error envelope, got: %s", stderr)
	}
}

func TestSmokeEndpointsListFarFuture(t *testing.T) {
	endpointsLogin(t)
	stdout, stderr, code := runCLI(t, "",
		"endpoints", "list",
		"--from", "2099-01-01T00:00:00Z",
		"--to", "2099-01-02T00:00:00Z",
		"--output", "json")
	if code != 0 {
		t.Fatalf("endpoints list (far future) exit %d\nstderr: %s", code, stderr)
	}
	var resp struct {
		Data       []map[string]any `json:"data"`
		Pagination map[string]any   `json:"pagination"`
	}
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		t.Fatalf("endpoints list far-future stdout not JSON: %v\n%s", err, stdout)
	}
	if len(resp.Data) != 0 {
		t.Errorf("far-future window returned %d rows, expected 0: %s", len(resp.Data), stdout)
	}
	if resp.Pagination == nil {
		t.Errorf("far-future response missing pagination wrapper: %s", stdout)
	}
}
