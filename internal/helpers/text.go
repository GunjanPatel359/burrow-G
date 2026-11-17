package helpers

import "strings"

// Truncate collapses whitespace and cuts s to max characters.
func Truncate(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")

	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}

// ShortPath replaces the home folder with "~".
func ShortPath(path, home string) string {
	if path == home {
		return "~"
	}
	for _, sep := range []string{"/", `\`} {
		if strings.HasPrefix(path, home+sep) {
			return "~" + strings.TrimPrefix(path, home)
		}
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
