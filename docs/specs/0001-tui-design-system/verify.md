# Verify: TUI design system · spec 0001 · updated 2026-11-25

_The spec has no numbered acceptance criteria, so these steps come from its standard definition (sections 2, 3, 4 and 9) and rollout step 1. `/check verify` runs them; `/test` locks the ones that last._

## Step 1: tokens

### Commands
- [ ] `make check` → gofmt clean, vet clean, every test passes (spec, Enforcement)
- [ ] `go test ./internal/tui/theme -run 'TestEveryPaletteFillsEverySlot|TestPaletteContrast|TestLooksCoverEverySlot' -v` → all 17 palettes fill 14 slots and pass every "drawn on" pair at truecolor and 256 (section 2)

### Terminal, by hand (run `make build`, then `./bin/burrow-g`)
- [ ] Truecolor terminal (`COLORTERM=truecolor`) → looks like before; muted text (time ago, hints) is a little brighter; the window background is painted, then restored on quit (section 3)
- [ ] 256 colors (`COLORTERM= TERM=xterm-256color ./bin/burrow-g`) → theme colors close to truecolor, background painted with the 256 index, still readable on the selected row (section 3)
- [ ] 16 colors (`COLORTERM= TERM=xterm ./bin/burrow-g`) → your terminal's own background stays; errors red, stars yellow, folders cyan; the selected row is reversed; `theme nord` changes nothing visible (sections 2, 3)
- [ ] `NO_COLOR=1 ./bin/burrow-g` → no colors anywhere; the selected row is reversed, headers bold, muted text faint; the background is never touched (section 3)
- [ ] `LANG=C LC_ALL= ./bin/burrow-g` → the selection marker is `>` instead of `▌` (section 4; other glyphs move in step 2)
- [ ] Light terminal background, no theme saved (`sqlite3` the index, or a fresh `HOME`) → starts on `catppuccin-latte`; dark terminal → `tokyo-night` (section 9)
- [ ] Same light terminal inside tmux → starts on `tokyo-night` (termenv can't ask there, so it counts as dark) (section 9)
- [ ] `theme nord` saved, then a light terminal → still `nord` (section 9)

## Value sourcing coverage
- color depth ← `EnvColorProfile()`: the four terminal steps above
- glyph set ← `LC_ALL`, `LC_CTYPE`, `LANG`: the `LANG=C` step, plus `TestASCIILocale`
- dark or light ← `bg.orig`: the light, dark and tmux steps, plus `TestInitial` and `TestFirstThemeFollowsTerminal`
- 256 values ← `Tone.X256`: the 256 step, plus `TestPaletteContrast`
- 16 color values ← shared map: the 16 color step, plus `TestGet_Color16`
- saved theme ← `settings` key `theme`: the saved `nord` step

## Step 2: primitives

### Commands
- [ ] `go test ./internal/tui/ui/ -v` → Row, SurfaceRow, PaneHeader, Divider, VRule, KeyHints, StatusDot, Dialog and Overlay tests pass (section 6)
- [ ] `go test ./internal/tui/ -run TestHintsComeFromBindings` → status hints and the help command's key list both come from `key.Binding` help, arrows from the glyph set (section 8)
- [ ] `rg -n '[▌★●│─·↑↓←→…]|lipgloss\.Color\(' --type go internal/tui | rg -v '_test.go|internal/tui/theme/'` → only comments match (section 1)

### Terminal, by hand
- [ ] Press `→`: the "Agents" header turns Muted bold and the sessions header Accent bold; `←` swaps them back (sections 6, 7)
- [ ] The status line hints draw keys in Accent and words in Muted, joined by `·`, for example `↑/↓ move · enter resume` (section 8)
- [ ] `ctrl+k`: the suggestions sit on the Surface background, filled through the gap and the right edge; the selected one uses the Selected background with the marker (section 6)
- [ ] `ctrl+k`, `help`, `enter`: the Keys list shows each hint once, spelled like the status line (section 8)
- [ ] `LANG=C LC_ALL= ./bin/burrow-g`: arrows read `^/v` and `</>`, rules are `|` and `-`, the separator is `.`, the star is `*`, long titles end in `...` and still fit the row (sections 4, Exceptions)
- [ ] `./bin/burrow-g list` (CLI): long titles still end in `…` (Exceptions)
- [ ] Dialog and Overlay have no user yet (delete will be the first); their look is covered by `TestDialog` and `TestOverlay` (section 6)

## Step 3: states

### Commands
- [ ] `go test ./internal/tui/components/statusbar/ -run TestSpinner -v` → Busy starts one spinner, a second Busy does not start another, ticks after OK and ticks from an earlier spinner are dropped (section 7, Spinner ownership)
- [ ] `go test ./internal/tui/components/sessionlist/ -run 'TestTitleParts|TestSearchHighlightsTitle|TestNoMatches' -v` → matched title runes drawn in Match, folder matches and runes past the cut never highlighted, `no matches for "…"` keeps the search box (section 7, Match positions)
- [ ] `go test ./internal/tui/ -run 'TestNarrowWindowShowsOnePane|TestShortWindowHidesDetails|TestTooSmallWindow|TestEmptyStates|TestBusyStatusSpins|TestThemeWithoutColorsSaysWhy' -v` → the small window steps, empty states, spinner start and theme message (sections 5, 7, 8)
- [ ] `go test ./internal/tui/ui/ -run 'TestEmptyState|TestRow_Unfocused' -v` → EmptyState centers and cuts; an unfocused selection is a Muted marker on SelectedInactive, marker only at 16 colors (sections 6, 7)

### Terminal, by hand
- [ ] Start `./bin/burrow-g`: the status dot spins (`⠋⠙⠹…`) while refreshing, then turns into a still `●` with `refreshed`; nothing moves after that (section 7)
- [ ] `LANG=C LC_ALL= ./bin/burrow-g`: the spinner is `| / - \` (section 4)
- [ ] With the agents pane focused, the selected session row keeps a Muted `▌` and a soft background; press `→` and it turns into the Accent marker on the Selected background, the agents row dims the same way (section 7)
- [ ] `→`, `/`, type `auth`: the matched letters in each title are drawn in Match (yellow on most themes); a session found only by its folder shows no highlight (section 7)
- [ ] `/`, type `zzzz`: `no matches for "zzzz"` and `esc clears the search` sit centered under the search box; `esc` brings the list back (section 7)
- [ ] Pick an agent with no sessions (for example one you never used): `no sessions yet` and `start one in <Agent> and press r` (section 7)
- [ ] Narrow the window below 60 columns: only the focused pane shows at full width, `→` and `←` switch panes, `← agents` is in the hints (section 5)
- [ ] Make it shorter than 12 lines: the details box goes away and the list takes its lines (section 5)
- [ ] Make it smaller than 40 × 8: only `make the window bigger` and `needs 40×8` show; `q` still quits; growing the window brings everything back (section 5)
- [ ] `COLORTERM= TERM=xterm ./bin/burrow-g`, then `ctrl+k`, `theme nord`: the status says `theme "nord" (colors need truecolor or 256)`; restart in truecolor and nord is in use (section 8)
- [ ] Same with `NO_COLOR=1` (section 8)

### Value sourcing coverage
- match positions ← the list filter's matched byte offsets: the `auth` step, plus `TestTitleParts` (collapsed spaces, wide runes, the cut, folder offsets)
- spinner state ← `spinner.Model` in `statusbar`: the spinner steps, plus `TestSpinner` and `TestBusyStatusSpins`
- window size ← `tea.WindowSizeMsg`: the narrow, short and too small steps
- empty state agent name ← `agentlist.DisplayName`: the "no sessions yet" step, plus `TestEmptyStates`
- search query ← `list.Model.FilterValue()`: the `zzzz` step, plus `TestNoMatches`
- color depth for the theme message ← `theme.Env.Depth`: the 16 color and `NO_COLOR` steps, plus `TestThemeWithoutColorsSaysWhy`
