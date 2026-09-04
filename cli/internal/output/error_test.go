package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderError_jsonShape(t *testing.T) {
	var buf bytes.Buffer
	err := RenderError(&buf, ModeJSON, ErrorEnvelope{
		Code:     "token_expired",
		Message:  "JWT expired or invalid",
		Hint:     "traceway login --profile stormwind",
		ExitCode: 4,
	})
	if err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\nout: %s", err, buf.String())
	}
	if got["error"] != "token_expired" {
		t.Errorf("error = %v", got["error"])
	}
	if got["message"] != "JWT expired or invalid" {
		t.Errorf("message = %v", got["message"])
	}
	if got["hint"] != "traceway login --profile stormwind" {
		t.Errorf("hint = %v", got["hint"])
	}
	if int(got["exit_code"].(float64)) != 4 {
		t.Errorf("exit_code = %v", got["exit_code"])
	}
}

func TestRenderError_jsonOmitsEmptyHint(t *testing.T) {
	var buf bytes.Buffer
	_ = RenderError(&buf, ModeJSON, ErrorEnvelope{
		Code:     "internal",
		Message:  "boom",
		ExitCode: 1,
	})
	var got map[string]any
	_ = json.Unmarshal(buf.Bytes(), &got)
	if _, ok := got["hint"]; ok {
		t.Errorf("hint should be omitted when empty, got: %v", got)
	}
}

func TestRenderError_proseHasErrorPrefix(t *testing.T) {
	var buf bytes.Buffer
	_ = RenderError(&buf, ModeTable, ErrorEnvelope{
		Code:     "token_expired",
		Message:  "session expired",
		Hint:     "traceway login --profile stormwind",
		ExitCode: 4,
	})
	out := buf.String()
	if !strings.HasPrefix(out, "Error:") {
		t.Errorf("prose form should start with 'Error:', got:\n%s", out)
	}
	if !strings.Contains(out, "session expired") {
		t.Errorf("missing message, got:\n%s", out)
	}
	if !strings.Contains(out, "Hint:") {
		t.Errorf("missing hint line, got:\n%s", out)
	}
}

func TestRenderError_proseOmitsHintLineWhenAbsent(t *testing.T) {
	var buf bytes.Buffer
	_ = RenderError(&buf, ModeTable, ErrorEnvelope{
		Code:     "internal",
		Message:  "boom",
		ExitCode: 1,
	})
	out := buf.String()
	if strings.Contains(out, "Hint:") {
		t.Errorf("Hint line should be omitted, got:\n%s", out)
	}
}
