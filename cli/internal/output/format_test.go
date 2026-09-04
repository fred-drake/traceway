package output

import (
	"testing"
)

func TestParseMode_validValues(t *testing.T) {
	cases := map[string]Mode{
		"json":  ModeJSON,
		"JSON":  ModeJSON,
		"yaml":  ModeYAML,
		"table": ModeTable,
	}
	for in, want := range cases {
		got, err := ParseMode(in)
		if err != nil {
			t.Errorf("ParseMode(%q) error: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("ParseMode(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestParseMode_invalid(t *testing.T) {
	if _, err := ParseMode("xml"); err == nil {
		t.Error("expected error for invalid mode")
	}
}

func TestResolveMode_explicitWins(t *testing.T) {
	got := ResolveMode("json", false)
	if got != ModeJSON {
		t.Errorf("got %v", got)
	}
	got = ResolveMode("table", false)
	if got != ModeTable {
		t.Errorf("got %v", got)
	}
}

func TestResolveMode_emptyDefaultsByTTY(t *testing.T) {
	if got := ResolveMode("", true); got != ModeTable {
		t.Errorf("TTY default = %v, want ModeTable", got)
	}
	if got := ResolveMode("", false); got != ModeJSON {
		t.Errorf("non-TTY default = %v, want ModeJSON", got)
	}
}
