// Package session is the common shape for all agents' sessions.
package session

import (
	"errors"
	"slices"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/portable"
)

// ErrMissing is returned by List when the agent's files aren't there: it isn't
// installed, or its folder can't be reached right now.
var ErrMissing = errors.New("agent files not found")

// ErrUnsupported means this agent can't do that yet. Callers check it with
// errors.Is and say so, like ErrMissing.
var ErrUnsupported = errors.New("not supported for this agent")

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
	// Export streams s in the portable format, oldest record first. It uses
	// the agent's own export when that gives structured data, else reads
	// the agent's files itself (spec 0003).
	Export(s Session, w *portable.Writer) error
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
