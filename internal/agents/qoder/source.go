package qoder

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/GunjanPatel359/burrow-G/internal/session"
)

// Source reads sessions from Home, usually ~/.qoder.
type Source struct {
	Home string
}

func (Source) Name() string { return "qoder" }

func (q Source) List() ([]session.Session, error) {
	if _, err := os.Stat(filepath.Join(q.Home, "projects")); os.IsNotExist(err) {
		return nil, session.ErrMissing
	}
	files, err := filepath.Glob(filepath.Join(q.Home, "projects", "*", "*.jsonl"))
	if err != nil {
		return nil, err
	}

	var list []session.Session
	for _, path := range files {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		title, cwd, err := parseFile(path)
		if err != nil || cwd == "" {
			continue
		}
		list = append(list, session.Session{
			Agent:    "qoder",
			ID:       strings.TrimSuffix(filepath.Base(path), ".jsonl"),
			Title:    title,
			Dir:      cwd,
			LastUsed: info.ModTime(),
		})
	}
	return list, nil
}

func (Source) ResumeCommand(s session.Session) []string {
	return []string{"qodercli", "--resume", s.ID}
}

func (Source) Delete(s session.Session, trashDir string) error {
	return errors.New("not implemented")
}
