package state

import (
	"errors"
	"os"
	"path/filepath"
)

// statePath returns the path to the state file, resolving XDG_STATE_HOME
// or falling back to $HOME/.local/state. It does not create the file.
func statePath() (string, error) {
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return filepath.Join(dir, "traceway", "state.json"), nil
	}
	home := os.Getenv("HOME")
	if home == "" {
		return "", errors.New("neither XDG_STATE_HOME nor HOME is set")
	}
	return filepath.Join(home, ".local", "state", "traceway", "state.json"), nil
}
