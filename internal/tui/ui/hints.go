package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"

	"github.com/GunjanPatel359/burrow-G/internal/tui/theme"
)

// arrows are the key names a binding's help may use for an arrow glyph.
var arrows = map[string]func(theme.GlyphSet) string{
	"up":    func(g theme.GlyphSet) string { return g.Up },
	"down":  func(g theme.GlyphSet) string { return g.Down },
	"left":  func(g theme.GlyphSet) string { return g.Left },
	"right": func(g theme.GlyphSet) string { return g.Right },
}

// KeyName spells a binding's help key with arrows from the glyph set:
// "up/down" becomes "↑/↓", "enter/right" becomes "enter/→".
func KeyName(g theme.GlyphSet, help string) string {
	parts := strings.Split(help, "/")
	for i, p := range parts {
		if arrow, ok := arrows[p]; ok {
			parts[i] = arrow(g)
		}
	}
	return strings.Join(parts, "/")
}

// HintText is a binding as plain "key action" text, for the help command.
func HintText(g theme.GlyphSet, b key.Binding) string {
	h := b.Help()
	return KeyName(g, h.Key) + " " + h.Desc
}

// KeyHints draws bindings as "↑/↓ move · enter open": keys in Accent, words in
// Muted. Bindings that are off or have no help are left out.
func KeyHints(t theme.Theme, keys ...key.Binding) string {
	return keyHints(t, lipgloss.NewStyle(), keys)
}

// keyHints draws the hints on top of base, so a dialog keeps its background.
func keyHints(t theme.Theme, base lipgloss.Style, keys []key.Binding) string {
	muted := t.Muted.Inherit(base)
	var out []string
	for _, b := range keys {
		h := b.Help()
		if !b.Enabled() || h.Key == "" {
			continue
		}
		out = append(out, t.Accent.Inherit(base).Render(KeyName(t.Glyph, h.Key))+muted.Render(" "+h.Desc))
	}
	return strings.Join(out, muted.Render(" "+t.Glyph.Sep+" "))
}
