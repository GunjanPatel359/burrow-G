// Package session is the common shape for all agents' sessions.
package session

import (
	"errors"
	"slices"
	"time"
)

// ErrMissing is returned by List when the agent's files aren't there: it isn't
// installed, or its folder can't be reached right now.
var ErrMissing = errors.New("agent files not found")

type Session struct {
	Agent      string
	ID         string
	Title      string
	Dir        string
	LastUsed   time.Time
	Bookmarked bool
}

// Key is unique across agents, e.g. "claude:abc".
func (s Session) Key() string {
	return s.Agent + ":" + s.ID
}

// Source is implemented by each agent reader.
type Source interface {
	Name() string
	List() ([]Session, error)
	ResumeCommand(s Session) []string
	Delete(s Session, trashDir string) error
}

// Sort puts bookmarked first, then newest first.
func Sort(list []Session, bookmarked func(Session) bool) {
	slices.SortStableFunc(list, func(a, b Session) int {
		aMarked, bMarked := bookmarked(a), bookmarked(b)
		if aMarked != bMarked {
			if aMarked {
				return -1
			}
			return 1
		}
		return b.LastUsed.Compare(a.LastUsed)
	})
}
