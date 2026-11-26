package handoff

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// The limits of spec 0004, section 3.
const (
	toolOutputLimit = 2048
	toolInputLimit  = 1024
	textLimit       = 8192
	summaryLimit    = 40960
	defaultBudget   = 100_000
)

// cut shortens s to at most limit bytes and says how much more there is.
// It cuts at a line break when one is in the last fifth, so code stays
// readable, and never inside a character.
func cut(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	end := limit
	for end > 0 && !utf8.RuneStart(s[end]) {
		end--
	}
	if nl := strings.LastIndexByte(s[:end], '\n'); nl > 0 && nl >= end-end/5 {
		end = nl
	}
	more := (len(s) - end + 1023) / 1024
	return s[:end] + fmt.Sprintf("\n… [%d KB more in the full record]", more)
}
