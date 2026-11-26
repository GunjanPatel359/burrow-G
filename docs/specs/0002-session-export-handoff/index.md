# 0002. Session export and handoff between agent CLIs

**Date**: 2026-11-26
**Status**: In Progress

## Summary

You can export any session from any of the five agents as a Markdown transcript, and you can hand a session off to any agent (the same one or another). A handoff starts a fresh session in the target agent, in the same folder, with the old conversation passed as its first prompt. burrow-G still never writes into any agent's files: the target agent saves its new session itself. Building this means one new transcript reader per agent, one shared renderer (with secret masking and a size budget), two CLI subcommands, and two TUI keys.

## Requirements

**User stories**:
- As a developer, I want to hand my current Claude session to Codex (or any other installed agent) so that Codex picks up where Claude stopped, in the same folder, without me retelling the story.
- As a developer, I want to export a session as a readable Markdown file so that I can keep it, share it, or paste it anywhere.
- As a developer, I want secrets masked before a transcript leaves for another vendor's model so that a handoff never spreads a key I pasted earlier.

**Acceptance criteria**:
- **AC-1**: `burrow-g export <id>` (id by prefix, as `resume`) prints the session to stdout as Markdown: a header with title, agent, session id, folder, and export time, then every message of the main conversation in order, each with its role (User or Assistant), its time when the agent records one, its prose text, and one line per tool call. Thinking blocks, tool outputs, and subagent (sidechain) entries never appear. Works for Claude, Qoder, OpenCode, Copilot, and Codex.
- **AC-2**: `burrow-g export <id> -o <file>` writes the Markdown to `<file>` (mode 0600) instead of stdout and prints the written path to stderr. `-o` is accepted before or after the id.
- **AC-3**: In the TUI, pressing `e` on the selected session, or running `export` from the command bar, writes `~/.local/share/burrow-g/exports/<agent>-<id>.md` (file 0600, folder 0700, an existing file is replaced) and the status bar shows the path.
- **AC-4**: `burrow-g handoff <id> <agent>` starts `<agent>` in the session's folder, replacing burrow-G in the current terminal, with one first prompt made of a fixed preamble (naming the source agent, session id, title, and folder) followed by the transcript from AC-1, trimmed per AC-6.
- **AC-5**: In the TUI, pressing `t` on the selected session, or running `handoff` from the command bar, opens a picker listing every agent whose CLI is found on PATH (the source agent included). `enter` hands off in a new terminal tab when the terminal supports tabs and otherwise in this terminal; `o` hands off in this terminal; `esc` cancels with nothing launched. `handoff @agent:<name>` skips the picker. Under `sudo`, a handoff never opens a new tab.
- **AC-6**: A handoff prompt is never larger than 100,000 bytes. When the transcript does not fit, the prompt keeps the first user message and as many of the newest messages as fit, and puts a single `[N messages omitted]` line in place of the dropped middle. If the kept messages alone are still too big, their text is cut and ends with `[truncated]`. Export output is never trimmed.
- **AC-7**: In both export and handoff, text matching the known secret patterns (AWS access keys, GitHub tokens, OpenAI and Anthropic keys, Slack tokens, `Bearer` tokens, PEM private key blocks) is replaced with `[redacted]`, and the number of replacements is reported (status bar in the TUI, stderr in the CLI).
- **AC-8**: After a handoff launches, a one line report states the messages sent, the prompt size, the messages omitted, and the secrets masked: in the status bar for a new tab, on stderr just before the terminal is taken over otherwise.
- **AC-9**: If the session's folder no longer exists, a handoff is refused with `folder no longer exists: <dir>` and nothing is launched. Export still works for that session.
- **AC-10**: Entries burrow-G cannot parse are skipped and counted; export shows `(N entries skipped)` under the header and the handoff report includes the count. If zero messages remain, both export and handoff fail with `no messages found in <agent> session <id>`, and nothing is written or launched.
- **AC-11**: `burrow-g handoff <id> <agent>` with an unknown agent name fails and lists the valid names; with an agent whose CLI is not on PATH it fails with `<agent> is not installed (not found on PATH)`. Nothing is launched in either case.
- **AC-12**: burrow-G writes no file inside any agent's folder or database. Transcripts are read with read only opens (`?mode=ro` for SQLite). The only files written are export files under burrow-G's data folder or at the `-o` path.

## Decision

**Chosen option**: Option 1: Handoff through a first prompt

burrow-G exports any session to Markdown and hands it off by starting the target agent with that Markdown, masked and trimmed to 100 KB, as its first prompt, writing nothing into any agent's files.

**Implementation skills**: `golang-code-style` (`samber/cc-skills-golang`, `.agents/skills/golang-code-style/`) · `golang-error-handling` (`samber/cc-skills-golang`, `.agents/skills/golang-error-handling/`) · `golang-testing` (`samber/cc-skills-golang`, `.agents/skills/golang-testing/`) · `golang-database` (`samber/cc-skills-golang`, `.agents/skills/golang-database/`)

## Feature design

**Data model sketch** (in memory only; no table, no patch, nothing stored in the index):

| Type | Field | Type | Required | Notes |
|---|---|---|---|---|
| `session.Transcript` | `Session` | `session.Session` | yes | agent, id, title, dir of the source |
| | `Messages` | `[]session.Message` | yes | oldest first; empty means AC-10 failure |
| | `Skipped` | `int` | yes | entries that could not be parsed (AC-10) |
| `session.Message` | `Role` | `string` | yes | `"user"` or `"assistant"` only |
| | `Time` | `time.Time` | no | zero when the agent records none |
| | `Text` | `string` | no | prose only; thinking dropped; may be empty when `Tools` is not |
| | `Tools` | `[]string` | no | one line per tool call, input only, no output |

New interface in `internal/session/transcript.go`, beside `Source`:

```go
type Transcriber interface {
	Transcript(s Session) (Transcript, error)
	StartCommand(dir, prompt string) []string
}
```

All five agent `Source` types implement it. A future agent can ship listing first and add it later; burrow-G checks with a type assertion and treats a missing implementation as "export and handoff not supported for <agent>". (`Skipped` was added to the confirmed shape to carry the AC-10 count; it holds no new data.)

Per agent transcript source (all read only):

| Agent | Where | Messages | Tool lines | Skip |
|---|---|---|---|---|
| Claude | `<Home>/projects/*/<id>.jsonl` | `type` `user` / `assistant`; `message.content` string or blocks of type `text` | `tool_use` blocks (`name`, `input`) | `thinking` blocks, `tool_result` only user records, `isSidechain: true`, `isMeta: true`, every other record type |
| Qoder | `<Home>/projects/*/<id>.jsonl` | same layout as Claude (user content is a string, assistant content an array) | `tool_use` blocks | as Claude |
| OpenCode | `opencode.db`: `session_message` (when `session_v2` exists), else `message` + `part` | v2: `data` JSON with `type` `user` / `assistant`, `content[]` of `text`; v1: `message.data.role`, `part.data.type` `text` | v2 `content[]` of type `tool`; v1 parts of type `tool` | `system`, `synthetic`, `idle` and other message types; reasoning parts |
| Copilot | `<dir of DBPath>/session-state/<id>/events.jsonl`; when absent, the `turns` table (`user_message`, `assistant_response`) | `user.message`, `assistant.message` events | `tool.execution_start` (`toolName`, `arguments`) | hook, usage, permission, notification, turn marker events |
| Codex | the session's `rollout-*.jsonl` | `response_item` with payload `type: message`, role `user` / `assistant`, `input_text` / `output_text` parts | payload `type: function_call` (`name`, `arguments`) | `developer` and `system` roles, user messages that start with `<environment_context>` or `<user_instructions>`, reasoning items |

Consecutive records with the same role are merged into one `Message` (Claude writes one reply as several records), except that text arriving after a tool call starts a new `Message`, so prose and tool lines keep the order they happened in. Readers pass each part of a record to `Transcript.Add` in order.

`StartCommand` per agent (the `dir` argument is unused today and kept for agents that need a folder flag):

| Agent | argv |
|---|---|
| Claude | `claude <prompt>` |
| Qoder | `qodercli -i <prompt>` (`--prompt-interactive`, from Qoder's CLI reference) |
| OpenCode | `opencode --prompt <prompt>` |
| Copilot | `copilot -i <prompt>` |
| Codex | `codex <prompt>` |

**State transitions**: none; export and handoff are one shot actions with no stored state.

**API surface** (CLI subcommands, command bar commands, keys, Go functions):

| Surface | Form | Key inputs | Key outputs | Auth | Key errors |
|---|---|---|---|---|---|
| CLI | `burrow-g export <id> [-o file]` | id prefix (req), `-o` path (opt) | Markdown on stdout, or file plus path on stderr; masked and skipped counts on stderr | local user | no or several matches (as `resume`), no messages (AC-10), agent without `Transcriber` |
| CLI | `burrow-g handoff <id> <agent>` | id prefix (req), agent name (req) | target agent replaces burrow-G; report on stderr | local user | unknown agent, not installed (AC-11), folder gone (AC-9), no messages |
| Key | `e` in the session list | selected session | export file, path in status bar | local user | as export |
| Key | `t` in the session list | selected session | agent picker | local user | no installed agents: status bar `no agent CLI found on PATH` |
| Command | `export` | selected session | as `e` | local user | no session selected: `select a session first` |
| Command | `handoff [@agent:<name>]` | selected session, optional agent tag | picker, or straight handoff when tagged | local user | as `t` plus unknown agent |
| Picker | `enter` / `o` / `esc` | highlighted agent | new tab (or here), here, cancel | local user | `OpenTab` failure shown in status bar, as resume |
| Go | `transcript.Markdown(t Transcript, now time.Time) (string, Stats)` | transcript | masked Markdown, counts | n/a | none |
| Go | `transcript.Prompt(t Transcript) (string, Stats)` | transcript | preamble plus masked, trimmed Markdown | n/a | none |
| Go | `transcript.Handoff(src, dst Transcriber, s Session, lookPath func(string) (string, error)) (argv []string, Stats, error)` | source, target, session, injectable PATH lookup | argv for the target | n/a | folder gone, not installed, no messages |

`Stats` carries `Messages`, `Bytes`, `Omitted`, `Masked`, `Skipped`.

Handoff preamble (fixed text, written first in the prompt):

```
You are continuing a coding session that started in <agent> (session <id>, "<title>") in <dir>.
Below is its transcript: user prompts, assistant replies, and one line per tool call (tool outputs are not included).
Read it, then briefly say where things stand and wait for my next instruction. Do not redo work that is already done.
```

Markdown layout (export and the body of the prompt):

```
# <title>

Agent: <agent> · Session: <id> · Folder: <dir> · Exported: <RFC 3339 time>
(N entries skipped)   (only when N > 0)

## User · 2026-11-20 14:03

<text>

## Assistant · 2026-11-20 14:04

<text>

- tool: Edit internal/db/db.go
- tool: Bash go test ./...
```

Time shows as local `2006-01-02 15:04`, and the ` · <time>` part is left out when the time is zero. Tool lines are `<tool name as the agent records it> <primary argument>`, where the primary argument is the first present of `file_path`, `filePath`, `path`, `command`, `cmd`, `pattern`, `url`, `query`, collapsed to one line and cut to 120 characters; with none present, the tool name alone.

**Value sourcing**:

| Action | Value produced / displayed | Source |
|---|---|---|
| export, handoff | session to act on | CLI id prefix through `db.Find` (as `resume`); TUI selected row |
| export header | title, agent, id, folder | `session.Session` fields from the index |
| export header | export time | `now` passed in (injectable; `time.Now()` at the call site) |
| export, prompt | messages, roles, times, text | the agent's `Transcript` reader (table above) |
| export, prompt | tool lines | the agent's tool call records, rule above |
| export, report | skipped count | `Transcript.Skipped` |
| export, prompt, report | masked count | number of pattern matches replaced in `transcript` redaction |
| prompt | omitted count, byte size | computed by the budget step from the rendered messages |
| TUI export | file path | `helpers.DataDir(home)/exports/<agent>-<id>.md` |
| handoff | target argv | target's `StartCommand(s.Dir, prompt)` |
| handoff | working folder | `s.Dir`, checked with `session.CheckDir` |
| picker | agent list | `Env.Sources` filtered to `Transcriber` implementers whose argv[0] passes `lookPath` |
| picker | new tab or here | `launch.Detect` result, forced to here under `sudo` (existing `main.go` rule) |
| command bar | selected session | new `command.Env.Selected func() (session.Session, bool)` |

**Key invariants**:
- No code path in this feature opens an agent's file or database for writing.
- `len(prompt) <= 100_000` bytes for every handoff, preamble included.
- Redaction runs before budgeting, so sizes count the masked text; it applies to text and tool lines alike.
- The first user message is always kept in a handoff prompt (cut with `[truncated]` only if it alone exceeds the budget).
- A handoff with zero messages, a missing folder, or a missing target CLI launches nothing.

**Security model**: local, single user tool. Everything runs as the person who started burrow-G; under `sudo` it uses root's home and sessions (existing rule) and never opens a tab. Export files may contain private conversation, so they are written 0600 in a 0700 folder. Transcripts leaving for another vendor are masked first (AC-7). Masking is best effort on known shapes; it is not a guarantee, and the report says how many it caught (only in what was sent). The prompt travels as a command line argument, the one first prompt channel every agent accepts, so on a machine shared with other users it is readable through `ps` and `/proc/<pid>/cmdline` while the target runs; the README says so. Export paths are built only from an id that names a file inside the exports folder. No compliance scope applies.

**Configuration required**: none. The 100,000 byte budget is a constant in `internal/transcript/budget.go`.

**Critical test scenarios**:
- Happy path: a Claude fixture with prompts, replies, a thinking block, a tool call, and a sidechain record exports to the expected Markdown (thinking and sidechain absent, tool line present), verifies **AC-1**; `Handoff` to a fake target returns `[codex, <preamble + transcript>]`, verifies **AC-4**.
- Failure case: a 300 KB transcript yields a prompt of at most 100,000 bytes that starts with the first user message and ends with the newest message, with one `[N messages omitted]` line, verifies **AC-6**; a fixture with one corrupt line exports with `(1 entries skipped)`, and an all corrupt fixture fails with no messages, verifies **AC-10**.
- Secrets: a message containing `AKIA` + 16 characters and a PEM block exports with both replaced by `[redacted]` and `Masked == 2`, verifies **AC-7**.
- Refusals: a missing folder, an unknown agent, and a `lookPath` that fails each return the stated error and an empty argv, verifies **AC-9**, **AC-11**.
- Read only: every reader's test opens its SQLite fixture through the same `?mode=ro` path the code uses, and a test that removes write permission from the fixture folder still passes, verifies **AC-12**.

## Build plan

Ordered as a Tracer Bullet (no build approach is recorded for this project, so this is the assumed default): first one thin thread from Claude through export and CLI handoff end to end, then the other four readers, then the TUI.

- [x] T1 · Add `Message`, `Transcript`, and the `Transcriber` interface, satisfies **AC-1** · deps: none · files: `internal/session/transcript.go`
- [x] T2 · Build the shared `transcript` package: Markdown renderer, tool line rule, secret masking, 100 KB budget with first plus newest trimming, preamble, `Handoff` with injectable `lookPath`, and `Stats`, with tests, satisfies **AC-1**, **AC-4**, **AC-6**, **AC-7**, **AC-9**, **AC-10**, **AC-11** · deps: T1 · files: `internal/transcript/render.go`, `internal/transcript/redact.go`, `internal/transcript/budget.go`, `internal/transcript/handoff.go`, `internal/transcript/*_test.go`
- [x] T3 · Claude `Transcript` and `StartCommand`, with JSONL fixtures (text, thinking, tool_use, tool_result only user record, sidechain, corrupt line), satisfies **AC-1**, **AC-10**, **AC-12** · deps: T1 · files: `internal/agents/claude/transcript.go`, `internal/agents/claude/transcript_test.go`, `internal/agents/claude/testdata/transcript/*`
- [x] T4 · CLI `export` (with `-o` before or after the id, 0600 file) and `handoff` subcommands, report lines on stderr, usage text, satisfies **AC-1**, **AC-2**, **AC-4**, **AC-7**, **AC-8**, **AC-9**, **AC-10**, **AC-11** · deps: T2, T3 · files: `cmd/burrow-g/transfer.go`, `cmd/burrow-g/transfer_test.go`, `cmd/burrow-g/main.go`, `cmd/burrow-g/commands.go`
- [x] T5 · Qoder `Transcript`; confirm the real `qodercli` first prompt flag (docs or an install) and only then add `StartCommand`; until confirmed, Qoder is a source but not a picker target, satisfies **AC-1**, **AC-4**, **AC-12** · deps: T1 · files: `internal/agents/qoder/transcript.go`, `internal/agents/qoder/transcript_test.go`, `internal/agents/qoder/testdata/transcript/*`
- [x] T6 · OpenCode `Transcript` for both `session_message` (v2) and `message` + `part` (v1), picked through `sqlite_master`, temp SQLite fixtures for both, plus `StartCommand`, satisfies **AC-1**, **AC-4**, **AC-12** · deps: T1 · files: `internal/agents/opencode/transcript.go`, `internal/agents/opencode/transcript_test.go`
- [x] T7 · Copilot `Transcript` from `events.jsonl`, falling back to the `turns` table, plus `StartCommand`, satisfies **AC-1**, **AC-4**, **AC-12** · deps: T1 · files: `internal/agents/copilot/transcript.go`, `internal/agents/copilot/transcript_test.go`, `internal/agents/copilot/testdata/transcript/*`
- [x] T8 · Codex `Transcript` from the rollout file (skipping developer, system, and injected context messages) plus `StartCommand`; check the fixture against a real local rollout, since Codex support is untested against real sessions, satisfies **AC-1**, **AC-4**, **AC-12** · deps: T1 · files: `internal/agents/codex/transcript.go`, `internal/agents/codex/transcript_test.go`, `internal/agents/codex/testdata/transcript/*`
- [x] T9 · Command bar: `Env.Selected`, `export` and `handoff` commands (`@agent:` tag), the `ExportRequested` and `HandoffRequested` events added to `event.All`, with tests, satisfies **AC-3**, **AC-5** · deps: T2 · files: `internal/command/command.go`, `internal/command/commands/export.go`, `internal/command/commands/handoff.go`, `internal/command/commands/commands.go`, `internal/command/commands/commands_test.go`, `internal/command/event/event.go`
- [x] T10 · TUI: `e` and `t` keys, an agent picker component (installed `Transcriber` agents only, `enter` / `o` / `esc`), export written in a background `tea.Cmd` with the path in the status bar, handoff through `OpenTab` or a "launch here" result; `tui.Run` returns the session plus the argv to run instead of the session alone, satisfies **AC-3**, **AC-5**, **AC-7**, **AC-8**, **AC-9** · deps: T2, T9 · files: `internal/tui/keys.go`, `internal/tui/app.go`, `internal/tui/commands.go`, `internal/tui/messages.go`, `internal/tui/components/agentpicker/model.go`, `internal/tui/components/agentpicker/model_test.go`, `internal/tui/app_test.go`
- [x] T11 · `main.go`: run the argv returned by `tui.Run` (resume or handoff here), print the handoff report to stderr before taking over the terminal, satisfies **AC-5**, **AC-8** · deps: T4, T10 · files: `cmd/burrow-g/main.go`
- [x] T12 · README (export and handoff, the per agent table) and a `CHANGELOG.md` entry, satisfies **AC-1**, **AC-4** · deps: T4, T10 · files: `README.md`, `CHANGELOG.md`

## Consequences

**Positive**:
- Any session can move to any installed agent in two keystrokes, and the read only promise still holds word for word.
- The transcript model and renderer are a base for later features (search inside conversations, a preview pane, Option 2 native import) without new storage.
- Agent format changes break one reader at a time and show up as `(N entries skipped)` rather than a crash.

**Negative / tradeoffs**:
- A handoff is a fresh session, not a resumed one; the target sees a summary of tool calls, never their outputs, so it may need to look at files again.
- Long sessions lose their middle in a handoff, and the target spends up to about 25k tokens of context on history.
- Five new parsers of formats burrow-G does not control; each agent release can need a reader fix (the same cost listing already pays, now larger).
- Masking catches known key shapes only; a password typed in plain words still goes across.
- The new session is not linked back to its source in burrow-G's list (no lineage in v1); the link lives only in the preamble text.

**Neutral**:
- `tui.Run` changes its return value to carry the argv; `main.go` adapts in the same change.
- `command.Env` gains `Selected`, which future per session commands can reuse.
- No database patch and no version bump of the index.

## Follow-up

- [x] Confirm Qoder's first prompt flag before T5 wires `StartCommand` (`qodercli` is not installed on the machine this was designed on). Settled from Qoder's CLI reference: `-i` / `--prompt-interactive` runs the prompt, then stays interactive.
- [x] Confirm `opencode --prompt` opens the interactive TUI with the prompt already sent (not only filled in), and that GNOME Terminal's `OpenTab` passes a 100 KB argument intact; if not, handoff for that path falls back to this terminal. Both settled in verify: OpenCode (offline run) shows the prompt as the first message of a new session, and a real GNOME tab passed a 99,768 byte prompt byte for byte.
- [x] After the build, `/sync` should record in `internal/agents/AGENTS.md` that adding an agent now also means `transcript.go` (`Transcript`, `StartCommand`), and add `internal/transcript/` to the root `AGENTS.md` key files. Done in `internal/agents/AGENTS.md` (convention and key files); root `AGENTS.md` has no key files table, so nothing was added there.
- [ ] Revisit Option 2 (native import into OpenCode and Copilot) only if people ask for handed off history inside those agents' own pickers.
- [ ] Lineage (recording that session B came from session A) was left out of v1; it needs a numbered patch and a way to learn the new session's id after the target saves it.

Decision history (context, options, rationale): [rationale.md](rationale.md). Verify steps: [verify.md](verify.md).
