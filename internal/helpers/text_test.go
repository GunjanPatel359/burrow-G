package helpers

import "testing"

func TestTruncate(t *testing.T) {
	cases := []struct {
		in   string
		max  int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello w…"},
		{"line one\nline  two", 50, "line one line two"},
		{"héllo wörld", 6, "héllo…"},
	}

	for _, c := range cases {
		got := Truncate(c.in, c.max)
		if got != c.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", c.in, c.max, got, c.want)
		}
	}
}

func TestShortPath(t *testing.T) {
	cases := []struct {
		path, home, want string
	}{
		{"/home/u/Desktop/x", "/home/u", "~/Desktop/x"},
		{"/home/u", "/home/u", "~"},
		{"/opt/x", "/home/u", "/opt/x"},
		{"/home/user2/x", "/home/u", "/home/user2/x"},
	}

	for _, c := range cases {
		got := ShortPath(c.path, c.home)
		if got != c.want {
			t.Errorf("ShortPath(%q, %q) = %q, want %q", c.path, c.home, got, c.want)
		}
	}
}

func TestBetween(t *testing.T) {
	cases := []struct {
		s, start, end, want string
	}{
		{"<a>x</a>", "<a>", "</a>", "x"},
		{"pre <name>/about</name> post", "<name>", "</name>", "/about"},
		{"no tags", "<a>", "</a>", ""},
		{"<a>never closed", "<a>", "</a>", ""},
	}

	for _, c := range cases {
		if got := Between(c.s, c.start, c.end); got != c.want {
			t.Errorf("Between(%q) = %q, want %q", c.s, got, c.want)
		}
	}
}
