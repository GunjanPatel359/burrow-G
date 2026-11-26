package transcript

import (
	"fmt"

	"github.com/GunjanPatel359/burrow-G/internal/session"
)

// Handoff returns the argv that starts dst in s's folder with s's conversation
// as its first prompt. It launches nothing; it refuses when the folder is gone,
// dst isn't installed, or nothing could be read.
func Handoff(src, dst session.Transcriber, s session.Session, lookPath func(string) (string, error)) ([]string, Stats, error) {
	if err := session.CheckDir(s); err != nil {
		return nil, Stats{}, err
	}
	name := agentName(dst)
	probe := dst.StartCommand(s.Dir, "")
	if len(probe) == 0 {
		return nil, Stats{}, fmt.Errorf("%s can't start with a first prompt yet", name)
	}
	if _, err := lookPath(probe[0]); err != nil {
		return nil, Stats{}, fmt.Errorf("%s is not installed (not found on PATH)", name)
	}

	t, err := src.Transcript(s)
	if err != nil {
		return nil, Stats{}, err
	}
	if err := CheckMessages(t); err != nil {
		return nil, Stats{Skipped: t.Skipped}, err
	}
	prompt, stats := Prompt(t)
	return dst.StartCommand(s.Dir, prompt), stats, nil
}

// Targets are the agents a session can be handed to: they take a first
// prompt and their CLI is on PATH.
func Targets(sources []session.Source, lookPath func(string) (string, error)) []session.Source {
	var out []session.Source
	for _, src := range sources {
		tr, ok := src.(session.Transcriber)
		if !ok {
			continue
		}
		argv := tr.StartCommand("", "")
		if len(argv) == 0 {
			continue
		}
		if _, err := lookPath(argv[0]); err == nil {
			out = append(out, src)
		}
	}
	return out
}

// Report is the one line summary shown after a handoff launches.
func Report(agent string, st Stats) string {
	line := fmt.Sprintf("handed off to %s: %d messages, %s prompt, %d omitted, %d secrets masked",
		agent, st.Messages, size(st.Bytes), st.Omitted, st.Masked)
	if st.Skipped > 0 {
		line += fmt.Sprintf(", %d entries skipped", st.Skipped)
	}
	return line
}

// ExportReport is the one line summary after an export.
func ExportReport(st Stats) string {
	line := fmt.Sprintf("exported %d messages, %d secrets masked", st.Messages, st.Masked)
	if st.Skipped > 0 {
		line += fmt.Sprintf(", %d entries skipped", st.Skipped)
	}
	return line
}

func size(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d B", n)
	}
	return fmt.Sprintf("%.1f KB", float64(n)/1000)
}

func agentName(t session.Transcriber) string {
	if n, ok := t.(interface{ Name() string }); ok {
		return n.Name()
	}
	return "agent"
}
