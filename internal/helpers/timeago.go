// Package helpers has small reusable functions.
package helpers

import (
	"fmt"
	"time"
)

// TimeAgo returns text like "2 hours ago".
func TimeAgo(t, now time.Time) string {
	d := now.Sub(t)
	days := int(d.Hours() / 24)

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return plural(int(d.Minutes()), "minute")
	case d < 24*time.Hour:
		return plural(int(d.Hours()), "hour")
	case days < 30:
		return plural(days, "day")
	case days < 365:
		return plural(days/30, "month")
	default:
		return plural(days/365, "year")
	}
}

func plural(n int, unit string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s ago", unit)
	}
	return fmt.Sprintf("%d %ss ago", n, unit)
}
