package theme

import (
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"testing"
)

var hexColor = regexp.MustCompile(`^#[0-9a-f]{6}$`)

func slotNames() []string {
	t := reflect.TypeFor[Palette]()
	names := make([]string, t.NumField())
	for i := range names {
		names[i] = t.Field(i).Name
	}
	return names
}

func tone(p Palette, slot string) Tone {
	return reflect.ValueOf(p).FieldByName(slot).Interface().(Tone)
}

// Reflection, so a new slot can't be left out of a palette.
func TestEveryPaletteFillsEverySlot(t *testing.T) {
	for name, p := range palettes {
		for _, s := range slotNames() {
			c := tone(p, s)
			if !hexColor.MatchString(c.Hex) {
				t.Errorf("%s.%s: hex %q is not #rrggbb", name, s, c.Hex)
			}
			if c.X256 < 16 {
				t.Errorf("%s.%s: 256 index %d is one of the 16 the terminal redefines", name, s, c.X256)
			}
		}
	}
}

func TestLooksCoverEverySlot(t *testing.T) {
	for _, s := range slotNames() {
		if _, ok := ansi16[s]; !ok {
			t.Errorf("ansi16 has no look for %s", s)
		}
		if _, ok := noColor[s]; !ok {
			t.Errorf("noColor has no look for %s", s)
		}
	}
}

// drawnOn lists, per background, the foregrounds actually drawn on it.
var drawnOn = map[string][]string{
	"Background":       {"Text", "Muted", "Border", "Accent", "Info", "Label", "Success", "Warning", "Error", "Match"},
	"Selected":         {"Text", "Muted", "Warning", "Accent", "Info", "Match"},
	"SelectedInactive": {"Text", "Muted", "Warning", "Info", "Match"},
	"Surface":          {"Text", "Muted", "Accent", "Info", "Border"},
}

// minContrast is 4.5:1 for text and 3:1 for secondary text and rules.
func minContrast(fg string) float64 {
	if fg == "Muted" || fg == "Border" {
		return 3
	}
	return 4.5
}

func TestPaletteContrast(t *testing.T) {
	for name, p := range palettes {
		for bg, fgs := range drawnOn {
			for _, fg := range fgs {
				f, b := tone(p, fg), tone(p, bg)
				if r := contrast(hexRGB(f.Hex), hexRGB(b.Hex)); r < minContrast(fg) {
					t.Errorf("%s: %s on %s is %.2f at truecolor, want %.1f", name, fg, bg, r, minContrast(fg))
				}
				if r := contrast(xterm(f.X256), xterm(b.X256)); r < minContrast(fg) {
					t.Errorf("%s: %s on %s is %.2f at 256 (%d on %d), want %.1f", name, fg, bg, r, f.X256, b.X256, minContrast(fg))
				}
			}
		}
	}
}

func TestContrastMath(t *testing.T) {
	if r := contrast(hexRGB("#000000"), hexRGB("#ffffff")); math.Abs(r-21) > 0.01 {
		t.Errorf("black on white = %.2f, want 21", r)
	}
	if got := fmt.Sprint(xterm(16), xterm(231), xterm(232)); got != "[0 0 0] [255 255 255] [8 8 8]" {
		t.Errorf("xterm table = %s", got)
	}
}

// contrast is the WCAG 2 contrast ratio of two sRGB colors.
func contrast(a, b [3]float64) float64 {
	x, y := luminance(a), luminance(b)
	return (max(x, y) + 0.05) / (min(x, y) + 0.05)
}

func luminance(c [3]float64) float64 {
	lin := func(v float64) float64 {
		v /= 255
		if v <= 0.04045 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(c[0]) + 0.7152*lin(c[1]) + 0.0722*lin(c[2])
}

func hexRGB(h string) [3]float64 {
	v, _ := strconv.ParseUint(h[1:], 16, 32)
	return [3]float64{float64(v >> 16 & 255), float64(v >> 8 & 255), float64(v & 255)}
}

// xterm is the standard RGB value of a 256 color index from 16 up.
func xterm(i uint8) [3]float64 {
	if i >= 232 {
		v := float64(8 + 10*(int(i)-232))
		return [3]float64{v, v, v}
	}
	level := []float64{0, 95, 135, 175, 215, 255}
	n := int(i) - 16
	return [3]float64{level[n/36], level[n/6%6], level[n%6]}
}
