# Terminal UI

## Overview

The Bubble Tea app: an agents pane, a sessions pane with search, a details box, a status bar and the `ctrl+k` command bar. `Run` returns a `Launch` (session, argv, report) to run in this terminal, a resume or a handoff, or nil; `cmd/burrow-g/main.go` prints the report and runs it after the UI exits. New to Bubble Tea? Read `docs/bubbletea-cheatsheet.md`.

## Key files

| File | Owns |
|---|---|
| `app.go` | `model`, `Update` (key routing), `apply` (command events), `Run` |
| `commands.go` | background jobs (`tea.Cmd`): refresh, open tab, bookmark, export, handoff targets (`findTargets`), handoff prompt |
| `messages.go` | messages the app sends itself (`refreshedMsg`, …) |
| `keys.go`, `layout.go` | key bindings, sizing of panes |
| `background.go` | sets the terminal background for the theme, restores it on exit |
| `components/<name>/model.go` | agentlist, sessionlist, details, statusbar, commandbar, agentpicker (handoff dialog); each its own model |
| `theme/palettes.go`, `theme/theme.go` | 17 palettes; `theme.Get` turns one into styles |
| `theme/env.go`, `theme/looks.go`, `theme/glyphs.go`, `theme/layout.go` | color depth and locale (`theme.Env`), the shared 16 color and no color looks, Unicode and ASCII glyph sets, layout sizes |
| `ui/` | shared drawing pieces (`Row`, `PaneHeader`, `KeyHints`, `StatusDot`, `Dialog`, `EmptyState`, …), each drawn from a `theme.Theme` |

## Conventions

- Design system: draw everything to `docs/design.md` (tokens, glyphs, primitives, states, copy). Colors, symbols and sizes live only in `theme/`; outside it, code draws through `ui` and theme styles and never names a color or a literal glyph (review enforces this; the `rg` check is in `docs/design.md`).
- Only the jobs in `commands.go` touch the database or agents' files, and they run as `tea.Cmd`, never inside `Update`. Each returns a message from `messages.go` carrying its error. PATH lookups too: handoff targets come from `findTargets` at startup and on refresh, kept in the shared `sel` state the command bar reads.
- Components keep their own messages (e.g. `agentlist.SelectedMsg`) and own `Update`; the app forwards keys to the focused pane.
- The open command bar takes every key first; `ctrl+c` always quits.
- Errors go to the status bar (`m.status.Error`), never panics or prints.
- Tests drive the real model: `testModel`/`testModelWith` with a `fakeSource` and a temp database, then `press`, `pressAll`, `deliver` to feed keys and messages.
- Adding a theme: a palette in `theme/palettes.go` filling all 14 slots (each a `Tone` with hex and 256 index) that passes `go test ./internal/tui/theme` (contrast included), then the README theme list.
- Status work: batch the `tea.Cmd` that `m.status.Busy` returns with the job, or the spinner never ticks.

## Gotchas

- `enter` opens a new tab only when `launch.Detect` found a terminal (GNOME today); with none it resumes in place, like `o`. Under sudo the terminal is always nil.
- The theme preview must be dropped back to `m.saved` on close and after a run, or a previewed theme sticks.
- `bg.restore()` runs before `main` resumes a session, so the agent starts with the user's own background.
- `Run` reads the color depth, the locale and the terminal background before `tea.NewProgram`: the background query reads from the terminal and would race with Bubble Tea's input. Tests pass a fixed `theme.Env` and a `dark` func to `newModel` instead.

## Agent skills

- [bubbletea](../../.agents/skills/bubbletea/): `ggprompts/tfe`, Bubble Tea and Lip Gloss patterns (Elm architecture, pane layouts, key and mouse handling, components). Its templates are general; this doc's conventions win where they differ.

_Drafted by /audit from the repo, worth a quick human pass. Edit freely: once a line stops matching this draft, later runs treat it as curated and will flag rather than overwrite it._
