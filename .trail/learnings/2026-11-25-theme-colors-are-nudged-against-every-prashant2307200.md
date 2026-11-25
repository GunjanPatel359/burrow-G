---
title: "Theme colors are nudged against every background they are drawn on, not just the window"
author: Prashant2307200
date: 2026-11-25
branch: feat/tui-design-tokens
cost: { minutes: 34, tokens: 362100 }
id: l-17c1a689
scope: repo
status: active
src: agent
touches:
  - ".agents/skills/bubbletea/SKILL.md"
  - ".agents/skills/bubbletea/references/components.md"
  - ".agents/skills/bubbletea/references/emoji-width-fix.md"
  - ".agents/skills/bubbletea/references/golden-rules.md"
  - ".agents/skills/bubbletea/references/troubleshooting.md"
  - ".agents/skills/golang-code-style/SKILL.md"
  - ".agents/skills/golang-code-style/evals/evals.json"
  - ".agents/skills/golang-code-style/references/details.md"
  - ".agents/skills/golang-database/SKILL.md"
  - ".agents/skills/golang-database/evals/evals.json"
---

## Decided
Each palette color must reach its minimum contrast (4.5:1 text, 3:1 muted and border) on every background it is actually drawn on: the window, the selected row, the unfocused selected row and the command bar surface. This holds at truecolor and at the 256 color index. Failing colors are nudged in lightness, hue kept.

## Watch out
The design spec audit checked only against the window background, so it expected small changes (tokyo-night Muted #5c6692). Checking the selected row too moves Muted much further on dark themes (tokyo-night #737ca7, dracula #8693b9), and 16 of 17 palettes change. This is the rule working, not a bug. Don't "restore" upstream hex values; the contrast test will fail.

The 256 indexes were picked by searching the few nearest indexes for each background and keeping the combination whose foregrounds land closest to their hex while passing contrast. Picking each background on its own (nearest only) gave tokyo-night a pink accent.

New slots started from existing ones: Border from Muted, Match from Warning, SelectedInactive a 50% mix of Background and Selected, Surface a 40% mix.

In the code: internal/tui/theme/palettes.go (comments mark "was" values), internal/tui/theme/palettes_test.go (drawnOn table, TestPaletteContrast).
