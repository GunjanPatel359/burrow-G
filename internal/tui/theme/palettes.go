package theme

import "github.com/charmbracelet/lipgloss"

// Palette is the set of colours a theme is built from.
type Palette struct {
	Accent   lipgloss.Color // headers, selection marker
	Info     lipgloss.Color // folders
	Label    lipgloss.Color // details labels
	Success  lipgloss.Color
	Warning  lipgloss.Color // stars, "refreshing…"
	Error    lipgloss.Color
	Muted    lipgloss.Color // secondary text, lines
	Text     lipgloss.Color
	Selected lipgloss.Color // selected row background
}

const Default = "tokyo-night"

// palettes holds every theme. Add a new theme by adding an entry here.
var palettes = map[string]Palette{
	"tokyo-night": {
		Accent:   "#7aa2f7",
		Info:     "#7dcfff",
		Label:    "#bb9af7",
		Success:  "#9ece6a",
		Warning:  "#e0af68",
		Error:    "#f7768e",
		Muted:    "#565f89",
		Text:     "#c0caf5",
		Selected: "#283457",
	},
}
