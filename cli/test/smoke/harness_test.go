//go:build smoke

package smoke

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "traceway-smoke-bin-")
	if err != nil {
		panic(err)
	}

	binaryPath = filepath.Join(tmp, "traceway")
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/traceway")
	cmd.Dir = repoRoot()
	if out, err := cmd.CombinedOutput(); err != nil {
		panic("smoke harness: go build failed: " + err.Error() + "\n" + string(out))
	}
	code := m.Run()
	_ = os.RemoveAll(tmp)
	os.Exit(code)
}

// repoRoot walks up from the current working dir to find go.mod.
func repoRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	for d := wd; d != "/"; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d
		}
	}
	panic("smoke harness: go.mod not found above " + wd)
}

// requireEnv reads the four smoke env vars and skips the test cleanly if
// any are missing. Returns (url, username, password, projectID).
func requireEnv(t *testing.T) (string, string, string, string) {
	t.Helper()
	url := os.Getenv("TRACEWAY_SMOKE_URL")
	user := os.Getenv("TRACEWAY_SMOKE_USERNAME")
	pass := os.Getenv("TRACEWAY_SMOKE_PASSWORD")
	proj := os.Getenv("TRACEWAY_SMOKE_PROJECT_ID")
	if url == "" || user == "" || pass == "" || proj == "" {
		t.Skip("smoke env not set (TRACEWAY_SMOKE_URL/USERNAME/PASSWORD/PROJECT_ID)")
	}
	return url, user, pass, proj
}

// freshXDG isolates each test by pointing XDG_CONFIG_HOME and XDG_STATE_HOME
// at fresh temp dirs. Cleanup is handled by t.TempDir.
func freshXDG(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
}

// runCLI invokes the built binary with the given args. stdinBody is fed
// to the process stdin; pass "" for none. Returns stdout, stderr, exit code.
func runCLI(t *testing.T, stdinBody string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Env = os.Environ() // inherit XDG_* and any other vars
	if stdinBody != "" {
		cmd.Stdin = strings.NewReader(stdinBody)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			code = ee.ExitCode()
		} else {
			t.Fatalf("runCLI: failed to run %v: %v", args, err)
		}
	}
	return stdout.String(), stderr.String(), code
}
