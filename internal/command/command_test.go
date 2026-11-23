package command

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/GunjanPatel359/burrow-G/internal/command/event"
)

var colour = Arg{
	Help:     "a colour theme",
	Required: true,
	Suggest: func(_ Env, typed string) []Suggestion {
		return Match([]Suggestion{{Text: "dracula"}, {Text: "nord"}}, typed)
	},
	Check: func(_ Env, v string) error {
		if !slices.Contains([]string{"dracula", "nord"}, v) {
			return fmt.Errorf("no theme %q", v)
		}
		return nil
	},
}

var where = Arg{
	Name: "where",
	Help: "a place",
	Suggest: func(_ Env, typed string) []Suggestion {
		return Match([]Suggestion{{Text: "alpha"}, {Text: "beta"}}, typed)
	},
	Check: func(_ Env, v string) error {
		if v == "bad" {
			return errors.New("bad place")
		}
		return nil
	},
}

var demo = Command{
	Name: "demo", Help: "a demo", Key: "d",
	Args: []Arg{colour, where},
	Run: func(_ Env, in Input) (event.Event, error) {
		return event.ShowHelp{Text: in.Value() + "|" + in.Get("where")}, nil
	},
}

var plainLess = Command{
	Name: "plain-less", Help: "takes nothing",
	Run: func(Env, Input) (event.Event, error) { return event.Refresh{}, nil },
}

var needsWhere = Command{
	Name: "needs",
	Args: []Arg{{Name: "where", Help: "a place", Required: true}},
	Run:  func(Env, Input) (event.Event, error) { return event.Refresh{}, nil },
}

func testService() *Service {
	return NewService(Env{}, demo, plainLess, needsWhere)
}

func runText(t *testing.T, line string) string {
	t.Helper()
	ev, err := testService().Run(line)
	if err != nil {
		t.Fatalf("Run(%q): %v", line, err)
	}
	h, ok := ev.(event.ShowHelp)
	if !ok {
		t.Fatalf("Run(%q) = %#v", line, ev)
	}
	return h.Text
}

func TestRun_ValueAndTags(t *testing.T) {
	if got := runText(t, "demo dracula @where:alpha"); got != "dracula|alpha" {
		t.Errorf("got %q", got)
	}
}

func TestRun_QuotesKeepSpaces(t *testing.T) {
	if got := runText(t, `demo dracula @where:"a b"`); got != "dracula|a b" {
		t.Errorf("got %q", got)
	}
}

func TestRun_TagsInAnyOrder(t *testing.T) {
	if got := runText(t, "demo @where:alpha dracula"); got != "dracula|alpha" {
		t.Errorf("got %q", got)
	}
}

func TestRun_Errors(t *testing.T) {
	cases := []struct{ line, want string }{
		{"", "no command"},
		{"nope", `unknown command "nope"`},
		{"demo dracula @x:1", "demo doesn't take @x"},
		{"demo dracula @where:a @where:b", "@where given twice"},
		{"demo", "demo needs a value: a colour theme"},
		{"needs", "needs needs @where: a place"},
		{`demo "dracula`, `missing closing "`},
		{"demo dracula @", `"@" is not a tag (use @name:value)`},
		{"demo dracula @:a", `"@:a" is not a tag (use @name:value)`},
		{"demo pink", `no theme "pink"`},
		{"demo dracula @where:bad", "bad place"},
		{"plain-less extra", "plain-less takes no value"},
	}
	for _, c := range cases {
		_, err := testService().Run(c.line)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("Run(%q) err = %v, want %q", c.line, err, c.want)
		}
	}
}

func TestRun_NothingCarriesOverBetweenRuns(t *testing.T) {
	s := testService()
	if _, err := s.Run("demo dracula @where:alpha"); err != nil {
		t.Fatal(err)
	}
	if got := runText(t, "demo nord"); got != "nord|" {
		t.Errorf("got %q, a value carried over", got)
	}
}

func TestUsage(t *testing.T) {
	if got := Usage(demo); got != "demo <value> @where:" {
		t.Errorf("Usage(demo) = %q", got)
	}
	if got := Usage(plainLess); got != "plain-less" {
		t.Errorf("Usage(plain-less) = %q", got)
	}
}

func TestCommands_InRegistrationOrder(t *testing.T) {
	cs := testService().Commands()
	if len(cs) != 3 || cs[0].Name != "demo" || cs[2].Name != "needs" {
		t.Errorf("commands = %v", cs)
	}
}

func TestMatch_PrefixFirstThenSubstring(t *testing.T) {
	in := []Suggestion{{Text: "gruvbox-dark"}, {Text: "dracula"}, {Text: "catppuccin-mocha"}}
	got := Match(in, "D")
	if len(got) != 2 || got[0].Text != "dracula" || got[1].Text != "gruvbox-dark" {
		t.Errorf("got %+v", got)
	}
	if len(Match(make([]Suggestion, 10), "")) != 10 {
		t.Error("want every match; the bar scrolls")
	}
}

func TestMatch_MultiByte(t *testing.T) {
	got := Match([]Suggestion{{Text: "Éclair"}, {Text: "日本語"}}, "é")
	if len(got) != 1 || got[0].Text != "Éclair" {
		t.Errorf("got %+v", got)
	}
}
