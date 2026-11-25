# 0001. Adopt a token based design system for the terminal UI

**Date**: 2026-11-25
**Status**: Accepted

## Summary

burrow-G's look gets one written standard: named colors (tokens) that every theme must fill, a named set of symbols with a plain ASCII version, fixed sizes for the layout, a small kit of shared drawing pieces, and rules for wording. It codifies the UI you have today, then closes the gaps: colors that hold up on 256 and 16 color terminals and with `NO_COLOR`, minimum contrast for every theme, a light default on light terminals, and sensible behavior in small windows. You build it in four thin steps, each one green and looking the same or better.

## Decision

**Chosen option**: Option 1: Codify the current UI as tokens plus shared primitives, rolled out in thin steps.

Colors, symbols and sizes live as tokens in `internal/tui/theme`; a new `internal/tui/ui` package draws every shared piece from a `theme.Theme`; components use only these two and never pick colors or symbols themselves.

**Implementation skills**: `bubbletea` (`ggprompts/tfe`, `.agents/skills/bubbletea/`) · `golang-testing` (`samber/cc-skills-golang`, `.agents/skills/golang-testing/`, standard `testing` only, per `AGENTS.md`)

## Rationale

Reasoning, options and the contrast audit: see [rationale.md](rationale.md).

## Standard definition

### 1. Packages and the one dependency rule

| Package | Owns | May import |
|---|---|---|
| `internal/tui/theme` | palettes, color profile handling, glyph sets, layout tokens, `Theme` (ready styles) | lipgloss, termenv (never bubbles) |
| `internal/tui/ui` (new) | primitives: `Row`, `PaneHeader`, `Divider`, `VRule`, `KeyHints`, `StatusDot`, `Dialog`, `Overlay`, `EmptyState` | theme, lipgloss, `charmbracelet/x/ansi`, `bubbles/key` |
| `internal/tui/components/*`, `internal/tui/*.go` | screens and state | ui, theme (styles and tokens only) |

Rule: outside `theme/` no code names a color (`lipgloss.Color`, a hex string, an ANSI index) or a literal glyph from the table in section 4. Layout code may still call `lipgloss.NewStyle()` for width and height only (no colors).

### 2. Color tokens

Every palette fills all 14 slots. A slot is a `Tone` with a truecolor hex and a hand picked 256 index; the 16 color value is not per theme, it comes from the shared map below.

```go
type Tone struct {
	Hex  string // truecolor, "#rrggbb"
	X256 uint8  // xterm 256 index, 16..255, hand picked per theme
}
```

| Slot | Used for | Min contrast | 16 colors (shared) | No color (attributes) |
|---|---|---|---|---|
| `Text` | titles, values | 4.5:1 | terminal default fg | plain |
| `Muted` | secondary text, hints, timestamps | 3:1 | default fg + faint | faint |
| `Border` | `│ ─` rules, dialog frame | 3:1 | default fg + faint | faint |
| `Accent` | focused pane header, selection marker, keys in hints | 4.5:1 | 4 (blue) | bold |
| `Info` | folders | 4.5:1 | 6 (cyan) | plain |
| `Label` | details labels (`id`, `folder`) | 4.5:1 | 5 (magenta) | plain |
| `Success` | `●` ok status | 4.5:1 | 2 (green) | plain |
| `Warning` | `★`, busy spinner | 4.5:1 | 3 (yellow) | plain |
| `Error` | error status | 4.5:1 | 1 (red) | plain |
| `Match` | search match inside a title | 4.5:1 | 3 (yellow) + underline | underline |
| `Selected` | row background, focused pane | (a background) | none, use reverse | reverse |
| `SelectedInactive` | row background, unfocused pane | (a background) | none, marker only | marker only |
| `Surface` | dialog and command bar suggestions background | (a background) | none (plain rows, dialog keeps its frame) | none |
| `Background` | whole window | (a background) | not painted | not painted |

Contrast is the WCAG 2 contrast ratio from relative luminance. At 256 colors it is computed from the standard xterm 256 RGB table. The minimum applies to every foreground on every background it is actually drawn on, at truecolor and at 256:

| Background | Foregrounds drawn on it |
|---|---|
| `Background` | every foreground slot |
| `Selected` | Text, Muted (time ago), Warning (`★`), Accent (marker), Info, Match |
| `SelectedInactive` | Text, Muted, Warning, Muted marker, Info, Match |
| `Surface` | Text, Muted, Accent (dialog title, keys), Info (folders in suggestions), Border |

A palette that fails is fixed by nudging the failing foreground's HSL lightness (same hue and saturation), away from the background, until it passes on **every** background in this table. The backgrounds are never nudged. Checked this way, 16 of the 17 palettes change (58 colors); only github-dark passes as it is. The largest moves are Muted on dark themes, because the selected row is lighter than the window: tokyo-night Muted `#565f89` becomes `#737ca7` (256: 103), dracula `#6272a4` becomes `#8693b9`. The light themes get darker status colors (catppuccin-latte Warning `#df8e1d` becomes `#7d4f10`). The full before and after list is in `rationale.md`, and each nudged value carries a `// was #…` comment in `palettes.go`.

Where the new slots start, before any nudge: `Border` from `Muted`, `Match` from `Warning`, `SelectedInactive` a 50% mix of `Background` and `Selected`, `Surface` a 40% mix. Each mix is rounded to hex first, and contrast is checked on the rounded value.

The 256 indexes come from a search, not from each color's nearest index on its own. For each background, take its few nearest indexes, all four distinct. For every combination, give each foreground the nearest index that passes all its pairs at 256. Keep the combination with the smallest total distance (CIE Lab), where background distance counts double. Picking each background alone gave tokyo-night a pink Accent. Some light themes still land far from their hex at 256 (catppuccin-latte Accent is index 18; rose-pine-dawn Text and Accent share index 24). That is accepted, because the contrast must hold.

### 3. Color profiles

Resolved once in `tui.Run`, before `tea.NewProgram`, into `theme.Env{Depth, ASCII}`. `Depth` comes from `lipgloss.DefaultRenderer().Output().EnvColorProfile()`:

| `Depth` | When | Colors | Background painting |
|---|---|---|---|
| `TrueColor` | profile is `TrueColor` | `Tone.Hex` | yes, `Background.Hex` |
| `Color256` | profile is `ANSI256` | `Tone.X256` | yes, `Background.X256` |
| `Color16` | profile is `ANSI` | shared 16 map | no, keep the terminal's own |
| `NoColor` | profile is `Ascii` (`NO_COLOR` set, or no color support) | none, attributes column | no |

One source of truth: `theme.Get(name, env)` resolves each `Tone` to one concrete `lipgloss.Color` for `env.Depth` (no `CompleteColor`, which would read the global renderer instead), and at `Color16` and `NoColor` swaps background colors for attributes (reverse, marker only). `tui.Run` then sets the renderer to match with `lipgloss.SetColorProfile`: the same profile, except `NoColor` sets `termenv.ANSI`, because at `Ascii` termenv drops every attribute (bold, faint, reverse, underline) along with the colors. The `NoColor` styles carry no colors, so nothing colored is ever emitted. `background.set`/`restore` run only when painting happened. Tests build a `Theme` with a fixed `Env` and set the renderer profile the same way.

### 4. Glyph set

One `GlyphSet` struct with two values, `Unicode` and `ASCII`. Components read `t.Glyph.<Name>`.

| Name | Unicode | ASCII | Used for |
|---|---|---|---|
| `Marker` | `▌` | `>` | selected row |
| `Star` | `★` | `*` | bookmark |
| `Dot` | `●` | `o` | status |
| `VLine` / `HLine` | `│` / `─` | `\|` / `-` | rules |
| `Sep` | `·` | `.` | between parts and hints |
| `Up` `Down` `Left` `Right` | `↑ ↓ ← →` | `^ v < >` | key hints |
| `Ellipsis` | `…` | `...` | cut text, in progress |
| `Times` | `×` | `x` | sizes (`40×8`) |
| `Corners` | `╭ ╮ ╰ ╯` | `+ + + +` | dialog frame |
| `Spinner` | `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏` at 12 fps (the frames of `spinner.MiniDot`) | `\| / - \` at 10 fps (`spinner.Line`) | busy status; stored as `[]string` frames plus fps so `theme` never imports bubbles |

ASCII is chosen when the first set variable of `LC_ALL`, `LC_CTYPE`, `LANG` has no `UTF-8`/`utf8` (any case), for example `C`, `POSIX`, `en_US.ISO-8859-1`. All three unset means Unicode. No setting, no command.

### 5. Layout tokens and small windows

| Token | Value | Meaning |
|---|---|---|
| `Gutter` | 1 | left padding of every row and header |
| `AgentsWidth` | 20 | agents pane width (fixed, today's value) |
| `DetailsHeight` | 4 | details box lines |
| `LabelWidth` | 8 | details label column |
| `NarrowWidth` | 60 | below: one pane at a time |
| `ShortHeight` | 12 | below: details box hidden |
| `MinWidth` × `MinHeight` | 40 × 8 | below: only the "too small" note |

Steps, checked on every `WindowSizeMsg`: width below 60 shows only the focused pane at full width (agents or sessions), headers unchanged, `←`/`→` switch as today, and the hints gain `← agents`. Height below 12 hides details and gives its lines to the list. Below 40 × 8 the whole screen is `EmptyState("make the window bigger", "needs 40×8")`; keys still work (`q` quits).

### 6. Primitives (`internal/tui/ui`)

```go
func Row(t theme.Theme, left, right []theme.Part, width int, selected, active bool) string // moved from theme
func SurfaceRow(t theme.Theme, left, right []theme.Part, width int, selected bool) string // command bar suggestions: Surface fill through gap and padding
func PaneHeader(t theme.Theme, title string, focused bool) string // " Agents", Accent bold if focused, Muted bold if not
func Divider(t theme.Theme, width int) string                     // HLine × width in Border
func VRule(t theme.Theme, height int) string                      // VLine column in Border
func KeyHints(t theme.Theme, keys ...key.Binding) string          // "↑/↓ move · enter open": key in Accent, words in Muted, Sep between
func StatusDot(t theme.Theme, s Status) string                    // Dot or spinner frame + text, styled by s.Kind (OK, Busy, Error)
func Dialog(t theme.Theme, title string, body []string, keys []key.Binding, width int) string
func Overlay(base, box string, width, height int) string          // splice box centered into base lines, ansi.Cut keeps escapes intact
func EmptyState(t theme.Theme, message, hint string, width, height int) string
```

`Dialog`: rounded frame in Border from `Corners`, filled with Surface, one column of padding, title in Accent bold, body in Text (wrapped to the inner width, never cut), `KeyHints` last. Width is `max(24, min(50, bodyWidth-4))`. Shown with `Overlay` over the body, so the list stays visible around it. `Overlay` clamps: a box wider or taller than the base is cut to the base size, never grows it.

`SurfaceRow` at `Color16` and `NoColor` is a plain row (no fill); its selection uses reverse, like `Row`. First user is the upcoming delete confirm: title `delete session`, body the quoted title and folder, keys `y delete · n cancel`. Help stays in the right pane, not a dialog.

### 7. States

| State | Look |
|---|---|
| selected, pane focused | Accent `Marker` + Selected background, Text bold |
| selected, pane unfocused | Muted `Marker` + SelectedInactive background |
| focused pane header | Accent bold; unfocused header Muted bold |
| busy | `StatusDot` with spinner in Warning + `refreshing…` / `opening…` |
| done | `●` Success + past tense (`refreshed`, `opened "x" in a new tab`) |
| error | `●` Error + `error: <message>` |
| search match | Match on the matched runes of the title only (folder matches are not highlighted; the folder is not in the row) |
| empty, agent has no sessions | `no sessions yet` · hint `start one in <Agent> and press r` |
| empty, search found nothing | `no matches for "<query>"` · hint `esc clears the search` |
| empty, no agent installed | `no agents found` · hint `burrow-g reads ~/.claude, ~/.qoder, ~/.codex, …` |
| empty, details with no session | `no sessions` in Muted (today's text) |

**Spinner ownership.** `statusbar` owns a `spinner.Model` built from `t.Glyph.Spinner`. `Busy(text)` starts it and returns its first tick command; `OK` and `Error` stop it. Ticks carry the spinner's ID, so a tick from a stopped spinner is dropped and nothing ticks while idle. `newModel` sets Busy before the program starts, so `model.Init` returns the status bar's tick alongside the first refresh.

**Match positions.** The list filter returns byte offsets into the filter text `Title + " " + ShortPath`. `sessionlist` converts them to rune positions, keeps those inside the title, maps them through whitespace collapsing and truncation, and drops any past the cut; the delegate styles those runes with `t.Match`.

Every meaning a color carries is also carried by a glyph or a word, so `NO_COLOR` and color blind users lose nothing.

### 8. Copy and key hints

- Lowercase, no trailing period: `refreshed`, `theme "nord"`.
- Titles and user values in double quotes: `bookmarked "Fix flaky test"`.
- Done is past tense; in progress ends with `Ellipsis`: `refreshing…`.
- Errors start with `error: ` and keep the underlying message.
- Hints are `key action` pairs joined by ` Sep `, keys spelled as in `keys.go` (`enter`, `esc`, `ctrl+k`), arrows from the glyph set.
- Hints come from the bindings: every binding in `keys.go` gets `key.WithHelp(key, action)`, with arrows written as glyph names that `KeyHints` fills from `t.Glyph`. The status line hints and the help command's key list (`command.Env.Keys`) are both built from these bindings, never from string constants split on `" · "`.
- Under `Color16` and `NoColor`, `theme <name>` still saves the choice and reports `theme "<name>" (colors need truecolor or 256)`, since every theme then looks the same.
- Counts after a title use ` Sep `: `Claude Code · 60`.

### 9. First run theme

`theme.Initial(saved string, dark func() bool) string`: a saved, known name wins and `dark` is never called; otherwise `dark()` picks `Default` (`tokyo-night`) or `DefaultLight` (`catppuccin-latte`). `dark` reads the terminal background that `newBackground` already queried (`bg.orig`, lightness below 0.5 means dark), so there is one terminal query at startup, not two, and it runs in `tui.Run` before the program starts (the query reads from the terminal and would race with Bubble Tea's input). The pick is not saved, so a later terminal change is picked up; only `theme <name>` saves. When the terminal does not answer, or runs inside tmux or screen (termenv skips the query there), the result is dark, so `tokyo-night`, today's behavior. Light terminals inside tmux therefore start dark; `theme catppuccin-latte` fixes it once.

### 10. Resolved inputs

| Value | Source |
|---|---|
| color depth | `termenv.Output.EnvColorProfile()` of the Lip Gloss default renderer (honors `NO_COLOR`, `CLICOLOR_FORCE`), mapped to `theme.Depth` |
| glyph set | `LC_ALL`, `LC_CTYPE`, `LANG` via an injected `getenv` |
| dark or light terminal | lightness of `bg.orig` (the background `newBackground` read), injected as `func() bool` |
| match positions | the list filter's matched indexes, converted as in section 7 |
| spinner state | `spinner.Model` inside `statusbar` |
| hint and help key text | `key.Binding.Help()` of the bindings in `keys.go`, arrows from `t.Glyph` |
| saved theme | `settings` table key `theme` (exists, no schema change) |
| 16 color values | shared map in `theme/` (section 2), not per palette |
| 256 values | `Tone.X256` per palette, hand picked in rollout step 1 |
| window size | `tea.WindowSizeMsg` |
| empty state agent name | `agentlist.DisplayName(m.agent.Name())` in `layout.go`, which draws the "no sessions yet" state |
| search query in "no matches" | `list.Model.FilterValue()` inside `sessionlist` |

### Canonical pattern

```go
// theme/palettes.go: a palette fills every slot with both depths.
// Struct fields need the Tone type spelled out; Go only drops it in maps and slices.
"tokyo-night": {
	Text:       Tone{"#c0caf5", 189},
	Muted:      Tone{"#737ca7", 103}, // was #565f89
	Accent:     Tone{"#7aa2f7", 111},
	Selected:   Tone{"#283457", 237},
	Background: Tone{"#1a1b26", 234},
	// … all 14 slots
},

// components/sessionlist/model.go: the row delegate draws with ui and tokens only.
func (d delegate) Render(w io.Writer, m list.Model, index int, li list.Item) {
	t := d.theme
	s := li.(item).s
	star := theme.Part{Text: "  "}
	if s.Bookmarked {
		star = theme.Part{Text: t.Glyph.Star + " ", Style: t.Star}
	}
	ago := helpers.TimeAgo(s.LastUsed, time.Now())
	title := helpers.Truncate(s.Title, m.Width()-len(ago)-6, t.Glyph.Ellipsis) // collapses whitespace, then cuts
	left := []theme.Part{star, {Text: title}}
	right := []theme.Part{{Text: ago, Style: t.Muted}}
	fmt.Fprint(w, ui.Row(t, left, right, m.Width(), index == m.Index(), d.active))
}

// The search empty state lives in sessionlist (it knows the query);
// "no sessions yet" lives in layout.go (it knows the agent).
if m.Filtered() && len(m.list.VisibleItems()) == 0 {
	return ui.EmptyState(t, `no matches for "`+m.list.FilterValue()+`"`, "esc clears the search", m.width, m.height)
}
```

**Replaces**:
- Literal glyphs in components and layout (`"▌"`, `"★ "`, `"● "`, `strings.Repeat("─", w)`, `"│\n"`, `" · "`, `"↑/↓"`).
- Hand built headers and rules in `layout.go` (`t.Header.Render(" Agents")`, `t.Muted.Render(strings.Repeat("─", …))`).
- `theme.Row` in `theme/` (moves to `ui.Row`).
- Hex only `Palette` fields of type `lipgloss.Color`, and Muted doubling as the border color.
- `"● " + text` in `statusbar`; the static `refreshing…` dot.
- Hard coded `leftWidth`, `details.Height` and label `Width(8)` outside the token table.
- Hint string constants in `keys.go` and `keyLines` splitting them on `" · "`.

**Enforcement**:
Palette tests in `internal/tui/theme/theme_test.go`: every palette fills all 14 slots with a valid hex and a 256 index in 16..255 (checked by reflection, so a new slot can't be skipped); every palette passes the section 2 minimums for every pair in the "drawn on" table, at truecolor and at 256; the shared 16 map covers every slot. The package rule in section 1 is enforced by `internal/tui/AGENTS.md` and review (no source scanning test, by choice).

**Rollout** (thin steps, each merges green and looks the same or better):
1. Tokens: `Tone`, 14 slots, 256 values, shared 16 map, `theme.Env` and `Depth` with the renderer profile rule, background rule, glyph sets with locale detection, layout tokens, nudged palettes, first run theme, palette tests.
2. Primitives: create `ui`, move `Row`, add `SurfaceRow`, `PaneHeader`, `Divider`, `VRule`, `KeyHints`, `StatusDot`; add `key.WithHelp` to `keys.go` and build hints and `command.Env.Keys` from it; move `layout.go`, `statusbar`, `details`, `agentlist`, `sessionlist`, `commandbar` onto them. Add `Dialog` and `Overlay` with tests (first used by delete).
3. States: spinner, unfocused selection and header, search match, empty states, small window steps, the `theme` message under 16 colors and no color.
4. Reference: write `docs/design.md` (token tables, glyph table, primitives, states, voice), and point `internal/tui/AGENTS.md` at it.

**Exceptions**:
- `burrow-g list` and other plain CLI output in `cmd/` keep printing plain text. `helpers.Truncate` gains a tail parameter: the CLI passes `…`, the TUI passes `t.Glyph.Ellipsis`. The tail counts toward the width (cut at `max - runes(tail)`), so the three character ASCII `...` never overflows a row.
- `internal/command` help text stays plain words; the TUI styles it when shown.
- Tests may use literal glyphs and colors in expected output.
- Out of scope: user defined theme files, mouse input, animation beyond the spinner.

## Consequences

**Positive**:
- Every theme is readable: text at 4.5:1, secondary at 3:1, checked by tests, on truecolor and 256 color terminals.
- Light terminal users start on a light theme; `NO_COLOR` and ASCII locales get a usable, honest UI.
- New screens (delete confirm, future agents) reuse `Dialog`, `EmptyState` and `KeyHints` instead of hand built strings.
- One place to change a symbol, a size or a color.

**Negative / tradeoffs**:
- 17 palettes × 14 slots × a hand picked 256 index is real upfront work, repeated for every new theme.
- Nudged palettes are no longer the exact upstream hex. 16 of 17 themes change, most visibly a brighter Muted on dark themes and darker status colors on light ones; people who know a theme well will notice.
- At 256 colors some light themes drift far from their truecolor look, because few xterm indexes pass 4.5:1 on a light selected row.
- 4 new slots (Border, Surface, SelectedInactive, Match) per palette; theme authors must think about them.
- The package rule is checked by review only, so stray literals can slip in.
- The spinner adds periodic redraws while busy.

**Neutral**:
- No storage change; the `settings` key `theme` keeps its meaning.
- `x/ansi` moves from indirect to direct dependency (already in `go.sum`).
- `theme.Get` gains an environment parameter; tests pass a fixed `Env` and set the renderer profile to match.
- Light terminals inside tmux or screen start on the dark default, because termenv cannot ask the terminal there.
- At 16 colors and no color all 17 themes look the same; the choice still saves for truecolor sessions.

## Follow-up

- [ ] Delete sessions: its spec should use `Dialog` and the copy rules here for the confirm.
- [ ] After step 4, `internal/tui/AGENTS.md` should point to `docs/design.md` and state the package rule (via `/sync`).
- [ ] Consider a source guard test later if review misses stray colors or glyphs.
- [ ] README theme list and "terminal background" note need updating after step 1 (light default, 16 color behavior).
