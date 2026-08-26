package notifications

import "testing"

func TestEmailAdapterValidate(t *testing.T) {
	eleven := make([]string, 11)
	for i := range eleven {
		eleven[i] = "ops@example.com"
	}
	cases := []struct {
		name       string
		recipients []string
		ok         bool
	}{
		{"bare address", []string{"ops@example.com"}, true},
		{"display name", []string{"Ops Team <ops@example.com>"}, true},
		{"comma-joined pair", []string{"a@example.com, b@example.com"}, false},
		{"header injection", []string{"ops@example.com\r\nBcc: x@example.com"}, false},
		{"missing at", []string{"ops.example.com"}, false},
		{"empty", nil, false},
		{"eleven recipients", eleven, false},
	}
	for _, c := range cases {
		err := (&EmailAdapter{Recipients: c.recipients}).Validate()
		if (err == nil) != c.ok {
			t.Errorf("%s: err = %v, want ok=%v", c.name, err, c.ok)
		}
	}
}
