package gateway

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// nextFire returns the next time the cron spec is due at or after `now`.
// Supported grammar (intentionally minimal — no third-party cron lib):
//
//   "@every <duration>"   — e.g. "@every 5m", "@every 2h30m"
//   "@daily HH:MM"        — e.g. "@daily 09:00"
//
// Returns (zero, false) for unparseable specs.
func nextFire(spec string, now time.Time) (time.Time, bool) {
	spec = strings.TrimSpace(spec)
	if !strings.HasPrefix(spec, "@") {
		return time.Time{}, false
	}
	parts := strings.SplitN(spec, " ", 2)
	if len(parts) != 2 {
		return time.Time{}, false
	}
	switch parts[0] {
	case "@every":
		d, err := time.ParseDuration(parts[1])
		if err != nil {
			return time.Time{}, false
		}
		// Fire at the next boundary >= now
		base := now.Truncate(d)
		if base.Before(now) {
			base = base.Add(d)
		}
		return base, true
	case "@daily":
		hh, mm, err := parseClock(parts[1])
		if err != nil {
			return time.Time{}, false
		}
		next := time.Date(now.Year(), now.Month(), now.Day(), hh, mm, 0, 0, now.Location())
		if !next.After(now) {
			next = next.Add(24 * time.Hour)
		}
		return next, true
	}
	return time.Time{}, false
}

func parseClock(s string) (int, int, error) {
	pieces := strings.Split(s, ":")
	if len(pieces) != 2 {
		return 0, 0, fmt.Errorf("expected HH:MM, got %q", s)
	}
	hh, err := strconv.Atoi(pieces[0])
	if err != nil || hh < 0 || hh > 23 {
		return 0, 0, fmt.Errorf("bad hour: %q", pieces[0])
	}
	mm, err := strconv.Atoi(pieces[1])
	if err != nil || mm < 0 || mm > 59 {
		return 0, 0, fmt.Errorf("bad minute: %q", pieces[1])
	}
	return hh, mm, nil
}
