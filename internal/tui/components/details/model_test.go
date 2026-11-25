package details

import (
	"strings"
	"testing"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/session"
	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
)

func TestView_Session(t *testing.T) {
	m := New(theme.Get(theme.Default, theme.Env{}))
	m.SetWidth(80)
	s := session.Session{ID: "aaa-111", Dir: "/tmp/proj", LastUsed: time.Now().Add(-2 * time.Hour)}
	m.Show(s, []string{"claude", "--resume", "aaa-111"})

	v := m.View()
	for _, want := range []string{"aaa-111", "/tmp/proj", "claude --resume aaa-111", "2 hours ago"} {
		if !strings.Contains(v, want) {
			t.Errorf("view is missing %q:\n%s", want, v)
		}
	}
	if n := strings.Count(v, "\n") + 1; n != theme.DetailsHeight {
		t.Errorf("view has %d lines, want %d", n, theme.DetailsHeight)
	}
}

func TestView_Empty(t *testing.T) {
	m := New(theme.Get(theme.Default, theme.Env{}))
	m.Clear()

	v := m.View()
	if !strings.Contains(v, "no sessions") {
		t.Errorf("empty view = %q", v)
	}
	if n := strings.Count(v, "\n") + 1; n != theme.DetailsHeight {
		t.Errorf("empty view has %d lines, want %d", n, theme.DetailsHeight)
	}
}
