package session

import "fmt"

// Owned is one user's copy of an agent's source.
type Owned struct {
	Owner  string
	Source Source
}

// Merged lists one agent's sessions from several users' homes (used under sudo).
// Each session's Owner says whose home it came from.
type Merged []Owned

func (m Merged) Name() string { return m[0].Source.Name() }

// List skips homes that fail (e.g. root never used this agent) unless all fail.
func (m Merged) List() ([]Session, error) {
	var all []Session
	var firstErr error
	ok := false
	for _, o := range m {
		list, err := o.Source.List()
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		ok = true
		for _, s := range list {
			s.Owner = o.Owner
			all = append(all, s)
		}
	}
	if !ok {
		return nil, firstErr
	}
	return all, nil
}

func (m Merged) ResumeCommand(s Session) []string { return m[0].Source.ResumeCommand(s) }

func (m Merged) Delete(s Session, trashDir string) error {
	for _, o := range m {
		if o.Owner == s.Owner {
			return o.Source.Delete(s, trashDir)
		}
	}
	return fmt.Errorf("no %s home for %q", m.Name(), s.Owner)
}
