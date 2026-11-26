# Command bar engine

## Overview

The engine behind `ctrl+k`: it parses a line like `theme nord @agent:claude`, checks arguments, runs the command, and offers suggestions while typing. Commands never touch the screen; they return an event that the TUI applies.

## Key files

| File | Owns |
|---|---|
| `command.go` | `Command`, `Arg`, `Input`, `Env`, `Service`, `Match`, `Usage` |
| `parse.go`, `run.go`, `suggest.go` | read a line (quotes, `@tag:value`), run it, suggest and complete |
| `commands/commands.go` | `New`: the list of every command |
| `commands/<name>.go` | one command each: `theme`, `refresh`, `export`, `handoff`, `help`, `quit` |
| `event/event.go` | events a command returns, and `All` |

## Conventions

- Adding a command: write `commands/<name>.go` with one `command.Command` (`Name`, `Help`, optional `Key`, `Args`, `Run`) and list it in `New`. Everything about the command stays in that file.
- An `Arg` with no `Name` is the bare value (`in.Value()`); one with a `Name` is a tag (`in.Get("agent")`). An `Arg` several commands share is one variable listed in each.
- Commands get what they need through `command.Env` (store, themes, current theme, keys, the selected session, handoff targets), never globals, so tests can build a fake `Env`.
- Tests run lines through `New(env).Run("…")` against a real temp database.

## Gotchas

- A new event needs three edits: the type in `event/event.go`, an entry in `event.All`, and a `case` in `apply` in `internal/tui/app.go`. An unhandled event shows "unknown event" in the status bar.
- `theme` previews while choosing (`ThemeChanged{Preview: true}`); only a real run saves the setting.

_Drafted by /audit from the repo, worth a quick human pass. Edit freely: once a line stops matching this draft, later runs treat it as curated and will flag rather than overwrite it._
