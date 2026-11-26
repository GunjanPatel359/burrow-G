---
title: Agent transcript formats differ from what their names suggest
author: Prashant2307200
date: 2026-11-26
branch: feat/session-handoff
cost: { minutes: 2, tokens: 2925 }
id: l-2cec2988
scope: repo
status: active
src: agent
touches:
  - internal/agents/qoder/transcript.go
  - internal/launch/gnome.go
---

## Decided
Read each agent's transcript by checking real local data, not field names, because three fields looked right in fixtures but were wrong on real sessions.
Verify export and handoff against the real burrow-g binary in a sandbox HOME with fixture sessions, and put stand-in scripts named claude, codex, opencode, copilot and qodercli first on PATH. A real handoff sends the conversation to an outside model provider and starts a real session in the agent's own store, so a test run should never start the real CLI.

## Watch out
- Copilot: subagent events are the ones with `parentToolCallId` set. Filtering on `parentAgentTaskId` instead dropped every user prompt, because Copilot also sets it on the person's own queued prompts (real sessions exported with 1 message instead of 4).
- OpenCode v2: the message kind (user, assistant, system, synthetic, idle) is the `type` column of `session_message`, not a field in `data`, otherwise every message reads as unknown. User text is `data.text`; assistant parts are `data.content[]`.
- Codex: the `exec` tool's input is JavaScript, not JSON, so the tool line shows only `exec`. Most Codex tool lines carry no argument for that reason.
- Copilot sessions can have an `events.jsonl` with no messages; fall back to the `turns` table then too, otherwise export fails with "no messages found".
- A real `gnome-terminal --tab -- claude ...` can resolve `claude` on the terminal server's PATH, not the caller's, which would start the real agent. Test the tab path with a stand-in `gnome-terminal` script, or pass an absolute path to a stand-in agent. With an absolute path, a real GNOME tab passed a 99,768 byte prompt through byte for byte.
- Shell `$(cat file)` strips trailing newlines, so a prompt sent that way looks 2 bytes short; pass exact bytes from Python instead.
- Fixture JSONL must be compact (`"type":"user"`, no space), because the lister matches that exact text and skips a session written with spaces.
- Qoder writes `timestamp` as a number (epoch ms) on its bookkeeping records (`runtime-config`, `active-leaf`), while user and assistant records use a time string. A reader that decodes `timestamp` as a time fails those whole lines and counts them as skipped although nothing was lost.

## In the code:
internal/agents/copilot/transcript.go, internal/agents/opencode/transcript.go, internal/agents/codex/transcript.go, internal/transcript/render.go (ToolLine)
internal/agents/qoder/transcript.go (record.Timestamp), internal/launch/gnome.go (OpenTab), screenshots: tmux capture-pane -e, then ANSI to HTML, then google-chrome --headless --screenshot
