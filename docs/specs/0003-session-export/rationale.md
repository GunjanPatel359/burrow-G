# 0003. Session export: rationale

The decision record behind [index.md](index.md).

## Context

A session's history is locked in one agent's own files. Issue #7 asks to export it and import it elsewhere: into another agent, or the same agent on another machine. Both are wanted. This spec covers export, in a format rich enough that a later import spec can build on it.

The forces:

- **Read only.** burrow-G never changes agents' files (`AGENTS.md`, Rules).
- **Every agent stores sessions differently** and changes the format between versions: Claude, Qoder and Codex use JSONL, OpenCode and Copilot use SQLite. OpenCode has renamed tables and has 50 migrations.
- **Sessions are big.** Locally the largest Claude file is 105 MB, the 90th percentile 10 MB.
- **Only OpenCode has a structured export command** (`opencode session export`). Copilot (this version), Codex and Qoder have none. Claude's `/export` runs only when typed inside a running session and saves the terminal view: for this spec's own design session it wrote 22 KB against a 1.8 MB `.jsonl`, with tool output cut to what the screen showed (36 `… +N lines (ctrl+o to expand)` marks) and no tool call ids, timestamps or compaction summaries. It can't be run by burrow-G, and nothing could be imported from it.
- **Import needs more than prose.** To appear as a real session in another agent, tool calls, their results and compaction summaries have to survive the trip.

## The spike (2026-11-26)

A throwaway Go converter ran Claude → portable → Claude and → OpenCode on real local sessions. It wrote nothing outside a sandbox except one deliberate import into OpenCode through its own CLI.

| Measured | Result |
|---|---|
| Largest Claude session, 105 MB, 45,144 lines | 8.0 MB portable (2 MB gzipped), 0.8 s, 17 MB peak memory |
| Its make up | 93% bookkeeping (request copies, attachments, duplicated tool output); the conversation is about 7 MB |
| All 66 local Claude sessions | 3.1 s total, 728 MB → 17 MB; every tool call paired with its result except one session cut off mid call |
| Claude → portable → Claude | 6,716 records in and out, 0 differences |
| Claude → portable → OpenCode | accepted by `opencode session import`; 6 compactions and 3,349 tool calls with results, read back by `opencode session export` |
| A real session (z/OS work, 30 MB) imported into OpenCode | 3.6 MB portable, 804 tool calls all paired, listed and opened by OpenCode |
| `opencode session export`, 17 MB session | 2.0 s, 311 MB; 0.27 s for a small one |

What the spike found about the formats, all now in index.md section 5:

- Walking `parentUuid` from the leaf gave only the last compaction segment (2,581 of 22,102 entries). `logicalParentUuid` on a compact boundary points forward, not back.
- Parallel tool calls put results on side branches; walking parents alone lost results in 15 of 66 sessions.
- All 1,520 thinking blocks in the largest session are signature only, with no text, so reasoning can't cross agents from Claude.
- OpenCode import requires fields its export always has (`projectID`, `cost`, `tokens`, `time` on tool parts) and globally unique message ids. That belongs to the import spec.

## Options considered

### Option 1: `Export` on `Source`, one portable format, native where structured

Each agent implements `Export(s, w *portable.Writer)`. Inside, it uses the agent's own export when that gives structured data, else its own reader.

**Pros**:
- One contract, enforced by the compiler, in the same place as `List` and `Delete`; adding an agent is the same checklist as today.
- Callers see one format, whatever the route.
- Native where it exists puts the burden of following OpenCode's storage on OpenCode.
- Rich enough for import: tool calls, results and compactions survive (the spike imported into OpenCode with them).

**Cons**:
- Every agent must at least stub the method.
- The native route needs the agent installed and is slow (seconds, hundreds of MB) next to reading files.

### Option 2: an optional `Exporter` interface, checked with a type assertion

**Pros**:
- Idiomatic Go for a capability only some types have; no stubs.

**Cons**:
- Gaps are invisible: nothing forces a new agent to think about export.
- The codebase already keeps every agent capability on `Source` (`Delete` is a stub today), so this would be the odd one out.

### Option 3: a Markdown transcript model (as in PR #24, spec 0002)

Readers produce prose plus one line per tool call; output is Markdown for reading or a first prompt for handoff.

**Pros**:
- Easy to read and share; works as a first prompt for every agent.

**Cons**:
- Drops tool output and compaction summaries, so nothing can be imported from it as a real session.
- Prose can't be turned back into structure, so import would need a second model later anyway.

### Option 4: always read the agent's files, OpenCode's database included

**Pros**:
- Fast (milliseconds), no dependency on the agent's CLI, the same mechanism `List` uses.

**Cons**:
- burrow-G would re-implement OpenCode's event sourced storage: `session_message` with `seq`, forks (`fork_session_id`, `fork_boundary`), undo state (`revert`), and every future migration.
- OpenCode keeps `session export` working for its own import; its tables have no such promise.

### Option 5: always use the agent's own export

**Pros**:
- No readers to maintain at all.

**Cons**:
- Possible for one agent of five. Claude's `/export` is text inside a running session, and the others have none.

## Rationale

The portable format is load bearing: it is the only option that keeps what an import needs, and the spike proved a real import works from it. Putting `Export` on `Source` follows how the codebase already treats agent capabilities and makes the gaps visible.

The per-agent route is chosen by one rule, structured native export first, because the two costs are different in kind. The native route costs seconds and memory per export, which only matters in bulk, and nothing exports in bulk. Reading a store you don't own costs maintenance with every release of that agent, and that cost never stops. For OpenCode the native export wins. For the others there is nothing native to use, so they get readers, each in its own package, each breaking alone.

Masking is left out on purpose: an export is a faithful copy for moving a session, and masking would change tool output. Mode 600 files and a README warning cover the risk of a file someone chose to write. Masking belongs to anything that sends history to another vendor's model, which is the import spec's question.

Option 4 is the runner up for OpenCode. If `session export` ever disappears or becomes too slow, the reader can switch to the database without changing anything outside its package.
