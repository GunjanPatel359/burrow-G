# Verify: Freebuff and Codebuff sessions · spec 0005 · updated 2026-11-27

_Steps come from the standard definition (sections 2 to 8) and issue #20's "Done when" list._

## Commands

- [ ] `make check` → gofmt clean, vet clean, every test passes
- [ ] `go test ./internal/agents/freebuff -v` → missing `projects` gives `ErrMissing`; a `log.jsonl` only chat is skipped; two projects with the same name list with their own folders; the same chat id in two projects lists only the newest; `projectRoot` beats `cwd`; a chat whose folder doesn't match its project is skipped; a decoy `cwd` later in `run-state.json` is ignored; stale or missing `chat-meta.json` gives `(untitled)`; `chat-title.txt` beats the first prompt; `ResumeCommand` and `StartCommand` tables for both, one, none (first installed wins, none gives `freebuff`), and no `LookPath` uses the real `PATH`; the export fixture gives every row of section 8 (sections 2, 4 to 8)

## By hand

Run `make build`. Use a fake `HOME` that links to a real `~/.config/manicode` (`qa-local/run.sh real` does this), never the real index.

- [ ] `./bin/burrow-g list` → every used Freebuff chat appears once, with its title in the order of section 5 (Freebuff's own, else the first prompt), its project folder, and its last used time; never used chats don't appear (sections 3, 4)
- [ ] TUI → the agents pane shows "Freebuff" with the right count, not cut short; the details pane shows `freebuff --continue <id>` (sections 1, 6)
- [ ] With only `codebuff` on `PATH` (`PATH=<dir with a codebuff stub>:/usr/bin`) → details show `codebuff --continue <id>` (section 6)
- [ ] Resume a chat (enter) → Freebuff opens **that** chat, in its folder, with its history (section 4.3)
- [ ] Missing `~/.config/manicode` → refresh keeps the saved Freebuff list, no error (section 4.1)
- [ ] `e` on a Freebuff chat, Save to file → `exported "<title>"`; the file under `~/.local/share/burrow-g/exports/freebuff-<id>.jsonl` has every prompt and reply, and each tool call has its result (section 8)
- [ ] Compact a throwaway chat in a current Freebuff (`/compact`), say a few more words, then export it → one `compaction` record, after the messages before `/compact` and before the ones after it; nothing before it is missing (section 8)
- [ ] With `codebuff` on `PATH`: `e` on any session, Continue in Freebuff → Codebuff starts in that folder, reads the handoff file and says where the work left off (section 7)
- [ ] With only `freebuff` on `PATH`: `e` on any session → no Continue in Freebuff row (section 7)
- [ ] With neither on `PATH`, in a terminal that opens tabs (GNOME, not under sudo), enter on a Freebuff chat → status says "freebuff is not installed (not found in PATH)" and no tab opens; `o`, or enter in any other terminal → burrow-G exits with the same message (section 6)
- [ ] A chat started in a current Freebuff shows Freebuff's own title in the list, not the first prompt (section 5)

## Cross product resume (needs a Codebuff account)

- [ ] Start a throwaway chat in Freebuff, then `codebuff --continue <id>` in the same folder → the history is there and a new prompt answers
- [ ] Reopen it with `freebuff --continue <id>` → both turns are there
- [ ] Start a throwaway chat in Codebuff and repeat the other way round

## Value sourcing coverage

- `Home` ← `main.go`: the `list` and TUI steps
- installed CLIs ← `Source.LookPath`: the `codebuff` only step, plus the `ResumeCommand` and `StartCommand` tables
- `Dir` ← `run-state.json`: the résumé step, plus the folder tests
- `Title` ← `chat-title.txt`, then `chat-meta.json`: the own title and `list` steps, plus the title and cache tests
- conversation ← `chat-messages.json`, summary ← `run-state.json`: the export and compaction steps, plus the export fixture
- `LastUsed` ← `chat-messages.json` mtime: the `list` step
