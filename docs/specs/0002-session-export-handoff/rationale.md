# 0002. Session export and handoff: rationale

The decision record behind [index.md](index.md).

## Context

> ⚠️ Premise note: the request says "import", but a true import (making a Claude session appear as a native, resumable OpenCode session with its history intact) is not something burrow-G can do faithfully. Tool calls, model ids, and message shapes are vendor specific, so a converted session would replay tool calls the target never made, on a model it may not have. Writing that conversion into the target's store would also break the project's core rule that agents' files are only read. The goal behind "import" is "keep working on this in another agent without losing the thread", and that is met by handing over the conversation as context, not by forging the target's storage. This spec designs that.

People switch agents in the middle of a task: one hits a rate limit, another is better at a kind of change, or a long session has grown too big to keep working in. Today the only way across is copying text by hand. burrow-G already finds every session across all five agents and knows each one's folder, so it is the natural place to carry a conversation from one agent to the next.

The forces are these. burrow-G only ever reads agents' files (`AGENTS.md`, Rules), and that rule is why people trust it with their agent stores. Each agent keeps transcripts in a different place and shape: Claude and Qoder use JSONL files, OpenCode and Copilot use SQLite plus side files, Codex uses rollout JSONL files, and every one of them changes its format between versions (OpenCode already renamed `session` to `session_v2`). A transcript sent to another agent goes to a different vendor's model, so secrets pasted into a session leak further than before. Linux caps a single command argument at 128 KiB, and every agent's context window is finite, so long sessions cannot be passed whole.

Two agents ship their own import commands (`opencode session import` takes its own export JSON; `copilot sessions import` takes an undocumented "semantic session" JSONL). Claude, Codex, and Qoder have none. If this is left undecided, each agent pair would get its own ad hoc converter, and every agent format change would break several of them at once.

## Options considered

### Option 1: Handoff through a first prompt

Render the source transcript as Markdown and start a new session in the target agent with that Markdown as its first prompt, in the source session's folder. Every agent accepts a first prompt on its command line (`claude "<p>"`, `codex "<p>"`, `opencode --prompt`, `copilot -i`).

**Pros**:
- Works for all five agents in both directions with one renderer and one small `StartCommand` per agent.
- Keeps the read only rule intact; the target agent writes its own session in its own format.
- Survives target format changes completely; only source readers ever break, and each breaks alone.
- The target model sees plain prose it can reason about, not foreign tool call records.

**Cons**:
- The new session is not a replay: the target cannot undo or inspect the source's tool calls, only read a summary.
- Long sessions are trimmed to fit the 100 KB budget, so the middle of a long session is lost to the target.
- The handoff spends part of the target's context window on history.

### Option 2: Native import where the target supports it, handoff elsewhere

Convert to OpenCode's export JSON and run `opencode session import`, or to Copilot's semantic JSONL and run `copilot sessions import`, so the result appears in that agent's own list with history; use Option 1 for Claude, Codex, and Qoder.

**Pros**:
- For two targets, the history shows up as a real session in that agent's own picker.
- The agent's own tool writes the store, so the read only rule still holds.

**Cons**:
- Two private formats to produce, one of them (Copilot's) with no public schema, and both tied to provider and model ids the source session may not have.
- Two behaviours to explain and test: some handoffs give a native session, others a fresh one.
- Every OpenCode or Copilot format change can break import silently, on top of reader breakage.

### Option 3: Write native session files for every target

burrow-G writes a real session file into each target's store (a new Claude JSONL under `~/.claude/projects/`, a Codex rollout, rows in OpenCode's database), so every agent resumes the converted session natively.

**Pros**:
- Highest fidelity; the converted session resumes with `--resume` like any other.

**Cons**:
- Breaks the project's founding rule and the trust that comes with it; a bad write can corrupt an agent's store.
- Ten format writers (five readers times five writers) that break with every agent release.
- Writing into another vendor's store while that agent is running risks concurrent write corruption.

### Option 4: Export only

Ship export to Markdown and stop; people paste the file into the next agent themselves.

**Pros**:
- Smallest build; no launch code, no picker, no budget.

**Cons**:
- Leaves the main job (continue in another agent) as manual copy and paste.
- Pasting by hand skips the budget and redaction, which is exactly where mistakes happen.

## Rationale

The read only rule is the load bearing force. Options 2 and 3 each trade it away (fully, or by betting on private formats) for history that the target model cannot really use: a model reading another vendor's tool call records gains little over a clean summary of what was done. Option 1 keeps the rule, needs one renderer instead of a writer per target, and turns the "agents change formats" force from a quadratic problem into a linear one: each reader breaks alone, and no target ever breaks.

The budget and redaction answer the two remaining forces. 100,000 bytes stays under Linux's 128 KiB limit for one argument with headroom for the preamble, and is about 25k tokens, small next to every current agent's context window. Keeping the first user message plus the newest messages preserves the goal and the current state, which is what a person continuing the work needs; the middle is the cheapest part to lose. Masking known key shapes before the text leaves for another vendor is cheap and catches the common leak (a key pasted into a prompt), while leaving tool outputs out already removes most of the rest.

Option 2 is the runner up. If people later ask for handed off sessions to show their history inside OpenCode or Copilot, it can be added on top of this design (same transcript model, one extra writer per target) without changing anything built here.
