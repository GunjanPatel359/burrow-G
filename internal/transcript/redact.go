package transcript

import "regexp"

const redacted = "[redacted]"

// secretPatterns are known key shapes. Masking is best effort: a password
// typed in plain words still goes across.
var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----`),
	regexp.MustCompile(`\b(?:AKIA|ASIA)[0-9A-Z]{16}\b`),
	regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]{36,}\b`),
	regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9_]{22,}\b`),
	regexp.MustCompile(`\bsk-(?:ant-|proj-)?[A-Za-z0-9_-]{20,}`),
	regexp.MustCompile(`\bxox[abposr]-[A-Za-z0-9-]{10,}`),
}

// bearer keeps the word "Bearer" so the text still reads naturally.
var bearer = regexp.MustCompile(`(?i)\b(Bearer\s+)[A-Za-z0-9._~+/-]{16,}=*`)

// redactor masks secrets and counts how many it replaced.
type redactor struct {
	count int
}

func (r *redactor) mask(s string) string {
	for _, re := range secretPatterns {
		s = re.ReplaceAllStringFunc(s, func(string) string {
			r.count++
			return redacted
		})
	}
	return bearer.ReplaceAllStringFunc(s, func(m string) string {
		r.count++
		return bearer.FindStringSubmatch(m)[1] + redacted
	})
}
