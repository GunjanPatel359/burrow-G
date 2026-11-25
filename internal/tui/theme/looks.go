package theme

// Look is how a slot is drawn when the palette's own colors are not used.
type Look struct {
	Color     string // ANSI index "0".."15"; empty keeps the terminal's own color
	Faint     bool
	Bold      bool
	Underline bool
	Reverse   bool
}

// ansi16 is shared by every theme: with 16 colors the terminal decides what
// "red" looks like, so per theme choices would be overridden anyway.
var ansi16 = map[string]Look{
	"Text":             {},
	"Muted":            {Faint: true},
	"Border":           {Faint: true},
	"Accent":           {Color: "4"},
	"Info":             {Color: "6"},
	"Label":            {Color: "5"},
	"Success":          {Color: "2"},
	"Warning":          {Color: "3"},
	"Error":            {Color: "1"},
	"Match":            {Color: "3", Underline: true},
	"Selected":         {Reverse: true},
	"SelectedInactive": {}, // the marker alone shows the selection
	"Surface":          {},
	"Background":       {},
}

// noColor carries every meaning with attributes; glyphs and words do the rest.
var noColor = map[string]Look{
	"Text":             {},
	"Muted":            {Faint: true},
	"Border":           {Faint: true},
	"Accent":           {Bold: true},
	"Info":             {},
	"Label":            {},
	"Success":          {},
	"Warning":          {},
	"Error":            {},
	"Match":            {Underline: true},
	"Selected":         {Reverse: true},
	"SelectedInactive": {},
	"Surface":          {},
	"Background":       {},
}
