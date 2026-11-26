# Transcripts: export and handoff

## Overview

Turns a session's conversation (`session.Transcript`, read by each agent's `transcript.go`) into Markdown for export, and into a first prompt for handing the session to another agent. Governed by [spec 0002](../../docs/specs/0002-session-export-handoff/index.md). Nothing here reads or writes agent files; readers live in `internal/agents/`.

## Key files

| File | Owns |
|---|---|
| `render.go` | `Markdown`, `Stats`, `CheckMessages`, `ToolLine` (tool name plus its main argument) |
| `redact.go` | secret patterns and the `redactor` that masks and counts |
| `budget.go` | `Prompt`: preamble, masking, the 100,000 byte `Budget`, first user message plus newest kept |
| `handoff.go` | `Handoff` (refusals, then argv), `Targets`, `Report`, `ExportReport` |
| `export.go` | `ExportPath`, `SaveExport`, `WritePrivate` (0600 files, 0700 folder) |

## Conventions

- Mask before cutting or budgeting: tool lines are masked whole, then cut to 120 characters, so a secret crossing a cut can't leak. Sizes count the masked text.
- `len(Prompt) <= Budget` always, preamble included. A new piece of prompt text must fit inside that count.
- `Stats.Masked` in a prompt counts only secrets in what was sent; `Stats.Messages` only messages actually kept.
- Side effects stay injectable: `Handoff` and `Targets` take `lookPath`; nothing here calls `exec` or reads `PATH` itself.
- A new secret shape goes in `secretPatterns` (or `labeled` when only its name gives it away) with a case in `redact_test.go`, and in the README's list.

## Gotchas

- The prompt is an argv element: control bytes are stripped (a NUL fails `exec`), and other local users can read it through `/proc` while the target runs.
- Export file names come from agent data; `SaveExport` refuses an id that would leave the exports folder.

_Drafted by /sync from the introducing change, worth a quick human pass._
