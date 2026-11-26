# Verify: session export · spec 0003 · updated 2026-11-26

_Steps follow the rollout in [index.md](index.md). Real sessions are only read; every write goes to a temp folder._

## Step 1: portable format, Claude, CLI

### Commands
- [ ] `make check` → gofmt clean, vet clean, every test passes
- [ ] `go test ./internal/portable -v` → a written file reads back equal; a file without the end line gives `ErrTruncated`; a wrong `format` is refused; an 860 KB line reads (section 2)
- [ ] `go test ./internal/agents/claude -run Export -v` → fixtures cover: compaction segments all exported in order, a parallel tool result on a side branch kept, one reply split by a tool result becomes one assistant record then one user record with every result, signature only thinking dropped and counted, injected blocks removed with the real text kept, a slash command kept as `/foo args`, a tool result image kept after its result, an unreadable line skipped and counted, the title equal to what `List` shows (section 5)
- [ ] `go test ./internal/agents/... -run Unsupported` → Qoder, Codex, Copilot return an error that `errors.Is(err, session.ErrUnsupported)` (section 1)

### Real sessions, by hand (`make build`)
- [ ] `./bin/burrow-g export <claude id> -o /tmp/s.jsonl` → file is mode 600; stderr says `exported N records …` with dropped counts; first line is a header with `burrow-session/1`, last line `kind: end` (sections 2, 7)
- [ ] `./bin/burrow-g export <claude id> | head -1` → header on stdout, nothing else on stdout (section 7)
- [ ] Export every local Claude session to a temp folder; for each, every `tool_call` id has a `tool_result` (a session cut off mid call may miss its last one) (section 5)
- [ ] The largest local session exports in under 2 s with under 50 MB peak memory (`/usr/bin/time -v`) (section 1, streaming)
- [ ] `sha256sum` of `~/.claude/projects` before and after the exports above → unchanged (section 1, read only)
- [ ] `./bin/burrow-g export <copilot id>` → `burrow-g: export isn't supported for Copilot CLI yet`, exit status not 0 (section 7)

## Step 2: OpenCode

### Commands
- [ ] `go test ./internal/agents/opencode -run Export -v` → a `testdata/` export converts: tool parts become a call and a result with the same id, errors set `error`, compaction kept, `idle` and `model-switched` dropped and counted; nil `Run` gives `ErrUnsupported` (section 6)

### By hand
- [ ] `./bin/burrow-g export <opencode id> -o /tmp/o.jsonl` → records match the session's messages in OpenCode (section 6)
- [ ] With `opencode` removed from `PATH` → `error: opencode not found on PATH` (section 6)

## Step 3: TUI

### Terminal, by hand
- [ ] `x` on a Claude session → status `exported N records to "~/.local/share/burrow-g/exports/claude-<id>.jsonl"`; the folder is mode 700, the file 600 (section 7)
- [ ] `x` on a Copilot session → `error: export isn't supported for Copilot CLI yet` (section 7)
- [ ] `ctrl+k` then `export` → same as `x` (section 7)
- [ ] The key hint for `x` shows in the status line and the help list, from `keys.go` (`docs/design.md`)
