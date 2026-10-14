package checkin

import "time"

// DueAfterDays matches the PRD's "¿han pasado 7 días desde el último
// check-in?" decision — a simple threshold checked on read, no scheduler.
const DueAfterDays = 7

// IsDue reports whether a new check-in is due, given the date (YYYY-MM-DD)
// of the last one and the current time.
func IsDue(lastLoggedOn string, now time.Time) (due bool, daysSince int, err error) {
	last, err := time.Parse("2006-01-02", lastLoggedOn)
	if err != nil {
		return false, 0, err
	}
	daysSince = int(now.UTC().Sub(last).Hours() / 24)
	return daysSince >= DueAfterDays, daysSince, nil
}
