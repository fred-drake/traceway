//go:build smoke

package smoke

import (
	"encoding/json"
	"testing"
)

func TestSmokeProfilesList(t *testing.T) {
	url, user, pass, _ := requireEnv(t)
	freshXDG(t)

	if _, _, code := runCLI(t, pass+"\n", "login", "--url", url, "--username", user, "--password-stdin"); code != 0 {
		t.Fatal("login failed")
	}

	stdout, stderr, code := runCLI(t, "", "profiles", "list", "--output", "json")
	if code != 0 {
		t.Fatalf("profiles list exit %d\nstderr: %s", code, stderr)
	}
	var resp struct {
		Current string `json:"current"`
		Data    []struct {
			Name    string `json:"name"`
			URL     string `json:"url"`
			Current bool   `json:"current"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		t.Fatalf("profiles list stdout not JSON: %v\n%s", err, stdout)
	}
	if resp.Current == "" {
		t.Errorf("profiles list response missing 'current': %s", stdout)
	}
	if len(resp.Data) == 0 {
		t.Fatalf("profiles list returned 0 profiles after login: %s", stdout)
	}
	foundCurrent := false
	for _, p := range resp.Data {
		if p.Current && p.Name == resp.Current {
			foundCurrent = true
		}
	}
	if !foundCurrent {
		t.Errorf("no profile marked current matches top-level current=%q: %s", resp.Current, stdout)
	}
}
