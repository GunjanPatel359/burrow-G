package command

import (
	"strings"
	"testing"
)

func texts(rows []Suggestion) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.Text
	}
	return out
}

func TestSuggest(t *testing.T) {
	cases := []struct {
		line string
		want []string
	}{
		{"de", []string{"demo"}},
		{"demo @", []string{"@where:"}},
		{"demo @wh", []string{"@where:"}},
		{"demo @where:", []string{"alpha", "beta"}},
		{"demo @where:al", []string{"alpha"}},
		{`demo @where:"al`, []string{"alpha"}},
		{"demo dr", []string{"dracula"}},
		{"demo ", []string{"dracula", "nord"}},
		{"demo @where:alpha ", []string{"dracula", "nord"}},
		{"nope x", nil},
		{"demo @where:alpha @", nil},
		{"demo @nope:x", nil},
		{"plain-less x", nil},
		{"demo ドラ", nil},
	}
	s := testService()
	for _, c := range cases {
		got := texts(s.Suggest(c.line))
		if strings.Join(got, ",") != strings.Join(c.want, ",") {
			t.Errorf("Suggest(%q) = %v, want %v", c.line, got, c.want)
		}
	}
}

func TestSuggest_CommandRowsCarryHelpAndKey(t *testing.T) {
	got := testService().Suggest("")
	if len(got) != 3 || got[0] != (Suggestion{Text: "demo", Hint: "a demo", Key: "d"}) {
		t.Errorf("got %+v", got)
	}
	tags := testService().Suggest("demo @")
	if len(tags) != 1 || tags[0].Hint != "a place" {
		t.Errorf("tags %+v", tags)
	}
}

func TestComplete(t *testing.T) {
	cases := []struct{ line, pick, want string }{
		{"", "demo", "demo "},
		{"de", "demo", "demo "},
		{"demo @wh", "@where:", "demo @where:"},
		{"demo @where:al", "alpha", "demo @where:alpha "},
		{"demo @where:", "a b", `demo @where:"a b" `},
		{`demo @where:"al`, "alpha", `demo @where:"alpha" `},
		{"demo dr", "dracula", "demo dracula "},
		{"demo ", "nord", "demo nord "},
		{"demo @where:alpha dr", "dracula", "demo @where:alpha dracula "},
		{`demo @where:"a b" dr`, "dracula", `demo @where:"a b" dracula `},
		{`demo "dr`, "dracula", `demo "dracula" `},
	}
	s := testService()
	for _, c := range cases {
		if got := s.Complete(c.line, Suggestion{Text: c.pick}); got != c.want {
			t.Errorf("Complete(%q, %q) = %q, want %q", c.line, c.pick, got, c.want)
		}
	}
}
