//go:build smoke

package smoke

import (
	"encoding/json"
	"strings"
	"testing"
)

// metricsLogin runs the login + projects-use prelude shared by all metrics
// subtests so we don't repeat it five times.
func metricsLogin(t *testing.T) {
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

func TestSmokeMetricsQueryMissingName(t *testing.T) {
	metricsLogin(t)
	_, stderr, code := runCLI(t, "", "metrics", "query", "--since", "1h", "--output", "json")
	if code != 2 {
		t.Fatalf("metrics query (no --name) exit = %d, want 2\nstderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, "usage_error") {
		t.Errorf("expected usage_error envelope, got: %s", stderr)
	}
}

func TestSmokeMetricsQueryBogusName(t *testing.T) {
	metricsLogin(t)
	stdout, stderr, code := runCLI(t, "",
		"metrics", "query", "--name", "__no_such_metric_xyz__",
		"--since", "1h", "--output", "json")
	if code != 0 {
		t.Fatalf("metrics query (bogus name) exit = %d, want 0\nstderr: %s", code, stderr)
	}
	var resp struct {
		Results []struct {
			Name   string         `json:"name"`
			Series map[string]any `json:"series"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		t.Fatalf("metrics query stdout not JSON: %v\n%s", err, stdout)
	}
	for _, r := range resp.Results {
		if len(r.Series) != 0 {
			t.Errorf("bogus metric %q returned non-empty series: %v", r.Name, r.Series)
		}
	}
}

func TestSmokeMetricsQueryBadAggregation(t *testing.T) {
	metricsLogin(t)
	_, stderr, code := runCLI(t, "",
		"metrics", "query", "--name", "any",
		"--aggregation", "nopenope", "--since", "1h", "--output", "json")
	if code != 2 {
		t.Fatalf("metrics query (bad --aggregation) exit = %d, want 2\nstderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, "usage_error") {
		t.Errorf("expected usage_error envelope, got: %s", stderr)
	}
}

func TestSmokeMetricsQueryBadTag(t *testing.T) {
	metricsLogin(t)
	_, stderr, code := runCLI(t, "",
		"metrics", "query", "--name", "any",
		"--tag", "missingequals", "--since", "1h", "--output", "json")
	if code != 2 {
		t.Fatalf("metrics query (bad --tag) exit = %d, want 2\nstderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, "usage_error") {
		t.Errorf("expected usage_error envelope, got: %s", stderr)
	}
}
