package client

import (
	"errors"
	"testing"
)

func TestAPIError_Error(t *testing.T) {
	e := &APIError{StatusCode: 500, Body: "boom"}
	got := e.Error()
	if got == "" {
		t.Fatal("Error() returned empty string")
	}
}

func TestSentinelErrors_areDistinct(t *testing.T) {
	all := []error{ErrUnauthorized, ErrForbidden, ErrNotFound, ErrRateLimited}
	for i, a := range all {
		for j, b := range all {
			if i == j {
				continue
			}
			if errors.Is(a, b) {
				t.Errorf("errors.Is(%v, %v) = true; expected distinct", a, b)
			}
		}
	}
}

func TestAPIError_doesNotMatchSentinels(t *testing.T) {
	apiErr := &APIError{StatusCode: 418}
	if errors.Is(apiErr, ErrUnauthorized) {
		t.Error("APIError(418) should not match ErrUnauthorized")
	}
}
