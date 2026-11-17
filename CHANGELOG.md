# Changelog

## Unreleased

### Added

- `sudo burrow-g` lists root's sessions (tagged `root`) together with your own, and resumes
  each as the user it belongs to. Run
  `burrow-g setup-sudo` once so sudo can find burrow-G and your agents. Under sudo,
  burrow-G uses root's home even if `HOME` still points at yours, so it never writes
  root-owned files into your home.

### Fixed

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
