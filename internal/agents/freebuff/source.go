// Package freebuff reads Freebuff and Codebuff chats. Both CLIs run the same
// code and keep their chats in the same folder, in the same format.
package freebuff

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/GunjanPatel359/burrow-G/internal/portable"
	"github.com/GunjanPatel359/burrow-G/internal/session"
)

// Source reads chats from Home, usually ~/.config/manicode.
type Source struct {
	Home string
}

func (Source) Name() string { return "freebuff" }

func (f Source) List() ([]session.Session, error) {
	projects := filepath.Join(f.Home, "projects")
	entries, err := os.ReadDir(projects)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, session.ErrMissing
	}
	if err != nil {
		return nil, err
	}

	var list []session.Session
	// Chat ids are start times, unique only inside one project folder, but
	// the index keys sessions by id and one duplicate would fail the whole
	// save. The newest chat for an id is kept.
	seen := map[string]int{} // chat id -> its index in list
	for _, project := range entries {
		if !project.IsDir() {
			continue
		}
		chatsDir := filepath.Join(projects, project.Name(), "chats")
		chats, err := os.ReadDir(chatsDir)
		if err != nil {
			continue
		}
		for _, c := range chats {
			if !c.IsDir() {
				continue
			}
			s, ok := readChat(filepath.Join(chatsDir, c.Name()), project.Name())
			if !ok {
				continue
			}
			if i, dup := seen[s.ID]; dup {
				if s.LastUsed.After(list[i].LastUsed) {
					list[i] = s
				}
				continue
			}
			seen[s.ID] = len(list)
			list = append(list, s)
		}
	}
	return list, nil
}

func (Source) ResumeCommand(s session.Session) []string {
	return []string{"freebuff", "--continue", s.ID}
}

// StartCommand always starts Codebuff: Freebuff takes no prompt, only the
// login and doctor commands. Without codebuff the Continue in dialog hides
// the row.
func (Source) StartCommand(prompt string) []string {
	return []string{"codebuff", prompt}
}

func (Source) Delete(s session.Session, trashDir string) error {
	return errors.New("delete not supported for freebuff yet")
}

// Export reads the chat's files itself: Freebuff's /export asks the model
// for a Markdown summary, not data. The reader is in export.go.
func (f Source) Export(s session.Session, w *portable.Writer) error {
	// List only shows a chat whose folder is named like its project folder,
	// so that names where it is. The id is part of a path: refuse one that
	// could leave chats/.
	project := filepath.Base(s.Dir)
	if s.ID == "" || s.ID == "." || s.ID == ".." || strings.ContainsAny(s.ID, `/\`) || s.Dir == "" {
		return fmt.Errorf("invalid freebuff chat id %q", s.ID)
	}
	dir := filepath.Join(f.Home, "projects", project, "chats", s.ID)
	found, ok := readChat(dir, project)
	if !ok {
		return fmt.Errorf("freebuff chat %q not found", s.ID)
	}
	return writeMessages(dir, found, w)
}
