# Verify: session export and handoff · spec 0002 · updated 2026-11-26
_Steps derived from spec 0002 acceptance criteria. `/check verify` runs these; `/test` locks the durable ones._

## Commands
- [ ] `burrow-g list`, then `burrow-g export <claude id>` → Markdown on stdout: `# <title>`, a line with Agent, Session, Folder, Exported, then `## User` / `## Assistant` blocks with times and `- tool:` lines; no thinking, tool output or subagent text → AC-1
- [ ] Repeat `burrow-g export <id>` for a Qoder, OpenCode, Copilot and Codex session → each prints its conversation; stderr says `exported N messages, M secrets masked` → AC-1, AC-7
- [ ] `burrow-g export <id> -o /tmp/s.md` and `burrow-g export -o /tmp/s.md <id>` → nothing on stdout, the path on stderr, `stat -c %a /tmp/s.md` is `600` → AC-2
- [ ] In a scratch session, paste `AKIAABCDEFGHIJKLMNOP` and a PEM private key block; export it → both show as `[redacted]`, and the masked count counts them → AC-7
- [ ] `burrow-g handoff <claude id> codex` → Codex starts in the session's folder with the preamble naming claude, the id, title and folder, then the transcript; just before, stderr shows `handed off to codex: N messages, X KB prompt, N omitted, N secrets masked` → AC-4, AC-8
- [ ] Hand off a long session (over 100 KB of Markdown) → report shows `omitted` > 0; in the target, the prompt starts with the first user message, has one `[N messages omitted]` line, and ends with the newest message → AC-6
- [ ] Export the same long session → nothing omitted or `[truncated]` → AC-6
- [ ] Hand off a session whose folder was removed → `folder no longer exists: <dir>`, nothing starts; `burrow-g export` of it still works → AC-9
- [ ] Export a session file with a hand corrupted line (copy it to a test home first, never edit the agent's file) → `(1 entries skipped)` under the header; an all corrupt one fails with `no messages found in <agent> session <id>` → AC-10
- [ ] `burrow-g handoff <id> nope` → `unknown agent "nope" (choose from: …)`; with an agent whose CLI is not on PATH (`PATH=/usr/bin burrow-g handoff <id> codex`) → `codex is not installed (not found on PATH)`; nothing starts → AC-11
- [ ] Before and after exports and a handoff, compare `find ~/.claude ~/.qoder ~/.codex ~/.copilot ~/.local/share/opencode -newer <marker>` → only files the target agent wrote for its new session change; burrow-G writes only under `~/.local/share/burrow-g/` or the `-o` path → AC-12

## UI / manual
- [ ] TUI, sessions pane, press `e` → status bar `exported N messages, M secrets masked to ~/.local/share/burrow-g/exports/<agent>-<id>.md`; file `600`, folder `700`; press `e` again → file replaced → AC-3
- [ ] `ctrl+k`, run `export` → same as `e`; with no session selected (empty agent) → `select a session first` → AC-3
- [ ] Press `t` → "Hand off session" dialog lists only agents installed on PATH (source included); `esc` closes it and nothing starts → AC-5
- [ ] In GNOME Terminal, `t`, pick Codex, `enter` → a new tab opens in the session's folder with Codex and the prompt; burrow-G stays open and its status bar shows the `handed off to codex: …` report → AC-5, AC-8
- [ ] `t`, pick an agent, `o` → burrow-G exits, the report prints on stderr, the agent starts in this terminal → AC-5, AC-8
- [ ] In a terminal without tab support (e.g. an IDE terminal), `t` then `enter` → hands off in this terminal → AC-5
- [ ] `ctrl+k`, `handoff @agent:codex` → skips the picker and hands off → AC-5
- [ ] `sudo burrow-g`, `t`, `enter` → hands off in this terminal, never a new tab → AC-5
- [ ] Follow-up check: `opencode --prompt` sends the prompt (not only fills the input); GNOME `OpenTab` carries a ~100 KB prompt intact → AC-4, AC-5

## Value sourcing
- [ ] Two sessions sharing an id prefix: `burrow-g export <short prefix>` → `N sessions match`, type more → session to act on (`db.Find`)
- [ ] Rename a Claude session with `/rename`, refresh, export → header shows the new title → header title from the index
- [ ] Export twice a minute apart → `Exported:` time changes, message times don't → export time from `now`
- [ ] Message times show in local time: `TZ=Asia/Kolkata burrow-g export <id>` vs `TZ=UTC` → times shift by 5:30 → message times
- [ ] A tool call with `file_path`, one with `command`, one with neither → `- tool: Edit a.go`, `- tool: Bash go test`, `- tool: TodoWrite` → tool line rule
- [ ] Handoff to Qoder starts `qodercli -i <prompt>`; to Copilot `copilot -i <prompt>`; to OpenCode `opencode --prompt <prompt>` → target argv from `StartCommand`
- [ ] Hand off from a session whose folder differs from the current folder → the agent starts in the session's folder → working folder from `s.Dir`
- [ ] Remove one agent's CLI from PATH, press `t` → it is missing from the picker → picker list from `lookPath`
- [ ] Select a different session, open `ctrl+k`, run `export` → exports the highlighted one, not the first → `Env.Selected`

## Acceptance-criteria coverage
- AC-1 commands 1, 2 · AC-2 command 3 · AC-3 UI 1, 2 · AC-4 command 5, UI 9 · AC-5 UI 3 to 9 · AC-6 commands 6, 7 · AC-7 commands 2, 4 · AC-8 command 5, UI 4, 5 · AC-9 command 8 · AC-10 command 9 · AC-11 command 10 · AC-12 command 11
