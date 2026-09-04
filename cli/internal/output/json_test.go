package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

type project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type wrappedResp struct {
	Data       []project `json:"data"`
	Pagination struct {
		Total int `json:"total"`
	} `json:"pagination"`
}

func TestRenderJSON_passThroughWhenNoFields(t *testing.T) {
	in := wrappedResp{
		Data: []project{{ID: "p1", Name: "alpha", URL: "https://a"}},
	}
	in.Pagination.Total = 1

	var buf bytes.Buffer
	if err := RenderJSON(&buf, in, nil); err != nil {
		t.Fatal(err)
	}

	var got wrappedResp
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON output: %v\noutput: %s", err, buf.String())
	}
	if got.Data[0].URL != "https://a" {
		t.Errorf("URL was stripped; pass-through expected")
	}
}

func TestRenderJSON_projectsFieldsInWrappedData(t *testing.T) {
	in := wrappedResp{
		Data: []project{
			{ID: "p1", Name: "alpha", URL: "https://a"},
			{ID: "p2", Name: "beta", URL: "https://b"},
		},
	}

	var buf bytes.Buffer
	if err := RenderJSON(&buf, in, []string{"id", "name"}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	if !strings.Contains(out, `"id"`) || !strings.Contains(out, `"name"`) {
		t.Errorf("expected id and name in projection, got: %s", out)
	}
	if strings.Contains(out, `"url"`) {
		t.Errorf("url should have been projected away, got: %s", out)
	}
	// Pagination stays at the top level even when fields are projected.
	if !strings.Contains(out, `"pagination"`) {
		t.Errorf("pagination should pass through, got: %s", out)
	}
}

func TestRenderJSON_projectsTopLevelObject(t *testing.T) {
	in := project{ID: "p1", Name: "alpha", URL: "https://a"}

	var buf bytes.Buffer
	if err := RenderJSON(&buf, in, []string{"id"}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, `"id"`) {
		t.Errorf("expected id, got: %s", out)
	}
	if strings.Contains(out, `"name"`) {
		t.Errorf("name should have been projected away, got: %s", out)
	}
}

func TestParseFieldsFlag(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"a", []string{"a"}},
		{"a,b,c", []string{"a", "b", "c"}},
		{" a , b ", []string{"a", "b"}},
	}
	for _, c := range cases {
		got := ParseFieldsFlag(c.in)
		if len(got) != len(c.want) {
			t.Errorf("ParseFieldsFlag(%q) = %v, want %v", c.in, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("ParseFieldsFlag(%q)[%d] = %q, want %q", c.in, i, got[i], c.want[i])
			}
		}
	}
}
