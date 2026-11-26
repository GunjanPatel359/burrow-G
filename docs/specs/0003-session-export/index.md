# 0003. Export any session to one portable format

**Date**: 2026-11-26
**Status**: Proposed

## Summary

Every agent reader gets an `Export` method that writes a session in one portable format: a JSON lines file with a header, then every message and compaction summary in order, tool calls and their full results included. Each agent decides on its own how it gets the data: through the agent's own export command when that command gives structured data (OpenCode today), and through its own reader of the agent's files when it doesn't (Claude first, the others one at a time). Callers never know which. This is the first half of moving sessions between agents and machines; importing a portable file into an agent is a later spec built on this format.

## Decision

**Chosen option**: Option 1: `Export` on `session.Source`, writing the portable format, native where the agent's export is structured. See [rationale.md](rationale.md).

**Implementation skills**: `golang-testing` (`.agents/skills/golang-testing/`, standard `testing` only) · `golang-error-handling` (`.agents/skills/golang-error-handling/`, sentinel errors) · `golang-database` only if a reader touches SQLite

## Rationale

Options, the spike that measured them, and why: see [rationale.md](rationale.md).

## Standard definition

### 1. The method

```go
// internal/session/session.go
type Source interface {
	Name() string
	List() ([]Session, error)
	ResumeCommand(s Session) []string
	Delete(s Session, trashDir string) error
	// Export streams s in the portable format, oldest record first.
	Export(s Session, w *portable.Writer) error
}

// ErrUnsupported means this agent can't do that yet. Callers check it with
// errors.Is and say so, like ErrMissing.
var ErrUnsupported = errors.New("not supported for this agent")
```

- Every agent implements it. An agent without a reader yet returns `ErrUnsupported` (wrapped is fine).
- `Export` writes the header, every record, and nothing else. The caller creates the `Writer` and calls `Close`, which writes the end line (section 2). A failed `Export` leaves no end line, so a reader can tell a cut off file.
- `Export` reads agents' files and never changes them. The native route runs the agent's own CLI, which may touch its own store the way any use of that agent does; burrow-G itself still opens nothing for writing.
- `Export` streams. It never holds a whole session in memory; the largest local Claude session is 105 MB.

### 2. The portable format (`internal/portable`)

One JSON object per line, UTF-8, `\n` endings, HTML not escaped. Line 1 is the header, the last line is the end line, every line between is a record. Unknown fields are ignored on read, so later versions can add fields; a change that breaks old readers bumps the version in `format`.

```jsonl
{"kind":"header","format":"burrow-session/1","agent":"claude","id":"3cbaaae5-…","title":"zOS support in motaCH","dir":"/home/u/motaCH/pluginengine","created":"2026-10-27T06:33:07Z","updated":"2026-11-26T05:35:37Z","exported":"2026-11-26T06:00:00Z","burrow":"0.0.3"}
{"kind":"message","role":"user","time":"2026-10-27T06:33:07Z","parts":[{"type":"text","text":"add z/OS support"}]}
{"kind":"message","role":"assistant","time":"…","model":"claude-opus-5","parts":[{"type":"text","text":"Reading the plugin first."},{"type":"tool_call","id":"toolu_01","name":"Bash","input":{"command":"ls"}}]}
{"kind":"message","role":"user","time":"…","parts":[{"type":"tool_result","id":"toolu_01","text":"main.go\n"}]}
{"kind":"compaction","time":"…","text":"Summary of the earlier conversation …"}
{"kind":"end","records":4,"dropped":{"thinking without text":618,"injected text":40}}
```

```go
type Header struct {
	Kind     string    `json:"kind"`   // "header"
	Format   string    `json:"format"` // "burrow-session/1"
	Agent    string    `json:"agent"`  // Source.Name(): tool names below are this agent's
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Dir      string    `json:"dir"`
	Created  time.Time `json:"created"`
	Updated  time.Time `json:"updated"`
	Exported time.Time `json:"exported"`
	Burrow   string    `json:"burrow"` // burrow-G version that wrote it
}

type Record struct {
	Kind  string    `json:"kind"`            // "message" or "compaction"
	Role  string    `json:"role,omitempty"`  // message: "user" or "assistant"
	Time  time.Time `json:"time"`
	Model string    `json:"model,omitempty"` // assistant: the model id the agent recorded
	Parts []Part    `json:"parts,omitempty"` // message
	Text  string    `json:"text,omitempty"`  // compaction: the summary
}

type Part struct {
	Type      string          `json:"type"`                 // text, reasoning, tool_call, tool_result, image
	Text      string          `json:"text,omitempty"`       // text, reasoning, tool_result
	ID        string          `json:"id,omitempty"`         // tool_call and its tool_result share it
	Name      string          `json:"name,omitempty"`       // tool_call: the agent's own tool name
	Input     json.RawMessage `json:"input,omitempty"`      // tool_call: the agent's own arguments
	Error     bool            `json:"error,omitempty"`      // tool_result: the tool failed
	MediaType string          `json:"media_type,omitempty"` // image
	Data      string          `json:"data,omitempty"`       // image: base64
}

type End struct {
	Kind    string         `json:"kind"` // "end"
	Records int            `json:"records"`
	Dropped map[string]int `json:"dropped,omitempty"` // what was left out, by reason
}
```

`Writer`: `NewWriter(io.Writer)`, `Header(Header) error`, `Record(Record) error`, `Drop(reason string)` (counts one thing left out), `Close() error` (writes the end line). It buffers and flushes on `Close`; records are written as they come.
`Reader`: `NewReader(io.Reader)` reads and checks the header (`format` must start with `burrow-session/`, else an error), then `Next() (Record, error)` returns records and `io.EOF` after the end line. A file with no end line returns `ErrTruncated` instead of `io.EOF`. Lines up to 64 MB are accepted (one real Claude line is 860 KB).

### 3. What goes in and what stays out

| Goes in | Stays out (counted in `dropped`) |
|---|---|
| the conversation the user sees: user text, assistant text | the agent's bookkeeping: request copies, attachments, cost and mode lines, hook output |
| every tool call with its own name and arguments, unchanged | thinking stored only as a signature, with no text (Claude) |
| every tool result, whole: text, error flag | text the agent injects into user turns (`<system-reminder>`, `<command-…>`, meta lines) |
| images, as base64 `image` parts: in a message, or right after their `tool_result` with the same `id` | sidechains and subagent transcripts (their parent's tool call and result stay in) |
| reasoning text, when the agent stores it as plain text (OpenCode) | — |
| compaction summaries, where they happened | — |

- Tool names and arguments are never mapped to another agent's tools; the header's `agent` says whose they are. Mapping is the importer's job.
- No secret masking. An export is a faithful copy for moving a session, and masking changes tool output. The file is written mode 600 and the README says it can hold code and secrets (section 6).
- The whole history is exported, every compaction segment. An importer that wants less (only since the last compaction) cuts it on its side.

### 4. Native or own reader, per agent

Rule: inside each agent's `Export` method, use the agent's own export when it gives **structured** data (messages, tool calls and results as data); otherwise the non native code lives right there, in that method and its package, reading the agent's files. Either way `Export` writes the portable format, and no caller knows or cares which route an agent took.

| Agent | Route | Status in this spec |
|---|---|---|
| OpenCode | native: `opencode session export --standalone <id>` (JSON), converted | step 2 |
| Claude Code | own reader of `~/.claude/projects/*/<id>.jsonl` (its `/export` is plain text, only inside a running session) | step 1 |
| Qoder | own reader (JSONL close to Claude's) | step 3 |
| Codex | own reader of rollout files | step 3 |
| Copilot CLI | own reader, until it ships a structured export | step 3 |

When an agent later ships a structured export, its package switches to it; nothing outside the package changes.

### 5. Claude reader (step 1)

Facts from real files (Claude Code 2.1.332), each covered by a fixture:

1. **Two passes over the file.** Pass 1 keeps only `uuid → parentUuid` (plus title, folder, times); pass 2 streams the lines on the live branch in file order. Memory stays about one map entry per line, not the file.
2. **The live branch** is walked back from the `leafUuid` of the last `last-prompt` line, or the last non sidechain `user` or `assistant` line when that is missing or unknown.
3. **Compaction starts a new chain.** A `system` line with subtype `compact_boundary` has `parentUuid: null`. Its `logicalParentUuid` points *forward* to an attachment after the summary, not back to the older history, so it is ignored. The older history is the segment before each boundary in file order: each segment is walked back from its own last non sidechain message. All segments are exported, oldest first.
4. **The summary** is the `user` line with `isCompactSummary: true`; it becomes a `compaction` record, not a user message.
5. **Parallel tool calls branch.** Claude chains call A → call B and hangs result A off call A as a side branch. A `user` line carrying a `tool_result` whose parent is on the live branch is kept even when it is not on the branch itself.
6. **One assistant reply spans several lines**, one per content block, sharing `message.id`, and tool results can sit between them (call A, result A, call B, all one reply: 181 times in the 15 largest local files). The export follows the API's shape, not file order: one `assistant` record with every block of the reply, then one `user` record with all of that reply's tool results, in call order. Pass 2 holds the results of the reply in progress until a line with a different `message.id` (or a real user turn) ends it. This is the grouping every importer needs: Anthropic and OpenAI style APIs both want all results of a reply right after it.
7. `thinking` blocks with empty `thinking` text (signature only) are dropped and counted; with text they become `reasoning` parts.
8. Lines that are not JSON are skipped and counted (`unreadable line`), never fatal.
9. Header title and `dir` come from the same `parseFile` that `List` uses (`/rename` sidecar, then the first `ai-title`, then the first prompt), so an export and the list never disagree.
10. **Injected text in user turns.** Whole injected blocks are removed by tag name: `<system-reminder>`, `<local-command-caveat>`, `<local-command-stdout>`, and lines with `isMeta: true` (counted as `injected text`). The real text around them stays. A turn left with no text and no tool result is dropped. A slash command (`<command-name>/foo</command-name>` with `<command-args>`) is kept as the user's text `/foo <args>`, because that is what the user typed.
11. **Images in a tool result** are kept: an `image` part right after its `tool_result` in the same record, with the same `id`, so an importer can put it back inside the result.

A session that can't be found returns an error naming the id; the file is located the way `List` found it (`projects/*/<id>.jsonl`).

### 5a. Qoder reader (step 3)

Qoder's JSONL follows Claude Code's format, so `qoder/export.go` follows section 5 with these differences, seen in all 7 local sessions (Qoder CLI, 2026-11):

1. The leaf is in `active-leaf` lines (`leafUuid`, the last one is current); `last-prompt` is read too.
2. `active-leaf` and `runtime-config` lines write `timestamp` as milliseconds; every other line as RFC 3339 text. Both are read, so no line is skipped for its timestamp.
3. A compaction marks two lines with `isCompactSummary`: the `system` boundary (no message, counted as bookkeeping) and the `user` summary, whose content is a list of text blocks.
4. `logicalParentUuid` on a boundary points back to the older segment. The reader doesn't use it either way (section 5, rule 3).
5. Thinking blocks keep their text, so Qoder exports carry `reasoning` parts.

### 6. OpenCode native route (step 2)

- `opencode.Source` gains a `Run func(name string, args ...string) ([]byte, error)` field; `main.go` sets it to an `exec` wrapper. Tests set it to return a saved export file from `testdata/`. Nil `Run` means `ErrUnsupported`, so a reader built without it still compiles and lists.
- It runs `opencode session export --standalone <id>`. `--standalone` uses a private server instead of attaching to (or starting) the user's background service.
- If `opencode` is not on `PATH`, the error says `opencode not found on PATH`.
- `main.go` passes `opencode.Exec`, which sends the command's output to a private temp file (removed after) instead of a pipe. Through a pipe OpenCode exits before the pipe is drained and cuts large exports short with exit status 0: a 17 MB session arrived as 16 MiB + 64 KiB, every time; to a file it arrives whole.
- Conversion: `user` → user message (its `text`, plus `files` as image parts when they are images); `assistant` → one assistant record with its `text` and `reasoning` parts as they are and each `tool` part as a `tool_call` (its `name` and `state.input`), then one user record with all of that message's results in the same order (`state.content` text, or `state.error.message` with `error: true`), the same grouping as Claude (section 5, rule 6); `compaction` → compaction record (`summary`). `system` and `synthetic` (text OpenCode wrote, like "Continue if you have next steps") are dropped as `injected text`; `idle`, `model-switched`, `agent-switched` and anything newer as `bookkeeping line`. Attached files that aren't images are dropped as `attached file`. A tool still running (no `completed` or `error` state) keeps its call and gets no result.
- Measured: 2.0 s and 311 MB for a 17 MB, 1,454 message session; 0.27 s for a small one. Fine for one export at a time.

### 7. Where you use it

Only from the TUI for now: there is no `burrow-g export` command and no command bar entry yet (see Follow-up).

- **`e` on a session** exports it to `~/.local/share/burrow-g/exports/<agent>-<id>.jsonl`. The folder is created mode 700, the file mode 600 (an existing file is narrowed to 600 before it is emptied). The export runs as a background job (`internal/tui/commands.go`), never inside `Update`, with the busy spinner while it runs.
- If the export fails, the half written file is removed, so a file in the folder is always a whole session.
- The key comes from `keys.go` like every other binding (`e`, help `export`), so the status line hints and the help command's key list show it.
- Status line, in `docs/design.md` voice:
  - busy: `exporting` + `t.Glyph.Ellipsis`
  - done: `exported 6716 records to "~/.local/share/burrow-g/exports/claude-3cbaaae5.jsonl"` (the path shortened with `~` like the details box)
  - not yet: `error: export isn't supported for Copilot CLI yet` (the agent's display name)
  - failure: `error: <message>`

### Canonical pattern

```go
// internal/tui: export one session to a private file, as a job.
func exportSession(src session.Source, s session.Session, path, version string) (portable.End, error) {
	f, err := createPrivate(path) // 600, existing file narrowed first
	if err != nil {
		return portable.End{}, err
	}
	w := portable.NewWriter(f, version)
	err = src.Export(s, w)
	var end portable.End
	if err == nil {
		end, err = w.Close()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(path) // never leave half a session behind
	}
	return end, err
}
```

`Close` returns the `End` it wrote, so the status line can report the count.

## Consequences

- Adding an agent now means five methods; `Export` may start as `ErrUnsupported`. `internal/agents/AGENTS.md` gets one line saying so.
- The portable format is a public contract once released: change it only by adding fields, or bump `burrow-session/N` and keep reading the old one.
- Claude's file format is undocumented and changes between versions. The reader skips what it doesn't know instead of failing, and the fixtures pin the shapes above; a new Claude version that breaks them shows up as missing tool results or a short export, which section 8 of `verify.md` checks on real files.
- OpenCode export depends on its CLI being installed and keeping `session export` working. That is a public command it needs for its own import, so it is the more stable surface compared with its database (50 migrations, a table rename, event sourced bookkeeping).

## Rollout

1. `internal/portable` with tests; `Source.Export` and `ErrUnsupported`; Claude reader with fixtures; every other agent returns `ErrUnsupported`; the TUI `e` key. README and CHANGELOG.
2. OpenCode native route with a `testdata/` export file.
3. Qoder, Codex, Copilot readers, one commit each.

## Follow-up

- [ ] A `burrow-g export <id> [-o file]` command and a command bar `export` entry, when moving sessions between machines needs the shell (decided 2026-11-26: TUI only for now).
- [ ] Spec 0004: import a portable file into an agent (OpenCode through `opencode session import`, others to be decided), with tool name mapping.
- [ ] Markdown rendering of a portable file, for reading and sharing, if people ask for it.
- [ ] PR #24 (spec 0002, Markdown export and handoff) is not built on: this spec has its own readers inside each agent's `Export`. If handoff comes back, it renders from `Export`, so each agent keeps one reader.
