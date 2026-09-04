//go:build smoke

package smoke

import (
	"strings"
	"testing"
)

// TestSmokeUnknownProfile verifies that a global --profile flag pointing at
// a name that doesn't exist produces a clean not_authenticated envelope
// (exit 4), not a panic or stack trace.
func TestSmokeUnknownProfile(t *testing.T) {
	url, user, pass, _ := requireEnv(t)
	freshXDG(t)
	if _, _, code := runCLI(t, pass+"\n", "login", "--url", url, "--username", user, "--password-stdin"); code != 0 {
		t.Fatal("login failed")
	}

	_, stderr, code := runCLI(t, "",
		"--profile", "no-such-profile",
		"projects", "list", "--output", "json")
	if code != 4 {
		t.Fatalf("--profile no-such-profile exit = %d, want 4\nstderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, "not_authenticated") {
		t.Errorf("expected not_authenticated envelope, got: %s", stderr)
	}
	if strings.Contains(stderr, "panic:") || strings.Contains(stderr, "goroutine ") {
		t.Errorf("stderr contains panic/stack trace: %s", stderr)
	}
}

// TestSmokeUnknownProject verifies that a global --project flag pointing at
// a UUID the user can't access fails cleanly. The exact exit code depends on
// what the server returns (400/403/404 are all plausible — observed: 400 →
// api_error → exit 1). What we actually care about is: non-zero exit, no
// panic, no connection_failed (which would mean the URL plumbing broke).
func TestSmokeUnknownProject(t *testing.T) {
	url, user, pass, _ := requireEnv(t)
	freshXDG(t)
	if _, _, code := runCLI(t, pass+"\n", "login", "--url", url, "--username", user, "--password-stdin"); code != 0 {
		t.Fatal("login failed")
	}

	_, stderr, code := runCLI(t, "",
		"--project", "00000000-0000-0000-0000-000000000000",
		"exceptions", "list", "--since", "1h",
		"--page-size", "1", "--output", "json")
	if code == 0 {
		t.Fatalf("--project <zeros> exited 0; expected a server-side error\nstderr: %s", stderr)
	}
	if strings.Contains(stderr, "panic:") || strings.Contains(stderr, "goroutine ") {
		t.Errorf("stderr contains panic/stack trace: %s", stderr)
	}
	if strings.Contains(stderr, "connection_failed") {
		t.Errorf("got connection_failed for a server-side auth/not-found case: %s", stderr)
	}
}
