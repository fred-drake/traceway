package output

import (
	"encoding/json"
	"io"
	"strings"
)

// ParseFieldsFlag splits "a, b, c" → []string{"a", "b", "c"}. Empty input → nil.
func ParseFieldsFlag(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// RenderJSON marshals v as compact JSON to w (one record per line, no indent).
// If fields is non-nil, projects each item of a top-level "data" array
// (Traceway's wrapper shape) — or the top-level object itself if it has no
// "data" key — to just the named fields. "pagination" and other top-level
// wrapper keys pass through unchanged.
//
// Compact output is intentional: it minimizes tokens for LLM consumers and
// matches what real APIs return. Humans wanting pretty output can pipe to
// `jq` (or use `--output table`).
func RenderJSON(w io.Writer, v any, fields []string) error {
	if fields == nil {
		return json.NewEncoder(w).Encode(v)
	}

	// Round-trip through JSON to a generic any so we can project by string keys.
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var generic any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return err
	}
	projected := projectFields(generic, fields)
	return json.NewEncoder(w).Encode(projected)
}

// projectFields keeps only the named keys, with awareness of Traceway's
// {data: [...], pagination: {...}} wrapper. See RenderJSON for the contract.
func projectFields(v any, fields []string) any {
	set := make(map[string]bool, len(fields))
	for _, f := range fields {
		set[f] = true
	}

	if m, ok := v.(map[string]any); ok {
		if data, hasData := m["data"]; hasData {
			if arr, ok := data.([]any); ok {
				out := make([]any, len(arr))
				for i, item := range arr {
					out[i] = projectMap(item, set)
				}
				m["data"] = out
				return m
			}
		}
		return projectMap(v, set)
	}
	if arr, ok := v.([]any); ok {
		out := make([]any, len(arr))
		for i, item := range arr {
			out[i] = projectMap(item, set)
		}
		return out
	}
	return v
}

func projectMap(v any, fields map[string]bool) any {
	m, ok := v.(map[string]any)
	if !ok {
		return v
	}
	out := make(map[string]any, len(fields))
	for k, val := range m {
		if fields[k] {
			out[k] = val
		}
	}
	return out
}
