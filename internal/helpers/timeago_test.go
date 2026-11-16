package helpers

import (
	"testing"
	"time"
)

func TestTimeAgo(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		ago  time.Duration
		want string
	}{
		{10 * time.Second, "just now"},
		{1 * time.Minute, "1 minute ago"},
		{5 * time.Minute, "5 minutes ago"},
		{1 * time.Hour, "1 hour ago"},
		{2 * time.Hour, "2 hours ago"},
		{24 * time.Hour, "1 day ago"},
		{9 * 24 * time.Hour, "9 days ago"},
		{60 * 24 * time.Hour, "2 months ago"},
		{800 * 24 * time.Hour, "2 years ago"},
	}

	for _, c := range cases {
		got := TimeAgo(now.Add(-c.ago), now)
		if got != c.want {
			t.Errorf("TimeAgo(%v ago) = %q, want %q", c.ago, got, c.want)
		}
	}
}
