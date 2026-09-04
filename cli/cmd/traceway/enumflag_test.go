package main

import (
	"strings"
	"testing"
)

func TestValidateEnumFlag_acceptsAllAllowed(t *testing.T) {
	allowed := []string{"avg", "sum", "p95"}
	for _, v := range allowed {
		if err := validateEnumFlag("--aggregation", v, allowed); err != nil {
			t.Errorf("validateEnumFlag(%q) = %v, want nil", v, err)
		}
	}
}

func TestValidateEnumFlag_rejectsUnknown(t *testing.T) {
	allowed := []string{"avg", "sum", "p95"}
	err := validateEnumFlag("--aggregation", "bogus", allowed)
	if err == nil {
		t.Fatal("expected error for unknown value")
	}
	msg := err.Error()
	if !strings.Contains(msg, "--aggregation") {
		t.Errorf("error %q should mention the flag name", msg)
	}
	for _, a := range allowed {
		if !strings.Contains(msg, a) {
			t.Errorf("error %q should list allowed value %q", msg, a)
		}
	}
}

func TestValidateEnumFlag_rejectsEmpty(t *testing.T) {
	if err := validateEnumFlag("--x", "", []string{"a", "b"}); err == nil {
		t.Fatal("empty value should not validate")
	}
}

func TestEnumFlagHint_format(t *testing.T) {
	got := enumFlagHint("traceway metrics query", "--aggregation", []string{"avg", "sum"})
	want := "traceway metrics query --aggregation <avg|sum>"
	if got != want {
		t.Errorf("enumFlagHint = %q, want %q", got, want)
	}
}
