package freebuff

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/session"
)

var _ session.Source = Source{}

// used is the modification time given to every chat-messages.json, with a
// fraction of a millisecond like the times Freebuff caches.
var used = time.Date(2026, 8, 31, 11, 8, 58, 778_500_000, time.UTC)

type meta int

const (
	noMeta     meta = iota
	freshMeta       // matches chat-messages.json
	nearMeta        // mtime off by 0.4 ms, still current
	staleSize       // chat-messages.json grew since
	staleMtime      // mtime off by 5 ms
)

type chat struct {
	project, id string
	runState    string // contents of run-state.json; "" leaves it out
	messages    bool   // write chat-messages.json
	meta        meta
	prompt      string
	title       string    // contents of chat-title.txt; "" leaves it out
	at          time.Time // modification time of chat-messages.json; zero means used
}

func runState(root, cwd string) string {
	return fmt.Sprintf(`{"sessionState":{"fileContext":{"projectRoot":%q,"cwd":%q,"fileTree":[]},"mainAgentState":{"agentType":"base2-free"}},"output":{"type":"lastMessage"}}`, root, cwd)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeChat lays out one chat folder the way Freebuff does.
func writeChat(t *testing.T, home string, c chat) {
	t.Helper()
	dir := filepath.Join(home, "projects", c.project, "chats", c.id)
	writeFile(t, filepath.Join(dir, "log.jsonl"), `{"msg":"never read"}`+"\n")
	if c.runState != "" {
		writeFile(t, filepath.Join(dir, "run-state.json"), c.runState)
	}
	if c.title != "" {
		writeFile(t, filepath.Join(dir, "chat-title.txt"), c.title)
	}
	if !c.messages {
		return
	}
	messages := filepath.Join(dir, "chat-messages.json")
	writeFile(t, messages, `[{"variant":"user","content":"hi"}]`)
	at := c.at
	if at.IsZero() {
		at = used
	}
	if err := os.Chtimes(messages, at, at); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(messages)
	if err != nil {
		t.Fatal(err)
	}
	size, ms := info.Size(), float64(info.ModTime().UnixNano())/1e6
	switch c.meta {
	case noMeta:
		return
	case nearMeta:
		ms += 0.4
	case staleSize:
		size++
	case staleMtime:
		ms += 5
	}
	m, err := json.Marshal(map[string]any{"messageCount": 1, "firstPrompt": c.prompt, "messagesSize": size, "messagesMtimeMs": ms})
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "chat-meta.json"), string(m))
}

func TestList(t *testing.T) {
	home := t.TempDir()
	chats := []chat{
		{project: "api", id: "2026-08-31T11-00-00.000Z", runState: runState("/w/api", "/w/api/cmd"), messages: true, meta: freshMeta, prompt: "fix the login"},
		// Two projects called db share projects/db/.
		{project: "db", id: "2026-08-31T12-00-00.000Z", runState: runState("/one/db", "/one/db"), messages: true, meta: freshMeta, prompt: "first db"},
		{project: "db", id: "2026-08-31T13-00-00.000Z", runState: runState("/two/db", "/two/db"), messages: true, meta: nearMeta, prompt: "second db"},
		{project: "db", id: "2026-08-31T14-00-00.000Z", runState: runState("/one/db", "/one/db"), messages: true, meta: noMeta},
		{project: "db", id: "2026-08-31T15-00-00.000Z", runState: runState("/one/db", "/one/db"), messages: true, meta: staleSize, prompt: "old prompt"},
		{project: "db", id: "2026-08-31T16-00-00.000Z", runState: runState("/one/db", "/one/db"), messages: true, meta: staleMtime, prompt: "old prompt"},
		{project: "db", id: "2026-08-31T17-00-00.000Z", runState: runState("/one/db", "/one/db"), messages: true, meta: freshMeta, prompt: ""},
		// Freebuff's own title wins over the first prompt, even a stale one.
		{project: "db", id: "2026-08-31T17-30-00.000Z", runState: runState("/one/db", "/one/db"), messages: true, meta: freshMeta, prompt: "first prompt", title: "Fix login redirect\n"},
		{project: "db", id: "2026-08-31T17-40-00.000Z", runState: runState("/one/db", "/one/db"), messages: true, meta: staleSize, prompt: "old prompt", title: "Rename the db package"},
		{project: "db", id: "2026-08-31T17-50-00.000Z", runState: runState("/one/db", "/one/db"), messages: true, meta: freshMeta, prompt: "blank title", title: " \n"},
		// The project root moved; cwd still matches the project folder.
		{project: "web", id: "2026-08-31T18-00-00.000Z", runState: runState("/w/elsewhere", "/w/web"), messages: true, meta: freshMeta, prompt: "moved root"},
		// Skipped: opened and never used, so only log.jsonl.
		{project: "db", id: "2026-08-31T19-00-00.000Z"},
		// Skipped: neither folder is called web, so --continue would open another chat.
		{project: "web", id: "2026-08-31T20-00-00.000Z", runState: runState("/w/a", "/w/b"), messages: true, meta: freshMeta, prompt: "wrong folder"},
		// Skipped: run-state.json missing or broken.
		{project: "web", id: "2026-08-31T21-00-00.000Z", messages: true, meta: freshMeta, prompt: "no run state"},
		{project: "web", id: "2026-08-31T22-00-00.000Z", runState: `{"sessionState":{"fileCon`, messages: true, meta: freshMeta, prompt: "broken"},
		{project: "web", id: "2026-08-31T23-00-00.000Z", runState: `{"sessionState":{"mainAgentState":{}}}`, messages: true, meta: freshMeta, prompt: "no file context"},
	}
	for _, c := range chats {
		writeChat(t, home, c)
	}
	// Stray entries that aren't chats.
	writeFile(t, filepath.Join(home, "projects", "notes.txt"), "x")
	writeFile(t, filepath.Join(home, "projects", "db", "chats", "stray.json"), "{}")
	if err := os.MkdirAll(filepath.Join(home, "projects", "empty"), 0o755); err != nil {
		t.Fatal(err)
	}

	src := Source{Home: home}
	if src.Name() != "freebuff" {
		t.Errorf("Name() = %q, want freebuff", src.Name())
	}
	got, err := src.List()
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]struct{ title, dir string }{
		"2026-08-31T11-00-00.000Z": {"fix the login", "/w/api"},
		"2026-08-31T12-00-00.000Z": {"first db", "/one/db"},
		"2026-08-31T13-00-00.000Z": {"second db", "/two/db"},
		"2026-08-31T14-00-00.000Z": {"(untitled)", "/one/db"},
		"2026-08-31T15-00-00.000Z": {"(untitled)", "/one/db"},
		"2026-08-31T16-00-00.000Z": {"(untitled)", "/one/db"},
		"2026-08-31T17-00-00.000Z": {"(untitled)", "/one/db"},
		"2026-08-31T17-30-00.000Z": {"Fix login redirect", "/one/db"},
		"2026-08-31T17-40-00.000Z": {"Rename the db package", "/one/db"},
		"2026-08-31T17-50-00.000Z": {"blank title", "/one/db"},
		"2026-08-31T18-00-00.000Z": {"moved root", "/w/web"},
	}
	if len(got) != len(want) {
		t.Errorf("got %d sessions, want %d: %+v", len(got), len(want), got)
	}
	for _, s := range got {
		w, ok := want[s.ID]
		if !ok {
			t.Errorf("unexpected session %+v", s)
			continue
		}
		if s.Agent != "freebuff" || s.Title != w.title || s.Dir != w.dir || !s.LastUsed.Equal(used) {
			t.Errorf("%s = %+v, want title %q, dir %q, last used %v", s.ID, s, w.title, w.dir, used)
		}
	}
}

// Chat ids are start times, unique only inside one project folder, while the
// index keys sessions by id: one duplicate failed the whole Freebuff save.
// The newest chat for an id is kept.
func TestList_SameIDInTwoProjects(t *testing.T) {
	const id = "2026-08-31T11-00-00.000Z"
	later := used.Add(time.Hour)
	tests := []struct {
		name       string
		one, two   time.Time // when each project's chat was last used
		wantFolder string
	}{
		{"first project newer", later, used, "/a/one"},
		{"second project newer", used, later, "/b/two"},
		{"same time keeps the first by name", used, used, "/a/one"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			writeChat(t, home, chat{project: "one", id: id, runState: runState("/a/one", "/a/one"), messages: true, meta: freshMeta, prompt: "in one", at: tt.one})
			writeChat(t, home, chat{project: "two", id: id, runState: runState("/b/two", "/b/two"), messages: true, meta: freshMeta, prompt: "in two", at: tt.two})
			writeChat(t, home, chat{project: "two", id: "2026-08-31T12-00-00.000Z", runState: runState("/b/two", "/b/two"), messages: true, meta: freshMeta, prompt: "other"})

			got, err := Source{Home: home}.List()
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 2 {
				t.Fatalf("got %d sessions, want 2 (one per id): %+v", len(got), got)
			}
			for _, s := range got {
				if s.ID == id && s.Dir != tt.wantFolder {
					t.Errorf("kept %s in %s, want the one in %s", id, s.Dir, tt.wantFolder)
				}
			}
		})
	}
}

func TestList_LongPromptIsCut(t *testing.T) {
	home := t.TempDir()
	long := "please refactor the whole storage layer so that\n\tevery reader streams instead of loading files into memory"
	writeChat(t, home, chat{project: "db", id: "a", runState: runState("/one/db", "/one/db"), messages: true, meta: freshMeta, prompt: long})

	got, err := Source{Home: home}.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d sessions, want 1", len(got))
	}
	want := "please refactor the whole storage layer so that every reade…"
	if got[0].Title != want {
		t.Errorf("Title = %q, want %q", got[0].Title, want)
	}
}

func TestList_NoProjectsFolder(t *testing.T) {
	if _, err := (Source{Home: t.TempDir()}).List(); !errors.Is(err, session.ErrMissing) {
		t.Errorf("err = %v, want session.ErrMissing", err)
	}
}

func TestList_NoSessionsYet(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, "projects"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Source{Home: home}.List()
	if err != nil || len(got) != 0 {
		t.Errorf("got %v, %v; want no sessions and no error", got, err)
	}
}

func TestFolders(t *testing.T) {
	for _, file := range []string{"run-state.json", "run-state-reordered.json"} {
		t.Run(file, func(t *testing.T) {
			root, cwd, err := folders(filepath.Join("testdata", file))
			if err != nil {
				t.Fatal(err)
			}
			if root != "/home/u/code/db" || cwd != "/home/u/code/db/sub" {
				t.Errorf("folders = %q, %q; want /home/u/code/db, /home/u/code/db/sub", root, cwd)
			}
		})
	}
}

// installed stands in for exec.LookPath: only names are on PATH.
func installed(names ...string) func(string) (string, error) {
	return func(name string) (string, error) {
		if slices.Contains(names, name) {
			return "/usr/bin/" + name, nil
		}
		return "", exec.ErrNotFound
	}
}

func TestResumeCommand(t *testing.T) {
	tests := []struct {
		name      string
		installed []string
		cli       string
	}{
		{"both installed", []string{"freebuff", "codebuff"}, "freebuff"},
		{"only freebuff", []string{"freebuff"}, "freebuff"},
		{"only codebuff", []string{"codebuff"}, "codebuff"},
		{"neither", nil, "freebuff"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := Source{LookPath: installed(tt.installed...)}
			got := src.ResumeCommand(session.Session{ID: "2026-08-31T11-08-58.778Z"})
			want := []string{tt.cli, "--continue", "2026-08-31T11-08-58.778Z"}
			if !slices.Equal(got, want) {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}
}

// With no LookPath, the CLI is looked up on the real PATH, as resume does.
func TestResumeCommand_NilLookPathUsesPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "codebuff"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	got := Source{}.ResumeCommand(session.Session{ID: "a"})
	if want := []string{"codebuff", "--continue", "a"}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v (codebuff is the one on PATH)", got, want)
	}
}

// Freebuff takes no prompt ("Allowed choices are login, doctor"), so a
// new chat always starts in Codebuff; without it the dialog hides the row.
func TestStartCommand(t *testing.T) {
	tests := []struct {
		name      string
		installed []string
	}{
		{"both installed", []string{"freebuff", "codebuff"}},
		{"only freebuff", []string{"freebuff"}},
		{"only codebuff", []string{"codebuff"}},
		{"neither", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Source{LookPath: installed(tt.installed...)}.StartCommand("read the handoff")
			want := []string{"codebuff", "read the handoff"}
			if !slices.Equal(got, want) {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}
}

func TestDelete_NotSupported(t *testing.T) {
	if err := (Source{}).Delete(session.Session{ID: "a"}, t.TempDir()); err == nil {
		t.Error("want an error: delete is not supported for freebuff yet")
	}
}
