package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStatePath_xdgSet(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/tmp/xdgstate")
	t.Setenv("HOME", "/tmp/home")

	got, err := statePath()
	if err != nil {
		t.Fatalf("statePath() error: %v", err)
	}
	want := filepath.Join("/tmp/xdgstate", "traceway", "state.json")
	if got != want {
		t.Errorf("statePath() = %q, want %q", got, want)
	}
}

func TestStatePath_xdgUnset(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", "/tmp/home")

	got, err := statePath()
	if err != nil {
		t.Fatalf("statePath() error: %v", err)
	}
	want := filepath.Join("/tmp/home", ".local", "state", "traceway", "state.json")
	if got != want {
		t.Errorf("statePath() = %q, want %q", got, want)
	}
}

func TestStatePath_neither(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", "")

	if _, err := statePath(); err == nil {
		t.Fatal("expected error when both XDG_STATE_HOME and HOME are empty")
	}
}

func TestLoad_missingFile_returnsEmpty(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	st, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if st == nil {
		t.Fatal("Load() returned nil state")
	}
	if len(st.Profiles) != 0 {
		t.Errorf("expected empty Profiles, got %v", st.Profiles)
	}
	if st.CurrentProfile != "" {
		t.Errorf("expected empty CurrentProfile, got %q", st.CurrentProfile)
	}
}

func TestLoad_existingFile_readsJSON(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)
	if err := os.MkdirAll(filepath.Join(dir, "traceway"), 0o700); err != nil {
		t.Fatal(err)
	}
	body := `{
		"current_profile": "stormwind",
		"profiles": {
			"stormwind": {
				"jwt": "abc.def.ghi",
				"current_project_id": "proj-1"
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(dir, "traceway", "state.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	st, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if st.CurrentProfile != "stormwind" {
		t.Errorf("CurrentProfile = %q, want %q", st.CurrentProfile, "stormwind")
	}
	p, ok := st.Profiles["stormwind"]
	if !ok {
		t.Fatal("stormwind profile not loaded")
	}
	if p.JWT != "abc.def.ghi" {
		t.Errorf("JWT = %q", p.JWT)
	}
	if p.CurrentProjectID != "proj-1" {
		t.Errorf("CurrentProjectID = %q", p.CurrentProjectID)
	}
}

func TestLoad_corruptFile_returnsError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)
	if err := os.MkdirAll(filepath.Join(dir, "traceway"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "traceway", "state.json"), []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(); err == nil {
		t.Fatal("expected Load() to fail on corrupt JSON")
	}
}

func TestSave_writesAtomicallyWith0600(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)

	st := &State{
		CurrentProfile: "default",
		Profiles: map[string]ProfileState{
			"default": {JWT: "tok", CurrentProjectID: "proj-1"},
		},
	}
	if err := st.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	path := filepath.Join(dir, "traceway", "state.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file perm = %o, want 0600", perm)
	}

	dirInfo, err := os.Stat(filepath.Join(dir, "traceway"))
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if perm := dirInfo.Mode().Perm(); perm != 0o700 {
		t.Errorf("dir perm = %o, want 0700", perm)
	}
}

func TestSave_thenLoad_roundTrips(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	want := &State{
		CurrentProfile: "stormwind",
		Profiles: map[string]ProfileState{
			"stormwind": {JWT: "tok", CurrentProjectID: "proj-1"},
		},
	}
	if err := want.Save(); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.CurrentProfile != want.CurrentProfile {
		t.Errorf("CurrentProfile mismatch: got %q want %q", got.CurrentProfile, want.CurrentProfile)
	}
	if got.Profiles["stormwind"] != want.Profiles["stormwind"] {
		t.Errorf("Profile mismatch: got %+v want %+v", got.Profiles["stormwind"], want.Profiles["stormwind"])
	}
}

func TestActive_explicitName(t *testing.T) {
	st := &State{
		CurrentProfile: "stormwind",
		Profiles: map[string]ProfileState{
			"stormwind": {JWT: "a"},
			"cloud":     {JWT: "b"},
		},
	}
	if got := st.Active("cloud"); got != "cloud" {
		t.Errorf("Active(cloud) = %q, want cloud", got)
	}
}

func TestActive_emptyName_usesCurrentProfile(t *testing.T) {
	st := &State{
		CurrentProfile: "stormwind",
		Profiles: map[string]ProfileState{
			"stormwind": {JWT: "a"},
		},
	}
	if got := st.Active(""); got != "stormwind" {
		t.Errorf("Active(\"\") = %q, want stormwind", got)
	}
}

func TestActive_emptyName_emptyCurrent_usesDefault(t *testing.T) {
	st := &State{
		Profiles: map[string]ProfileState{
			"default": {JWT: "d"},
		},
	}
	if got := st.Active(""); got != "default" {
		t.Errorf("Active(\"\") = %q, want default", got)
	}
}
