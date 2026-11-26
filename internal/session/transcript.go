package session

import "time"

// Message is one turn of the main conversation: prose plus one line per tool call.
type Message struct {
	Role  string // "user" or "assistant"
	Time  time.Time
	Text  string
	Tools []string
}

// Transcript is a session's conversation, oldest message first.
type Transcript struct {
	Session  Session
	Messages []Message
	Skipped  int // entries that could not be parsed
}

// Transcriber is implemented by agents that can read a whole conversation
// and start a new session with a first prompt.
type Transcriber interface {
	Name() string
	Transcript(s Session) (Transcript, error)
	// StartCommand returns nil when the agent can't take a first prompt.
	StartCommand(dir, prompt string) []string
}

// Add appends m, merging it into the last message when the role repeats,
// because agents write one reply as several records. Text after a tool call
// starts a new message instead, so prose and tool lines keep their order.
func (t *Transcript) Add(m Message) {
	if m.Text == "" && len(m.Tools) == 0 {
		return
	}
	n := len(t.Messages)
	if n == 0 || t.Messages[n-1].Role != m.Role || (m.Text != "" && len(t.Messages[n-1].Tools) > 0) {
		t.Messages = append(t.Messages, m)
		return
	}
	last := &t.Messages[n-1]
	if m.Text != "" {
		if last.Text != "" {
			last.Text += "\n\n"
		}
		last.Text += m.Text
	}
	last.Tools = append(last.Tools, m.Tools...)
	if last.Time.IsZero() {
		last.Time = m.Time
	}
}
