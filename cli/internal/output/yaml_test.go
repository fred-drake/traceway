package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderYAML_passThrough(t *testing.T) {
	in := project{ID: "p1", Name: "alpha", URL: "https://a"}

	var buf bytes.Buffer
	if err := RenderYAML(&buf, in, nil); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "id: p1") {
		t.Errorf("expected 'id: p1' in YAML, got:\n%s", out)
	}
	if !strings.Contains(out, "name: alpha") {
		t.Errorf("expected 'name: alpha' in YAML, got:\n%s", out)
	}
	if !strings.Contains(out, "url: https://a") {
		t.Errorf("expected 'url: https://a' in YAML, got:\n%s", out)
	}
}

func TestRenderYAML_projectsFields(t *testing.T) {
	in := project{ID: "p1", Name: "alpha", URL: "https://a"}

	var buf bytes.Buffer
	if err := RenderYAML(&buf, in, []string{"id"}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "id: p1") {
		t.Errorf("expected id in projection, got:\n%s", out)
	}
	if strings.Contains(out, "name:") {
		t.Errorf("name should be projected away, got:\n%s", out)
	}
}
