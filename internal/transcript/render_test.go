package transcript

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/session"
)

func sample() session.Transcript {
	at := time.Date(2026, 11, 20, 14, 3, 0, 0, time.Local)
	return session.Transcript{
		Session: session.Session{Agent: "claude", ID: "abc", Title: "Fix the build", Dir: "/tmp/proj"},
		Messages: []session.Message{
			{Role: "user", Time: at, Text: "make it build"},
			{Role: "assistant", Time: at.Add(time.Minute), Text: "Done.", Tools: []string{"Edit internal/db/db.go", "Bash go test ./..."}},
			{Role: "user", Text: "thanks"},
		},
	}
}

func TestMarkdown(t *testing.T) {
	now := time.Date(2026, 11, 26, 9, 0, 0, 0, time.UTC)
	got, st := Markdown(sample(), now)
	want := "# Fix the build\n\n" +
		"Agent: claude · Session: abc · Folder: /tmp/proj · Exported: 2026-11-26T09:00:00Z\n\n" +
		"## User · 2026-11-20 14:03\n\nmake it build\n\n" +
		"## Assistant · 2026-11-20 14:04\n\nDone.\n\n" +
		"- tool: Edit internal/db/db.go\n- tool: Bash go test ./...\n\n" +
		"## User\n\nthanks\n\n"
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
	if st != (Stats{Messages: 3, Bytes: len(want)}) {
		t.Errorf("stats = %+v", st)
	}
}

func TestMarkdown_Skipped(t *testing.T) {
	tr := sample()
	tr.Skipped = 2
	got, st := Markdown(tr, time.Time{})
	if !strings.Contains(got, "· Folder: /tmp/proj\n(2 entries skipped)\n\n## User") {
		t.Errorf("no skipped line under the header:\n%s", got)
	}
	if st.Skipped != 2 {
		t.Errorf("Skipped = %d, want 2", st.Skipped)
	}
}

func TestCheckMessages(t *testing.T) {
	tr := session.Transcript{Session: session.Session{Agent: "codex", ID: "x1"}}
	err := CheckMessages(tr)
	if err == nil || err.Error() != "no messages found in codex session x1" {
		t.Errorf("err = %v", err)
	}
	if CheckMessages(sample()) != nil {
		t.Error("want nil for a transcript with messages")
	}
}

func TestToolLine(t *testing.T) {
	long := strings.Repeat("a", 200)
	tests := []struct {
		name, tool, input, want string
	}{
		{"file path first", "Edit", `{"command":"x","file_path":"a.go"}`, "Edit a.go"},
		{"camel case path", "read", `{"filePath":"b.go"}`, "read b.go"},
		{"command collapsed", "Bash", `{"command":"go  test\n ./..."}`, "Bash go test ./..."},
		{"quoted json", "shell", `"{\"cmd\":\"ls -la\"}"`, "shell ls -la"},
		{"command as list", "shell", `{"command":["bash","-lc","ls"]}`, "shell bash -lc ls"},
		{"no primary arg", "TodoWrite", `{"todos":[]}`, "TodoWrite"},
		{"not json", "Odd", `nope`, "Odd"},
		{"cut to 120", "Bash", `{"command":"` + long + `"}`, "Bash " + strings.Repeat("a", 119) + "…"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToolLine(tt.tool, json.RawMessage(tt.input)); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAdd_MergesSameRoleInOrder(t *testing.T) {
	var tr session.Transcript
	tr.Add(session.Message{Role: "assistant", Text: "one"})
	tr.Add(session.Message{Role: "assistant", Text: "more"})
	tr.Add(session.Message{Role: "assistant", Tools: []string{"Bash ls"}})
	tr.Add(session.Message{Role: "assistant", Tools: []string{"Read a.go"}})
	tr.Add(session.Message{Role: "assistant", Text: "two"}) // after a tool: starts anew
	tr.Add(session.Message{Role: "user"})
	tr.Add(session.Message{Role: "user", Text: "ok"})

	var got []string
	for _, m := range tr.Messages {
		got = append(got, m.Role+":"+m.Text+":"+strings.Join(m.Tools, ","))
	}
	want := "assistant:one\n\nmore:Bash ls,Read a.go|assistant:two:|user:ok:"
	if strings.Join(got, "|") != want {
		t.Errorf("got  %q\nwant %q", strings.Join(got, "|"), want)
	}
}
