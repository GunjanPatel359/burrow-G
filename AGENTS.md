# burrow-G

One terminal UI that lists and resumes AI coding agent sessions (Claude Code, Qoder, OpenCode, Copilot CLI, Codex) across every project.

## Stack

- **Language / Runtime**: Go 1.26 (module `github.com/GunjanPatel359/burrow-G`)
- **Framework**: Bubble Tea TUI (`charmbracelet/bubbletea`, `bubbles`, `lipgloss`, `muesli/termenv`)
- **Key dependencies**: `modernc.org/sqlite` (pure Go SQLite, builds with `CGO_ENABLED=0`)
- **Package manager**: Go modules; also shipped on npm as prebuilt binaries (`npm/`, `scripts/npm.sh`)

## Build approach

<TBD, set by /scope>

## Commands

```bash
make check     # gofmt check + go vet + go test (same as CI)
make build     # ./bin/burrow-g
make install   # into $(go env GOPATH)/bin
go test ./internal/db/...   # one package
make dist      # release archives + checksums in ./dist
scripts/npm.sh build <version>   # npm packages from ./dist
```

## Specs

Stored in `docs/specs/`. Format: `docs/specs/NNNN-title.md`.

## Rules

- Agents' files are only ever read, never changed. Open their SQLite files with `?mode=ro`.
- `gofmt` clean is enforced in CI; run `make check` before pushing.
- Tests use the standard `testing` package only: real temp files (`t.TempDir()`), real SQLite, fixtures in `testdata/`, no mocking library.
- Make side effects injectable for tests: pass funcs like `getenv`, `run`, `current` instead of calling `os`/`exec` directly (see `launch.Detect`, `homeDir`).
- One thing per file where the code is a list: one agent per `internal/agents/<agent>/`, one command per `internal/command/commands/<name>.go`, one terminal per `internal/launch/<terminal>.go`.
- Comments are short plain English and explain why, not what.
- Release version lives only in `const version` in `cmd/burrow-g/commands.go`; `make dist` reads it. Bump it before tagging, and add a `CHANGELOG.md` entry.
- Commit subjects use `Area: imperative summary` (e.g. `OpenCode: read sessions from session_v2`). PRs target `main`.
- Under `sudo`, burrow-G uses root's home even if `HOME` points elsewhere, sets `HOME` to it, and never opens new tabs (`cmd/burrow-g/sudo.go`, `main.go`).
- Platforms: Linux tested, macOS untested, Windows unsupported (resume uses `syscall.Exec`).

## Agent skills

Where a skill disagrees with `## Rules` above, the rules win.

- [golang-code-style](.agents/skills/golang-code-style/): `samber/cc-skills-golang`, Go style: line breaks, declarations, control flow, when comments help
- [golang-error-handling](.agents/skills/golang-error-handling/): `samber/cc-skills-golang`, wrapping with `%w`, `errors.Is/As`, sentinel errors (like `session.ErrMissing`)
- [golang-testing](.agents/skills/golang-testing/): `samber/cc-skills-golang`, table driven and parallel tests, fixtures, fuzzing. Ignore its testify and mock advice; this repo uses standard `testing` only
- [golang-database](.agents/skills/golang-database/): `samber/cc-skills-golang`, `database/sql`: parameterized queries, NULLable columns, transactions. Ignore its sqlx/pgx advice; this repo uses plain `database/sql` on `modernc.org/sqlite`

MCP servers: gopls (recommended; `claude mcp add gopls -- gopls mcp`, needs gopls v0.20+)

## Context files

- [internal/agents/AGENTS.md](internal/agents/AGENTS.md): one read only reader per agent; how to add an agent and keep refresh safe
- [internal/db/AGENTS.md](internal/db/AGENTS.md): burrow-G's own SQLite index, bookmarks, and the versioned patch chain
- [internal/command/AGENTS.md](internal/command/AGENTS.md): command bar engine, one file per command, events back to the TUI
- [internal/tui/AGENTS.md](internal/tui/AGENTS.md): Bubble Tea app, components, themes, background jobs
- [docs/bubbletea-cheatsheet.md](docs/bubbletea-cheatsheet.md): Bubble Tea from scratch, walked through this codebase

_Drafted by /audit from the repo, worth a quick human pass. Edit freely: once a line stops matching this draft, later runs treat it as curated and will flag rather than overwrite it._
