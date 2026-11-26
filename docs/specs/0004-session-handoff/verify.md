# Verify: session handoff · spec 0004 · updated 2026-11-26

_Steps follow the rollout in [index.md](index.md). Agents are only started by hand; tests use a fake PATH and a fake terminal._

## Step 1: the handoff file

- [x] `go test ./internal/handoff -v` → starts at the latest summary (older records and summaries absent); no summary section without a compaction; the header (record not instructions, title, folder, last used, full record); a tool result under its call, a failed one marked `failed:`, reasoning left out, an image as `[image]`; cuts at 2 KB output, 1 KB input (as text), 8 KB text, 40 KB summary with `… [N KB more in the full record]`; never inside a character; at most 100,000 bytes with the first user message kept and the gap marked; a cut off export refused with `ErrTruncated`; the exact prompt (section 3)
- [x] The real z/OS Claude session (5.8 MB export) → a 74.9 KB handoff file: header, summary, recent conversation, 15 long items cut (section 3)

## Step 2: start commands and `Launch`

- [x] `go test ./internal/agents/... -run StartCommand` → `claude <p>`, `qodercli -i <p>`, `opencode --prompt <p>`, `copilot -i <p>`, `codex <p>` (section 4, each checked against the CLI's `--help`)
- [x] `go test ./internal/tui` → `enter` and `o` on a session return a `Launch` with the folder, `ResumeCommand` and the title (section 5)

## Step 3: the dialog

- [x] `go test ./internal/tui/ui ./internal/tui/components/picker -v` → `Choice` marks only the selected row, 24 to 50 cells; the picker moves within its items, `enter` and `o` choose (`Here` for `o`), `esc` closes
- [x] `go test ./internal/tui -run 'EOpens|Dialog|Continue|NoAgents' -v` → only agents on PATH are offered; Save to file exports; continuing here writes the handoff file (600) and launches the agent with a prompt naming it; with tabs it opens a tab and reports `continued "<title>" in <agent>`; `o` continues here; a deleted folder, or an agent that can't export, reports an error and writes and starts nothing; `esc` does nothing; `q` inside the dialog doesn't quit (sections 1, 2)
- [x] By hand (`make build && ./bin/burrow-g`, 2026-11-26): `e` opens the dialog over the screen; Save to file exports; Continue in an agent starts it in the session's folder and it reads the handoff file — checked by the user
