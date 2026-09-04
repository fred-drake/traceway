//go:build smoke

package smoke

import (
	"encoding/json"
	"testing"
)

func TestSmokeProjectsList(t *testing.T) {
	url, user, pass, _ := requireEnv(t)
	freshXDG(t)

	if _, _, code := runCLI(t, pass+"\n", "login", "--url", url, "--username", user, "--password-stdin"); code != 0 {
		t.Fatal("login failed (see TestSmokeLogin)")
	}

	stdout, stderr, code := runCLI(t, "", "projects", "list", "--output", "json")
	if code != 0 {
		t.Fatalf("projects list exit %d\nstderr: %s", code, stderr)
	}
	var arr []map[string]any
	if err := json.Unmarshal([]byte(stdout), &arr); err != nil {
		t.Fatalf("projects list stdout not a JSON array: %v\n%s", err, stdout)
	}
	if len(arr) == 0 {
		t.Fatal("projects list returned 0 projects; smoke account should have at least one")
	}
	if _, ok := arr[0]["id"]; !ok {
		t.Errorf("first project missing 'id' field: %v", arr[0])
	}
}
