# Index database

## Overview

burrow-G's own SQLite file at `~/.local/share/burrow-g/sessions.db` (root's home under sudo). It caches every agent's sessions so the UI shows instantly, and holds the only data that can't be rebuilt: bookmarks and settings. `patches/` brings files made by older releases up to date on every start.

## Key files

| File | Owns |
|---|---|
| `db.go` | `Open`, schema, `prepare` (version check, backup, patches, create tables) |
| `sessions.go` | `Save`, `List`, `Find`, `Counts`, `SetBookmark`, `Remove`, `Refresh` |
| `settings.go` | small key/value settings (e.g. the saved theme) |
| `patches/patches.go` | `Patch` interface, the `chain`, `Run`, `GetPatchVersion`, `VersionCheck` |
| `patches/patch002.go` | 0.0.1 → 0.0.2: drops rebuildable tables |

## Conventions

- `sessions` is a rebuildable index; `Save` replaces one agent's rows in a transaction. `bookmarks` and `settings` live in their own tables so a refresh never touches them.
- `Save` drops bookmarks whose session is gone; `Remove` drops a session and its bookmark together.
- Every storage change in a release gets exactly one new numbered patch file (`patch003.go`, …) with `Version()` and `Invoke(ctx, pct)`, plus one line appended to `chain`. Also update `schema` in `db.go` so new files get the final shape.
- A patch runs inside `pct.Tx`; `executePatch` commits it together with its version.
- Tests open real files under `t.TempDir()`; `patches_test.go` uses `rawDB` to build old shaped files.

## Gotchas

- Never edit a released patch. Fixes go in the next release's patch.
- A brand new file (no tables) counts as the latest version, so patches only ever see older files. A file with tables but no `meta` row is 0.0.1.
- A file whose version is newer than this binary is refused with "please upgrade", not opened.
- A backup goes to `sessions.db.bak` (via `VACUUM INTO`) only when patches are due.
- `SetMaxOpenConns(1)`: SQLite allows one writer at a time.

_Drafted by /audit from the repo, worth a quick human pass. Edit freely: once a line stops matching this draft, later runs treat it as curated and will flag rather than overwrite it._
