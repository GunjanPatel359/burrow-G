# Changelog

## Unreleased

### Added

- Windows support: `.zip` builds for amd64 and arm64, `npm install -g burrow-g` on Windows,
  data in `%LOCALAPPDATA%\burrow-g`, and `enter` opens a new tab in Windows Terminal.

### Fixed

- Agent databases (OpenCode, Copilot) and burrow-G's own database now open from folders
  whose names contain spaces, `#`, `?` or `%`.

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
