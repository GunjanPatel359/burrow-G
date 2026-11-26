package transcript

import (
	"strings"
	"testing"
)

func TestMask(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"aws", "key AKIAABCDEFGHIJKLMNOP here", "key [redacted] here"},
		{"github classic", "ghp_" + strings.Repeat("a", 36), "[redacted]"},
		{"github fine grained", "github_pat_" + strings.Repeat("b", 30), "[redacted]"},
		{"anthropic", "sk-ant-api03-" + strings.Repeat("c", 30), "[redacted]"},
		{"openai", "sk-proj-" + strings.Repeat("d", 30), "[redacted]"},
		{"slack", "xoxb-1234567890-abcdef", "[redacted]"},
		{"bearer", "Authorization: Bearer abcdefghijklmnopqrstuvwxyz", "Authorization: Bearer [redacted]"},
		{"pem", "-----BEGIN RSA PRIVATE KEY-----\nMIIE\nabc\n-----END RSA PRIVATE KEY-----", "[redacted]"},
		{"plain words stay", "sk-short and Bearer token", "sk-short and Bearer token"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r redactor
			if got := r.mask(tt.in); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMarkdown_MasksAndCounts(t *testing.T) {
	tr := sample()
	tr.Messages[0].Text = "use AKIAABCDEFGHIJKLMNOP and\n-----BEGIN PRIVATE KEY-----\nxyz\n-----END PRIVATE KEY-----"
	got, st := Markdown(tr, tr.Session.LastUsed)
	if strings.Contains(got, "AKIA") || strings.Contains(got, "xyz") || strings.Count(got, redacted) != 2 {
		t.Errorf("secrets left in:\n%s", got)
	}
	if st.Masked != 2 {
		t.Errorf("Masked = %d, want 2", st.Masked)
	}

	tr.Messages[1].Tools = []string{"Bash curl -H 'Authorization: Bearer abcdefghijklmnopqrstuvwxyz'"}
	prompt, pst := Prompt(tr)
	if strings.Contains(prompt, "abcdefghijklmnop") || pst.Masked != 3 {
		t.Errorf("prompt masked %d, want 3 including the tool line", pst.Masked)
	}
}
