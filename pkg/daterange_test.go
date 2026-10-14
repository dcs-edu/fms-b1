package pkg

import (
	"errors"
	"testing"
	"time"
)

func TestResolveDateRange(t *testing.T) {
	// Friday 2026-10-02 15:30 — a fixed "now" is only possible because ResolveDateRange takes it as a parameter
	now := time.Date(2026, time.October, 2, 15, 30, 0, 0, time.UTC)
	day := func(m time.Month, d int) time.Time { return time.Date(2026, m, d, 0, 0, 0, 0, time.UTC) }

	tests := []struct {
		key        string
		start, end time.Time
	}{
		{"today", day(time.October, 2), day(time.October, 3)},
		{"this_week", day(time.September, 27), day(time.October, 4)}, // weeks start on Sunday
		{"this_month", day(time.October, 1), day(time.November, 1)},
		{"three_months_to_date", day(time.July, 1), day(time.October, 3)},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			start, end, err := ResolveDateRange(tt.key, now)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !start.Equal(tt.start) || !end.Equal(tt.end) {
				t.Errorf("got [%v, %v), want [%v, %v)", start, end, tt.start, tt.end)
			}
		})
	}

	t.Run("unknown key", func(t *testing.T) {
		if _, _, err := ResolveDateRange("last_year", now); !errors.Is(err, ErrUnknownRange) {
			t.Errorf("got %v, want ErrUnknownRange", err)
		}
	})
}
