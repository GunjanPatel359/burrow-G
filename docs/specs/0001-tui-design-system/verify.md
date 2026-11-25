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
