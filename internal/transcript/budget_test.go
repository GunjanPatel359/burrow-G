package transcript

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GunjanPatel359/burrow-G/internal/session"
)

func TestPrompt_Fits(t *testing.T) {
	got, st := Prompt(sample())
	if !strings.HasPrefix(got, "You are continuing a coding session that started in claude (session abc, \"Fix the build\") in /tmp/proj.\n") {
		t.Errorf("preamble missing:\n%s", got)
	}
	if !strings.HasSuffix(got, "## User\n\nthanks\n\n") || strings.Contains(got, "omitted") {
		t.Errorf("want the whole transcript:\n%s", got)
	}
	if strings.Contains(got, "Exported:") {
		t.Error("a prompt has no export time")
	}
	if st.Messages != 3 || st.Omitted != 0 || st.Bytes != len(got) {
		t.Errorf("stats = %+v", st)
	}
}

func TestPrompt_TrimsMiddle(t *testing.T) {
	tr := session.Transcript{Session: session.Session{Agent: "claude", ID: "abc", Title: "Long", Dir: "/tmp"}}
	tr.Messages = append(tr.Messages, session.Message{Role: "user", Text: "FIRST goal"})
	for i := range 300 {
		role := "assistant"
		if i%2 == 1 {
			role = "user"
		}
		tr.Messages = append(tr.Messages, session.Message{Role: role, Text: fmt.Sprintf("msg %d %s", i, strings.Repeat("x", 1000))})
	}
	tr.Messages = append(tr.Messages, session.Message{Role: "user", Text: "NEWEST state"})

	got, st := Prompt(tr)
	if len(got) > Budget {
		t.Fatalf("prompt is %d bytes, over %d", len(got), Budget)
	}
	if !strings.Contains(got, "## User\n\nFIRST goal") || !strings.HasSuffix(got, "NEWEST state\n\n") {
		t.Error("want the first user message and the newest message kept")
	}
	if strings.Count(got, "messages omitted]") != 1 {
		t.Error("want exactly one omitted line")
	}
	line := fmt.Sprintf("[%d messages omitted]", st.Omitted)
	if st.Omitted == 0 || !strings.Contains(got, line) {
		t.Errorf("omitted = %d, line %q missing", st.Omitted, line)
	}
	if st.Messages+st.Omitted != len(tr.Messages) {
		t.Errorf("kept %d + omitted %d != %d", st.Messages, st.Omitted, len(tr.Messages))
	}
}

func TestPrompt_CutsHugeMessages(t *testing.T) {
	huge := strings.Repeat("é", 80_000) // two bytes each
	tr := session.Transcript{
		Session: session.Session{Agent: "codex", ID: "x", Title: "t", Dir: "/tmp"},
		Messages: []session.Message{
			{Role: "user", Text: huge},
			{Role: "assistant", Text: "middle"},
			{Role: "assistant", Text: huge},
		},
	}
	tr.Messages[1].Role = "user"

	got, st := Prompt(tr)
	if len(got) > Budget {
		t.Fatalf("prompt is %d bytes, over %d", len(got), Budget)
	}
	if strings.Count(got, "[truncated]") != 2 || !strings.Contains(got, "[1 messages omitted]") {
		t.Errorf("want both kept messages cut and one omitted:\n%.300s", got)
	}
	if !strings.ContainsRune(got, 'é') || strings.ContainsRune(got, '�') {
		t.Error("cut split a character")
	}
	if st.Messages != 2 || st.Omitted != 1 {
		t.Errorf("stats = %+v", st)
	}
}

func TestPrompt_SingleHugeMessage(t *testing.T) {
	tr := session.Transcript{
		Session:  session.Session{Agent: "codex", ID: "x", Title: "t", Dir: "/tmp"},
		Messages: []session.Message{{Role: "user", Text: strings.Repeat("a", 300_000)}},
	}
	got, _ := Prompt(tr)
	if len(got) > Budget || !strings.HasSuffix(got, "[truncated]\n\n") {
		t.Errorf("len %d, want <= %d ending in [truncated]", len(got), Budget)
	}
}

func TestMarkdown_NeverTrimmed(t *testing.T) {
	tr := session.Transcript{
		Session:  session.Session{Agent: "codex", ID: "x", Title: "t", Dir: "/tmp"},
		Messages: []session.Message{{Role: "user", Text: strings.Repeat("a", 300_000)}},
	}
	got, _ := Markdown(tr, tr.Session.LastUsed)
	if len(got) < 300_000 || strings.Contains(got, "[truncated]") {
		t.Error("export must keep everything")
	}
}
