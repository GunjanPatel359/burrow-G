package main

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/db"
	"github.com/GunjanPatel359/burrow-G/internal/session"
	"github.com/GunjanPatel359/burrow-G/internal/transcript"
)

// parseExportArgs reads "<id> [-o file]", with -o before or after the id.
func parseExportArgs(args []string) (id, out string, err error) {
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "-o":
			if i+1 >= len(args) {
				return "", "", errors.New("-o needs a file name")
			}
			i++
			out = args[i]
		case id == "":
			id = a
		default:
			return "", "", fmt.Errorf("unexpected argument %q", a)
		}
	}
	if id == "" {
		return "", "", errors.New("usage: burrow-g export <id> [-o file]")
	}
	return id, out, nil
}

// runExport prints the session as Markdown, or writes it to out.
func runExport(store *db.DB, sources []session.Source, id, out string, stdout, stderr io.Writer, now time.Time) error {
	s, src, err := pickSession(store, sources, id)
	if err != nil {
		return err
	}
	tr, err := transcriberFor(src)
	if err != nil {
		return err
	}
	t, err := tr.Transcript(s)
	if err != nil {
		return err
	}
	if err := transcript.CheckMessages(t); err != nil {
		return err
	}

	md, st := transcript.Markdown(t, now)
	if out == "" {
		if _, err := io.WriteString(stdout, md); err != nil {
			return err
		}
	} else {
		if err := transcript.WritePrivate(out, md); err != nil {
			return err
		}
		fmt.Fprintln(stderr, out)
	}
	fmt.Fprintln(stderr, transcript.ExportReport(st))
	return nil
}

// prepareHandoff finds the session and the target and builds the argv to run.
// It launches nothing.
func prepareHandoff(store *db.DB, sources []session.Source, id, agent string, lookPath func(string) (string, error)) (session.Session, []string, transcript.Stats, error) {
	dst, err := handoffTarget(sources, agent)
	if err != nil {
		return session.Session{}, nil, transcript.Stats{}, err
	}
	s, src, err := pickSession(store, sources, id)
	if err != nil {
		return s, nil, transcript.Stats{}, err
	}
	srcTr, err := transcriberFor(src)
	if err != nil {
		return s, nil, transcript.Stats{}, err
	}
	argv, st, err := transcript.Handoff(srcTr, dst, s, lookPath)
	return s, argv, st, err
}

func transcriberFor(src session.Source) (session.Transcriber, error) {
	tr, ok := src.(session.Transcriber)
	if !ok {
		return nil, fmt.Errorf("export and handoff not supported for %s", src.Name())
	}
	return tr, nil
}

// handoffTarget is the named agent, if it can take a first prompt.
func handoffTarget(sources []session.Source, agent string) (session.Transcriber, error) {
	var valid []string
	var target session.Transcriber
	for _, src := range sources {
		tr, ok := src.(session.Transcriber)
		if !ok || len(tr.StartCommand("", "")) == 0 {
			if src.Name() == agent {
				return nil, fmt.Errorf("%s can't start with a first prompt yet", agent)
			}
			continue
		}
		valid = append(valid, src.Name())
		if src.Name() == agent {
			target = tr
		}
	}
	if target == nil {
		return nil, fmt.Errorf("unknown agent %q (choose from: %s)", agent, strings.Join(valid, ", "))
	}
	return target, nil
}
