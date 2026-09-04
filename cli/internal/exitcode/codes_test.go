package exitcode

import "testing"

func TestCodes_areStable(t *testing.T) {
	cases := []struct {
		name string
		got  int
		want int
	}{
		{"Success", Success, 0},
		{"Generic", Generic, 1},
		{"Usage", Usage, 2},
		{"Connection", Connection, 3},
		{"Auth", Auth, 4},
		{"NotFound", NotFound, 5},
		{"RateLimited", RateLimited, 6},
		{"Server", Server, 7},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.name, c.got, c.want)
		}
	}
}
