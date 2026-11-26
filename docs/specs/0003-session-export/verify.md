# Verify: session export · spec 0003 · updated 2026-11-26

_Steps follow the rollout in [index.md](index.md). Real sessions are only read; every export lands in burrow-G's own exports folder or a temp folder._

## Step 1: portable format, Claude, TUI `e`

### Commands
- [ ] `make check` → gofmt clean, vet clean, every test passes
- [ ] `go test ./internal/portable -v` → a written file reads back equal; a file without the end line gives `ErrTruncated`; a wrong `format` is refused; a 1 MB line reads (section 2)
- [ ] `go test ./internal/agents/claude -run Export -v` → fixtures cover: compaction segments all exported in order, a parallel tool result on a side branch kept, one reply split by a tool result becomes one assistant record then one user record with every result, signature only thinking dropped and counted, injected blocks removed with the real text kept, assistant text kept word for word, a slash command kept as `/foo args`, a tool result image kept after its result, an unreadable line skipped and counted, the title equal to what `List` shows (section 5)
- [ ] `go test ./internal/agents/... -run Unsupported` → Qoder, Codex, Copilot, OpenCode return an error that `errors.Is(err, session.ErrUnsupported)` (section 1)
- [ ] `go test ./internal/tui -run Export -v` → `e` on a session writes `<exports>/<agent>-<id>.jsonl` with folder 700 and file 600, reports `exported N records to "…"`; an unsupported agent reports `error: export isn't supported for <name> yet` and leaves no file; a failing export leaves no file (section 7)

### Terminal, by hand (`make build`, then `./bin/burrow-g`)
- [ ] `e` on a Claude session → spinner `exporting…`, then `exported N records to "~/.local/share/burrow-g/exports/claude-<id>.jsonl"`; `stat` shows the folder 700 and the file 600; the file's first line is a `burrow-session/1` header, its last line `kind: end` (sections 2, 7)
- [ ] `e` on a Copilot session → `error: export isn't supported for Copilot CLI yet`, no file (section 7)
- [ ] The `e export` hint shows in the status line (when it fits) and in the help command's key list (`docs/design.md`)
- [ ] Export every local Claude session (a loop over the same job in a test binary, or `e` on a sample) into a temp `HOME`: every `tool_call` id has a `tool_result` (a session cut off mid call may miss its last one) (section 5)
- [ ] The largest local session exports in under 2 s with under 50 MB peak memory (section 1, streaming)
- [ ] `sha256sum` of `~/.claude/projects` before and after the exports above, excluding a session Claude is writing right now → unchanged (section 1, read only)

## Step 2: OpenCode

### Commands
- [ ] `go test ./internal/agents/opencode -run Export -v` → a `testdata/` export converts: tool parts become a call and a result with the same id, errors set `error`, compaction kept, `idle` and `model-switched` dropped and counted; nil `Run` gives `ErrUnsupported` (section 6)

### By hand
- [ ] `e` on an OpenCode session → records match the session's messages in OpenCode (section 6)
- [ ] With `opencode` removed from `PATH` → `error: opencode not found on PATH` (section 6)

## Step 3: Qoder, Codex, Copilot

_Each reader adds its own checks here when its rules are written into index.md._
