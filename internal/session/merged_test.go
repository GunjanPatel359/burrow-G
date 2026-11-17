package session

import (
	"errors"
	"slices"
	"testing"
)

type stub struct {
	list []Session
	err  error
}

func (stub) Name() string                            { return "claude" }
func (s stub) List() ([]Session, error)              { return s.list, s.err }
func (stub) ResumeCommand(s Session) []string        { return []string{"claude", "--resume", s.ID} }
func (stub) Delete(s Session, trashDir string) error { return errors.New("stub " + s.ID) }

func TestMerged_ListsEveryHomeWithOwner(t *testing.T) {
	m := Merged{
		{Owner: "root", Source: stub{list: []Session{{ID: "r1"}}}},
		{Owner: "gunjan", Source: stub{list: []Session{{ID: "g1"}, {ID: "g2"}}}},
	}

	list, err := m.List()
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, s := range list {
		got = append(got, s.Owner+":"+s.ID)
	}
	if want := []string{"root:r1", "gunjan:g1", "gunjan:g2"}; !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
	if m.Name() != "claude" {
		t.Errorf("Name = %q", m.Name())
	}
}

func TestMerged_OneHomeMissing(t *testing.T) {
	m := Merged{
		{Owner: "root", Source: stub{err: errors.New("no ~/.qoder")}},
		{Owner: "gunjan", Source: stub{list: []Session{{ID: "g1"}}}},
	}
	list, err := m.List()
	if err != nil || len(list) != 1 {
		t.Errorf("list = %v, err = %v; want gunjan's session and no error", list, err)
	}

	none := Merged{{Owner: "root", Source: stub{err: errors.New("gone")}}}
	if _, err := none.List(); err == nil {
		t.Error("every home failed: want an error")
	}
}

func TestMerged_DeleteGoesToOwner(t *testing.T) {
	m := Merged{{Owner: "root", Source: stub{}}}
	if err := m.Delete(Session{ID: "x", Owner: "root"}, ""); err == nil || err.Error() != "stub x" {
		t.Errorf("err = %v, want root's source to handle it", err)
	}
	if err := m.Delete(Session{ID: "x", Owner: "nobody"}, ""); err == nil {
		t.Error("unknown owner: want an error")
	}
}
