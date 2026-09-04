package client

import (
	"testing"
	"time"
)

func TestTimeRange_zeroValueIsValid(t *testing.T) {
	tr := TimeRange{}
	if !tr.From.IsZero() || !tr.To.IsZero() {
		t.Error("zero TimeRange should have zero From/To")
	}
}

func TestTimeRangeFromSince_setsFromToNow(t *testing.T) {
	now := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	tr := TimeRangeFromSinceAt(time.Hour, now)
	if !tr.To.Equal(now) {
		t.Errorf("To = %v, want %v", tr.To, now)
	}
	wantFrom := now.Add(-time.Hour)
	if !tr.From.Equal(wantFrom) {
		t.Errorf("From = %v, want %v", tr.From, wantFrom)
	}
}

func TestTimeRangeFromExplicit(t *testing.T) {
	from := time.Date(2026, 5, 13, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC)
	tr := TimeRangeFromExplicit(from, to)
	if !tr.From.Equal(from) {
		t.Errorf("From = %v, want %v", tr.From, from)
	}
	if !tr.To.Equal(to) {
		t.Errorf("To = %v, want %v", tr.To, to)
	}
}
