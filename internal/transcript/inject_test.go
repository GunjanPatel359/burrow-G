package transcript

import "testing"

func TestStripInjected(t *testing.T) {
	tags := []string{"system-reminder", "skill", "bash-stdout", "bash-stderr"}
	tests := []struct{ in, want string }{
		{"fix the build", "fix the build"},
		{"<div> is misaligned", "<div> is misaligned"},
		{"<skills> list them", "<skills> list them"},
		{"<system-reminder>be careful</system-reminder>", ""},
		{"<system-reminder>be careful</system-reminder>\n\nreal question here", "real question here"},
		{"<skill>\n<name>develop</name>\n</skill>", ""},
		{`<skill name="x">body</skill> then this`, "then this"},
		{"<bash-stdout>ok</bash-stdout><bash-stderr></bash-stderr>", ""},
		{"<system-reminder>never closed, takes the rest", ""},
		{"text with <system-reminder>inside</system-reminder>", "text with <system-reminder>inside</system-reminder>"},
	}
	for _, tt := range tests {
		if got := StripInjected(tt.in, tags); got != tt.want {
			t.Errorf("StripInjected(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
