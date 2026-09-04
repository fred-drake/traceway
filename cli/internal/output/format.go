// Package output renders command results as JSON, YAML, or tables, plus the
// stable error envelope.
package output

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// Mode controls how Render formats its output.
type Mode int

const (
	ModeJSON Mode = iota
	ModeYAML
	ModeTable
)

func (m Mode) String() string {
	switch m {
	case ModeJSON:
		return "json"
	case ModeYAML:
		return "yaml"
	case ModeTable:
		return "table"
	default:
		return "unknown"
	}
}

// ParseMode converts the user-facing flag value to a Mode.
func ParseMode(s string) (Mode, error) {
	switch strings.ToLower(s) {
	case "json":
		return ModeJSON, nil
	case "yaml":
		return ModeYAML, nil
	case "table":
		return ModeTable, nil
	default:
		return 0, fmt.Errorf("invalid output mode %q (valid: json, yaml, table)", s)
	}
}

// ResolveMode picks an effective Mode. An explicit user value wins; otherwise
// default to table on TTY and json on non-TTY.
//
// The error returned by ParseMode for invalid explicit values is intentionally
// suppressed here — the caller should validate via ParseMode at flag-parse time.
func ResolveMode(explicit string, isTTY bool) Mode {
	if explicit != "" {
		if m, err := ParseMode(explicit); err == nil {
			return m
		}
	}
	if isTTY {
		return ModeTable
	}
	return ModeJSON
}

// IsTerminal reports whether the given file descriptor is a terminal.
// Wraps golang.org/x/term so callers don't import it directly.
func IsTerminal(fd uintptr) bool {
	return term.IsTerminal(int(fd))
}

// StdoutIsTerminal is a convenience shortcut.
func StdoutIsTerminal() bool { return IsTerminal(os.Stdout.Fd()) }

// StderrIsTerminal is a convenience shortcut.
func StderrIsTerminal() bool { return IsTerminal(os.Stderr.Fd()) }
