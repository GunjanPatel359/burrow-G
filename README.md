# burrow-G

**Find and resume your AI coding agent sessions — across every agent and every project — from one terminal UI.**

```
 Agents             │ Claude Code · 60
▌Claude Code     60 │ ★ Refactor auth middleware                             5 minutes ago
 Qoder            7 │▌  Fix flaky integration test                          18 minutes ago
 OpenCode        12 │   Add dark mode toggle                                    1 day ago
 Copilot CLI      4 │   Investigate memory leak in worker                      3 days ago
 Codex            0 │   Write release notes                                    4 days ago
                    │──────────────────────────────────────────────────────────────────────
                    │ id      3f6c2a1e-8b4d-4e7a-9c1f-2d5b6a7e8f90
                    │ folder  ~/code/api-server
                    │ resume  claude --resume 3f6c2a1e-8b4d-4e7a-9c1f-2d5b6a7e8f90
                    │ used    18 minutes ago · 2026-11-15 10:36
───────────────────────────────────────────────────────────────────────────────────────────
 ● refreshed   ↑/↓ move · enter new tab · o resume here · b bookmark · / search · q quit
```

> **Status: v0.0.1 — early preview.** Listing, searching, resuming and bookmarks work.
> Deleting sessions is on the way. Expect rough edges and please [open an issue](../../issues).

## Why

Agent CLIs keep each conversation in their own place, and most of them only show sessions
for the folder you are standing in. After a week of work across several projects you
forget *where* a session was started, which agent it was in, and you have no single place
to see them all.

burrow-G reads every agent's sessions, shows them in one list, and drops you back into any
of them — in the right folder, with the right command.

## Features

- **All agents in one place** — Claude Code, Qoder, OpenCode, Copilot CLI and Codex.
- **Instant** — sessions are indexed in a small local database; only the agent you pick is
  re-read in the background.
- **Real titles** — uses each agent's own title (including Claude Code's `/rename`), falling
  back to your first prompt.
- **Search** — `/` filters by title or folder.
- **Resume anywhere** — `enter` opens the session in a **new terminal tab** and keeps burrow-G
  open; `o` resumes it right here. It always starts in the session's original folder.
- **Safe** — agents' files are only ever read, never changed.

## Supported agents

| Agent | Reads from | Resumes with |
|---|---|---|
| Claude Code | `~/.claude/projects/` | `claude --resume <id>` |
| Qoder | `~/.qoder/projects/` | `qodercli --resume <id>` |
| OpenCode | `~/.local/share/opencode/opencode.db` (read-only) | `opencode --session <id>` |
| Copilot CLI | `~/.copilot/session-store.db` (read-only) | `copilot --resume <id>` |
| Codex | `~/.codex/sessions/` | `codex resume <id>` |

Codex support follows Codex's documented session format but has not yet been tested
against real Codex sessions — reports welcome.

## Install

**npm** (no Go needed):

```sh
npm install -g burrow-g
```

**Go** 1.26 or newer:

```sh
go install github.com/GunjanPatel359/burrow-G/cmd/burrow-g@latest
```

Make sure `$(go env GOPATH)/bin` is on your `PATH`.

**Binary:** download the archive for your system from
[Releases](https://github.com/GunjanPatel359/burrow-G/releases), unpack it and put
`burrow-g` somewhere on your `PATH`.

**Platforms:** Linux (tested). macOS builds and should work, but is untested. Windows is
not supported yet (resuming relies on Unix `exec`).

## Usage

```
burrow-g                 open the terminal UI
burrow-g list            print every session (refreshes all agents)
burrow-g resume <id>     resume a session by id or id prefix
burrow-g setup-sudo      let 'sudo burrow-g' work (once)
burrow-g version         print the version
burrow-g help            show help
```

### Keys

| Key | Agents pane | Sessions pane |
|---|---|---|
| `↑` / `↓` | pick an agent | move |
| `enter` / `→` / `tab` | go to its sessions | open in a new tab (or here, see below) |
| `o` | — | resume in this terminal |
| `b` | — | bookmark (★) — bookmarked sessions stay at the top |
| `/` | — | search title or folder |
| `←` / `esc` | — | back to agents |
| `r` | refresh | refresh |
| `q` / `ctrl+c` | quit | quit |

**New tabs** currently work in **GNOME Terminal**. In other terminals (and IDE terminals
such as GoLand's), `enter` resumes in the current terminal instead.

### Sessions started with sudo

Agents run as root (for example from an IDE started with `sudo`) save their sessions in
root's home (`/root/.claude`, …), which your normal user can't read. To see and resume
them, run burrow-G as root:

```sh
burrow-g setup-sudo   # once, asks for your password
sudo burrow-g
```

sudo only looks for programs in system folders, not in your PATH. `setup-sudo` puts a small
script at `/usr/local/bin/burrow-g` that runs your burrow-G and adds your PATH (after the
system folders), so the agent CLIs are found too. Run it again if burrow-G moves (for
example after switching Node versions with nvm). It never overwrites a file it didn't make.

Under sudo, burrow-G always uses root's home (even with `sudo -E`), keeps its own index
there, and resumes in the same terminal instead of opening a new tab.

## How it works

- Each agent has a small reader in `internal/agents/<agent>/` that turns its files into a
  common `Session` (agent, id, title, folder, last used).
- burrow-G keeps an index in `~/.local/share/burrow-g/sessions.db` (SQLite). The UI shows that
  index instantly and refreshes the selected agent in the background. A refresh that finds no
  sessions at all leaves the index as it is.
- Bookmarks live in their own table, so refreshing never touches them. A bookmarked session
  that disappears is hidden, and comes back with its ★ if the session returns.
- Resuming changes into the session's folder and replaces burrow-G with the agent's own
  command, so when you quit the agent you are back at your normal prompt.
- Nothing leaves your machine.

## Development

```sh
make check   # gofmt + go vet + go test
make build   # ./bin/burrow-g
make install # into $(go env GOPATH)/bin
```

```
cmd/burrow-g/            entry point and CLI commands
internal/
  session/             shared Session type, Source interface, resume
  agents/<agent>/      one reader per agent — add a new agent here
  db/                  burrow-G's own SQLite index
    patches/           bring index files from older versions up to date
  launch/              open a command in a new terminal tab (gnome.go, …)
  helpers/             small text/time/file helpers
  tui/                 Bubble Tea UI
    app.go layout.go keys.go messages.go commands.go
    theme/             colour palettes → styles
    components/        agentlist, sessionlist, details, statusbar
```

### Changing how data is stored

burrow-G brings index files from older versions up to date itself, every time it starts
(`internal/db/patches/`):

- The release version is `version` in `cmd/burrow-g/commands.go`; change it before tagging.
  `make dist` reads it from there.
- Each release that changes storage gets **one patch file**, e.g. `patch003.go`, with
  `Version()` and `Invoke(ctx, pct)`, plus one line in `chain` in `patches.go`.
- The file stores its version in the `meta` table (`sqlite3 sessions.db "select * from meta"`);
  a file without one is 0.0.1. On start: back up to `sessions.db.bak` if patches are due → run
  the patches newer than the file's version, each in a transaction with its version → create
  missing tables → store this release's version.
- A new file (no tables yet) is already on the latest version, so a patch only ever sees an
  older file and can change it directly.
- A file made by a newer burrow-G is refused with "please upgrade" instead of an SQL error.
- Never change a released patch — fixes go in the next release's patch.

New to Bubble Tea? [docs/bubbletea-cheatsheet.md](docs/bubbletea-cheatsheet.md) explains
the model/update/view idea from scratch and walks through this codebase.

## Roadmap

- Delete sessions safely (to a trash folder, with restore)
- More agents: Gemini CLI, Qwen Code, Cursor CLI
- More terminals for new tabs: kitty, WezTerm, tmux
- Switchable colour themes

## License

[MIT](LICENSE)
