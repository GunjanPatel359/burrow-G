# burrow-G design system

This is the reference for how the terminal UI looks and talks. Read it before you add a screen, a theme, a symbol or a status message. The reasons behind each choice live in [spec 0001](specs/0001-tui-design-system/index.md); this page is the short version you work from.

The code is the source of truth. If this page and the code disagree, the code wins and this page needs a fix.

## The one rule

Colors, symbols and sizes live in `internal/tui/theme`. The shared drawing pieces live in `internal/tui/ui`. Everything else (components, `layout.go`, `app.go`) draws only through those two.

| Package | Owns | May import |
|---|---|---|
| `internal/tui/theme` | palettes, color depth, glyph sets, layout sizes, `Theme` (ready styles) | lipgloss, termenv (never bubbles) |
| `internal/tui/ui` | `Row`, `SurfaceRow`, `PaneHeader`, `Divider`, `VRule`, `KeyHints`, `StatusDot`, `StatusLine`, `Dialog`, `Overlay`, `EmptyState` | theme, lipgloss, `charmbracelet/x/ansi`, `bubbles/key` |
| `internal/tui/components/*`, `internal/tui/*.go` | screens and state | ui, theme (styles and tokens only) |

So outside `theme/` you never write `lipgloss.Color(...)`, a hex string, an ANSI index, or a literal symbol from the glyph table below. `lipgloss.NewStyle()` is still fine for width and height. Tests may use literal colors and symbols in expected output.

Nothing checks this automatically; review does. A quick look before you push:

```bash
rg -n '[▌★●│─·↑↓←→…×╭╮╰╯]|lipgloss\.Color\(' --type go internal/tui | rg -v '_test.go|internal/tui/theme/'
```

Only comments should match.

## Color tokens

Every palette fills all 14 slots. Each slot is a `Tone`: a truecolor hex plus a hand picked xterm 256 index. At 16 colors and without color, palettes are not used at all; every theme shares the looks in `theme/looks.go`.

| Slot | Style in `Theme` | Used for | Min contrast | 16 colors | No color |
|---|---|---|---|---|---|
| `Text` | `Text` | titles, values | 4.5:1 | terminal default | plain |
| `Muted` | `Muted` | secondary text, hints, time ago | 3:1 | faint | faint |
| `Border` | `Border` | `│ ─` rules, dialog frame | 3:1 | faint | faint |
| `Accent` | `Accent`, `Header` (bold), `Marker` | focused header, selection marker, keys in hints | 4.5:1 | 4 blue | bold |
| `Info` | `Folder` | folders | 4.5:1 | 6 cyan | plain |
| `Label` | `Label` (8 cells wide) | details labels (`id`, `folder`) | 4.5:1 | 5 magenta | plain |
| `Success` | `OK` | `●` done status | 4.5:1 | 2 green | plain |
| `Warning` | `Star`, `Busy` | `★`, busy spinner | 4.5:1 | 3 yellow | plain |
| `Error` | `Error` | error status | 4.5:1 | 1 red | plain |
| `Match` | `Match` | matched letters in a title | 4.5:1 | 3 yellow, underline | underline |
| `Selected` | `Selected` (with Text, bold) | selected row, focused pane | a background | reverse | reverse |
| `SelectedInactive` | `SelectedInactive` | selected row, unfocused pane | a background | marker only | marker only |
| `Surface` | `Surface` | dialog, command bar suggestions | a background | none | none |
| `Background` | `Background` (a color, or empty) | the whole window | a background | not painted | not painted |

Contrast is the WCAG 2 ratio, checked at truecolor and at 256 for every pair that is really drawn together:

| Background | Foregrounds drawn on it |
|---|---|
| `Background` | every foreground slot |
| `Selected` | Text, Muted, Warning, Accent, Info, Match |
| `SelectedInactive` | Text, Muted, Warning, the Muted marker, Info, Match |
| `Surface` | Text, Muted, Accent, Info, Border |

Here is the default dark theme, so you can see what a palette looks like. The other 16 are in `theme/palettes.go`.

| Slot | `tokyo-night` | 256 |
|---|---|---|
| Text | `#c0caf5` | 189 |
| Muted | `#737ca7` (was `#565f89`) | 103 |
| Border | `#626d9d` | 103 |
| Accent | `#7aa2f7` | 111 |
| Info | `#7dcfff` | 117 |
| Label | `#bb9af7` | 140 |
| Success | `#9ece6a` | 149 |
| Warning | `#e0af68` | 179 |
| Error | `#f7768e` | 211 |
| Match | `#e0af68` | 179 |
| Selected | `#283457` | 237 |
| SelectedInactive | `#21283f` | 236 |
| Surface | `#20253a` | 235 |
| Background | `#1a1b26` | 234 |

### Color depth

`tui.Run` reads the terminal once, before the program starts, into `theme.Env{Depth, ASCII}`. `theme.Get(name, env)` then resolves every slot for that depth, so nothing later guesses.

| `Depth` | When | Colors | Window background |
|---|---|---|---|
| `TrueColor` | the terminal says truecolor | `Tone.Hex` | painted, restored on quit |
| `Color256` | 256 colors | `Tone.X256` | painted with the 256 index |
| `Color16` | 16 colors | shared map | the terminal's own |
| `NoColor` | `NO_COLOR` set, or no colors | none, attributes only | the terminal's own |

Every meaning a color carries is also carried by a symbol or a word. That is why `NO_COLOR` and color blind users lose nothing: an error says `error:`, a selection has a marker, a match is underlined.

### First theme

A saved theme always wins. With none saved, a light terminal starts on `catppuccin-latte` and a dark one on `tokyo-night`. The pick is not saved, so changing terminals is followed; only `theme <name>` saves. Inside tmux or screen the terminal can't be asked, so it starts dark.

## Glyphs

Read every symbol from `t.Glyph`. When the locale is not UTF-8 (the first set one of `LC_ALL`, `LC_CTYPE`, `LANG`, for example `C`), the ASCII set is used instead.

| Name | Unicode | ASCII | Used for |
|---|---|---|---|
| `Marker` | `▌` | `>` | selected row |
| `Star` | `★` | `*` | bookmark |
| `Dot` | `●` | `o` | status |
| `VLine` / `HLine` | `│` / `─` | `\|` / `-` | rules |
| `Sep` | `·` | `.` | between parts and hints |
| `Up` `Down` `Left` `Right` | `↑ ↓ ← →` | `^ v < >` | key hints |
| `Ellipsis` | `…` | `...` | cut text, work in progress |
| `Times` | `×` | `x` | sizes, as in `40×8` |
| `Corners` | `╭ ╮ ╰ ╯` | `+ + + +` | dialog frame |
| `Spinner` | `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏` at 12 fps | `\| / - \` at 10 fps | busy status |

The ASCII ellipsis is three cells wide. `helpers.Truncate` counts the tail inside the width, so a cut title still fits its row.

## Layout

Sizes are in cells, from `theme/layout.go`.

| Token | Value | Meaning |
|---|---|---|
| `Gutter` | 1 | left padding of every row and header |
| `AgentsWidth` | 20 | agents pane width |
| `DetailsHeight` | 4 | details box lines |
| `LabelWidth` | 8 | details label column |
| `NarrowWidth` | 60 | below this, one pane at a time |
| `ShortHeight` | 12 | below this, the details box is hidden |
| `MinWidth` × `MinHeight` | 40 × 8 | below this, only the "too small" note |

The screen is two panes side by side, a divider, then the status line. As the window shrinks:

1. Narrower than 60: only the focused pane shows, at full width. `←` and `→` switch panes as usual.
2. Shorter than 12: the details box goes away and the list takes its lines.
3. Smaller than 40 × 8: only `make the window bigger` and `needs 40×8`. Keys still work, so `q` quits.

## Primitives

Draw shared pieces with these, from `internal/tui/ui`. Each takes the `theme.Theme` first.

| Function | Draws |
|---|---|
| `Row(t, left, right, width, selected, active)` | one list row: left parts, right parts at the end, marker and highlight when selected. `active` is whether its pane has focus. |
| `SurfaceRow(t, left, right, width, selected)` | a row on the Surface background, filled through the gap and padding. Command bar suggestions. |
| `PaneHeader(t, title, focused)` | ` Agents`: Accent bold when focused, Muted bold when not. |
| `Divider(t, width)` | `─` across the width, in Border. |
| `VRule(t, height)` | a `│` column, in Border. |
| `KeyHints(t, bindings...)` | `↑/↓ move · enter resume`: keys in Accent, words in Muted, `Sep` between. Bindings that are off are left out. |
| `HintText(g, binding)` | the same hint as plain text, for the help command's key list. |
| `KeyName(g, help)` | a help key with arrows from the glyph set: `up/down` becomes `↑/↓`. |
| `StatusDot(t, status)` | the dot (or spinner frame) and text, colored by `OK`, `Busy` or `Error`. |
| `StatusLine(t, status, width, bindings...)` | the status line: the status on the left, the key hints flush against the right edge, so they stay put while the status comes and goes. Hints that don't fit are left out from the end; with none left, the status is cut. |
| `Dialog(t, title, body, keys, width)` | a rounded frame on Surface: title in Accent bold, body in Text (wrapped, never cut), key hints last. 24 to 50 cells wide. |
| `Overlay(base, box, width, height)` | puts a box in the middle of a screen, keeping the rest visible. Never makes the screen bigger. |
| `EmptyState(t, message, hint, width, height)` | a message and a Muted hint, centered. Lines that don't fit are cut. |

Row text is a list of `theme.Part{Text, Style}`. An empty style uses the row's own color, and every part is drawn on top of the row's style, so the selected background runs through colored parts like the star.

A row from `sessionlist` looks like this:

```go
left := append([]theme.Part{star}, title...)
right := []theme.Part{{Text: ago, Style: t.Muted}}
fmt.Fprint(w, ui.Row(t, left, right, m.Width(), index == m.Index(), d.active))
```

`Dialog` and `Overlay` have no user yet. Deleting a session will be the first: title `delete session`, body the quoted title and folder, keys `y delete · n cancel`.

## States

| State | Look |
|---|---|
| selected, pane focused | Accent `▌` on Selected, Text bold |
| selected, pane unfocused | Muted `▌` on SelectedInactive (16 colors and no color: the marker alone) |
| pane header | Accent bold when focused, Muted bold when not |
| busy | spinner in Warning, then `refreshing…` or `opening…` |
| done | `●` in Success, then past tense: `refreshed`, `opened "x" in a new tab`. Clears itself after 2 s (`statusbar.FadeAfter`) unless something newer replaced it; errors stay until replaced |
| error | `●` in Error, then `error: <message>` |
| search match | matched letters of the title in Match. Matches only in the folder, or in the part cut off, are not shown. |
| agent has no sessions | `no sessions yet` · `start one in <Agent> and press r` |
| search found nothing | `no matches for "<query>"` · `esc clears the search`, under the search box |
| no agents | `no agents found` · `burrow-g reads ~/.claude, ~/.qoder, ~/.codex, …` |
| details with no session | `no sessions` in Muted |
| window too small | `make the window bigger` · `needs 40×8` |

**The spinner.** `statusbar` owns it. `Busy(text)` starts a new spinner and returns its first tick; calling it again while busy only changes the text. `OK`, `Error` and `Clear` stop it. `OK` returns the command that fades its message; return it from `Update` like any other. Ticks from a stopped or older spinner are dropped, so nothing redraws while idle. In `app.go`, batch the command `Busy` returns with the job it describes.

**Empty states.** Each one lives where its words come from. The search one lives in `sessionlist`, because it knows the query. The agent ones live in `layout.go`, because it knows the agent.

## Voice

The status line and empty states talk like this:

- Lowercase, no period at the end: `refreshed`, `no sessions yet`.
- Titles and values the user typed go in double quotes: `bookmarked "Fix flaky test"`, `theme "nord"`. Use `%q`.
- Done is past tense. Work in progress ends with `t.Glyph.Ellipsis`: `"refreshing" + t.Glyph.Ellipsis`.
- Errors start with `error: ` and keep the underlying message. Send them to `m.status.Error(err)`, never a panic or a print.
- A hint tells the next thing to do, in a few words: `esc clears the search`.
- Counts after a title use ` Sep `: `Claude Code · 60`.
- When something saves but can't show, say why: at 16 colors and without color, `theme nord` reports `theme "nord" (colors need truecolor or 256)`.

### Key hints

Hints are `key action` pairs. They come from the bindings in `keys.go`, never from strings:

```go
Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
Move:    key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("up/down", "move")),
```

List hints **most important first** in `keys.go` (`agentHints`, `sessionHints`): a narrow status line keeps the first ones and drops the rest, and `ctrl+k` then `help` still lists every key. The way back (`← agents`) comes right after moving and opening, because a narrow window shows one pane at a time.

Spell keys the way `keys.go` does (`enter`, `esc`, `ctrl+k`). For arrows, write `up`, `down`, `left` or `right` in the help key; `KeyHints` and `HintText` swap them for the glyph set's arrows. The status line and the help command's key list are both built from these same bindings, so they always agree.

The command bar's own help text in `internal/command` stays plain words; the TUI styles it when shown. `burrow-g list` and other CLI output stay plain text too.

## Recipes

**Add a theme.** Add a palette to `theme/palettes.go` with all 14 slots, each a `Tone{hex, index}`. Start the new slots from existing ones: Border from Muted, Match from Warning, SelectedInactive as a 50% mix of Background and Selected, Surface as a 40% mix. Run `go test ./internal/tui/theme`. If `TestPaletteContrast` fails a pair, nudge the foreground's lightness away from its background (same hue and saturation), never the background, and mark it `// was #…`. Then add the name to the README theme list.

**Add a symbol.** Add a field to `GlyphSet` in `theme/glyphs.go`, with a value in both `Unicode` and `ASCII`, and add a row to the glyph table here.

**Add a slot.** Add it to `Palette`, fill it in all 17 palettes, add it to both maps in `theme/looks.go`, give it a style in `theme.Get`, and add its pairs to `drawnOn` in `theme/palettes_test.go` (and to the table above). `TestEveryPaletteFillsEverySlot` and `TestLooksCoverEverySlot` fail until every palette and both maps fill it.

**Add a screen or a piece of one.** Build it from the primitives above. If it needs a look none of them has, add a primitive to `ui` with a test, rather than styling inside the component. Give it an empty state and check it at 40 × 8, under `LANG=C`, at 16 colors and with `NO_COLOR`.

**Check your change by hand:**

```bash
make build
./bin/burrow-g                                 # truecolor
COLORTERM= TERM=xterm-256color ./bin/burrow-g  # 256 colors
COLORTERM= TERM=xterm ./bin/burrow-g           # 16 colors
NO_COLOR=1 ./bin/burrow-g                      # no color
LANG=C LC_ALL= ./bin/burrow-g                  # ASCII glyphs
```
