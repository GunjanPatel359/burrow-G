package helpers

import "strings"

// Truncate collapses whitespace and cuts s to max characters, ending a cut
// with tail. The tail counts toward max, so a longer tail never overflows.
func Truncate(s string, max int, tail string) string {
	s = strings.Join(strings.Fields(s), " ")

	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 0 {
		return ""
	}
	keep := max - len([]rune(tail))
	if keep < 0 {
		return string(runes[:max]) // too narrow for the tail
	}
	return string(runes[:keep]) + tail
}

// ShortPath replaces the home folder with "~".
func ShortPath(path, home string) string {
	if path == home {
		return "~"
	}
	if strings.HasPrefix(path, home+"/") {
		return "~" + strings.TrimPrefix(path, home)
	}
	return path
}

// Between returns the text between start and end, or "" if not found.
func Between(s, start, end string) string {
	_, after, ok := strings.Cut(s, start)
	if !ok {
		return ""
	}
	inside, _, ok := strings.Cut(after, end)
	if !ok {
		return ""
	}
	return inside
}
