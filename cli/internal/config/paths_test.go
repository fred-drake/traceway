package config

import (
	"path/filepath"
	"testing"
)

func TestConfigPath_xdgSet(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
	t.Setenv("HOME", "/tmp/home")

	got, err := configPath()
	if err != nil {
		t.Fatalf("configPath() error: %v", err)
	}
	want := filepath.Join("/tmp/xdg", "traceway", "config.json")
	if got != want {
		t.Errorf("configPath() = %q, want %q", got, want)
	}
}

func TestConfigPath_xdgUnset(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/tmp/home")

	got, err := configPath()
	if err != nil {
		t.Fatalf("configPath() error: %v", err)
	}
	want := filepath.Join("/tmp/home", ".config", "traceway", "config.json")
	if got != want {
		t.Errorf("configPath() = %q, want %q", got, want)
	}
}

func TestConfigPath_neither(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")

	if _, err := configPath(); err == nil {
		t.Fatal("expected error when both XDG_CONFIG_HOME and HOME are empty")
	}
}
