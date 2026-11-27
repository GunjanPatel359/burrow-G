# Agent readers

## Overview

Each folder here reads one agent CLI's sessions and turns them into the shared `session.Session` (agent, id, title, folder, last used). Every reader implements `session.Source` from `internal/session/session.go`. They are wired up with their home or database path in `cmd/burrow-g/main.go`.

## Key files

| File | Owns |
|---|---|
| `internal/session/session.go` | `Session`, `Source` interface, `ErrMissing`, `Sort` |
| `claude/`, `qoder/` | JSONL files under `~/.claude/projects/*/` and `~/.qoder/projects/*/`; `parse.go` reads title and folder |
| `opencode/source.go` | `~/.local/share/opencode/opencode.db`, table `session_v2` or `session` |
| `copilot/source.go` | `~/.copilot/session-store.db`, table `sessions`; `Export` reads `~/.copilot/session-state/<id>/events.jsonl` |
| `codex/source.go` | `~/.codex/sessions/**/rollout-*.jsonl` plus `session_index.jsonl` thread names |
| `freebuff/` | `~/.config/manicode/projects/*/chats/*/`, shared by Freebuff and Codebuff; `parse.go` reads one chat folder for `List` and `Export` (title, the folder streamed out of `run-state.json`), `export.go` reads `chat-messages.json`; `pickCLI` runs `freebuff`, else `codebuff`, found through the `LookPath` field (spec 0005) |

## Conventions

- Adding an agent: new folder with `source.go` (`Name`, `List`, `ResumeCommand`, `StartCommand`, `Delete`, `Export`), a test file with `var _ session.Source = Source{}`, then add it to `sources` in `cmd/burrow-g/main.go` and the README table.
- `Name()` is the stable key stored in the index (`claude`, `qoder`, …). Never rename it.
- Read only, always. SQLite opens as `file:<path>?mode=ro`. `Delete` returns a "not implemented" error for now.
- `Export` writes the portable format (spec 0003): the agent's own structured export if it has one, else an inline reader in the agent's package. Return `session.ErrUnsupported` until then. Every `Source` method, `Export` included, lives in the agent's `source.go`; a long reader behind it goes in one `export.go` (Claude), like `parse.go` behind `List`.
- Parsers stop reading a file as soon as they have title and folder. Title order: the agent's own title (Claude `/rename` sidecar, ai-title, thread name), then the first prompt, then `(untitled)`.
- Skip unreadable or folderless files instead of failing the whole list.
- Tests build fixtures in `testdata/` or a temp SQLite file with only the columns `List` reads.

## Gotchas

- Return `session.ErrMissing` when the agent's folder or database is absent. `db.Refresh` keeps the saved list on `ErrMissing`; returning an empty list instead wipes it. An empty list is only right when the folder exists with no sessions.
- Agents rename tables across versions (OpenCode `session` → `session_v2`). Detect which exists through `sqlite_master` and support both; only interpolate a table name picked from a fixed list.
- Close errors from read only connections are ignored (`defer db.Close()`); never fail a refresh over them.
- Skip child sessions: OpenCode rows with a `parent_id`, Codex subagent rollouts.
- Freebuff/Codebuff `--continue <id>` looks for the chat under `projects/<base name of the folder it runs in>` and silently opens the latest chat when it isn't there. Keep `Dir` checked against the project folder name.
- Never open Freebuff's `log.jsonl` (account id and e-mail) or `~/.config/manicode/credentials.json`.
- Codex support follows its documented format and is untested against real sessions.

_Drafted by /audit from the repo, worth a quick human pass. Edit freely: once a line stops matching this draft, later runs treat it as curated and will flag rather than overwrite it._
