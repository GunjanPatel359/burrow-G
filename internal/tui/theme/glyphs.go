package theme

// GlyphSet names every symbol the UI draws, so a non UTF-8 locale can swap
// them all for ASCII.
type GlyphSet struct {
	Marker   string // selected row
	Star     string // bookmark
	Dot      string // status
	VLine    string
	HLine    string
	Sep      string // between parts and hints
	Up       string
	Down     string
	Left     string
	Right    string
	Ellipsis string // cut text, in progress
	Times    string // sizes, as in 40×8
	Corners  [4]string
	Spinner  Spinner
}

// Spinner is a busy animation. Frames plus a rate keep theme free of bubbles.
type Spinner struct {
	Frames []string
	FPS    int
}

var Unicode = GlyphSet{
	Marker:   "▌",
	Star:     "★",
	Dot:      "●",
	VLine:    "│",
	HLine:    "─",
	Sep:      "·",
	Up:       "↑",
	Down:     "↓",
	Left:     "←",
	Right:    "→",
	Ellipsis: "…",
	Times:    "×",
	Corners:  [4]string{"╭", "╮", "╰", "╯"},
	Spinner:  Spinner{Frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}, FPS: 12},
}

var ASCII = GlyphSet{
	Marker:   ">",
	Star:     "*",
	Dot:      "o",
	VLine:    "|",
	HLine:    "-",
	Sep:      ".",
	Up:       "^",
	Down:     "v",
	Left:     "<",
	Right:    ">",
	Ellipsis: "...",
	Times:    "x",
	Corners:  [4]string{"+", "+", "+", "+"},
	Spinner:  Spinner{Frames: []string{"|", "/", "-", `\`}, FPS: 10},
}
