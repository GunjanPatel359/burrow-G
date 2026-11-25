package theme

// Tone is one palette color at the two depths a palette controls. At 16
// colors and without color, the shared looks in looks.go are used instead.
type Tone struct {
	Hex  string // truecolor, "#rrggbb"
	X256 uint8  // xterm 256 index, 16..255, picked per theme to keep contrast
}

// Palette is the set of colors a theme is built from. Every theme fills every
// slot; theme_test.go checks the contrast of each pair a slot is drawn on.
type Palette struct {
	Text             Tone // titles, values
	Muted            Tone // secondary text, hints, timestamps
	Border           Tone // rules and the dialog frame
	Accent           Tone // focused pane header, selection marker, keys in hints
	Info             Tone // folders
	Label            Tone // details labels
	Success          Tone
	Warning          Tone // stars, busy spinner
	Error            Tone
	Match            Tone // search match inside a title
	Selected         Tone // selected row background, focused pane
	SelectedInactive Tone // selected row background, unfocused pane
	Surface          Tone // dialog and command bar suggestions background
	Background       Tone // the whole window, while burrow-G runs
}

const (
	Default      = "tokyo-night"
	DefaultLight = "catppuccin-latte"
)

// palettes holds every theme. Colors marked "was" were nudged from the
// upstream value to reach the minimum contrast. Add a theme by adding an entry
// here, then run the theme tests.
var palettes = map[string]Palette{
	"ayu-mirage": {
		Text:             Tone{"#cbccc6", 252},
		Muted:            Tone{"#868f9e", 246}, // was #707a8c
		Border:           Tone{"#707a8c", 244},
		Accent:           Tone{"#73d0ff", 117},
		Info:             Tone{"#95e6cb", 115},
		Label:            Tone{"#d4bfff", 183},
		Success:          Tone{"#bae67e", 149},
		Warning:          Tone{"#ffd580", 222},
		Error:            Tone{"#f28779", 210},
		Match:            Tone{"#ffd580", 222},
		Selected:         Tone{"#34455a", 238},
		SelectedInactive: Tone{"#2a3545", 237},
		Surface:          Tone{"#273141", 236},
		Background:       Tone{"#1f2430", 235},
	},
	"catppuccin-latte": {
		Text:             Tone{"#4c4f69", 239},
		Muted:            Tone{"#6e748a", 243}, // was #9ca0b0
		Border:           Tone{"#7c8197", 243},
		Accent:           Tone{"#094dd3", 18}, // was #1e66f5
		Info:             Tone{"#026086", 24}, // was #04a5e5
		Label:            Tone{"#8839ef", 92},
		Success:          Tone{"#327d22", 22}, // was #40a02b
		Warning:          Tone{"#7d4f10", 94}, // was #df8e1d
		Error:            Tone{"#d20f39", 124},
		Match:            Tone{"#7d4f10", 94},
		Selected:         Tone{"#ccd0da", 254},
		SelectedInactive: Tone{"#dee1e8", 255},
		Surface:          Tone{"#e1e4ea", 253},
		Background:       Tone{"#eff1f5", 231},
	},
	"catppuccin-mocha": {
		Text:             Tone{"#cdd6f4", 189},
		Muted:            Tone{"#9093a6", 246}, // was #6c7086
		Border:           Tone{"#72778e", 103},
		Accent:           Tone{"#8fb8fa", 111}, // was #89b4fa
		Info:             Tone{"#89dceb", 116},
		Label:            Tone{"#cba6f7", 183},
		Success:          Tone{"#a6e3a1", 150},
		Warning:          Tone{"#f9e2af", 223},
		Error:            Tone{"#f38ba8", 211},
		Match:            Tone{"#f9e2af", 223},
		Selected:         Tone{"#45475a", 237},
		SelectedInactive: Tone{"#313344", 236},
		Surface:          Tone{"#2e2e40", 235},
		Background:       Tone{"#1e1e2e", 234},
	},
	"dracula": {
		Text:             Tone{"#f8f8f2", 231},
		Muted:            Tone{"#8693b9", 110}, // was #6272a4
		Border:           Tone{"#707eac", 103},
		Accent:           Tone{"#caa8fa", 183}, // was #bd93f9
		Info:             Tone{"#8be9fd", 116},
		Label:            Tone{"#ff79c6", 212},
		Success:          Tone{"#50fa7b", 84},
		Warning:          Tone{"#f1fa8c", 228},
		Error:            Tone{"#ff5555", 203},
		Match:            Tone{"#f1fa8c", 228},
		Selected:         Tone{"#44475a", 238},
		SelectedInactive: Tone{"#363948", 237},
		Surface:          Tone{"#333644", 236},
		Background:       Tone{"#282a36", 235},
	},
	"everforest-dark": {
		Text:             Tone{"#d3c6aa", 187},
		Muted:            Tone{"#96a199", 247}, // was #859289
		Border:           Tone{"#859289", 245},
		Accent:           Tone{"#9dcbc5", 152}, // was #7fbbb3
		Info:             Tone{"#9eceaa", 151}, // was #83c092
		Label:            Tone{"#d699b6", 175},
		Success:          Tone{"#a7c080", 150},
		Warning:          Tone{"#dcbf84", 222}, // was #dbbc7f
		Error:            Tone{"#e67e80", 210},
		Match:            Tone{"#dcbf84", 222},
		Selected:         Tone{"#475258", 239},
		SelectedInactive: Tone{"#3a444a", 238},
		Surface:          Tone{"#374147", 237},
		Background:       Tone{"#2d353b", 236},
	},
	"github-dark": {
		Text:             Tone{"#c9d1d9", 252},
		Muted:            Tone{"#8b949e", 246},
		Border:           Tone{"#8b949e", 246},
		Accent:           Tone{"#58a6ff", 75},
		Info:             Tone{"#79c0ff", 75},
		Label:            Tone{"#d2a8ff", 183},
		Success:          Tone{"#3fb950", 77},
		Warning:          Tone{"#d29922", 136},
		Error:            Tone{"#f85149", 203},
		Match:            Tone{"#d29922", 136},
		Selected:         Tone{"#21262d", 235},
		SelectedInactive: Tone{"#171c22", 234},
		Surface:          Tone{"#151920", 233},
		Background:       Tone{"#0d1117", 232},
	},
	"github-light": {
		Text:             Tone{"#24292f", 235},
		Muted:            Tone{"#6e7781", 243},
		Border:           Tone{"#6e7781", 243},
		Accent:           Tone{"#0969da", 26},
		Info:             Tone{"#1a7980", 23}, // was #1b7c83
		Label:            Tone{"#8250df", 98},
		Success:          Tone{"#1a7f37", 22},
		Warning:          Tone{"#956400", 94}, // was #9a6700
		Error:            Tone{"#cf222e", 124},
		Match:            Tone{"#956400", 94},
		Selected:         Tone{"#ddf4ff", 195},
		SelectedInactive: Tone{"#eefaff", 254},
		Surface:          Tone{"#f1fbff", 255},
		Background:       Tone{"#ffffff", 231},
	},
	"gruvbox-dark": {
		Text:             Tone{"#ebdbb2", 187},
		Muted:            Tone{"#a29588", 247}, // was #928374
		Border:           Tone{"#928374", 102},
		Accent:           Tone{"#a6bfb5", 152}, // was #83a598
		Info:             Tone{"#98c688", 150}, // was #8ec07c
		Label:            Tone{"#d3869b", 175},
		Success:          Tone{"#b8bb26", 142},
		Warning:          Tone{"#fabd2f", 214},
		Error:            Tone{"#fb533f", 203}, // was #fb4934
		Match:            Tone{"#fabd2f", 214},
		Selected:         Tone{"#504945", 239},
		SelectedInactive: Tone{"#3c3937", 237},
		Surface:          Tone{"#383534", 236},
		Background:       Tone{"#282828", 235},
	},
	"kanagawa": {
		Text:             Tone{"#dcd7ba", 187},
		Muted:            Tone{"#9a9991", 145}, // was #727169
		Border:           Tone{"#7b7a71", 244},
		Accent:           Tone{"#a7bce4", 153}, // was #7e9cd8
		Info:             Tone{"#96c2d4", 152}, // was #7fb4ca
		Label:            Tone{"#957fb8", 139},
		Success:          Tone{"#98bb6c", 107},
		Warning:          Tone{"#e6c384", 222},
		Error:            Tone{"#e46876", 167},
		Match:            Tone{"#e6c384", 222},
		Selected:         Tone{"#2d4f67", 24},
		SelectedInactive: Tone{"#263748", 237},
		Surface:          Tone{"#253241", 236},
		Background:       Tone{"#1f1f28", 234},
	},
	"monokai": {
		Text:             Tone{"#f8f8f2", 231},
		Muted:            Tone{"#98947f", 246}, // was #75715e
		Border:           Tone{"#827e69", 101},
		Accent:           Tone{"#66d9ef", 81},
		Info:             Tone{"#a6e22e", 148},
		Label:            Tone{"#ae81ff", 141},
		Success:          Tone{"#a6e22e", 148},
		Warning:          Tone{"#e6db74", 185},
		Error:            Tone{"#fa4989", 204}, // was #f92672
		Match:            Tone{"#e6db74", 185},
		Selected:         Tone{"#49483e", 238},
		SelectedInactive: Tone{"#383830", 237},
		Surface:          Tone{"#35352d", 236},
		Background:       Tone{"#272822", 235},
	},
	"nord": {
		Text:             Tone{"#d8dee9", 253},
		Muted:            Tone{"#8e99ae", 247}, // was #616e88
		Border:           Tone{"#7b88a1", 103},
		Accent:           Tone{"#8fc4d3", 152}, // was #88c0d0
		Info:             Tone{"#9ac3c2", 152}, // was #8fbcbb
		Label:            Tone{"#b690af", 139}, // was #b48ead
		Success:          Tone{"#a3be8c", 108},
		Warning:          Tone{"#ebcb8b", 222},
		Error:            Tone{"#cf888f", 174}, // was #bf616a
		Match:            Tone{"#ebcb8b", 222},
		Selected:         Tone{"#434c5e", 239},
		SelectedInactive: Tone{"#39404f", 238},
		Surface:          Tone{"#363e4c", 237},
		Background:       Tone{"#2e3440", 235},
	},
	"one-dark": {
		Text:             Tone{"#abb2bf", 249},
		Muted:            Tone{"#888f9d", 246}, // was #5c6370
		Border:           Tone{"#767f8f", 244},
		Accent:           Tone{"#70b7f1", 117}, // was #61afef
		Info:             Tone{"#66bdc8", 116}, // was #56b6c2
		Label:            Tone{"#c678dd", 177},
		Success:          Tone{"#98c379", 150},
		Warning:          Tone{"#e5c07b", 179},
		Error:            Tone{"#e17079", 210}, // was #e06c75
		Match:            Tone{"#e5c07b", 179},
		Selected:         Tone{"#3e4451", 238},
		SelectedInactive: Tone{"#333842", 237},
		Surface:          Tone{"#313640", 236},
		Background:       Tone{"#282c34", 235},
	},
	"rose-pine": {
		Text:             Tone{"#e0def4", 189},
		Muted:            Tone{"#706c88", 103}, // was #6e6a86
		Border:           Tone{"#6e6a86", 60},
		Accent:           Tone{"#c4a7e7", 183},
		Info:             Tone{"#9ccfd8", 152},
		Label:            Tone{"#ebbcba", 181},
		Success:          Tone{"#3a8aaa", 31}, // was #31748f
		Warning:          Tone{"#f6c177", 222},
		Error:            Tone{"#eb6f92", 168},
		Match:            Tone{"#f6c177", 222},
		Selected:         Tone{"#26233a", 235},
		SelectedInactive: Tone{"#201d2f", 234},
		Surface:          Tone{"#1e1c2d", 233},
		Background:       Tone{"#191724", 232},
	},
	"rose-pine-dawn": {
		Text:             Tone{"#575279", 24},
		Muted:            Tone{"#7f7990", 243}, // was #9893a5
		Border:           Tone{"#8a8499", 243},
		Accent:           Tone{"#6d5786", 24}, // was #907aa9
		Info:             Tone{"#3c676f", 23}, // was #56949f
		Label:            Tone{"#c3453f", 88}, // was #d7827e
		Success:          Tone{"#286983", 24},
		Warning:          Tone{"#89550e", 94},  // was #ea9d34
		Error:            Tone{"#ad546e", 131}, // was #b4637a
		Match:            Tone{"#89550e", 94},
		Selected:         Tone{"#dfdad9", 254},
		SelectedInactive: Tone{"#ede7e3", 255},
		Surface:          Tone{"#efeae5", 253},
		Background:       Tone{"#faf4ed", 231},
	},
	"solarized-dark": {
		Text:             Tone{"#8b9b9d", 247}, // was #839496
		Muted:            Tone{"#657e86", 66},  // was #586e75
		Border:           Tone{"#5f777e", 66},
		Accent:           Tone{"#479fde", 75},  // was #268bd2
		Info:             Tone{"#2ca9a0", 37},  // was #2aa198
		Label:            Tone{"#8387cd", 104}, // was #6c71c4
		Success:          Tone{"#859900", 106},
		Warning:          Tone{"#c09200", 136}, // was #b58900
		Error:            Tone{"#e56562", 167}, // was #dc322f
		Match:            Tone{"#c09200", 136},
		Selected:         Tone{"#073642", 235},
		SelectedInactive: Tone{"#04313c", 233},
		Surface:          Tone{"#032f3b", 236},
		Background:       Tone{"#002b36", 234},
	},
	"solarized-light": {
		Text:             Tone{"#586b72", 241}, // was #657b83
		Muted:            Tone{"#768888", 244}, // was #93a1a1
		Border:           Tone{"#7d8e8e", 245},
		Accent:           Tone{"#1e6da5", 24},  // was #268bd2
		Info:             Tone{"#1e736d", 23},  // was #2aa198
		Label:            Tone{"#6368c0", 61},  // was #6c71c4
		Success:          Tone{"#687800", 58},  // was #859900
		Warning:          Tone{"#836300", 94},  // was #b58900
		Error:            Tone{"#da2825", 160}, // was #dc322f
		Match:            Tone{"#836300", 94},
		Selected:         Tone{"#eee8d5", 254},
		SelectedInactive: Tone{"#f6efdc", 255},
		Surface:          Tone{"#f7f0dd", 230},
		Background:       Tone{"#fdf6e3", 231},
	},
	"tokyo-night": {
		Text:             Tone{"#c0caf5", 189},
		Muted:            Tone{"#737ca7", 103}, // was #565f89
		Border:           Tone{"#626d9d", 103},
		Accent:           Tone{"#7aa2f7", 111},
		Info:             Tone{"#7dcfff", 117},
		Label:            Tone{"#bb9af7", 140},
		Success:          Tone{"#9ece6a", 149},
		Warning:          Tone{"#e0af68", 179},
		Error:            Tone{"#f7768e", 211},
		Match:            Tone{"#e0af68", 179},
		Selected:         Tone{"#283457", 237},
		SelectedInactive: Tone{"#21283f", 236},
		Surface:          Tone{"#20253a", 235},
		Background:       Tone{"#1a1b26", 234},
	},
}
