# Changelog

## Unreleased

### Added

- Freebuff and Codebuff chats, in one "Freebuff" row. Resume uses `freebuff`. They can be
  saved to a file and continued elsewhere like any session. "Continue in Freebuff" starts
  `codebuff`, the only one that takes a first prompt, so it shows only when `codebuff` is
  installed (#20).
- Continue a session in another agent: `e` on a session opens a dialog with **Save to file**
  and **Continue in** every agent installed on your `PATH`. Continuing writes a short handoff
  file (the latest summary and the recent conversation) and starts that agent in the session's
  folder, in a new tab or here, reading it. No agent's files are written (spec 0004).
- Done messages in the status line clear themselves after 2 s, and the key hints stay at the
  right edge.
- Export a session: press `e` on it, then Save to file. The whole conversation, every tool call with its output,
  and the compaction summaries (only the latest for Freebuff, which keeps no others) go to
  `~/.local/share/burrow-g/exports/<agent>-<id>.jsonl`, in
  one format for every agent (`burrow-session/1`, spec 0003). The file is readable only by
  you. OpenCode sessions go through OpenCode's own `session export`; the other agents' files
  are only read.

## v0.0.2 — 2026-11-23

### Added

- Command bar: press `ctrl+k` for `theme`, `refresh`, `help` and `quit`. Arguments use
  `@tag:value` and quotes for spaces.
- 16 new colour themes, four of them light (17 in all, see the README). `theme <name>`
  previews live and the choice is remembered. A theme also sets the terminal's background
  while burrow-G runs; the original comes back on quit.
- The `/ search` line is always visible above the sessions list.
- Bookmarks: press `b` on a session to keep it at the top of its agent's list (★); press `b`
  again to remove it. Bookmarks survive refreshes and restarts, and go away with their
  session when it is deleted.
- Index files from older versions are brought up to date automatically when burrow-G starts,
  after a backup to `sessions.db.bak`. An older burrow-G opening a newer index now says to
  upgrade instead of failing with an SQL error. A 0.0.1 index holds nothing that can't be
  rebuilt, so it is cleared and refilled from the agents' files on the first refresh.
- `sudo burrow-g` lists and resumes sessions that agents saved under root. Run
  `burrow-g setup-sudo` once so sudo can find burrow-G and your agents. Under sudo,
  burrow-G uses root's home even if `HOME` still points at yours, so it never writes
  root-owned files into your home.

### Fixed

- A refresh while an agent's folder is missing no longer empties its list; a refresh that
  finds the folder with no sessions in it does.
- Refreshing while a search is applied no longer shows an empty list.
- `burrow-g version` showed `dev` when installed with `go install`; it now shows the
  real version.

## v0.0.1 — 2026-11-16

First public preview.

### Added

- Terminal UI with two panes: agents on the left, their sessions on the right, and a
  details box (id, folder, resume command, last used) for the selected session.
- Reads sessions from **Claude Code**, **Qoder**, **OpenCode**, **Copilot CLI** and
  **Codex**.
- Session titles from each agent, including Claude Code's `/rename`; falls back to the
  first prompt.
- `/` search by title or folder.
- `enter` opens a session in a new GNOME Terminal tab (burrow-G stays open); `o` resumes it
  in the current terminal. Other terminals resume in place.
- Local SQLite index in `~/.local/share/burrow-g/` — the list shows instantly and only the
  selected agent is refreshed in the background.
- `burrow-g list`, `burrow-g resume <id>`, `burrow-g version`, `burrow-g help`.

### Known limitations

- Linux is tested; macOS builds but is untested; Windows is not supported.
- New tabs only in GNOME Terminal.
- Codex support has not been tested against real Codex sessions yet.
