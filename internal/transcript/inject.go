package transcript

import "strings"

// StripInjected removes blocks an agent adds to the person's turn, such as
// <system-reminder>…</system-reminder>, from the start of s and returns what
// the person wrote after them. A tag counts only as a whole name ("<skill>" or
// "<skill ", never "<skills>"), so real text starting with "<" is kept. An
// injected block with no closing tag takes the rest of s with it.
func StripInjected(s string, tags []string) string {
	s = strings.TrimSpace(s)
	for {
		tag := leadingTag(s, tags)
		if tag == "" {
			return s
		}
		_, after, ok := strings.Cut(s, "</"+tag+">")
		if !ok {
			return ""
		}
		s = strings.TrimSpace(after)
	}
}

func leadingTag(s string, tags []string) string {
	for _, tag := range tags {
		rest, ok := strings.CutPrefix(s, "<"+tag)
		if ok && (rest == "" || strings.ContainsRune("> \n\t", rune(rest[0]))) {
			return tag
		}
	}
	return ""
}
