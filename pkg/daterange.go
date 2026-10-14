package pkg

import (
	"errors"
	"fmt"
	"time"
)

var ErrUnknownRange = errors.New("unknown date range")

// ResolveDateRange turns a range key into a half-open [start, end) window around `now`.
// Ranges are computed in now's time zone, so pass time.Now().In(schoolLocation) rather than a UTC time.
func ResolveDateRange(rangeKey string, now time.Time) (time.Time, time.Time, error) {

	switch rangeKey {
	case "today":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return start, start.AddDate(0, 0, 1), nil

	case "this_week":
		weekday := int(now.Weekday())
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -weekday)
		return start, start.AddDate(0, 0, 7), nil

	case "this_month":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return start, start.AddDate(0, 1, 0), nil

	case "three_months_to_date":
		todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		tomorrowStart := todayStart.AddDate(0, 0, 1)
		currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return currentMonthStart.AddDate(0, -3, 0), tomorrowStart, nil

	default:
		return time.Time{}, time.Time{}, fmt.Errorf("%w: %q", ErrUnknownRange, rangeKey)
	}
}
