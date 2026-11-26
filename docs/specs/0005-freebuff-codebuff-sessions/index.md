# 0005. Read Freebuff and Codebuff sessions

**Date**: 2026-11-25
**Status**: Accepted
**Issue**: #20
**Number**: 0005, not 0002: the closed PR #24 used 0002 for its draft, and spec 0003 still refers to it by that number
**Updated**: 2026-11-27: `StartCommand` and `Export` (sections 7 and 8), added to every agent by specs [0003](../0003-session-export/index.md) and [0004](../0004-session-handoff/index.md); Freebuff's own chat title; section 7 corrected after a real handoff failed: Freebuff takes no prompt, so a new chat starts in Codebuff; a missing CLI is named before a new tab opens (section 6)

## Summary

burrow-G gains a read only reader for Freebuff, the free CLI from the Codebuff team. Its paid sibling Codebuff runs the same code and keeps its chats in the same folder, so one reader lists both, in one agents pane row called "Freebuff" ("Freebuff / Codebuff" would be cut short in the 20 column pane). Resume uses `freebuff`, or `codebuff` when only that one is installed. The choice between the two CLIs is one small function inside the reader. Starting a new chat (for "Continue in Freebuff") always uses `codebuff`, the only one that takes a prompt, and `Export` reads the chat's own files, since Freebuff has no structured export.

## Decision

**Chosen option**: Option 1: one reader for the shared folder, a CLI picker inside it, folder taken from `run-state.json` and checked against the chat's project folder.

**Implementation skills**: `golang-code-style` · `golang-error-handling` · `golang-testing` (standard `testing` only, per `AGENTS.md`)

## Rationale

What the files and both CLIs' code show, the options, and the cross product resume question: see [rationale.md](rationale.md).

## Standard definition

### 1. Packages

| Package | Owns | May import |
|---|---|---|
| `internal/agents/freebuff` (new) | `source.go`: `Name`, `List`, `ResumeCommand`, `StartCommand`, `Delete`, `Export`. `parse.go`: `readChat`, one chat folder as `List` shows it (titles, the head of `run-state.json`), and `pickCLI` (section 2). `export.go`: the export reader | `session`, `portable`, `helpers`, standard library |
| `internal/session`, `internal/tui` | `CheckCommand`: a missing CLI says "<cli> is not installed (not found in PATH)", for a resume here and, before the tab opens, a new tab (section 6) | (unchanged) |
| `cmd/burrow-g/main.go` | adds `freebuff.Source{Home: ~/.config/manicode, LookPath: exec.LookPath}` to `sources`, after `codex` | `freebuff` |
| `internal/tui/components/agentlist` | display name `"freebuff": "Freebuff"` | (unchanged) |

`source.go` holds only the six `Source` methods (repo convention); file parsing lives in `parse.go`, as in `claude/` and `qoder/`, and the export reader in `export.go`, as in the other agents. `List` and `Export` read a chat folder through the same `readChat`, so an export's header and the list never disagree.

### 2. The CLI picker

```go
// agents/freebuff/parse.go
// pickCLI returns the first of names that lookPath finds. If none is found it
// returns names[0], so the résumé error names the CLI people should install.
// A nil lookPath means exec.LookPath.
func pickCLI(lookPath func(string) (string, error), names ...string) string
```

- "Installed" means "found on `PATH`", the same lookup `session.Resume` makes with `argv[0]`. A binary that only sits in the agent's own folder doesn't count, because resume couldn't run it.
- Order is preference: the first name wins when several are installed.
- It is called from `ResumeCommand`, so the details pane always shows the command that would run. `StartCommand` doesn't use it: only Codebuff can start a chat with a prompt (section 7).
- It stays private to the reader: Freebuff is the only agent with two CLIs. If a second one appears, it moves to `internal/session` then, rather than becoming shared code with one caller now.

### 3. Where sessions come from

| Field | Source |
|---|---|
| sessions | `<Home>/projects/<project>/chats/<chat-id>/`, one folder per chat; `<Home>` is `~/.config/manicode` |
| `ID` | the chat folder name, an ISO time with `:` replaced by `-` (`2026-08-31T11-08-58.778Z`) |
| `Dir` | `run-state.json` → `sessionState.fileContext.projectRoot`, else `.cwd`; the first whose base name equals `<project>` (section 4) |
| `Title` | `chat-title.txt`, the title Freebuff gets from its server (newer versions); else `chat-meta.json` → `firstPrompt` when the cache is current (section 5); else `(untitled)` |
| `LastUsed` | modification time of `chat-messages.json` |
| `Agent` | `"freebuff"` (the stable index key; never renamed, also for Codebuff chats) |

### 4. Rules for `List`

1. `<Home>/projects` missing → `session.ErrMissing` (a refresh keeps the saved list). Any other error reading it → that error.
2. A chat without `chat-messages.json` is skipped: Freebuff leaves an opened, never used chat as a folder with only `log.jsonl`.
3. `Dir` must have the base name `<project>`. Both CLIs find the chat for `--continue <id>` under `projects/<basename(project root)>/chats/`, and when the id isn't there they **silently open the most recent chat instead**. A chat whose folder fails this check is skipped, so burrow-G never offers a résumé that opens a different conversation.
4. `run-state.json` is 0.6 to 0.8 MB. It is read with a streaming `json.Decoder` that stops once it has both `projectRoot` and `cwd`. Both sit in the first few hundred bytes today, and the reader stays correct if the order changes (it just reads further).
5. A chat whose files can't be read or parsed is skipped; it never fails the whole list. That includes a chat with no `run-state.json`: Freebuff writes it separately from the messages and can resume without it, but without it the chat's folder can't be known, and a guess from the project folder name could break rule 3.
6. Only `chat-messages.json` (stat only), `chat-title.txt`, `chat-meta.json` and `run-state.json` are opened (`Export` also reads the first two whole, section 8). `log.jsonl` (account id and e-mail) and `<Home>/credentials.json` are never opened.
7. Two projects with the same folder name share `projects/<name>/`; each chat still gets its own real folder from `run-state.json`.
8. A chat id is the time the chat started, unique only inside one project folder, while the index keys sessions by agent and id. When two project folders hold the same id, only the most recently used chat is listed (the first project by name on a tie); listing both would fail the whole save.

### 5. Title cache check

`chat-meta.json` is `{messageCount, firstPrompt, messagesSize, messagesMtimeMs}`, a cache that Freebuff trusts only while `messagesSize` equals the size of `chat-messages.json` and `messagesMtimeMs` equals its modification time in milliseconds. burrow-G makes the same check, with the time allowed to differ by less than 1 ms (a float written by JavaScript, read back in Go). Missing, unreadable or stale cache, or an empty `firstPrompt` → `(untitled)`. `List` never parses `chat-messages.json` itself (it reaches megabytes).

`chat-title.txt` comes first when it holds text: Freebuff asks its server for a title after the first prompt and shows it ahead of the prompt in its own chat list. It is written already cleaned up (one line, at most 60 characters); burrow-G trims it and cuts it to 60 like the prompt. Versions before late 2026 never wrote it.

### 6. Resume

`ResumeCommand` returns `[pickCLI(LookPath, "freebuff", "codebuff"), "--continue", <ID>]`, run in `Dir` by `session.Resume` or a new tab, like every agent.

| Installed on `PATH` | Command |
|---|---|
| both | `freebuff --continue <id>` |
| only `freebuff` | `freebuff --continue <id>` |
| only `codebuff` | `codebuff --continue <id>` |
| neither | `freebuff --continue <id>`: resume fails with "freebuff is not installed (not found in PATH)", in the status bar before a new tab opens, or after burrow-G exits for a resume here, like a missing folder |

Opening a chat in the other product works and continues the conversation; the next turn runs that product's agent and billing (rationale, "Cross product resume").

### 7. Start a new chat

`StartCommand(prompt)` returns `["codebuff", prompt]`, whatever is installed. Codebuff takes the first message as its words (`codebuff [prompt...]`). Freebuff takes none: its only argument is `login` or `doctor`, and its parser drops any prompt (checked in 0.2.38's `--help` and its argument parser; `freebuff <prompt>` fails with "Allowed choices are login, doctor").

The handoff dialog (spec 0004) offers "Continue in Freebuff" when that first word is on `PATH`, so the row shows only when `codebuff` is installed and is hidden with only `freebuff`. It keeps the shared "Freebuff" label. Resume is unaffected: `--continue <id>` works in both (section 6).

### 8. Export

Freebuff has no structured export: its commands are `login` and `doctor`, and `/export` asks the model to write a Markdown summary. So `Export` reads the chat's files (spec 0003, section 4: own reader).

- **Where**: `<Home>/projects/<base name of Dir>/chats/<ID>/`, the folder `List` found (section 4, rule 3). An id with `/`, `\` or `..`, or a chat `List` would skip, is an error naming the id.
- **Conversation**: `chat-messages.json`, the chat as Freebuff shows it, read one message at a time. After a compaction Freebuff keeps the earlier messages there ("Earlier messages stay here; the agent continues from a summary of them"), so the whole history is in it. Older versions also stored a copy of the run state in each AI message; it is skipped.
- **Compaction**: the summary is only in `run-state.json` → `sessionState.mainAgentState.messageHistory`, as the user message holding `<conversation_summary>`; its text is the part inside `<historical_memory>`, its time the message's `sentAt`. It becomes a `compaction` record before the first message newer than it. Freebuff keeps only the latest summary, so earlier ones can't be exported.
- **Times**: from the message id (`user-<ms>`, `ai-<ms>-<hex>`, `sys-<ms>`, `divider-<ms>`); the `timestamp` field is only a local `HH:MM`. An id without a time takes the previous message's. Header: title, folder and `Updated` from `readChat`, `Created` from the first message (`Updated` when there is none).

| In `chat-messages.json` | Becomes |
|---|---|
| `user` message | user text (`content`) |
| its `attachments`, `textAttachments`, `fileAttachments` | dropped as `attached file` (paths, and pasted text only as a preview) |
| `ai` message | one turn, split into replies the way the model made them: tool calls join the reply of the text before them, the next text or reasoning after a tool call starts a new reply. Each reply is followed by one user record with its results (spec 0003, section 5, rule 6) |
| `text` block | `text`, or `reasoning` when `textType` is `reasoning` |
| `tool` block | `tool_call` (`toolCallId`, `toolName`, `input`); its `output`, the text Freebuff shows, as the `tool_result`. A tool still running has no output and gets no result |
| `ask-user` block | `tool_call` `ask_user` with `{questions}`, result `{answers}` or `{skipped}` |
| `agent` block (a subagent; `spawn_agents` leaves no tool block of its own) | `tool_call` `spawn_agents` with `{agents: [{agent_type, prompt, params}]}`, id `agentId`; the subagent's text as its result. Its other steps are dropped as `sidechain line` |
| `ai` message with `sys-` id (notices like "Context compacted"), `error` messages, `mode-divider` and unknown blocks | dropped as `bookkeeping line` |
| a message that doesn't fit the shape | dropped as `unreadable line`; a file cut off midway is an error |

Not in the export, because `chat-messages.json` doesn't keep them: the model id, and the error flag of a tool result (a failed tool's result is its error message).

### 9. Resolved inputs

| Value | Source |
|---|---|
| `Home` | `filepath.Join(home, ".config", "manicode")` in `main.go`; both launchers use `os.homedir()/.config/manicode` on Linux and macOS |
| installed CLIs | `Source.LookPath`, `exec.LookPath` in `main.go`, a fake in tests |
| project folder name | the `projects/<project>` directory name |
| ID, title, folder, last used | section 3 |

### Canonical pattern

```go
// agents/freebuff/parse.go
func pickCLI(lookPath func(string) (string, error), names ...string) string {
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	for _, name := range names {
		if _, err := lookPath(name); err == nil {
			return name
		}
	}
	return names[0]
}

// agents/freebuff/source.go
func (f Source) ResumeCommand(s session.Session) []string {
	// Freebuff and Codebuff share the chats folder and both resume any chat in it.
	return []string{pickCLI(f.LookPath, "freebuff", "codebuff"), "--continue", s.ID}
}

// agents/freebuff/source_test.go
func TestResumeCommand_OnlyCodebuff(t *testing.T) {
	src := Source{LookPath: installed("codebuff")}
	got := src.ResumeCommand(session.Session{ID: "2026-08-31T11-08-58.778Z"})
	want := []string{"codebuff", "--continue", "2026-08-31T11-08-58.778Z"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
```

**Enforcement**: tests in `internal/agents/freebuff/` (`var _ session.Source = Source{}`, chat folders built in `t.TempDir()` for every rule in section 4 and the title order in section 5 (the cache check needs mtimes that git doesn't keep), `run-state.json` shapes with decoy `cwd` keys in `testdata/`, `ResumeCommand` and `StartCommand` tables for sections 2, 6 and 7 (order, fallback to `freebuff`, a nil `LookPath` using the real `PATH`), and an export fixture in `testdata/export/` with every row of section 8). The local QA kit gets a Freebuff fixture.

**Rollout**: one PR: the spec, then one commit with the reader (all six methods), picker, wiring, display name and docs. Specs 0003 and 0004 landed while this was in review, so `StartCommand` and `Export` are part of that commit: without them it doesn't build.

**Docs**: README "Supported agents" row ("Freebuff / Codebuff", which has room there; `~/.config/manicode/projects/`, `freebuff --continue <id>` or `codebuff`), `internal/agents/AGENTS.md` key files row, `CHANGELOG.md` line; the agent tables in specs 0003 (section 4) and 0004 (section 4) get a Freebuff row.

**Exceptions** (out of scope):
- Freebuff Desktop (`~/.config/freebuff-desktop/`): its threads live in per project SQLite files that the CLIs don't read, and the app accepts no outside link to open a thread (rationale), so burrow-G could list them but never open them.
- `FREEBUFF_CONFIG_DIR` and the non production `~/.config/manicode-<env>` folders.
- `Delete` returns an error, like every agent today.

## Consequences

**Positive**:
- Freebuff and Codebuff users see and resume their chats next to every other agent's.
- Resume never lands in a different chat because of a mismatched folder.

**Negative / tradeoffs**:
- Freebuff and Codebuff chats can't be told apart, so a Codebuff chat resumes in Freebuff when both are installed.
- Chats whose project root and cwd both differ from their project folder name (a chat that switched folders in the app) are hidden rather than shown with a risky resume.
- Of two chats with the same id in different project folders, the older one is hidden (section 4, rule 8). It takes two chats started in the same millisecond.
- Freebuff updates itself almost daily; a layout change shows up as missing chats, not as an error.

**Neutral**:
- No index schema change: the new agent key is `freebuff`.
- `ResumeCommand` now runs a `PATH` lookup each time the details pane draws; it is a few `stat` calls.

## Follow-up

- [ ] A real cross product resume (Freebuff chat in Codebuff and back) by someone with a Codebuff account (verify.md).
- [ ] If a deleted chat folder still shows before a refresh, resume opens Freebuff's most recent chat; consider a chat folder check before resume when delete support lands.
- [ ] Export a chat compacted by a current Freebuff and check the summary lands where the compaction happened (verify.md): the local chats predate compaction.
- [ ] Revisit Freebuff Desktop if it ever accepts a `freebuff://` link that opens a thread.
