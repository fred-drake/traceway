//go:build smoke

// Package smoke contains end-to-end tests that talk to a real Traceway
// instance via the built CLI binary. They are gated behind the "smoke"
// build tag so `go test ./...` skips them entirely.
//
// Run with: go test -tags smoke ./test/smoke/... (or `just smoke-test`).
//
// Required env vars (tests are skipped, not failed, if any are missing):
//
//	TRACEWAY_SMOKE_URL         e.g. https://traceway.stormwind.local
//	TRACEWAY_SMOKE_USERNAME    email used to log in
//	TRACEWAY_SMOKE_PASSWORD    password
//	TRACEWAY_SMOKE_PROJECT_ID  UUID of a project the user can access
package smoke
