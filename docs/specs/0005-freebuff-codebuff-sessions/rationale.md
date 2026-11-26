# 0005. Freebuff and Codebuff sessions: rationale

Decision record for [index.md](index.md). This file explains why.

## Context

Issue #20 asks for a Freebuff reader. It left three questions open: whether the paid `codebuff` CLI shares the folder and should be covered, whether the folder is the same on macOS, and whether `freebuff --continue <id>` really resumes. The answers below come from the files on a Linux machine (Freebuff 0.0.162, then 0.2.38 after a self update) and from reading the JavaScript bundled inside the current release binaries, Freebuff 0.2.38 and Codebuff 1.0.688, downloaded from codebuff.com and never run. Nothing below has been run against a Codebuff account.

## Findings

### One folder, one format

- Both npm launchers default the config folder to `path.join(os.homedir(), '.config', 'manicode')` on every platform, and install their binary there (`~/.config/manicode/freebuff`, `~/.config/manicode/codebuff`). macOS uses the same path.
- Both binaries build the chat folder the same way: `<config>/projects/<path.basename(projectRoot)>/chats/<new Date().toISOString().replace(/:/g, "-")>/`, holding `chat-messages.json`, `chat-meta.json` and `run-state.json` (plus `log.jsonl`).
- Both use the same `chat-meta.json` schema, `{messageCount, firstPrompt, messagesSize, messagesMtimeMs}`, and trust it only while size and mtime match `chat-messages.json`.
- Both accept `--continue [conversation-id]` and `--agent <agent-id>`.
- The project root is the folder the CLI started in (`process.cwd()` after `--cwd`), or the folder chosen later with the in-app change directory command.

### `--continue` falls back silently

Given an id, the CLI looks for `projects/<basename(project root)>/chats/<id>`. If that folder isn't there, it logs at debug level and opens the most recently saved chat of that project instead. A résumé started in the wrong folder therefore opens the wrong conversation with no error. This is why `Dir` is checked against the project folder name, and why `projectRoot` comes before `cwd`.

### Products can't be told apart

There is no product field in any chat file. The only hint is `sessionState.mainAgentState.agentType` (Freebuff chats here: `base2-free-mimo`, `base3-free-mimo`), but:

- both binaries contain both sets of agent ids (shared code, a build flag picks the product);
- both accept any agent with `--agent`;
- every new prompt overwrites `agentType` with the resuming CLI's agent (next section), so one cross product resume erases the hint;
- `agentType` sits after `fileContext`, about 0.5 MB into `run-state.json`, so reading it costs far more than the folder.

Codebuff's last npm release was 2026-09-29, Freebuff's 2026-11-24, and the `CodebuffAI/codebuff` repository now redirects to `CodebuffAI/freebuff`.

### Cross product resume

Read from both binaries' code:

1. Loading a chat (`--continue`) reads `run-state.json` and `chat-messages.json` from the chat folder with no check of which product wrote them.
2. On the next prompt, the CLI sets `mainAgentState.agentType` to the `--agent` id or its own mode's agent, then runs with the loaded state as `previousRun`. The history carries over; the turn runs on the resuming product's agent, models and billing (Codebuff credits, or Freebuff's free models with ads).
3. Both write the chat back in the same format, so the other CLI can still open it later.

So opening a Freebuff chat in Codebuff, or the reverse, works and keeps the conversation. Not checked: whether the server refuses a history produced by the other product's models. Switching models inside one product already sends such a history, so a refusal is unlikely. verify.md lists the manual check.

### Freebuff Desktop

The desktop app (AppImage 0.0.205) keeps threads in `~/.config/freebuff-desktop/projects/<name>-<uuid>/desktop-v2.db` (SQLite, table `threads`; the column set differs between files). The CLIs never read it. Its `freebuff://` handler accepts only `freebuff://auth/complete` (sign in) and `freebuff://agentic-lab?skillId=…&version=…`; thread windows open from inside the app only (`?thread=<id>&project=<path>` on its own local URL). burrow-G could list desktop threads but could not open them, so they stay out.

## Options considered

### Option 1: One reader, CLI picker, checked folder (chosen)

One `freebuff` reader for the shared folder, labeled "Freebuff" in the agents pane. A small `pickCLI` in the reader chooses `freebuff`, else `codebuff`, by `PATH`. `Dir` from `projectRoot`, else `cwd`, kept only when its base name matches the project folder.

- Matches what is on disk: one folder, one format, either CLI resumes any chat.
- Resume never opens the wrong chat.
- The picker is one small function, tested through `ResumeCommand`. It stays in the reader because Freebuff is the only agent with two CLIs; shared code with one caller would be a guess about future agents.

### Option 2: Two readers split by `agentType`

A `freebuff` and a `codebuff` row, each resuming with its own CLI.

- Looks tidier, but the split is a guess (shared ids, `--agent`), a cross product resume moves chats between rows, and each chat needs a 0.5 MB read.
- Rejected.

### Option 3: Freebuff only, `cwd` as the folder (issue text)

- Simplest. Codebuff users would see their chats labeled Freebuff and resume them in a CLI they may not have, and a chat whose `cwd` differs from the project root would resume into the wrong conversation.
- Rejected.

### Option 4: Pick the CLI by the binary in `~/.config/manicode`

- The launchers put binaries there, but a leftover binary after `npm uninstall` would win, and resume runs what `PATH` finds, not that file.
- Rejected in favor of the `PATH` lookup resume already makes.

## Rationale

Option 1 is the only one that is honest about what the files can tell (nothing about the product), safe on resume (folder checked). Its one cost, Codebuff chats resuming in Freebuff when both are installed, is harmless by the cross product finding, and is what the user chose: Freebuff by default, Codebuff as the fallback.

## Export: which file (added 2026-11-27)

Specs 0003 and 0004 gave every agent `Export` and `StartCommand`. Freebuff's CLI has no export command, and its `/export` asks the model for Markdown, so it gets its own reader. Two files hold a conversation:

| | `chat-messages.json` (what Freebuff shows) | `run-state.json` → `messageHistory` (what the model sees) |
|---|---|---|
| whole history | yes: earlier messages stay after a compaction | no: a compaction replaces them with one summary message |
| tool calls | name, input, and the result as the text shown | name, input, result as data |
| model id, real times | no (times only in message ids) | yes |
| also holds | notices, mode dividers | Freebuff's own instructions and reminders |

Spec 0003 asks for the whole history with summaries where they happened, so the conversation comes from `chat-messages.json` and only the summary from `run-state.json`. Mixing the two message by message to win back model ids and result data was rejected: the run state covers only the turns since the last compaction, so part of an export would carry them and part wouldn't, and matching the two lists is a guess wherever Freebuff dropped or rewrote messages.

What this rests on: the shapes of both files on this machine (two used chats, from a Freebuff version older than compaction) and Freebuff 0.0.127's own code for the rest (`ai-<ms>-<hex>` ids, `ask-user` and `agent` blocks, the summary message, `chat-title.txt`). Exporting the two real chats gives every one of their 19 tool calls with its result. A compacted chat is still to be checked (verify.md).
