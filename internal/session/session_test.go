package session

import (
	"testing"
	"time"
)

func TestKey(t *testing.T) {
	s := Session{Agent: "claude", ID: "abc"}
	if got := s.Key(); got != "claude:abc" {
		t.Errorf("Key() = %q, want %q", got, "claude:abc")
	}
}

func TestSort_BookmarkedFirstThenNewest(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	oldBookmarked := Session{ID: "old-bookmarked", LastUsed: now.Add(-30 * 24 * time.Hour)}
	newest := Session{ID: "new", LastUsed: now.Add(-1 * time.Hour)}
	middle := Session{ID: "middle", LastUsed: now.Add(-5 * 24 * time.Hour)}

	list := []Session{middle, newest, oldBookmarked}
	Sort(list, func(s Session) bool { return s.ID == "old-bookmarked" })

	want := []string{"old-bookmarked", "new", "middle"}
	for i, id := range want {
		if list[i].ID != id {
			t.Fatalf("position %d = %q, want %q (full order: %v)", i, list[i].ID, id, ids(list))
		}
	}
}

func ids(list []Session) []string {
	out := make([]string, len(list))
	for i, s := range list {
		out[i] = s.ID
	}
	return out
}
