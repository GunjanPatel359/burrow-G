# 0004. Session handoff: rationale

The decision record behind [index.md](index.md).

## Context

Issue #7 wants a session's context carried into another agent. Spec 0003 gives every agent an `Export` to one portable format with tool calls, their results and compaction summaries. What's left is getting that history into a target agent so the user can keep working.

The forces:

- **Read only.** burrow-G never changes agents' files; that is why people trust it with their agent stores.
- **Only OpenCode can import natively** (`opencode session import`, proven in the spec 0003 spike). Claude Code, Qoder, Codex and Copilot CLI (1.0.80) have no import command.
- **Every agent accepts a first prompt** on its command line (checked against each `--help`, 2026-11-26).
- **Sessions are huge, context windows are not.** The real z/OS session exports to 5.8 MB (about 1.5M tokens). Its latest compaction summary is 19 KB; everything after it is 79 KB, 70 KB with tool output cut at 2 KB each.
- **A command line argument** is capped at 128 KiB on Linux and is visible to other users in the process list.
- **History goes to another vendor's model**, so secrets pasted into a session travel further than before. Masking them is left for later (decided 2026-11-26); this version says so in the README.

## Options considered

### Option 1: handoff through a file, from the export, for every agent

Render a bounded Markdown file from the portable export (latest summary plus recent messages, long items cut); start the agent with a short prompt that points at the file.

**Pros**: one path for all five agents in both directions; the read only rule holds (the target writes its own session); no reader of its own, so only spec 0003's readers can break; no size limit or process list exposure from the prompt; the summary means a long session loses nothing in the middle that the agent itself hadn't already condensed.

**Cons**: not a replay, the target reads history rather than resuming it; agents may ask permission to read a file outside the project; OpenCode, which could import the full history, gets the same summary as everyone.

### Option 2: Option 1 plus native import for OpenCode

**Pros**: OpenCode sessions keep the full history as a real OpenCode session.

**Cons**: two behaviours to explain and test; tied to OpenCode's private import format (`projectID`, `cost`, `tokens`, globally unique ids, found by trial in the spike). Left for later (Follow-up): it fits inside OpenCode's `source.go` without changing anything here.

### Option 3: the transcript as the first prompt (PR #24)

**Pros**: no file read needed.

**Cons**: 100 KB cap from the argument limit, and the transcript is visible in the process list; PR #24's own readers keep one line per tool call with no output, and cut the middle of long sessions blindly.

### Option 4: write native session files for every agent

**Pros**: the highest fidelity, a real `--resume`.

**Cons**: breaks the read only rule; one writer per agent, each tied to a private format; writing a store while its agent runs risks corrupting it.

## Rationale

Option 1 keeps the rule that makes burrow-G safe to run and needs no new reader: it renders from the export spec 0003 already built. The budget answers the size force with the agent's own condensing (the compaction summary) instead of a blind cut, and the file answers the argument limit and the process list together. The user chose handoff only, with no native import for now (2026-11-26); Option 2 stays the natural next step.

The dialog on `e` replaces a separate handoff key: saving and continuing are the same question, "where does this session go?". The picker and the per agent start commands follow PR #24 by Prashant Dobariya, which designed and checked them first.
