# Verify: session export · spec 0003 · updated 2026-11-26

_Steps follow the rollout in [index.md](index.md). Real sessions are only read; every export lands in burrow-G's own exports folder or a temp folder. Results below are from 2026-11-26 on the author's machine (Claude Code 2.1.332, OpenCode 2.1.2, Qoder CLI, Copilot CLI 1.0.80, codex-cli 0.142.5)._

## Step 1: portable format, Claude, TUI `e`

### Commands
- [x] `make check` → gofmt clean, vet clean, every test passes
- [x] `go test ./internal/portable -v` → a written file reads back equal; a file without the end line gives `ErrTruncated`; a wrong `format` is refused; a 1 MB line reads (section 2)
- [x] `go test ./internal/agents/claude -run Export -v` → fixtures cover: compaction segments all exported in order, a parallel tool result on a side branch kept, one reply split by a tool result becomes one assistant record then one user record with every result, signature only thinking dropped and counted, injected blocks removed with the real text kept, assistant text kept word for word, a slash command kept as `/foo args`, a tool result image kept after its result, an unreadable line skipped and counted, the title equal to what `List` shows (section 5)
- [x] `go test ./internal/agents/opencode -run Unsupported` → a `Source` without `Run` returns an error that `errors.Is(err, session.ErrUnsupported)` (section 1); every other agent now exports
- [x] `go test ./internal/tui -run Export -v` → `e` on a session writes `<exports>/<agent>-<id>.jsonl` with folder 700 and file 600, reports `exported "<title>"` and clears it after 2 s; an existing file is narrowed to 600 and emptied; an unsupported agent reports `error: export isn't supported for <name> yet` and leaves no file; a failing export leaves no file; `e` while searching is typed into the search box (section 7)

### Real sessions
- [x] The job `e` runs, on a real 30 MB Claude session → 1,867 records, 5.8 MB, file 600, folder 700, 0.3 s (section 7)
- [x] Every local Claude session exported (67): none failed; every `tool_call` id has a `tool_result` except one session cut off mid call (1 call) (section 5)
- [x] The largest local session (105 MB, 45,144 lines) → 6,716 records, 0.76 s, 23 MB peak memory including the test binary (limits: 2 s, 50 MB); the file starts with a `burrow-session/1` header and ends with `kind: end` (sections 1, 2)
- [x] `sha256sum` of every file under `~/.claude/projects`, `~/.qoder/projects`, `~/.copilot/session-state` before and after the exports (712 files) → unchanged, except the transcript of the Claude Code session doing the verifying, which Claude itself was writing (section 1, read only)

### Terminal, by hand (`make build`, then `./bin/burrow-g`)
- [x] `e` on a Claude session → spinner `exporting…`, then a done message; the file appears in `~/.local/share/burrow-g/exports/` (section 7). Checked by hand 2026-11-26; that run asked for a shorter message that clears itself, now `exported "<title>"` for 2 s
- [x] The `e export` hint shows in the status line (checked by hand, 2026-11-26) and the help list (`TestExportKeyHint`) (`docs/design.md`)

## Step 2: OpenCode

- [x] `go test ./internal/agents/opencode -run 'Export|Exec' -v` → a `testdata/` export converts: tool parts become a call and a result with the same id, errors set `error`, a running tool keeps its call without a result, compaction kept, `idle` and `model-switched` dropped and counted; `Exec` returns whole output and carries stderr in its error (section 6)
- [x] A real 17 MB session through `opencode.Exec` → 2,592 records. Through a pipe OpenCode cut the same export at 16 MiB + 64 KiB with exit status 0 (3 of 3 runs), which is why `Exec` goes through a temp file (section 6)
- [x] With `opencode` off `PATH` → `opencode not found on PATH` (section 6)

## Step 3: Qoder, Codex, Copilot

- [x] `go test ./internal/agents/qoder -run Export -v` → millisecond timestamps read (no unreadable lines), compaction from the paired lines, thinking kept as reasoning, parallel calls grouped (section 5a)
- [x] Every local Qoder session exported (7): every tool call has its result (section 5a)
- [x] `go test ./internal/agents/codex -run Export -v` → a rollout fixture in the documented format: injected messages dropped, encrypted only reasoning dropped, `function_call` and `custom_tool_call` with their outputs, `{content, success: false}` as an error, compaction kept, a call cut off keeps no made up result (section 5b)
- [ ] Codex on real sessions — _none on this machine; the reader follows the documented format, type names checked in the codex-cli 0.142.5 binary_ (section 5b)
- [x] `go test ./internal/agents/copilot -run Export -v` → subagent events dropped as sidechain lines with the parent's call and result kept, `error` as `{message}` or plain text, attachments counted, a session without `events.jsonl` exports empty (section 5c)
- [x] Every local Copilot session exported (4): every tool call has its result; the one without an event log exports with 0 records (section 5c)
