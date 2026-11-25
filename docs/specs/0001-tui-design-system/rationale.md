# 0001. Design system: rationale

Decision record for [index.md](index.md). `/develop` builds from `index.md`; this file explains why.

## Context

burrow-G is a Bubble Tea terminal UI (Go, Lip Gloss, termenv). Its look already has good bones: a `Palette` of 10 semantic colors, 17 named themes, a `theme.Theme` of ready styles, a shared `Row` helper, and no raw colors outside `internal/tui/theme`. What it lacks is a written standard, so every new screen decides its own symbols, spacing and states. Glyphs (`▌ ★ ● │ ─ · ↑ ↓`) appear as literals in five files, the layout sizes (`leftWidth = 20`, a details box of 4 lines, an 8 column label) are scattered constants, and `Muted` does double duty as the border color.

The UI also assumes the best terminal. Palettes are truecolor hex only, so 256 and 16 color terminals get whatever rounding Lip Gloss does, and with 16 colors the app still repaints the background with its own color, fighting the user's terminal theme. `NO_COLOR` turns colors off, but then selection, errors and stars only stand out where a glyph already carries them. A light terminal always starts on the dark `tokyo-night`. Small windows clip.

Contrast is the hidden problem. Measured against the minimums chosen here (4.5:1 for text, 3:1 for secondary), 10 of the 17 palettes fail at least one slot today, including the dark default (Muted at 2.8) and the chosen light default (5 slots). The upstream palettes were tuned for code syntax in editors, not for status words on a background.

Deleting sessions is next on the roadmap and needs a confirm surface. Without a standard it would be the first hand built overlay, and every later screen would copy it.

## Options considered

### Option 1: Codify the current UI as tokens plus shared primitives, in thin steps

Keep today's look as the baseline. Grow the palette into 14 slots with hand picked 256 values and a shared 16 color map, name every glyph with an ASCII variant, name the layout sizes, and add an `internal/tui/ui` package of primitives the components draw with. Roll out in four steps that each merge green.

**Pros**:
- Nothing users rely on changes by surprise; each step is small enough to review.
- Builds on what already works (`Theme`, `Row`, the theme command and live preview).
- Fixes fallbacks, contrast and small windows at the root, in one place.

**Cons**:
- Most upfront work of the options: 17 palettes × 14 slots × a 256 value.
- Four PRs instead of one; the codebase is mixed for a while.

### Option 2: Adopt a ready made component kit's styles

Use the Bubbles default styles (list delegates, help, spinner) or a styling kit as the look, and drop most of `theme`.

**Pros**:
- Less code to own; consistent with other Charm apps.

**Cons**:
- Fights the 17 themes and the custom `Row` (marker plus highlight running through colored parts).
- Fallback and contrast behavior is whatever the kit does, not something burrow-G controls or tests.
- A visible redesign nobody asked for.

### Option 3: Document only

Write the rules into `AGENTS.md` and `docs/design.md`, change no code, enforce by review.

**Pros**:
- Cheapest; no risk to the running UI.

**Cons**:
- Leaves every real gap open: no 256 or 16 color fallbacks, failing contrast, scattered glyphs.
- Rules nobody can check drift within months.

### Option 4: Option 1 as a single migration PR

Same end state, delivered at once.

**Pros**:
- No period with old and new patterns mixed.

**Cons**:
- A large diff touching every component, hard to review and hard to bisect if the UI breaks on some terminal.

## Rationale

The forces are a small team, a UI that already looks right on the author's truecolor terminal, real users on terminals that are not (16 color, `NO_COLOR`, light backgrounds, non UTF-8 locales), and a destructive feature (delete) about to land. That rules out Option 2, which trades control for a redesign, and Option 3, which records the gaps without closing any. Option 1 keeps the look people know and makes it hold everywhere; the thin rollout (over Option 4) keeps each review small and each step reversible, which matters because terminal rendering bugs only show up on someone else's setup.

The fallbacks are split on purpose. At 256 colors the palette is fixed and identical everywhere, so a hand picked index per theme pays off and can be contrast tested. At 16 colors the user's terminal decides what "red" looks like, so per theme choices would be overridden anyway; one shared semantic map (Error is red, Success is green) is what 16 color users expect, and keeping their own background avoids clashing with it. At no color, attributes (reverse, bold, faint, underline) plus glyphs and words carry every meaning.

Nudging failing colors (over relaxing status colors to 3:1 or dropping themes) keeps all 17 themes and gives them all readable status text, which is where users look after a refresh or an error. The first audit checked only against the window background. Building step 1 showed that the selected row matters as much: it is where the eye sits, and the time ago text, the star and the marker are drawn on it. Checking every pair in the "drawn on" table changes 16 palettes instead of 10, and moves Muted on dark themes much further (tokyo-night `#737ca7`, not the `#5c6692` the first audit expected). The rule was kept over the smaller drift. The other ways out were weaker: darkening the selected row until the old Muted passes makes the selection almost vanish (tokyo-night would need a Selected barely lighter than its Background), and drawing muted parts in Text on selected rows hides the difference between a title and its timestamp. The cost, themes drifting from their upstream hex, shows in Muted on the dark themes and in the status colors on the light ones. It is the honest price of them being readable. catppuccin-latte stays the light default because it pairs well with the dark default; its five failing slots get darker variants like every other failing palette.

Enforcement is the palette tests only, by choice. They guard the part that is easy to get silently wrong (a missing slot, a 256 index that fails contrast). The package rule (no colors or glyphs outside `theme`) is left to review and `AGENTS.md`; a source scanning test stays a follow up if that proves leaky.

## Contrast audit, before (2026-11-25, against Background only)

The first audit, made while designing. It checked each slot against `Background` only (plus Text on Selected), so it undercounted; the after audit below is the one the tests enforce. Ratio against Background unless noted. Minimums: 4.5 for Text, Accent, Info, Label, Success, Warning, Error; 3 for Muted; 4.5 for Text on Selected.

| Palette | Failing slots today |
|---|---|
| ayu-mirage | none |
| catppuccin-latte | Accent 4.3, Info 2.5, Success 3.0, Warning 2.3, Muted 2.3 |
| catppuccin-mocha | none |
| dracula | none |
| everforest-dark | none |
| github-dark | none |
| github-light | none |
| gruvbox-dark | Error 4.3 |
| kanagawa | none |
| monokai | Error 3.9 |
| nord | Label 4.4, Error 3.1, Muted 2.4 |
| one-dark | Error 4.4, Muted 2.3 |
| rose-pine | Success 3.4 |
| rose-pine-dawn | Accent 3.5, Info 3.1, Label 2.6, Warning 2.1, Error 3.8, Muted 2.7 |
| solarized-dark | Accent 4.1, Label 3.4, Error 3.2, Muted 2.8, Text on Selected 4.1 |
| solarized-light | Text 4.1, Accent 3.4, Info 2.9, Label 4.1, Success 3.0, Warning 3.0, Error 4.3, Muted 2.5, Text on Selected 3.6 |
| tokyo-night | Muted 2.8 |

New slots (Border, Surface, SelectedInactive, Match) are not in the audit; they are set and tested in rollout step 1.

## Contrast audit, after (2026-11-25, every "drawn on" pair, rollout step 1)

Every foreground checked on every background it is drawn on (index.md, section 2), at truecolor and at 256, with the minimums above (3 for Muted and Border). Only slots that moved from upstream are listed; Border and Match start from Muted and Warning, so they are left out. `TestPaletteContrast` in `internal/tui/theme/palettes_test.go` enforces this.

| Palette | Nudged slots (upstream → now) |
|---|---|
| ayu-mirage | Muted `#707a8c` → `#868f9e` |
| catppuccin-latte | Muted `#9ca0b0` → `#6e748a`, Accent `#1e66f5` → `#094dd3`, Info `#04a5e5` → `#026086`, Success `#40a02b` → `#327d22`, Warning `#df8e1d` → `#7d4f10` |
| catppuccin-mocha | Muted `#6c7086` → `#9093a6`, Accent `#89b4fa` → `#8fb8fa` |
| dracula | Muted `#6272a4` → `#8693b9`, Accent `#bd93f9` → `#caa8fa` |
| everforest-dark | Muted `#859289` → `#96a199`, Accent `#7fbbb3` → `#9dcbc5`, Info `#83c092` → `#9eceaa`, Warning `#dbbc7f` → `#dcbf84` |
| github-dark | none |
| github-light | Info `#1b7c83` → `#1a7980`, Warning `#9a6700` → `#956400` |
| gruvbox-dark | Muted `#928374` → `#a29588`, Accent `#83a598` → `#a6bfb5`, Info `#8ec07c` → `#98c688`, Error `#fb4934` → `#fb533f` |
| kanagawa | Muted `#727169` → `#9a9991`, Accent `#7e9cd8` → `#a7bce4`, Info `#7fb4ca` → `#96c2d4` |
| monokai | Muted `#75715e` → `#98947f`, Error `#f92672` → `#fa4989` |
| nord | Muted `#616e88` → `#8e99ae`, Accent `#88c0d0` → `#8fc4d3`, Info `#8fbcbb` → `#9ac3c2`, Label `#b48ead` → `#b690af`, Error `#bf616a` → `#cf888f` |
| one-dark | Muted `#5c6370` → `#888f9d`, Accent `#61afef` → `#70b7f1`, Info `#56b6c2` → `#66bdc8`, Error `#e06c75` → `#e17079` |
| rose-pine | Muted `#6e6a86` → `#706c88`, Success `#31748f` → `#3a8aaa` |
| rose-pine-dawn | Muted `#9893a5` → `#7f7990`, Accent `#907aa9` → `#6d5786`, Info `#56949f` → `#3c676f`, Label `#d7827e` → `#c3453f`, Warning `#ea9d34` → `#89550e`, Error `#b4637a` → `#ad546e` |
| solarized-dark | Text `#839496` → `#8b9b9d`, Muted `#586e75` → `#657e86`, Accent `#268bd2` → `#479fde`, Info `#2aa198` → `#2ca9a0`, Label `#6c71c4` → `#8387cd`, Warning `#b58900` → `#c09200`, Error `#dc322f` → `#e56562` |
| solarized-light | Text `#657b83` → `#586b72`, Muted `#93a1a1` → `#768888`, Accent `#268bd2` → `#1e6da5`, Info `#2aa198` → `#1e736d`, Label `#6c71c4` → `#6368c0`, Success `#859900` → `#687800`, Warning `#b58900` → `#836300`, Error `#dc322f` → `#da2825` |
| tokyo-night | Muted `#565f89` → `#737ca7` |

16 of 17 palettes, 58 colors. Most of the extra moves come from the selected row (Muted, Accent, Info on dark themes) and the command bar surface.

tokyo-night at 256 colors, as built: Text 189, Muted 103, Border 103, Accent 111, Info 117, Label 140, Success 149, Warning 179, Error 211, Match 179, Selected 237, SelectedInactive 236, Surface 235, Background 234. Every pair passes at 256; Muted's lowest pair is 3.3.

## Glyph inventory (before)

Literal glyphs outside tests: `─` and `│` in `layout.go`, `·` `↑` `↓` `←` `→` in `keys.go`, `▌` and `★` in `theme/theme.go`, `★` in `sessionlist`, `↑` `↓` in `commandbar`, `●` in `statusbar`, `·` in `details`, `…` in `app.go`.
