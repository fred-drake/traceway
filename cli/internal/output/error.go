package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// ErrorEnvelope is the stable error contract written to stderr on any failure.
// Code is a snake_case stable identifier; LLMs may branch on it.
type ErrorEnvelope struct {
	Code     string `json:"error"`
	Message  string `json:"message"`
	Hint     string `json:"hint,omitempty"`
	ExitCode int    `json:"exit_code"`
}

// RenderError writes the envelope to w. Compact JSON for ModeJSON/ModeYAML;
// prose for ModeTable. (YAML mode uses JSON for errors — easier for callers
// to parse and matches gh's behavior for machine-formatted errors.)
func RenderError(w io.Writer, mode Mode, env ErrorEnvelope) error {
	if mode == ModeJSON || mode == ModeYAML {
		return json.NewEncoder(w).Encode(env)
	}
	if _, err := fmt.Fprintf(w, "Error: %s\n", env.Message); err != nil {
		return err
	}
	if env.Hint != "" {
		if _, err := fmt.Fprintf(w, "  Hint: %s\n", env.Hint); err != nil {
			return err
		}
	}
	return nil
}
