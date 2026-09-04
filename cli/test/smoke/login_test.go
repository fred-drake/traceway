//go:build smoke

package smoke

import (
	"strings"
	"testing"
)

// TestSmokeLogin authenticates against the real instance and verifies the
// resulting session works for one follow-up call. This is the prerequisite
// every other smoke test depends on.
func TestSmokeLogin(t *testing.T) {
	url, user, pass, _ := requireEnv(t)
	freshXDG(t)

	stdout, stderr, code := runCLI(t,
		pass+"\n",
		"login", "--url", url, "--username", user, "--password-stdin",
	)
	if code != 0 {
		t.Fatalf("login exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "Logged in as "+user) {
		t.Errorf("login stdout = %q", stdout)
	}
}
