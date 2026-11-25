package ui

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	"github.com/muesli/termenv"

	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
)

var (
	move  = key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("up/down", "move"))
	open  = key.NewBinding(key.WithKeys("enter", "right"), key.WithHelp("enter/right", "sessions"))
	quit  = key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit"))
	quiet = key.NewBinding(key.WithKeys("x")) // no help: never shown
)

func TestKeyHints(t *testing.T) {
	th := plainTheme()
	if got := visible(KeyHints(th, move, open, quiet, quit)); got != "↑/↓ move · enter/→ sessions · q quit" {
		t.Errorf("KeyHints = %q", got)
	}
	ascii := theme.Get(theme.Default, theme.Env{ASCII: true})
	if got := visible(KeyHints(ascii, move, quit)); got != "^/v move . q quit" {
		t.Errorf("ASCII KeyHints = %q", got)
	}
	off := quit
	off.SetEnabled(false)
	if got := visible(KeyHints(th, move, off)); got != "↑/↓ move" {
		t.Errorf("disabled binding shown: %q", got)
	}
}

func TestKeyHints_KeysStandOut(t *testing.T) {
	withProfile(t, termenv.TrueColor)
	th := plainTheme()
	if KeyHints(th, quit) == th.Muted.Render("q quit") {
		t.Error("the key should be drawn in Accent, not Muted")
	}
}

func TestHintText(t *testing.T) {
	if got := HintText(theme.Unicode, open); got != "enter/→ sessions" {
		t.Errorf("HintText = %q", got)
	}
	if got := KeyName(theme.Unicode, "shift+tab/left"); got != "shift+tab/←" {
		t.Errorf("KeyName = %q", got)
	}
}
