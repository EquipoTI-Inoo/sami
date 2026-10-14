package checkin

import (
	"testing"
	"time"
)

func TestIsDue(t *testing.T) {
	now := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name          string
		lastLoggedOn  string
		wantDue       bool
		wantDaysSince int
	}{
		{"today", "2026-03-15", false, 0},
		{"six days ago", "2026-03-09", false, 6},
		{"exactly seven days ago", "2026-03-08", true, 7},
		{"well over a week ago", "2026-02-01", true, 42},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			due, daysSince, err := IsDue(tc.lastLoggedOn, now)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if due != tc.wantDue || daysSince != tc.wantDaysSince {
				t.Fatalf("IsDue(%q) = (%v, %d), want (%v, %d)", tc.lastLoggedOn, due, daysSince, tc.wantDue, tc.wantDaysSince)
			}
		})
	}
}

func TestIsDueRejectsBadDate(t *testing.T) {
	if _, _, err := IsDue("not-a-date", time.Now()); err == nil {
		t.Fatal("expected an error for an unparseable date")
	}
}
