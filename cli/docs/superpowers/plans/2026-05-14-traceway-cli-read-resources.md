# traceway-cli Read Resources Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wrap the four query resources Traceway exposes (exceptions, logs, endpoints, metrics), wiring them up as `traceway` subcommands that satisfy the four use cases approved during brainstorming: "what's broken in prod?", "why is endpoint X slow?", "what did service Y log?", "any anomalies?".

**Architecture:** Same library-first split established in Plan 1 — each resource is a method on `pkg/client.Client` plus a thin `cmd/traceway/<resource>.go` that parses flags, calls the client, renders output. Shared infrastructure (time-range parsing, pagination flags, per-command session loading) is extracted into reusable helpers so the 4 resource commands don't duplicate setup boilerplate.

**Tech Stack:** Go (stdlib `net/http`), Cobra, existing `internal/{config,state,output,exitcode}` packages, existing `pkg/client.Client`/`do()` infrastructure.

---

## Critical context: upstream API shape (verified 2026-05-14)

The design doc was wrong about how `projectId` is passed. Confirmed against `tracewayapp/traceway` source:

- `projectId` is a **URL query parameter** (`?projectId=<uuid>`), NOT in the request body. Read by `extractProjectId(c)` via `c.Query("projectId")` in `backend/app/middleware/require_project_access.middleware.go`.
- All four query endpoints are `POST` with a JSON body containing search params.
- All four return `PaginatedResponse[T]`:
  ```
  { "data": [...], "pagination": { "page", "pageSize", "total", "totalPages" } }
  ```
  EXCEPT `metrics/query` which returns `{ "results": [...] }` with no pagination.
- `PaginationParams` (request) has `page` + `pageSize`. `Pagination` (response) has `page`, `pageSize`, `total`, `totalPages`.
- Time fields are RFC3339 (`time.Time` in upstream Go, marshalled as RFC3339 strings).

Endpoint-specific request shapes (from `tracewayapp/traceway/backend/app/controllers/`):

```go
// /api/exception-stack-traces (FindGrouppedExceptionStackTraces)
type ExceptionSearchRequest struct {
    FromDate        time.Time        `json:"fromDate"`
    ToDate          time.Time        `json:"toDate"`
    OrderBy         string           `json:"orderBy"`
    Pagination      PaginationParams `json:"pagination"`
    Search          string           `json:"search"`
    SearchType      string           `json:"searchType"`
    IncludeArchived bool             `json:"includeArchived"`
}

// /api/exception-stack-traces/:hash (FindByHash)
type ExceptionDetailRequest struct {
    Pagination PaginationParams `json:"pagination"`  // for occurrences list
}

// /api/logs (List)
type LogSearchRequest struct {
    FromDate         time.Time        `json:"fromDate"`
    ToDate           time.Time        `json:"toDate"`
    OrderBy          string           `json:"orderBy"`
    SortDirection    string           `json:"sortDirection"`
    Search           string           `json:"search"`
    SearchType       string           `json:"searchType"`
    MinSeverity      uint8            `json:"minSeverity"`
    ServiceName      string           `json:"serviceName"`
    TraceId          string           `json:"traceId"`
    Pagination       PaginationParams `json:"pagination"`
    // (other fields exist; we won't expose them in v1)
}

// /api/endpoints (FindAllEndpoints)  — also FindGroupedByEndpoint with same shape
type EndpointSearchRequest struct {
    FromDate      time.Time        `json:"fromDate"`
    ToDate        time.Time        `json:"toDate"`
    OrderBy       string           `json:"orderBy"`
    SortDirection string           `json:"sortDirection"`
    Pagination    PaginationParams `json:"pagination"`
    Search        string           `json:"search"`
}

// /api/metrics/query (Query)  — note: 'from'/'to' (not fromDate/toDate), no pagination
type MetricQueryRequest struct {
    Queries         []MetricQueryItem `json:"queries"`
    From            time.Time         `json:"from"`
    To              time.Time         `json:"to"`
    IntervalMinutes int               `json:"intervalMinutes"`
}
type MetricQueryItem struct {
    Name        string            `json:"name"`
    Aggregation string            `json:"aggregation"`
    TagFilters  map[string]string `json:"tagFilters"`
    GroupBy     string            `json:"groupBy"`
}
```

**Endpoint-grouping decision:** for `traceway endpoints list` we use `/api/endpoints/grouped` (returns `[]EndpointStats` with p50/p95/p99 per endpoint) — that's what humans/LLMs actually want. The bare `/api/endpoints` returns individual request samples (one row per request). The "grouped" route gives the per-endpoint stats view that matches use case #2 ("why is endpoint X slow?").

---

## File Map

```
pkg/client/
├── pagination.go          (create)  — Pagination + PaginationParams types
├── pagination_test.go     (create)
├── time.go                (create)  — TimeRange type + JSON serialization helpers
├── time_test.go           (create)
├── exceptions.go          (create)  — types, ListExceptions, GetException
├── exceptions_test.go     (create)
├── logs.go                (create)  — types, QueryLogs
├── logs_test.go           (create)
├── endpoints.go           (create)  — types, ListEndpoints (uses /endpoints/grouped)
├── endpoints_test.go      (create)
├── metrics.go             (create)  — types, QueryMetrics
└── metrics_test.go        (create)

cmd/traceway/
├── session.go             (create)  — loadSession() helper: config + state + active profile + project ID resolution
├── session_test.go        (create)
├── timerange.go           (create)  — addTimeRangeFlags() + resolveTimeRange()
├── timerange_test.go      (create)
├── querycommon.go         (create)  — addPaginationFlags() + addProjectFlag() (project flag is per-command)
├── exceptions.go          (replace stub)  — list + show subcommands
├── exceptions_test.go     (create)
├── logs.go                (replace stub)  — query subcommand
├── logs_test.go           (create)
├── endpoints.go           (create) [+ register in root]  — list subcommand
├── endpoints_test.go      (create)
├── metrics.go             (create) [+ register in root]  — query subcommand
└── metrics_test.go        (create)
└── root.go                (modify)  — register newExceptionsCmd, newLogsCmd, newEndpointsCmd, newMetricsCmd
```

**File responsibilities (new):**

- **`pkg/client/pagination.go`** — `Pagination` (response shape), `PaginationParams` (request shape). Only types; no behavior.
- **`pkg/client/time.go`** — `TimeRange{From, To time.Time}`. Used in request bodies (it's not a Cobra concern — that lives in `cmd/traceway/timerange.go`).
- **`pkg/client/{exceptions,logs,endpoints,metrics}.go`** — one file per resource: domain types + the `Client` methods that wrap the HTTP endpoints.
- **`cmd/traceway/session.go`** — single source of truth for "load config, load state, resolve active profile, resolve project ID." Every query command calls this.
- **`cmd/traceway/timerange.go`** — `addTimeRangeFlags()` registers `--since/--from/--to`; `resolveTimeRange()` validates and returns the `client.TimeRange`. Mutual-exclusivity rules live here.
- **`cmd/traceway/querycommon.go`** — `addPaginationFlags()` adds `--page` and `--page-size`. Also `addProjectFlag()` since `--project` is now per-command (the original persistent flag stays for backward compat? — see decision below).

**Decision on `--project` placement:** Plan 1 made `--project` a persistent root flag. We keep that for backward compat AND because every query command uses it. Per-command we just read `flagProject` directly. (No new flag declaration needed — it already exists.)

**Decision on `--since`/`--from`/`--to`/`--page`/`--page-size` placement:** These only apply to query commands. We add them per-command via the `addTimeRangeFlags`/`addPaginationFlags` helpers. They will NOT show up in `traceway login --help`, only in commands that use them.

---

## Task 1: pkg/client — Pagination types

**Files:**
- Create: `pkg/client/pagination.go`
- Test: `pkg/client/pagination_test.go`

- [ ] **Step 1: Write the failing tests**

Create `pkg/client/pagination_test.go`:

```go
package client

import (
	"encoding/json"
	"testing"
)

func TestPaginationParams_marshalsJSON(t *testing.T) {
	p := PaginationParams{Page: 0, PageSize: 50}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"page":0,"pageSize":50}`
	if string(b) != want {
		t.Errorf("got %s, want %s", b, want)
	}
}

func TestPagination_unmarshalsJSON(t *testing.T) {
	in := `{"page":2,"pageSize":50,"total":312,"totalPages":7}`
	var p Pagination
	if err := json.Unmarshal([]byte(in), &p); err != nil {
		t.Fatal(err)
	}
	if p.Page != 2 || p.PageSize != 50 || p.Total != 312 || p.TotalPages != 7 {
		t.Errorf("got %+v", p)
	}
}
```

- [ ] **Step 2: Run the tests, verify they fail**

Run: `nix develop --command go test ./pkg/client/... -run Pagination`
Expected: build failure — undefined types.

- [ ] **Step 3: Implement the types**

Create `pkg/client/pagination.go`:

```go
package client

// PaginationParams is the request-side pagination control.
type PaginationParams struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}

// Pagination is the response-side pagination block. Traceway returns this
// on every paginated list endpoint.
type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"totalPages"`
}
```

- [ ] **Step 4: Run the tests, verify they pass**

Run: `nix develop --command go test ./pkg/client/... -run Pagination`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/client/pagination.go pkg/client/pagination_test.go
git commit -m "feat(client): Pagination and PaginationParams types"
```

---

## Task 2: pkg/client — TimeRange type

**Files:**
- Create: `pkg/client/time.go`
- Test: `pkg/client/time_test.go`

- [ ] **Step 1: Write the failing tests**

Create `pkg/client/time_test.go`:

```go
package client

import (
	"testing"
	"time"
)

func TestTimeRange_zeroValueIsValid(t *testing.T) {
	tr := TimeRange{}
	if !tr.From.IsZero() || !tr.To.IsZero() {
		t.Error("zero TimeRange should have zero From/To")
	}
}

func TestTimeRangeFromSince_setsFromToNow(t *testing.T) {
	now := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	tr := TimeRangeFromSinceAt(time.Hour, now)
	if !tr.To.Equal(now) {
		t.Errorf("To = %v, want %v", tr.To, now)
	}
	wantFrom := now.Add(-time.Hour)
	if !tr.From.Equal(wantFrom) {
		t.Errorf("From = %v, want %v", tr.From, wantFrom)
	}
}

func TestTimeRangeFromExplicit(t *testing.T) {
	from := time.Date(2026, 5, 13, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC)
	tr := TimeRangeFromExplicit(from, to)
	if !tr.From.Equal(from) {
		t.Errorf("From = %v, want %v", tr.From, from)
	}
	if !tr.To.Equal(to) {
		t.Errorf("To = %v, want %v", tr.To, to)
	}
}
```

- [ ] **Step 2: Run the tests, verify they fail**

Run: `nix develop --command go test ./pkg/client/... -run TimeRange`
Expected: build failure.

- [ ] **Step 3: Implement TimeRange**

Create `pkg/client/time.go`:

```go
package client

import "time"

// TimeRange is an inclusive [From, To] interval used in resource queries.
// It marshals to RFC3339 strings via the request structs that embed it.
type TimeRange struct {
	From time.Time
	To   time.Time
}

// TimeRangeFromSince returns a TimeRange ending now and starting `d` ago.
// Equivalent to TimeRangeFromSinceAt(d, time.Now()).
func TimeRangeFromSince(d time.Duration) TimeRange {
	return TimeRangeFromSinceAt(d, time.Now())
}

// TimeRangeFromSinceAt is the testable form: caller supplies "now".
func TimeRangeFromSinceAt(d time.Duration, now time.Time) TimeRange {
	return TimeRange{From: now.Add(-d), To: now}
}

// TimeRangeFromExplicit constructs a TimeRange from two explicit instants.
// Caller is responsible for ensuring From <= To.
func TimeRangeFromExplicit(from, to time.Time) TimeRange {
	return TimeRange{From: from, To: to}
}
```

- [ ] **Step 4: Run the tests, verify they pass**

Run: `nix develop --command go test ./pkg/client/... -run TimeRange`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/client/time.go pkg/client/time_test.go
git commit -m "feat(client): TimeRange type for resource queries"
```

---

## Task 3: cmd/traceway — session loader

**Files:**
- Create: `cmd/traceway/session.go`
- Test: `cmd/traceway/session_test.go`

- [ ] **Step 1: Write the failing tests**

Create `cmd/traceway/session_test.go`:

```go
package main

import (
	"errors"
	"net/http/httptest"
	"net/http"
	"testing"

	"github.com/tracewayapp/traceway/cli/internal/config"
	"github.com/tracewayapp/traceway/cli/internal/state"
)

func TestLoadSession_happyPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Cleanup(func() { flagProfile = ""; flagProject = "" })

	cfg := &config.Config{Profiles: map[string]config.Profile{
		"default": {URL: "https://x", Username: "u"},
	}}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	st := &state.State{
		CurrentProfile: "default",
		Profiles:       map[string]state.ProfileState{"default": {JWT: "tok", CurrentProjectID: "proj-1"}},
	}
	if err := st.Save(); err != nil {
		t.Fatal(err)
	}

	sess, err := loadSession()
	if err != nil {
		t.Fatalf("loadSession: %v", err)
	}
	if sess.URL != "https://x" {
		t.Errorf("URL = %q", sess.URL)
	}
	if sess.JWT != "tok" {
		t.Errorf("JWT = %q", sess.JWT)
	}
	if sess.ProjectID != "proj-1" {
		t.Errorf("ProjectID = %q", sess.ProjectID)
	}
	if sess.ProfileName != "default" {
		t.Errorf("ProfileName = %q", sess.ProfileName)
	}
}

func TestLoadSession_flagProjectOverridesState(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Cleanup(func() { flagProfile = ""; flagProject = "" })

	cfg := &config.Config{Profiles: map[string]config.Profile{
		"default": {URL: "https://x"},
	}}
	_ = cfg.Save()
	st := &state.State{
		CurrentProfile: "default",
		Profiles:       map[string]state.ProfileState{"default": {JWT: "tok", CurrentProjectID: "proj-default"}},
	}
	_ = st.Save()

	flagProject = "proj-override"
	sess, err := loadSession()
	if err != nil {
		t.Fatal(err)
	}
	if sess.ProjectID != "proj-override" {
		t.Errorf("ProjectID = %q, want proj-override", sess.ProjectID)
	}
}

func TestLoadSession_missingProfile_returnsNotAuthenticated(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Cleanup(func() { flagProfile = ""; flagProject = "" })

	_, err := loadSession()
	if !errors.Is(err, errSessionNoProfile) {
		t.Errorf("got %v, want errSessionNoProfile", err)
	}
}

func TestLoadSession_profileWithoutJWT_returnsNotAuthenticated(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Cleanup(func() { flagProfile = ""; flagProject = "" })

	cfg := &config.Config{Profiles: map[string]config.Profile{
		"default": {URL: "https://x"},
	}}
	_ = cfg.Save()
	// State exists but has no JWT for this profile.
	st := &state.State{CurrentProfile: "default", Profiles: map[string]state.ProfileState{}}
	_ = st.Save()

	_, err := loadSession()
	if !errors.Is(err, errSessionNoJWT) {
		t.Errorf("got %v, want errSessionNoJWT", err)
	}
}

func TestLoadSession_missingProject_returnsNoProject(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Cleanup(func() { flagProfile = ""; flagProject = "" })

	cfg := &config.Config{Profiles: map[string]config.Profile{
		"default": {URL: "https://x"},
	}}
	_ = cfg.Save()
	st := &state.State{
		CurrentProfile: "default",
		Profiles:       map[string]state.ProfileState{"default": {JWT: "tok"}}, // no current_project_id
	}
	_ = st.Save()

	_, err := loadSession()
	if !errors.Is(err, errSessionNoProject) {
		t.Errorf("got %v, want errSessionNoProject", err)
	}
}

// Sanity-check: a session built against an httptest server and an HTTP probe
// works end-to-end (smoke for caller patterns).
func TestLoadSession_pointsAtRealServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(204)
	}))
	defer srv.Close()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Cleanup(func() { flagProfile = ""; flagProject = "" })

	cfg := &config.Config{Profiles: map[string]config.Profile{"default": {URL: srv.URL}}}
	_ = cfg.Save()
	st := &state.State{CurrentProfile: "default", Profiles: map[string]state.ProfileState{"default": {JWT: "tok", CurrentProjectID: "p"}}}
	_ = st.Save()

	sess, err := loadSession()
	if err != nil {
		t.Fatal(err)
	}
	if sess.URL != srv.URL {
		t.Errorf("URL mismatch")
	}
}
```

- [ ] **Step 2: Run the tests, verify they fail**

Run: `nix develop --command go test ./cmd/traceway/... -run Session`
Expected: build failure.

- [ ] **Step 3: Implement loadSession**

Create `cmd/traceway/session.go`:

```go
package main

import (
	"errors"
	"fmt"

	"github.com/tracewayapp/traceway/cli/internal/config"
	"github.com/tracewayapp/traceway/cli/internal/state"
)

// session bundles everything a query command needs after resolving config,
// state, the active profile, and the project ID. Built by loadSession.
type session struct {
	ProfileName string
	URL         string
	Username    string
	JWT         string
	ProjectID   string
}

// Sentinel errors so the caller can map them to the right error envelope.
var (
	errSessionNoProfile = errors.New("session: no profile configured")
	errSessionNoJWT     = errors.New("session: profile has no stored token")
	errSessionNoProject = errors.New("session: no project selected")
)

// loadSession reads config + state, resolves the active profile and project,
// and returns a session. Returns one of the errSession* sentinels on common
// "you need to configure something" failures so callers can render the
// matching error envelope.
func loadSession() (*session, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	st, err := state.Load()
	if err != nil {
		return nil, fmt.Errorf("loading state: %w", err)
	}

	name := resolveProfileName(st)
	cp, hasCfg := cfg.Profiles[name]
	if !hasCfg {
		return nil, fmt.Errorf("%w: %q", errSessionNoProfile, name)
	}

	sp, hasState := st.Profiles[name]
	if !hasState || sp.JWT == "" {
		return nil, fmt.Errorf("%w: %q", errSessionNoJWT, name)
	}

	projectID := flagProject
	if projectID == "" {
		projectID = sp.CurrentProjectID
	}
	if projectID == "" {
		return nil, fmt.Errorf("%w: profile %q has no current project", errSessionNoProject, name)
	}

	return &session{
		ProfileName: name,
		URL:         cp.URL,
		Username:    cp.Username,
		JWT:         sp.JWT,
		ProjectID:   projectID,
	}, nil
}
```

- [ ] **Step 4: Run the tests, verify they pass**

Run: `nix develop --command go test ./cmd/traceway/... -run Session`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/traceway/session.go cmd/traceway/session_test.go
git commit -m "feat(cmd): loadSession helper unifying config/state/profile/project resolution"
```

---

## Task 4: cmd/traceway — time range flag helper

**Files:**
- Create: `cmd/traceway/timerange.go`
- Test: `cmd/traceway/timerange_test.go`

- [ ] **Step 1: Write the failing tests**

Create `cmd/traceway/timerange_test.go`:

```go
package main

import (
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// helper to build a fake command with the time-range flags wired
func newCmdWithTimeFlags(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{Use: "fake"}
	addTimeRangeFlags(cmd)
	return cmd
}

func TestResolveTimeRange_defaultIsLastHour(t *testing.T) {
	cmd := newCmdWithTimeFlags(t)
	if err := cmd.ParseFlags(nil); err != nil {
		t.Fatal(err)
	}
	tr, err := resolveTimeRange(cmd)
	if err != nil {
		t.Fatal(err)
	}
	delta := tr.To.Sub(tr.From)
	if delta < 59*time.Minute || delta > 61*time.Minute {
		t.Errorf("default range should be ~1h, got %v", delta)
	}
}

func TestResolveTimeRange_sinceFlag(t *testing.T) {
	cmd := newCmdWithTimeFlags(t)
	if err := cmd.ParseFlags([]string{"--since", "30m"}); err != nil {
		t.Fatal(err)
	}
	tr, err := resolveTimeRange(cmd)
	if err != nil {
		t.Fatal(err)
	}
	delta := tr.To.Sub(tr.From)
	if delta < 29*time.Minute || delta > 31*time.Minute {
		t.Errorf("--since 30m should give ~30m range, got %v", delta)
	}
}

func TestResolveTimeRange_fromTo(t *testing.T) {
	cmd := newCmdWithTimeFlags(t)
	if err := cmd.ParseFlags([]string{
		"--from", "2026-05-13T00:00:00Z",
		"--to", "2026-05-13T23:59:59Z",
	}); err != nil {
		t.Fatal(err)
	}
	tr, err := resolveTimeRange(cmd)
	if err != nil {
		t.Fatal(err)
	}
	wantFrom, _ := time.Parse(time.RFC3339, "2026-05-13T00:00:00Z")
	wantTo, _ := time.Parse(time.RFC3339, "2026-05-13T23:59:59Z")
	if !tr.From.Equal(wantFrom) {
		t.Errorf("From = %v, want %v", tr.From, wantFrom)
	}
	if !tr.To.Equal(wantTo) {
		t.Errorf("To = %v, want %v", tr.To, wantTo)
	}
}

func TestResolveTimeRange_sinceAndFromAreMutuallyExclusive(t *testing.T) {
	cmd := newCmdWithTimeFlags(t)
	if err := cmd.ParseFlags([]string{
		"--since", "1h",
		"--from", "2026-05-13T00:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveTimeRange(cmd); err == nil {
		t.Fatal("expected mutual-exclusivity error")
	}
}

func TestResolveTimeRange_fromWithoutTo(t *testing.T) {
	cmd := newCmdWithTimeFlags(t)
	if err := cmd.ParseFlags([]string{"--from", "2026-05-13T00:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveTimeRange(cmd); err == nil {
		t.Fatal("expected error: --from requires --to")
	}
}

func TestResolveTimeRange_invalidFromFormat(t *testing.T) {
	cmd := newCmdWithTimeFlags(t)
	if err := cmd.ParseFlags([]string{
		"--from", "not-a-date",
		"--to", "2026-05-13T23:59:59Z",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveTimeRange(cmd); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestResolveTimeRange_invalidSinceDuration(t *testing.T) {
	cmd := newCmdWithTimeFlags(t)
	if err := cmd.ParseFlags([]string{"--since", "invalid"}); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveTimeRange(cmd); err == nil {
		t.Fatal("expected duration parse error")
	}
}
```

- [ ] **Step 2: Run the tests, verify they fail**

Run: `nix develop --command go test ./cmd/traceway/... -run TimeRange`
Expected: build failure.

- [ ] **Step 3: Implement the helper**

Create `cmd/traceway/timerange.go`:

```go
package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/tracewayapp/traceway/cli/pkg/client"
)

// errInvalidTimeRange is returned for any malformed combination of
// --since / --from / --to. Callers map this to the invalid_time_range
// error envelope code with exit 2 (usage).
var errInvalidTimeRange = errors.New("invalid time range")

// addTimeRangeFlags registers --since, --from, --to on the given command.
// The default (no flags) is "since 1h".
func addTimeRangeFlags(cmd *cobra.Command) {
	f := cmd.Flags()
	f.String("since", "", "Relative time range, e.g. 1h, 24h, 7d (default: 1h, mutually exclusive with --from/--to)")
	f.String("from", "", "Start of explicit time range, RFC3339 (mutually exclusive with --since)")
	f.String("to", "", "End of explicit time range, RFC3339 (required with --from)")
}

// resolveTimeRange validates the combination of --since/--from/--to on cmd
// and returns the resulting TimeRange. The default (none of the flags) is
// "last 1 hour".
func resolveTimeRange(cmd *cobra.Command) (client.TimeRange, error) {
	since, _ := cmd.Flags().GetString("since")
	from, _ := cmd.Flags().GetString("from")
	to, _ := cmd.Flags().GetString("to")

	if since != "" && (from != "" || to != "") {
		return client.TimeRange{}, fmt.Errorf("%w: --since cannot be combined with --from/--to", errInvalidTimeRange)
	}
	if (from != "") != (to != "") {
		return client.TimeRange{}, fmt.Errorf("%w: --from and --to must be used together", errInvalidTimeRange)
	}

	if from != "" {
		fromT, err := time.Parse(time.RFC3339, from)
		if err != nil {
			return client.TimeRange{}, fmt.Errorf("%w: --from: %v", errInvalidTimeRange, err)
		}
		toT, err := time.Parse(time.RFC3339, to)
		if err != nil {
			return client.TimeRange{}, fmt.Errorf("%w: --to: %v", errInvalidTimeRange, err)
		}
		return client.TimeRangeFromExplicit(fromT, toT), nil
	}

	dur := time.Hour
	if since != "" {
		d, err := time.ParseDuration(since)
		if err != nil {
			return client.TimeRange{}, fmt.Errorf("%w: --since: %v", errInvalidTimeRange, err)
		}
		dur = d
	}
	return client.TimeRangeFromSince(dur), nil
}
```

- [ ] **Step 4: Run the tests, verify they pass**

Run: `nix develop --command go test ./cmd/traceway/... -run TimeRange`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/traceway/timerange.go cmd/traceway/timerange_test.go
git commit -m "feat(cmd): time range flag helpers (--since / --from / --to)"
```

---

## Task 5: cmd/traceway — pagination flag helper

**Files:**
- Create: `cmd/traceway/querycommon.go`
- Test: `cmd/traceway/querycommon_test.go`

- [ ] **Step 1: Write the failing tests**

Create `cmd/traceway/querycommon_test.go`:

```go
package main

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestResolvePagination_defaults(t *testing.T) {
	cmd := &cobra.Command{Use: "fake"}
	addPaginationFlags(cmd)
	if err := cmd.ParseFlags(nil); err != nil {
		t.Fatal(err)
	}
	p := resolvePagination(cmd)
	if p.Page != 0 {
		t.Errorf("Page = %d, want 0", p.Page)
	}
	if p.PageSize != 50 {
		t.Errorf("PageSize = %d, want 50", p.PageSize)
	}
}

func TestResolvePagination_explicit(t *testing.T) {
	cmd := &cobra.Command{Use: "fake"}
	addPaginationFlags(cmd)
	if err := cmd.ParseFlags([]string{"--page", "3", "--page-size", "100"}); err != nil {
		t.Fatal(err)
	}
	p := resolvePagination(cmd)
	if p.Page != 3 {
		t.Errorf("Page = %d, want 3", p.Page)
	}
	if p.PageSize != 100 {
		t.Errorf("PageSize = %d, want 100", p.PageSize)
	}
}
```

- [ ] **Step 2: Run the tests, verify they fail**

Run: `nix develop --command go test ./cmd/traceway/... -run Pagination`
Expected: build failure.

- [ ] **Step 3: Implement the helper**

Create `cmd/traceway/querycommon.go`:

```go
package main

import (
	"github.com/spf13/cobra"

	"github.com/tracewayapp/traceway/cli/pkg/client"
)

// addPaginationFlags registers --page and --page-size on the given command.
// Defaults: page=0, page-size=50.
func addPaginationFlags(cmd *cobra.Command) {
	f := cmd.Flags()
	f.Int("page", 0, "Page number (0-indexed)")
	f.Int("page-size", 50, "Page size (max records per response)")
}

// resolvePagination reads the --page/--page-size flags from cmd and returns
// a PaginationParams. Assumes addPaginationFlags was called on the command.
func resolvePagination(cmd *cobra.Command) client.PaginationParams {
	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")
	return client.PaginationParams{Page: page, PageSize: pageSize}
}
```

- [ ] **Step 4: Run the tests, verify they pass**

Run: `nix develop --command go test ./cmd/traceway/... -run Pagination`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/traceway/querycommon.go cmd/traceway/querycommon_test.go
git commit -m "feat(cmd): pagination flag helpers (--page, --page-size)"
```

---

## Task 6: pkg/client — exceptions list (with projectId query param)

**Files:**
- Create: `pkg/client/exceptions.go`
- Test: `pkg/client/exceptions_test.go`

This task introduces the **first endpoint that uses `?projectId=<uuid>`**. Future tasks for logs/endpoints/metrics follow the same pattern.

- [ ] **Step 1: Write the failing tests**

Create `pkg/client/exceptions_test.go`:

```go
package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestListExceptions_sendsProjectIdAsQueryParam(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/exception-stack-traces" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("projectId") != "proj-1" {
			t.Errorf("projectId query = %q", r.URL.Query().Get("projectId"))
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["fromDate"] == nil || body["toDate"] == nil {
			t.Errorf("body missing fromDate/toDate: %v", body)
		}
		_, _ = w.Write([]byte(`{
			"data": [
				{"exceptionHash":"h1","stackTrace":"...","count":42,"firstSeen":"2026-05-13T00:00:00Z","lastSeen":"2026-05-13T01:00:00Z"}
			],
			"pagination":{"page":0,"pageSize":50,"total":1,"totalPages":1}
		}`))
	}))
	defer srv.Close()

	c := New(srv.URL, WithJWT("tok"))
	resp, err := c.ListExceptions(context.Background(), "proj-1", ListExceptionsRequest{
		TimeRange:  TimeRange{From: time.Now().Add(-time.Hour), To: time.Now()},
		Pagination: PaginationParams{Page: 0, PageSize: 50},
	})
	if err != nil {
		t.Fatalf("ListExceptions: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("got %d exceptions, want 1", len(resp.Data))
	}
	if resp.Data[0].ExceptionHash != "h1" {
		t.Errorf("ExceptionHash = %q", resp.Data[0].ExceptionHash)
	}
	if resp.Data[0].Count != 42 {
		t.Errorf("Count = %d", resp.Data[0].Count)
	}
	if resp.Pagination.Total != 1 {
		t.Errorf("Pagination.Total = %d", resp.Pagination.Total)
	}
}

func TestListExceptions_emptyResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[],"pagination":{"page":0,"pageSize":50,"total":0,"totalPages":0}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, WithJWT("tok"))
	resp, err := c.ListExceptions(context.Background(), "proj-1", ListExceptionsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Data) != 0 {
		t.Errorf("expected empty Data")
	}
}

func TestListExceptions_passesSearchParameters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["search"] != "NullPointer" {
			t.Errorf("search = %v", body["search"])
		}
		if body["searchType"] != "text" {
			t.Errorf("searchType = %v", body["searchType"])
		}
		if body["includeArchived"] != true {
			t.Errorf("includeArchived = %v", body["includeArchived"])
		}
		_, _ = w.Write([]byte(`{"data":[],"pagination":{}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, WithJWT("tok"))
	_, err := c.ListExceptions(context.Background(), "proj-1", ListExceptionsRequest{
		Search:          "NullPointer",
		SearchType:      "text",
		IncludeArchived: true,
	})
	if err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Run the tests, verify they fail**

Run: `nix develop --command go test ./pkg/client/... -run Exceptions`
Expected: build failure.

- [ ] **Step 3: Implement exceptions.go**

Create `pkg/client/exceptions.go`:

```go
package client

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// ExceptionGroup matches Traceway's models.ExceptionGroup. Hourly trends are
// only present on list responses; on the detail endpoint they're absent.
type ExceptionGroup struct {
	ExceptionHash string                `json:"exceptionHash"`
	StackTrace    string                `json:"stackTrace"`
	FirstSeen     time.Time             `json:"firstSeen"`
	LastSeen      time.Time             `json:"lastSeen"`
	Count         uint64                `json:"count"`
	HourlyTrend   []ExceptionTrendPoint `json:"hourlyTrend,omitempty"`
}

// ExceptionTrendPoint is one entry in an exception's hourly trend.
type ExceptionTrendPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Count     uint64    `json:"count"`
}

// ListExceptionsRequest is the body for POST /api/exception-stack-traces.
// projectId travels as a URL query param (handled by ListExceptions), not in
// the body — the upstream RequireProjectAccess middleware reads it via
// c.Query("projectId").
type ListExceptionsRequest struct {
	TimeRange       TimeRange        `json:"-"`               // serialized as fromDate/toDate via MarshalJSON
	Pagination      PaginationParams `json:"pagination"`
	OrderBy         string           `json:"orderBy,omitempty"`
	Search          string           `json:"search,omitempty"`
	SearchType      string           `json:"searchType,omitempty"`
	IncludeArchived bool             `json:"includeArchived,omitempty"`
}

// MarshalJSON expands TimeRange.From / TimeRange.To into top-level fromDate /
// toDate so the wire shape matches Traceway's ExceptionSearchRequest.
func (r ListExceptionsRequest) MarshalJSON() ([]byte, error) {
	type alias ListExceptionsRequest
	wire := struct {
		FromDate time.Time `json:"fromDate"`
		ToDate   time.Time `json:"toDate"`
		alias
	}{
		FromDate: r.TimeRange.From,
		ToDate:   r.TimeRange.To,
		alias:    alias(r),
	}
	return jsonMarshalNoHTMLEscape(wire)
}

// ListExceptionsResponse mirrors the upstream PaginatedResponse[ExceptionGroup].
type ListExceptionsResponse struct {
	Data       []ExceptionGroup `json:"data"`
	Pagination Pagination       `json:"pagination"`
}

// ListExceptions returns one page of grouped exceptions for the given project.
func (c *Client) ListExceptions(ctx context.Context, projectID string, req ListExceptionsRequest) (*ListExceptionsResponse, error) {
	path := "/api/exception-stack-traces?projectId=" + url.QueryEscape(projectID)
	var resp ListExceptionsResponse
	if err := c.do(ctx, http.MethodPost, path, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
```

You also need a tiny `jsonMarshalNoHTMLEscape` helper. Add it as the last function in `pkg/client/client.go` (NOT in exceptions.go — it's reusable for every resource that has a custom MarshalJSON). Open `pkg/client/client.go` and append at the bottom:

```go
// jsonMarshalNoHTMLEscape is a json.Marshal that doesn't escape <, >, & in
// strings. The default json.Marshal turns "p < 5" into "p < 5" which
// matches what httptest decoders see anyway, but produces noise in test
// failure messages and in --output json. We use this for any custom
// MarshalJSON on request bodies.
func jsonMarshalNoHTMLEscape(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	// json.Encoder appends a trailing newline; trim it so callers can compose.
	out := buf.Bytes()
	if len(out) > 0 && out[len(out)-1] == '\n' {
		out = out[:len(out)-1]
	}
	return out, nil
}
```

`bytes` and `encoding/json` are already imported in client.go from Plan 1; no import changes needed.

- [ ] **Step 4: Run the tests, verify they pass**

Run: `nix develop --command go test ./pkg/client/... -count=1`
Expected: PASS for all client tests including the new exceptions tests.

- [ ] **Step 5: Commit**

```bash
git add pkg/client/exceptions.go pkg/client/exceptions_test.go pkg/client/client.go
git commit -m "feat(client): ExceptionGroup, ListExceptionsRequest/Response, ListExceptions"
```

---

## Task 7: pkg/client — exception detail (show by hash)

**Files:**
- Modify: `pkg/client/exceptions.go` (add detail types + GetException method)
- Modify: `pkg/client/exceptions_test.go` (add tests)

- [ ] **Step 1: Write the failing tests**

Append to `pkg/client/exceptions_test.go`:

```go
func TestGetException_sendsHashInPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/exception-stack-traces/abc123" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("projectId") != "proj-1" {
			t.Errorf("projectId = %q", r.URL.Query().Get("projectId"))
		}
		_, _ = w.Write([]byte(`{
			"group": {"exceptionHash":"abc123","stackTrace":"trace","count":7,"firstSeen":"2026-05-13T00:00:00Z","lastSeen":"2026-05-13T01:00:00Z"},
			"occurrences": [
				{"id":"00000000-0000-0000-0000-000000000001","exceptionHash":"abc123","stackTrace":"trace","recordedAt":"2026-05-13T00:30:00Z"}
			],
			"pagination": {"page":0,"pageSize":20,"total":1,"totalPages":1}
		}`))
	}))
	defer srv.Close()

	c := New(srv.URL, WithJWT("tok"))
	resp, err := c.GetException(context.Background(), "proj-1", "abc123", PaginationParams{Page: 0, PageSize: 20})
	if err != nil {
		t.Fatalf("GetException: %v", err)
	}
	if resp.Group == nil || resp.Group.ExceptionHash != "abc123" {
		t.Errorf("Group missing or wrong hash: %+v", resp.Group)
	}
	if len(resp.Occurrences) != 1 {
		t.Fatalf("got %d occurrences, want 1", len(resp.Occurrences))
	}
	if resp.Occurrences[0].ExceptionHash != "abc123" {
		t.Errorf("Occurrence ExceptionHash = %q", resp.Occurrences[0].ExceptionHash)
	}
}

func TestGetException_404_returnsErrNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := New(srv.URL, WithJWT("tok"))
	_, err := c.GetException(context.Background(), "proj-1", "missing", PaginationParams{})
	if err == nil || err.Error() == "" {
		t.Error("expected an error")
	}
	// errors.Is check
	if !isNotFound(err) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// helper because errors.Is needs the import in our test file too
func isNotFound(err error) bool {
	for ; err != nil; err = unwrap(err) {
		if err == ErrNotFound {
			return true
		}
	}
	return false
}

func unwrap(err error) error {
	type unwrapper interface{ Unwrap() error }
	if u, ok := err.(unwrapper); ok {
		return u.Unwrap()
	}
	return nil
}
```

(The `isNotFound` helper avoids adding `errors` to the test file's imports just for this; we already have it implicitly from earlier tests in the same file via `errors.Is`. If your editor complains, replace `isNotFound(err)` with `errors.Is(err, ErrNotFound)` and add `"errors"` to the import block.)

- [ ] **Step 2: Run the tests, verify they fail**

Run: `nix develop --command go test ./pkg/client/... -run Exception`
Expected: build failure — `c.GetException undefined`.

- [ ] **Step 3: Implement GetException**

Append to `pkg/client/exceptions.go`:

```go
import (
	"github.com/google/uuid"  // add to existing imports
)

// ExceptionStackTrace is one occurrence of a grouped exception.
type ExceptionStackTrace struct {
	Id                 uuid.UUID         `json:"id"`
	ExceptionHash      string            `json:"exceptionHash"`
	StackTrace         string            `json:"stackTrace"`
	RecordedAt         time.Time         `json:"recordedAt"`
	TraceId            *uuid.UUID        `json:"traceId,omitempty"`
	TraceType          string            `json:"traceType,omitempty"`
	ServerName         string            `json:"serverName,omitempty"`
	AppVersion         string            `json:"appVersion,omitempty"`
	IsMessage          bool              `json:"isMessage,omitempty"`
	Attributes         map[string]string `json:"attributes,omitempty"`
	DistributedTraceId *uuid.UUID        `json:"distributedTraceId,omitempty"`
	SessionId          *uuid.UUID        `json:"sessionId,omitempty"`
}

// GetExceptionRequest is the body for POST /api/exception-stack-traces/:hash.
type getExceptionRequest struct {
	Pagination PaginationParams `json:"pagination"`
}

// GetExceptionResponse is the upstream ExceptionDetailResponse minus the
// session-recording blob (we don't expose recordings in v1).
type GetExceptionResponse struct {
	Group       *ExceptionGroup       `json:"group"`
	Occurrences []ExceptionStackTrace `json:"occurrences"`
	Pagination  Pagination            `json:"pagination"`
}

// GetException returns the group + paginated occurrences for the given hash.
func (c *Client) GetException(ctx context.Context, projectID, hash string, page PaginationParams) (*GetExceptionResponse, error) {
	path := "/api/exception-stack-traces/" + url.PathEscape(hash) + "?projectId=" + url.QueryEscape(projectID)
	var resp GetExceptionResponse
	if err := c.do(ctx, http.MethodPost, path, getExceptionRequest{Pagination: page}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
```

You'll need to `nix develop --command go get github.com/google/uuid` (it's already a transitive dep but worth making it a direct one explicitly).

- [ ] **Step 4: Add the uuid dependency**

Run:
```bash
nix develop --command go get github.com/google/uuid
nix develop --command go mod tidy
```
Expected: `go.mod` shows `github.com/google/uuid` as direct.

- [ ] **Step 5: Run the tests, verify they pass**

Run: `nix develop --command go test ./pkg/client/... -count=1`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add pkg/client/exceptions.go pkg/client/exceptions_test.go go.mod go.sum
git commit -m "feat(client): GetException + ExceptionStackTrace type"
```

---

## Task 8: cmd/traceway — exceptions list and show

**Files:**
- Modify: `cmd/traceway/exceptions.go` (replace stub)
- Create: `cmd/traceway/exceptions_test.go`
- Modify: `cmd/traceway/root.go` (already imports newExceptionsCmd from the stub; no change needed)

- [ ] **Step 1: Write the failing tests**

Create `cmd/traceway/exceptions_test.go`:

```go
package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tracewayapp/traceway/cli/internal/config"
	"github.com/tracewayapp/traceway/cli/internal/state"
)

// seedSessionFor configures config + state for a default profile pointing at
// baseURL with a stored JWT and a current_project_id.
func seedSessionFor(t *testing.T, baseURL string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	cfg := &config.Config{Profiles: map[string]config.Profile{
		"default": {URL: baseURL, Username: "fred@example.com"},
	}}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	st := &state.State{
		CurrentProfile: "default",
		Profiles:       map[string]state.ProfileState{"default": {JWT: "tok", CurrentProjectID: "proj-1"}},
	}
	if err := st.Save(); err != nil {
		t.Fatal(err)
	}
}

func TestExceptionsList_jsonShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/exception-stack-traces" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("projectId") != "proj-1" {
			t.Errorf("projectId = %q", r.URL.Query().Get("projectId"))
		}
		_, _ = w.Write([]byte(`{
			"data":[
				{"exceptionHash":"h1","stackTrace":"t1","count":7,"firstSeen":"2026-05-13T00:00:00Z","lastSeen":"2026-05-13T01:00:00Z"}
			],
			"pagination":{"page":0,"pageSize":50,"total":1,"totalPages":1}
		}`))
	}))
	defer srv.Close()
	seedSessionFor(t, srv.URL)
	t.Cleanup(func() { flagProject = "" })

	stdout, _, err := runCmd(t, "", "exceptions", "list", "--output", "json")
	if err != nil {
		t.Fatalf("exceptions list: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, `"h1"`) {
		t.Errorf("expected hash in output, got: %s", out)
	}
	if !strings.Contains(out, `"pagination"`) {
		t.Errorf("expected pagination passthrough, got: %s", out)
	}
}

func TestExceptionsList_table(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"data":[
				{"exceptionHash":"abcdef1234567890","stackTrace":"NullPointerException at line 42","count":7,"firstSeen":"2026-05-13T00:00:00Z","lastSeen":"2026-05-13T01:00:00Z"}
			],
			"pagination":{"total":1}
		}`))
	}))
	defer srv.Close()
	seedSessionFor(t, srv.URL)

	stdout, _, err := runCmd(t, "", "exceptions", "list", "--output", "table")
	if err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	// First line is header
	if !strings.Contains(out, "HASH") || !strings.Contains(out, "COUNT") {
		t.Errorf("table missing headers: %s", out)
	}
	if !strings.Contains(out, "7") {
		t.Errorf("table missing count: %s", out)
	}
	// Hash should be truncated (first 12 chars by convention)
	if !strings.Contains(out, "abcdef123456") {
		t.Errorf("table missing truncated hash prefix: %s", out)
	}
}

func TestExceptionsList_noSession_writesEnvelope(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	_, stderr, err := runCmd(t, "", "exceptions", "list", "--output", "json")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr.String(), `"not_authenticated"`) {
		t.Errorf("expected not_authenticated envelope, got: %s", stderr.String())
	}
}

func TestExceptionsList_invalidTimeRange_writesEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[],"pagination":{}}`))
	}))
	defer srv.Close()
	seedSessionFor(t, srv.URL)

	_, stderr, err := runCmd(t, "", "exceptions", "list", "--output", "json", "--since", "1h", "--from", "2026-05-13T00:00:00Z", "--to", "2026-05-13T23:59:59Z")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr.String(), `"invalid_time_range"`) {
		t.Errorf("expected invalid_time_range envelope, got: %s", stderr.String())
	}
}

func TestExceptionsShow_passesHashInPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/exception-stack-traces/abc" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{
			"group":{"exceptionHash":"abc","stackTrace":"trace","count":3,"firstSeen":"2026-05-13T00:00:00Z","lastSeen":"2026-05-13T01:00:00Z"},
			"occurrences":[],
			"pagination":{"page":0,"pageSize":20,"total":0,"totalPages":0}
		}`))
	}))
	defer srv.Close()
	seedSessionFor(t, srv.URL)

	stdout, _, err := runCmd(t, "", "exceptions", "show", "abc", "--output", "json")
	if err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	if !strings.Contains(out, `"abc"`) {
		t.Errorf("expected hash in output: %s", out)
	}
}

func TestExceptionsShow_404_writesNotFoundEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	seedSessionFor(t, srv.URL)

	_, stderr, err := runCmd(t, "", "exceptions", "show", "missing", "--output", "json")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr.String(), `"not_found"`) {
		t.Errorf("expected not_found envelope, got: %s", stderr.String())
	}
}
```

- [ ] **Step 2: Run the tests, verify they fail**

Run: `nix develop --command go test ./cmd/traceway/... -run Exceptions`
Expected: failures — `exceptions` command is still a hidden stub.

- [ ] **Step 3: Implement the command**

Replace `cmd/traceway/exceptions.go`:

```go
package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tracewayapp/traceway/cli/internal/exitcode"
	"github.com/tracewayapp/traceway/cli/internal/output"
	"github.com/tracewayapp/traceway/cli/pkg/client"
)

func newExceptionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exceptions",
		Short: "Query exception groups and occurrences",
	}
	cmd.AddCommand(newExceptionsListCmd())
	cmd.AddCommand(newExceptionsShowCmd())
	return cmd
}

func newExceptionsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recent exception groups",
		RunE:  runExceptionsList,
	}
	addTimeRangeFlags(cmd)
	addPaginationFlags(cmd)
	cmd.Flags().String("search", "", "Free-text search filter")
	cmd.Flags().String("search-type", "text", "Search type: text or regex")
	cmd.Flags().Bool("include-archived", false, "Include archived exceptions")
	cmd.Flags().String("order-by", "lastSeen", "Sort field (lastSeen, firstSeen, count)")
	return cmd
}

func runExceptionsList(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	mode := output.ResolveMode(flagOutput, output.StdoutIsTerminal())

	sess, err := loadSession()
	if err != nil {
		return renderSessionError(cmd.ErrOrStderr(), mode, err)
	}

	tr, err := resolveTimeRange(cmd)
	if err != nil {
		return renderTimeRangeError(cmd.ErrOrStderr(), mode, err)
	}
	page := resolvePagination(cmd)
	search, _ := cmd.Flags().GetString("search")
	searchType, _ := cmd.Flags().GetString("search-type")
	includeArchived, _ := cmd.Flags().GetBool("include-archived")
	orderBy, _ := cmd.Flags().GetString("order-by")

	c := client.New(sess.URL, client.WithJWT(sess.JWT))
	resp, err := c.ListExceptions(ctx, sess.ProjectID, client.ListExceptionsRequest{
		TimeRange:       tr,
		Pagination:      page,
		Search:          search,
		SearchType:      searchType,
		IncludeArchived: includeArchived,
		OrderBy:         orderBy,
	})
	if err != nil {
		return renderAPIError(cmd.ErrOrStderr(), mode, err, false)
	}

	switch mode {
	case output.ModeJSON:
		return output.RenderJSON(cmd.OutOrStdout(), resp, output.ParseFieldsFlag(flagFields))
	case output.ModeYAML:
		return output.RenderYAML(cmd.OutOrStdout(), resp, output.ParseFieldsFlag(flagFields))
	default:
		tw := output.NewTabWriter(cmd.OutOrStdout())
		_, _ = fmt.Fprintln(tw, "HASH\tCOUNT\tLAST SEEN\tFIRST SEEN\tFIRST LINE")
		for _, e := range resp.Data {
			hash := e.ExceptionHash
			if len(hash) > 12 {
				hash = hash[:12]
			}
			_, _ = fmt.Fprintf(tw, "%s\t%d\t%s\t%s\t%s\n",
				hash, e.Count,
				e.LastSeen.Format("2006-01-02 15:04:05"),
				e.FirstSeen.Format("2006-01-02 15:04:05"),
				firstLine(e.StackTrace),
			)
		}
		return tw.Flush()
	}
}

func newExceptionsShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <hash>",
		Short: "Show a single exception group with its occurrences",
		Args:  cobra.ExactArgs(1),
		RunE:  runExceptionsShow,
	}
	addPaginationFlags(cmd)
	return cmd
}

func runExceptionsShow(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	mode := output.ResolveMode(flagOutput, output.StdoutIsTerminal())

	sess, err := loadSession()
	if err != nil {
		return renderSessionError(cmd.ErrOrStderr(), mode, err)
	}
	page := resolvePagination(cmd)
	page.PageSize = pickDefault(page.PageSize, 20) // detail uses 20 by default

	c := client.New(sess.URL, client.WithJWT(sess.JWT))
	resp, err := c.GetException(ctx, sess.ProjectID, args[0], page)
	if err != nil {
		return renderAPIError(cmd.ErrOrStderr(), mode, err, false)
	}

	switch mode {
	case output.ModeJSON:
		return output.RenderJSON(cmd.OutOrStdout(), resp, output.ParseFieldsFlag(flagFields))
	case output.ModeYAML:
		return output.RenderYAML(cmd.OutOrStdout(), resp, output.ParseFieldsFlag(flagFields))
	default:
		// Group header, then occurrences table.
		if resp.Group != nil {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(),
				"HASH:        %s\nCOUNT:       %d\nFIRST SEEN:  %s\nLAST SEEN:   %s\n\nSTACK TRACE:\n%s\n\nOCCURRENCES (%d):\n",
				resp.Group.ExceptionHash, resp.Group.Count,
				resp.Group.FirstSeen.Format("2006-01-02 15:04:05"),
				resp.Group.LastSeen.Format("2006-01-02 15:04:05"),
				resp.Group.StackTrace,
				len(resp.Occurrences),
			)
		}
		tw := output.NewTabWriter(cmd.OutOrStdout())
		_, _ = fmt.Fprintln(tw, "ID\tRECORDED AT\tSERVER\tTRACE TYPE")
		for _, occ := range resp.Occurrences {
			traceType := occ.TraceType
			if traceType == "" {
				traceType = "-"
			}
			_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
				occ.Id.String(),
				occ.RecordedAt.Format("2006-01-02 15:04:05"),
				pickStr(occ.ServerName, "-"),
				traceType,
			)
		}
		return tw.Flush()
	}
}

// firstLine returns the first line of s, useful for fitting a stack trace
// into a table column.
func firstLine(s string) string {
	for i, ch := range s {
		if ch == '\n' || ch == '\r' {
			return s[:i]
		}
	}
	return s
}

// pickStr returns alt if s is empty, else s. Saves a guard at every callsite.
func pickStr(s, alt string) string {
	if s == "" {
		return alt
	}
	return s
}

// pickDefault returns alt if v is zero, else v.
func pickDefault(v, alt int) int {
	if v == 0 {
		return alt
	}
	return v
}

// renderSessionError maps loadSession sentinel errors to envelopes.
func renderSessionError(errOut interface {
	Write([]byte) (int, error)
}, mode output.Mode, err error) error {
	switch {
	case errors.Is(err, errSessionNoProfile), errors.Is(err, errSessionNoJWT):
		_ = output.RenderError(errOut, mode, output.ErrorEnvelope{
			Code:     "not_authenticated",
			Message:  err.Error(),
			Hint:     "traceway login",
			ExitCode: exitcode.Auth,
		})
		lastExitCode = exitcode.Auth
		return errors.New("not_authenticated")
	case errors.Is(err, errSessionNoProject):
		_ = output.RenderError(errOut, mode, output.ErrorEnvelope{
			Code:     "no_project",
			Message:  err.Error(),
			Hint:     "traceway projects use <project-id> (or pass --project)",
			ExitCode: exitcode.Usage,
		})
		lastExitCode = exitcode.Usage
		return errors.New("no_project")
	}
	_ = output.RenderError(errOut, mode, output.ErrorEnvelope{
		Code: "internal", Message: err.Error(), ExitCode: exitcode.Generic,
	})
	lastExitCode = exitcode.Generic
	return errors.New("internal")
}

// renderTimeRangeError maps errInvalidTimeRange (from resolveTimeRange) to an envelope.
func renderTimeRangeError(errOut interface {
	Write([]byte) (int, error)
}, mode output.Mode, err error) error {
	_ = output.RenderError(errOut, mode, output.ErrorEnvelope{
		Code:     "invalid_time_range",
		Message:  err.Error(),
		Hint:     "use --since DURATION (e.g. 1h, 24h) or --from RFC3339 --to RFC3339",
		ExitCode: exitcode.Usage,
	})
	lastExitCode = exitcode.Usage
	return errors.New("invalid_time_range")
}
```

- [ ] **Step 4: Run the tests, verify they pass**

Run: `nix develop --command go test ./cmd/traceway/... -count=1`
Expected: PASS for the new exceptions tests (and all existing tests still pass).

- [ ] **Step 5: Commit**

```bash
git add cmd/traceway/exceptions.go cmd/traceway/exceptions_test.go
git commit -m "feat(cmd): exceptions list and exceptions show <hash>"
```

---

## Task 9: pkg/client — logs query

**Files:**
- Create: `pkg/client/logs.go`
- Test: `pkg/client/logs_test.go`

- [ ] **Step 1: Write the failing tests**

Create `pkg/client/logs_test.go`:

```go
package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestQueryLogs_basic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/logs" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("projectId") != "proj-1" {
			t.Errorf("projectId = %q", r.URL.Query().Get("projectId"))
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["fromDate"] == nil || body["toDate"] == nil {
			t.Errorf("body missing fromDate/toDate: %v", body)
		}
		_, _ = w.Write([]byte(`{
			"data":[
				{"id":"00000000-0000-0000-0000-000000000001","timestamp":"2026-05-13T12:00:00Z","severityText":"ERROR","severityNumber":17,"serviceName":"api","body":"boom"}
			],
			"pagination":{"page":0,"pageSize":50,"total":1,"totalPages":1}
		}`))
	}))
	defer srv.Close()

	c := New(srv.URL, WithJWT("tok"))
	resp, err := c.QueryLogs(context.Background(), "proj-1", QueryLogsRequest{
		TimeRange:   TimeRange{From: time.Now().Add(-time.Hour), To: time.Now()},
		Pagination:  PaginationParams{Page: 0, PageSize: 50},
		ServiceName: "api",
	})
	if err != nil {
		t.Fatalf("QueryLogs: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("got %d logs", len(resp.Data))
	}
	if resp.Data[0].ServiceName != "api" {
		t.Errorf("ServiceName = %q", resp.Data[0].ServiceName)
	}
	if resp.Data[0].Body != "boom" {
		t.Errorf("Body = %q", resp.Data[0].Body)
	}
}

func TestQueryLogs_passesFilters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["serviceName"] != "api" {
			t.Errorf("serviceName = %v", body["serviceName"])
		}
		if int(body["minSeverity"].(float64)) != 13 { // WARN per OTel severity numbers
			t.Errorf("minSeverity = %v", body["minSeverity"])
		}
		if body["traceId"] != "abc123" {
			t.Errorf("traceId = %v", body["traceId"])
		}
		_, _ = w.Write([]byte(`{"data":[],"pagination":{}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, WithJWT("tok"))
	_, err := c.QueryLogs(context.Background(), "proj-1", QueryLogsRequest{
		ServiceName: "api",
		MinSeverity: 13,
		TraceId:     "abc123",
	})
	if err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Run the tests, verify they fail**

Run: `nix develop --command go test ./pkg/client/... -run Logs`
Expected: build failure.

- [ ] **Step 3: Implement logs.go**

Create `pkg/client/logs.go`:

```go
package client

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
)

// LogRecord matches the upstream models.LogRecord (subset — we drop fields
// we don't surface in v1, like resource/scope schema URLs).
type LogRecord struct {
	Id                 uuid.UUID         `json:"id"`
	Timestamp          time.Time         `json:"timestamp"`
	SeverityText       string            `json:"severityText"`
	SeverityNumber     uint8             `json:"severityNumber"`
	ServiceName        string            `json:"serviceName"`
	Body               string            `json:"body"`
	TraceId            string            `json:"traceId,omitempty"`
	SpanId             string            `json:"spanId,omitempty"`
	ResourceAttributes map[string]string `json:"resourceAttributes,omitempty"`
	ScopeName          string            `json:"scopeName,omitempty"`
	LogAttributes      map[string]string `json:"logAttributes,omitempty"`
}

// QueryLogsRequest is the body for POST /api/logs.
type QueryLogsRequest struct {
	TimeRange     TimeRange        `json:"-"`
	Pagination    PaginationParams `json:"pagination"`
	OrderBy       string           `json:"orderBy,omitempty"`
	SortDirection string           `json:"sortDirection,omitempty"`
	Search        string           `json:"search,omitempty"`
	SearchType    string           `json:"searchType,omitempty"`
	MinSeverity   uint8            `json:"minSeverity,omitempty"`
	ServiceName   string           `json:"serviceName,omitempty"`
	TraceId       string           `json:"traceId,omitempty"`
}

// MarshalJSON expands TimeRange into top-level fromDate/toDate.
func (r QueryLogsRequest) MarshalJSON() ([]byte, error) {
	type alias QueryLogsRequest
	wire := struct {
		FromDate time.Time `json:"fromDate"`
		ToDate   time.Time `json:"toDate"`
		alias
	}{r.TimeRange.From, r.TimeRange.To, alias(r)}
	return jsonMarshalNoHTMLEscape(wire)
}

// QueryLogsResponse mirrors the upstream PaginatedResponse[LogRecord].
type QueryLogsResponse struct {
	Data       []LogRecord `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// QueryLogs returns one page of log records for the given project and filters.
func (c *Client) QueryLogs(ctx context.Context, projectID string, req QueryLogsRequest) (*QueryLogsResponse, error) {
	path := "/api/logs?projectId=" + url.QueryEscape(projectID)
	var resp QueryLogsResponse
	if err := c.do(ctx, http.MethodPost, path, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
```

- [ ] **Step 4: Run the tests, verify they pass**

Run: `nix develop --command go test ./pkg/client/... -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/client/logs.go pkg/client/logs_test.go
git commit -m "feat(client): LogRecord, QueryLogsRequest/Response, QueryLogs"
```

---

## Task 10: cmd/traceway — logs query

**Files:**
- Modify: `cmd/traceway/logs.go` (currently a hidden stub from Plan 1 — replace it)
- Create: `cmd/traceway/logs_test.go`
- Modify: `cmd/traceway/root.go` (register the command)

The `logs` stub doesn't currently exist — Plan 1 only stubbed login/logout/profiles/projects. We need to add `cmd.AddCommand(newLogsCmd())` to root.go and create the file fresh.

- [ ] **Step 1: Write the failing tests**

Create `cmd/traceway/logs_test.go`:

```go
package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLogsQuery_basic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/logs" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{
			"data":[
				{"id":"00000000-0000-0000-0000-000000000001","timestamp":"2026-05-13T12:00:00Z","severityText":"ERROR","severityNumber":17,"serviceName":"api","body":"failed to connect"}
			],
			"pagination":{"total":1}
		}`))
	}))
	defer srv.Close()
	seedSessionFor(t, srv.URL)

	stdout, _, err := runCmd(t, "", "logs", "query", "--output", "json")
	if err != nil {
		t.Fatalf("logs query: %v", err)
	}
	if !strings.Contains(stdout.String(), "failed to connect") {
		t.Errorf("expected log body in output: %s", stdout.String())
	}
}

func TestLogsQuery_table(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"data":[
				{"id":"00000000-0000-0000-0000-000000000001","timestamp":"2026-05-13T12:00:00Z","severityText":"ERROR","severityNumber":17,"serviceName":"api","body":"failed to connect"}
			],
			"pagination":{"total":1}
		}`))
	}))
	defer srv.Close()
	seedSessionFor(t, srv.URL)

	stdout, _, err := runCmd(t, "", "logs", "query", "--output", "table")
	if err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	if !strings.Contains(out, "TIMESTAMP") || !strings.Contains(out, "SEVERITY") || !strings.Contains(out, "SERVICE") {
		t.Errorf("table missing headers: %s", out)
	}
	if !strings.Contains(out, "ERROR") || !strings.Contains(out, "api") || !strings.Contains(out, "failed") {
		t.Errorf("table missing row data: %s", out)
	}
}

func TestLogsQuery_passesServiceFilter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Decode body and assert serviceName was passed through
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		body := string(buf[:n])
		if !strings.Contains(body, `"serviceName":"api"`) {
			t.Errorf("expected serviceName=api in body, got: %s", body)
		}
		_, _ = w.Write([]byte(`{"data":[],"pagination":{}}`))
	}))
	defer srv.Close()
	seedSessionFor(t, srv.URL)

	if _, _, err := runCmd(t, "", "logs", "query", "--service", "api"); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Run the tests, verify they fail**

Run: `nix develop --command go test ./cmd/traceway/... -run Logs`
Expected: failure — newLogsCmd is not registered.

- [ ] **Step 3: Implement logs.go**

Create `cmd/traceway/logs.go`:

```go
package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tracewayapp/traceway/cli/internal/output"
	"github.com/tracewayapp/traceway/cli/pkg/client"
)

func newLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Query log records",
	}
	cmd.AddCommand(newLogsQueryCmd())
	return cmd
}

func newLogsQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query logs by time, service, severity, or trace",
		RunE:  runLogsQuery,
	}
	addTimeRangeFlags(cmd)
	addPaginationFlags(cmd)
	cmd.Flags().String("service", "", "Filter by service name")
	cmd.Flags().Uint8("min-severity", 0, "Minimum OTel severity number (1=TRACE, 5=DEBUG, 9=INFO, 13=WARN, 17=ERROR, 21=FATAL)")
	cmd.Flags().String("trace-id", "", "Filter to a specific OpenTelemetry trace ID")
	cmd.Flags().String("search", "", "Free-text search in body")
	cmd.Flags().String("search-type", "body", "Search type: body or attribute")
	cmd.Flags().String("order-by", "timestamp", "Sort field")
	cmd.Flags().String("sort-direction", "desc", "Sort direction: asc or desc")
	return cmd
}

func runLogsQuery(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	mode := output.ResolveMode(flagOutput, output.StdoutIsTerminal())

	sess, err := loadSession()
	if err != nil {
		return renderSessionError(cmd.ErrOrStderr(), mode, err)
	}
	tr, err := resolveTimeRange(cmd)
	if err != nil {
		return renderTimeRangeError(cmd.ErrOrStderr(), mode, err)
	}
	page := resolvePagination(cmd)
	service, _ := cmd.Flags().GetString("service")
	minSev, _ := cmd.Flags().GetUint8("min-severity")
	traceID, _ := cmd.Flags().GetString("trace-id")
	search, _ := cmd.Flags().GetString("search")
	searchType, _ := cmd.Flags().GetString("search-type")
	orderBy, _ := cmd.Flags().GetString("order-by")
	sortDir, _ := cmd.Flags().GetString("sort-direction")

	c := client.New(sess.URL, client.WithJWT(sess.JWT))
	resp, err := c.QueryLogs(ctx, sess.ProjectID, client.QueryLogsRequest{
		TimeRange:     tr,
		Pagination:    page,
		ServiceName:   service,
		MinSeverity:   minSev,
		TraceId:       traceID,
		Search:        search,
		SearchType:    searchType,
		OrderBy:       orderBy,
		SortDirection: sortDir,
	})
	if err != nil {
		return renderAPIError(cmd.ErrOrStderr(), mode, err, false)
	}

	switch mode {
	case output.ModeJSON:
		return output.RenderJSON(cmd.OutOrStdout(), resp, output.ParseFieldsFlag(flagFields))
	case output.ModeYAML:
		return output.RenderYAML(cmd.OutOrStdout(), resp, output.ParseFieldsFlag(flagFields))
	default:
		tw := output.NewTabWriter(cmd.OutOrStdout())
		_, _ = fmt.Fprintln(tw, "TIMESTAMP\tSEVERITY\tSERVICE\tBODY")
		for _, l := range resp.Data {
			_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
				l.Timestamp.Format("2006-01-02 15:04:05"),
				pickStr(l.SeverityText, "-"),
				pickStr(l.ServiceName, "-"),
				firstLine(l.Body),
			)
		}
		return tw.Flush()
	}
}
```

- [ ] **Step 4: Register the command in root.go**

Open `cmd/traceway/root.go` and add `cmd.AddCommand(newLogsCmd())` to the existing `AddCommand` block (after `newProjectsCmd()`).

- [ ] **Step 5: Run the tests, verify they pass**

Run: `nix develop --command go test ./cmd/traceway/... -count=1`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add cmd/traceway/logs.go cmd/traceway/logs_test.go cmd/traceway/root.go
git commit -m "feat(cmd): logs query command"
```

---

## Task 11: pkg/client — endpoints list (using /endpoints/grouped)

**Files:**
- Create: `pkg/client/endpoints.go`
- Test: `pkg/client/endpoints_test.go`

We use `/api/endpoints/grouped` (returns per-endpoint p50/p95/p99 stats) instead of `/api/endpoints` (returns individual request rows). The grouped view is what answers use case #2.

- [ ] **Step 1: Write the failing tests**

Create `pkg/client/endpoints_test.go`:

```go
package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestListEndpoints_callsGroupedRoute(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/endpoints/grouped" {
			t.Errorf("path = %q (want grouped)", r.URL.Path)
		}
		if r.URL.Query().Get("projectId") != "proj-1" {
			t.Errorf("projectId = %q", r.URL.Query().Get("projectId"))
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["fromDate"] == nil || body["toDate"] == nil {
			t.Errorf("body missing fromDate/toDate: %v", body)
		}
		// p50/p95/p99 are time.Duration in upstream — JSON encoded as nanoseconds (int64)
		_, _ = w.Write([]byte(`{
			"data":[
				{"endpoint":"GET /api/projects","count":120,"p50Duration":50000000,"p95Duration":150000000,"p99Duration":300000000,"avgDuration":80000000,"lastSeen":"2026-05-13T12:00:00Z","impact":0.42,"impactReason":"high p95"}
			],
			"pagination":{"page":0,"pageSize":50,"total":1,"totalPages":1}
		}`))
	}))
	defer srv.Close()

	c := New(srv.URL, WithJWT("tok"))
	resp, err := c.ListEndpoints(context.Background(), "proj-1", ListEndpointsRequest{
		TimeRange:  TimeRange{From: time.Now().Add(-time.Hour), To: time.Now()},
		Pagination: PaginationParams{Page: 0, PageSize: 50},
	})
	if err != nil {
		t.Fatalf("ListEndpoints: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("got %d endpoints", len(resp.Data))
	}
	if resp.Data[0].Endpoint != "GET /api/projects" {
		t.Errorf("Endpoint = %q", resp.Data[0].Endpoint)
	}
	if resp.Data[0].Count != 120 {
		t.Errorf("Count = %d", resp.Data[0].Count)
	}
	if resp.Data[0].P50Duration != 50*time.Millisecond {
		t.Errorf("P50Duration = %v, want 50ms", resp.Data[0].P50Duration)
	}
	if resp.Data[0].Impact != 0.42 {
		t.Errorf("Impact = %v", resp.Data[0].Impact)
	}
}
```

- [ ] **Step 2: Run the tests, verify they fail**

Run: `nix develop --command go test ./pkg/client/... -run Endpoints`
Expected: build failure.

- [ ] **Step 3: Implement endpoints.go**

Create `pkg/client/endpoints.go`:

```go
package client

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// EndpointStats matches the upstream models.EndpointStats. Durations are
// time.Duration values which Go marshals/unmarshals as nanoseconds.
type EndpointStats struct {
	Endpoint     string        `json:"endpoint"`
	Count        uint64        `json:"count"`
	P50Duration  time.Duration `json:"p50Duration"`
	P95Duration  time.Duration `json:"p95Duration"`
	P99Duration  time.Duration `json:"p99Duration"`
	AvgDuration  time.Duration `json:"avgDuration"`
	LastSeen     time.Time     `json:"lastSeen"`
	Impact       float64       `json:"impact"`
	ImpactReason string        `json:"impactReason"`
}

// ListEndpointsRequest is the body for POST /api/endpoints/grouped.
type ListEndpointsRequest struct {
	TimeRange     TimeRange        `json:"-"`
	Pagination    PaginationParams `json:"pagination"`
	OrderBy       string           `json:"orderBy,omitempty"`
	SortDirection string           `json:"sortDirection,omitempty"`
	Search        string           `json:"search,omitempty"`
}

// MarshalJSON expands TimeRange into top-level fromDate/toDate.
func (r ListEndpointsRequest) MarshalJSON() ([]byte, error) {
	type alias ListEndpointsRequest
	wire := struct {
		FromDate time.Time `json:"fromDate"`
		ToDate   time.Time `json:"toDate"`
		alias
	}{r.TimeRange.From, r.TimeRange.To, alias(r)}
	return jsonMarshalNoHTMLEscape(wire)
}

// ListEndpointsResponse mirrors PaginatedResponse[EndpointStats].
type ListEndpointsResponse struct {
	Data       []EndpointStats `json:"data"`
	Pagination Pagination      `json:"pagination"`
}

// ListEndpoints returns p50/p95/p99 stats grouped by endpoint route. We use
// the /grouped variant rather than the bare /endpoints (which returns one row
// per request).
func (c *Client) ListEndpoints(ctx context.Context, projectID string, req ListEndpointsRequest) (*ListEndpointsResponse, error) {
	path := "/api/endpoints/grouped?projectId=" + url.QueryEscape(projectID)
	var resp ListEndpointsResponse
	if err := c.do(ctx, http.MethodPost, path, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
```

- [ ] **Step 4: Run the tests, verify they pass**

Run: `nix develop --command go test ./pkg/client/... -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/client/endpoints.go pkg/client/endpoints_test.go
git commit -m "feat(client): EndpointStats, ListEndpointsRequest/Response, ListEndpoints"
```

---

## Task 12: cmd/traceway — endpoints list

**Files:**
- Create: `cmd/traceway/endpoints.go`
- Create: `cmd/traceway/endpoints_test.go`
- Modify: `cmd/traceway/root.go` (register newEndpointsCmd)

- [ ] **Step 1: Write the failing tests**

Create `cmd/traceway/endpoints_test.go`:

```go
package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEndpointsList_table(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"data":[
				{"endpoint":"GET /api/projects","count":120,"p50Duration":50000000,"p95Duration":150000000,"p99Duration":300000000,"avgDuration":80000000,"lastSeen":"2026-05-13T12:00:00Z","impact":0.42,"impactReason":"high p95"}
			],
			"pagination":{"total":1}
		}`))
	}))
	defer srv.Close()
	seedSessionFor(t, srv.URL)

	stdout, _, err := runCmd(t, "", "endpoints", "list", "--output", "table")
	if err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	if !strings.Contains(out, "ENDPOINT") || !strings.Contains(out, "P50") || !strings.Contains(out, "P95") || !strings.Contains(out, "P99") {
		t.Errorf("table missing headers: %s", out)
	}
	if !strings.Contains(out, "/api/projects") || !strings.Contains(out, "120") {
		t.Errorf("table missing row data: %s", out)
	}
	// Latency should be human-formatted (50ms, not 50000000)
	if strings.Contains(out, "50000000") {
		t.Errorf("table should format ns as human duration: %s", out)
	}
}

func TestEndpointsList_jsonShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"endpoint":"GET /","count":1}],"pagination":{}}`))
	}))
	defer srv.Close()
	seedSessionFor(t, srv.URL)

	stdout, _, err := runCmd(t, "", "endpoints", "list", "--output", "json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), `"GET /"`) {
		t.Errorf("missing endpoint in JSON: %s", stdout.String())
	}
}
```

- [ ] **Step 2: Run the tests, verify they fail**

Run: `nix develop --command go test ./cmd/traceway/... -run Endpoints`
Expected: failure (no endpoints command registered).

- [ ] **Step 3: Implement endpoints.go**

Create `cmd/traceway/endpoints.go`:

```go
package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/tracewayapp/traceway/cli/internal/output"
	"github.com/tracewayapp/traceway/cli/pkg/client"
)

func newEndpointsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "endpoints",
		Short: "Query HTTP endpoint performance",
	}
	cmd.AddCommand(newEndpointsListCmd())
	return cmd
}

func newEndpointsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List endpoints with p50/p95/p99 latency stats",
		RunE:  runEndpointsList,
	}
	addTimeRangeFlags(cmd)
	addPaginationFlags(cmd)
	cmd.Flags().String("search", "", "Free-text search filter for endpoint names")
	cmd.Flags().String("order-by", "impact", "Sort field (impact, count, p95, lastSeen)")
	cmd.Flags().String("sort-direction", "desc", "Sort direction: asc or desc")
	return cmd
}

func runEndpointsList(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	mode := output.ResolveMode(flagOutput, output.StdoutIsTerminal())

	sess, err := loadSession()
	if err != nil {
		return renderSessionError(cmd.ErrOrStderr(), mode, err)
	}
	tr, err := resolveTimeRange(cmd)
	if err != nil {
		return renderTimeRangeError(cmd.ErrOrStderr(), mode, err)
	}
	page := resolvePagination(cmd)
	search, _ := cmd.Flags().GetString("search")
	orderBy, _ := cmd.Flags().GetString("order-by")
	sortDir, _ := cmd.Flags().GetString("sort-direction")

	c := client.New(sess.URL, client.WithJWT(sess.JWT))
	resp, err := c.ListEndpoints(ctx, sess.ProjectID, client.ListEndpointsRequest{
		TimeRange:     tr,
		Pagination:    page,
		Search:        search,
		OrderBy:       orderBy,
		SortDirection: sortDir,
	})
	if err != nil {
		return renderAPIError(cmd.ErrOrStderr(), mode, err, false)
	}

	switch mode {
	case output.ModeJSON:
		return output.RenderJSON(cmd.OutOrStdout(), resp, output.ParseFieldsFlag(flagFields))
	case output.ModeYAML:
		return output.RenderYAML(cmd.OutOrStdout(), resp, output.ParseFieldsFlag(flagFields))
	default:
		tw := output.NewTabWriter(cmd.OutOrStdout())
		_, _ = fmt.Fprintln(tw, "ENDPOINT\tCOUNT\tP50\tP95\tP99\tIMPACT\tLAST SEEN")
		for _, e := range resp.Data {
			_, _ = fmt.Fprintf(tw, "%s\t%d\t%s\t%s\t%s\t%.2f\t%s\n",
				e.Endpoint, e.Count,
				formatDuration(e.P50Duration),
				formatDuration(e.P95Duration),
				formatDuration(e.P99Duration),
				e.Impact,
				e.LastSeen.Format("2006-01-02 15:04:05"),
			)
		}
		return tw.Flush()
	}
}

// formatDuration renders a Duration as a human-readable string.
// time.Duration's String() does this already (e.g. "50ms"); we just shorten
// for very small values.
func formatDuration(d time.Duration) string {
	if d == 0 {
		return "0"
	}
	return d.String()
}
```

- [ ] **Step 4: Register the command**

In `cmd/traceway/root.go`, add `cmd.AddCommand(newEndpointsCmd())` to the AddCommand block.

- [ ] **Step 5: Run the tests, verify they pass**

Run: `nix develop --command go test ./cmd/traceway/... -count=1`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add cmd/traceway/endpoints.go cmd/traceway/endpoints_test.go cmd/traceway/root.go
git commit -m "feat(cmd): endpoints list command"
```

---

## Task 13: pkg/client — metrics query

**Files:**
- Create: `pkg/client/metrics.go`
- Test: `pkg/client/metrics_test.go`

Metrics has a different shape: it uses `from`/`to` (not `fromDate`/`toDate`), takes a `queries` array (we expose only single-query in v1), and returns `{results: [...]}` (no pagination wrapper).

- [ ] **Step 1: Write the failing tests**

Create `pkg/client/metrics_test.go`:

```go
package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestQueryMetrics_singleQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/metrics/query" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("projectId") != "proj-1" {
			t.Errorf("projectId = %q", r.URL.Query().Get("projectId"))
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		// metrics uses 'from'/'to', not fromDate/toDate
		if body["from"] == nil || body["to"] == nil {
			t.Errorf("body missing from/to: %v", body)
		}
		queries, _ := body["queries"].([]any)
		if len(queries) != 1 {
			t.Fatalf("expected 1 query, got %d", len(queries))
		}
		q := queries[0].(map[string]any)
		if q["name"] != "http.request.duration" {
			t.Errorf("query name = %v", q["name"])
		}
		if q["aggregation"] != "p95" {
			t.Errorf("aggregation = %v", q["aggregation"])
		}
		_, _ = w.Write([]byte(`{
			"results":[
				{
					"name":"http.request.duration",
					"unit":"ms",
					"series":{
						"all":[
							{"timestamp":"2026-05-13T12:00:00Z","value":42.5},
							{"timestamp":"2026-05-13T12:05:00Z","value":47.0}
						]
					}
				}
			]
		}`))
	}))
	defer srv.Close()

	c := New(srv.URL, WithJWT("tok"))
	resp, err := c.QueryMetrics(context.Background(), "proj-1", QueryMetricsRequest{
		TimeRange:       TimeRange{From: time.Now().Add(-time.Hour), To: time.Now()},
		IntervalMinutes: 5,
		Queries: []MetricQueryItem{
			{Name: "http.request.duration", Aggregation: "p95"},
		},
	})
	if err != nil {
		t.Fatalf("QueryMetrics: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("got %d results", len(resp.Results))
	}
	r := resp.Results[0]
	if r.Name != "http.request.duration" {
		t.Errorf("Name = %q", r.Name)
	}
	if r.Unit != "ms" {
		t.Errorf("Unit = %q", r.Unit)
	}
	series, ok := r.Series["all"]
	if !ok {
		t.Fatal(`series["all"] missing`)
	}
	if len(series) != 2 {
		t.Errorf("got %d points, want 2", len(series))
	}
	if series[0].Value != 42.5 {
		t.Errorf("series[0].Value = %v", series[0].Value)
	}
}

func TestQueryMetrics_multiQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		queries, _ := body["queries"].([]any)
		if len(queries) != 2 {
			t.Errorf("expected 2 queries, got %d", len(queries))
		}
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer srv.Close()

	c := New(srv.URL, WithJWT("tok"))
	_, err := c.QueryMetrics(context.Background(), "proj-1", QueryMetricsRequest{
		Queries: []MetricQueryItem{
			{Name: "metric.a", Aggregation: "avg"},
			{Name: "metric.b", Aggregation: "sum"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Run the tests, verify they fail**

Run: `nix develop --command go test ./pkg/client/... -run Metrics`
Expected: build failure.

- [ ] **Step 3: Implement metrics.go**

Create `pkg/client/metrics.go`:

```go
package client

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// MetricQueryItem is one query within a QueryMetricsRequest.
type MetricQueryItem struct {
	Name        string            `json:"name"`
	Aggregation string            `json:"aggregation,omitempty"`
	TagFilters  map[string]string `json:"tagFilters,omitempty"`
	GroupBy     string            `json:"groupBy,omitempty"`
}

// QueryMetricsRequest is the body for POST /api/metrics/query.
//
// Note: metrics uses `from`/`to` (NOT fromDate/toDate like the other endpoints)
// and has no pagination — results are time-bucketed via IntervalMinutes.
type QueryMetricsRequest struct {
	TimeRange       TimeRange         `json:"-"`
	IntervalMinutes int               `json:"intervalMinutes,omitempty"`
	Queries         []MetricQueryItem `json:"queries"`
}

// MarshalJSON expands TimeRange into top-level from/to (NOT fromDate/toDate).
func (r QueryMetricsRequest) MarshalJSON() ([]byte, error) {
	type alias QueryMetricsRequest
	wire := struct {
		From time.Time `json:"from"`
		To   time.Time `json:"to"`
		alias
	}{r.TimeRange.From, r.TimeRange.To, alias(r)}
	return jsonMarshalNoHTMLEscape(wire)
}

// TimeSeriesPoint is one data point in a metric query result.
type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// MetricQueryResult is one query's results, optionally grouped by tag.
// The map key is the group label ("all" if no GroupBy was specified).
type MetricQueryResult struct {
	Name   string                       `json:"name"`
	Unit   string                       `json:"unit"`
	Series map[string][]TimeSeriesPoint `json:"series"`
}

// QueryMetricsResponse is the upstream MetricQueryResponse.
type QueryMetricsResponse struct {
	Results []MetricQueryResult `json:"results"`
}

// QueryMetrics runs one or more metric queries against the project.
func (c *Client) QueryMetrics(ctx context.Context, projectID string, req QueryMetricsRequest) (*QueryMetricsResponse, error) {
	path := "/api/metrics/query?projectId=" + url.QueryEscape(projectID)
	var resp QueryMetricsResponse
	if err := c.do(ctx, http.MethodPost, path, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
```

- [ ] **Step 4: Run the tests, verify they pass**

Run: `nix develop --command go test ./pkg/client/... -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/client/metrics.go pkg/client/metrics_test.go
git commit -m "feat(client): MetricQueryItem, QueryMetricsRequest/Response, QueryMetrics"
```

---

## Task 14: cmd/traceway — metrics query

**Files:**
- Create: `cmd/traceway/metrics.go`
- Create: `cmd/traceway/metrics_test.go`
- Modify: `cmd/traceway/root.go` (register newMetricsCmd)

- [ ] **Step 1: Write the failing tests**

Create `cmd/traceway/metrics_test.go`:

```go
package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsQuery_jsonOutput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"results":[
				{"name":"http.request.duration","unit":"ms","series":{"all":[{"timestamp":"2026-05-13T12:00:00Z","value":42.5}]}}
			]
		}`))
	}))
	defer srv.Close()
	seedSessionFor(t, srv.URL)

	stdout, _, err := runCmd(t, "", "metrics", "query", "--name", "http.request.duration", "--aggregation", "p95", "--output", "json")
	if err != nil {
		t.Fatalf("metrics query: %v", err)
	}
	if !strings.Contains(stdout.String(), "http.request.duration") {
		t.Errorf("expected metric name in output: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "42.5") {
		t.Errorf("expected value in output: %s", stdout.String())
	}
}

func TestMetricsQuery_table(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"results":[
				{"name":"http.request.duration","unit":"ms","series":{
					"all":[
						{"timestamp":"2026-05-13T12:00:00Z","value":42.5},
						{"timestamp":"2026-05-13T12:05:00Z","value":47.0}
					]
				}}
			]
		}`))
	}))
	defer srv.Close()
	seedSessionFor(t, srv.URL)

	stdout, _, err := runCmd(t, "", "metrics", "query", "--name", "http.request.duration")
	if err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	if !strings.Contains(out, "METRIC") || !strings.Contains(out, "GROUP") || !strings.Contains(out, "POINTS") {
		t.Errorf("table missing headers: %s", out)
	}
	if !strings.Contains(out, "http.request.duration") {
		t.Errorf("table missing metric name: %s", out)
	}
	if !strings.Contains(out, "47") {
		t.Errorf("table missing latest value: %s", out)
	}
}

func TestMetricsQuery_requiresName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer srv.Close()
	seedSessionFor(t, srv.URL)

	_, stderr, err := runCmd(t, "", "metrics", "query", "--output", "json")
	if err == nil {
		t.Fatal("expected --name to be required")
	}
	if !strings.Contains(stderr.String(), `"usage_error"`) {
		t.Errorf("expected usage_error envelope, got: %s", stderr.String())
	}
}
```

- [ ] **Step 2: Run the tests, verify they fail**

Run: `nix develop --command go test ./cmd/traceway/... -run Metrics`
Expected: failure.

- [ ] **Step 3: Implement metrics.go**

Create `cmd/traceway/metrics.go`:

```go
package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tracewayapp/traceway/cli/internal/exitcode"
	"github.com/tracewayapp/traceway/cli/internal/output"
	"github.com/tracewayapp/traceway/cli/pkg/client"
)

func newMetricsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metrics",
		Short: "Query metric time series",
	}
	cmd.AddCommand(newMetricsQueryCmd())
	return cmd
}

func newMetricsQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query a single metric over time",
		RunE:  runMetricsQuery,
	}
	addTimeRangeFlags(cmd)
	cmd.Flags().String("name", "", "Metric name (required)")
	cmd.Flags().String("aggregation", "avg", "Aggregation: avg, sum, count, min, max, p50, p95, p99")
	cmd.Flags().StringSlice("tag", nil, "Tag filter as key=value (repeatable)")
	cmd.Flags().String("group-by", "", "Tag to group series by")
	cmd.Flags().Int("interval-minutes", 0, "Time bucket size in minutes (0 = auto)")
	return cmd
}

func runMetricsQuery(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	mode := output.ResolveMode(flagOutput, output.StdoutIsTerminal())

	sess, err := loadSession()
	if err != nil {
		return renderSessionError(cmd.ErrOrStderr(), mode, err)
	}
	tr, err := resolveTimeRange(cmd)
	if err != nil {
		return renderTimeRangeError(cmd.ErrOrStderr(), mode, err)
	}

	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		_ = output.RenderError(cmd.ErrOrStderr(), mode, output.ErrorEnvelope{
			Code: "usage_error", Message: "--name is required",
			Hint: "traceway metrics query --name <metric-name>", ExitCode: exitcode.Usage,
		})
		lastExitCode = exitcode.Usage
		return errors.New("usage_error")
	}
	agg, _ := cmd.Flags().GetString("aggregation")
	groupBy, _ := cmd.Flags().GetString("group-by")
	intervalMin, _ := cmd.Flags().GetInt("interval-minutes")
	tags, _ := cmd.Flags().GetStringSlice("tag")

	tagFilters, err := parseTagFilters(tags)
	if err != nil {
		_ = output.RenderError(cmd.ErrOrStderr(), mode, output.ErrorEnvelope{
			Code: "usage_error", Message: err.Error(),
			Hint: "use --tag key=value (repeatable)", ExitCode: exitcode.Usage,
		})
		lastExitCode = exitcode.Usage
		return errors.New("usage_error")
	}

	c := client.New(sess.URL, client.WithJWT(sess.JWT))
	resp, err := c.QueryMetrics(ctx, sess.ProjectID, client.QueryMetricsRequest{
		TimeRange:       tr,
		IntervalMinutes: intervalMin,
		Queries: []client.MetricQueryItem{
			{Name: name, Aggregation: agg, TagFilters: tagFilters, GroupBy: groupBy},
		},
	})
	if err != nil {
		return renderAPIError(cmd.ErrOrStderr(), mode, err, false)
	}

	switch mode {
	case output.ModeJSON:
		return output.RenderJSON(cmd.OutOrStdout(), resp, output.ParseFieldsFlag(flagFields))
	case output.ModeYAML:
		return output.RenderYAML(cmd.OutOrStdout(), resp, output.ParseFieldsFlag(flagFields))
	default:
		// Summary table — for actual time-series data, --output json is recommended.
		tw := output.NewTabWriter(cmd.OutOrStdout())
		_, _ = fmt.Fprintln(tw, "METRIC\tUNIT\tGROUP\tPOINTS\tLATEST")
		for _, r := range resp.Results {
			if len(r.Series) == 0 {
				_, _ = fmt.Fprintf(tw, "%s\t%s\t-\t0\t-\n", r.Name, pickStr(r.Unit, "-"))
				continue
			}
			for group, pts := range r.Series {
				latest := "-"
				if len(pts) > 0 {
					latest = fmt.Sprintf("%g", pts[len(pts)-1].Value)
				}
				_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%s\n", r.Name, pickStr(r.Unit, "-"), group, len(pts), latest)
			}
		}
		return tw.Flush()
	}
}

// parseTagFilters parses ["k=v", "x=y"] into {"k":"v","x":"y"}. Returns an
// error if any element is malformed.
func parseTagFilters(in []string) (map[string]string, error) {
	if len(in) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(in))
	for _, item := range in {
		k, v, ok := strings.Cut(item, "=")
		if !ok || k == "" {
			return nil, fmt.Errorf("invalid --tag %q: expected key=value", item)
		}
		out[k] = v
	}
	return out, nil
}
```

- [ ] **Step 4: Register the command**

In `cmd/traceway/root.go`, add `cmd.AddCommand(newMetricsCmd())` to the AddCommand block.

- [ ] **Step 5: Run the tests, verify they pass**

Run: `nix develop --command go test ./cmd/traceway/... -count=1`
Expected: PASS.

- [ ] **Step 6: Final full-suite check**

Run: `nix develop --command go test ./... -count=1 && nix develop --command go vet ./... && nix develop --command just build`
Expected: all tests pass, vet clean, binary builds.

Try the binary:
```bash
nix develop --command ./bin/traceway --help
nix develop --command ./bin/traceway exceptions --help
nix develop --command ./bin/traceway logs --help
nix develop --command ./bin/traceway endpoints --help
nix develop --command ./bin/traceway metrics --help
```
Expected: each shows the right subcommands and flags.

- [ ] **Step 7: Commit**

```bash
git add cmd/traceway/metrics.go cmd/traceway/metrics_test.go cmd/traceway/root.go
git commit -m "feat(cmd): metrics query command"
```

---

## Self-Review (performed inline during writing)

**Spec coverage check:**

| Spec/use case | Implementing tasks |
|---|---|
| "What's broken in prod?" — exceptions list/show | Tasks 6–8 |
| "Why is endpoint X slow?" — endpoints list with p50/p95/p99 | Tasks 11–12 |
| "What did service Y log?" — logs query with service/severity/trace filters | Tasks 9–10 |
| "Anomalies?" — metrics query | Tasks 13–14 |
| Time range: --since OR --from/--to, default 1h | Task 4 |
| Pagination: --page-size N (default 50), --page N (default 0) | Tasks 1, 5 |
| Pass-through JSON shape with --fields projection | Reuses Plan 1's `output.RenderJSON` |
| Stable error envelope with new codes (no_project, invalid_time_range) | Task 8 (renderSessionError, renderTimeRangeError) |
| One HTTP call per command (no auto-pagination) | Tasks 6, 9, 11, 13 |

**Placeholder scan:** No "TBD"/"TODO" steps. The two notes about "we don't expose recordings in v1" and "session recording blob omitted" are explicit scope decisions, not deferred work.

**Type consistency:** Spot-checked key names across tasks — `ListExceptionsRequest`, `ListExceptionsResponse`, `GetExceptionResponse`, `QueryLogsRequest/Response`, `ListEndpointsRequest/Response`, `QueryMetricsRequest/Response`, `EndpointStats`, `LogRecord`, `ExceptionGroup`, `ExceptionStackTrace`, `MetricQueryItem`, `MetricQueryResult`, `TimeSeriesPoint`, `Pagination`, `PaginationParams`, `TimeRange`. All consistent.

The `jsonMarshalNoHTMLEscape` helper is added to `pkg/client/client.go` in Task 6 and reused by Tasks 9, 11, 13 — single source of truth.

The `loadSession`, `resolveTimeRange`, `resolvePagination`, `renderSessionError`, `renderTimeRangeError` helpers are introduced once (Tasks 3, 4, 5, 8) and reused by every subsequent command. Plan 1's `renderAPIError` (which we reuse without changes) handles the API-error mapping side.

---

## End-of-plan checkpoint

After Task 14, you should be able to:

```bash
just build
./bin/traceway exceptions list --since 1h
./bin/traceway exceptions show <hash> --output json --fields exceptionHash,count
./bin/traceway logs query --service api --since 30m --output json | jq '.data[].body'
./bin/traceway endpoints list --since 1h
./bin/traceway metrics query --name http.request.duration --aggregation p95 --since 1h --output json
```

End state: a working observability CLI for the four use cases approved during brainstorming. Plan 3 builds on this for mutations (`exceptions archive`/`resolve`), the smoke-test harness, and the README.
