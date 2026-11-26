---
title: "Moving a session to another agent is a handoff, not an import into that agent's store"
author: Prashant2307200
date: 2026-11-26
branch: feat/tui-design-tokens
cost: { minutes: 13, tokens: 197482 }
id: l-b6669099
scope: repo
status: active
src: agent
touches:
  - docs/specs/0002-session-export-handoff.md
---

## Decided
Moving a session between agent CLIs starts a fresh session in the target agent, in the same folder, with the old conversation (Markdown, secrets masked, capped at 100 KB) as its first prompt. burrow-G writes nothing into any agent's files; the target saves its own session. A handoff launches straight after you pick the target, with no confirm step; a one line report (messages, size, omitted, masked) follows.

## Tried and ruled out
- Writing native session files into each agent's store: breaks the read only rule, and every agent release would break the writers.
- Native import via `opencode session import` / `copilot sessions import`: private formats (Copilot's has no public schema), two behaviours to explain. Kept as the runner up if people want history inside those agents' own pickers.
- A y/n confirm before handoff: picking the target is already deliberate; the report after is enough.

## Watch out
- One command argument is capped at 128 KiB on Linux, so the prompt budget (100,000 bytes, preamble included) must stay under it.
- Qoder's first prompt flag and `opencode --prompt` behaviour were not verified when this was designed.

In the code: spec docs/specs/0002-session-export-handoff.md; new interface session.Transcriber (Transcript, StartCommand); shared code planned in internal/transcript/.
