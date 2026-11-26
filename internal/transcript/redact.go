package transcript

import "regexp"

const redacted = "[redacted]"

// secretPatterns are known key shapes. Masking is best effort: a password
// typed in plain words still goes across.
var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----`),
	// A key cut off before its END line is masked to the end of the text.
	regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*`),
	regexp.MustCompile(`\b(?:AKIA|ASIA)[0-9A-Z]{16}\b`),
	regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]{36,}\b`),
	regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9_]{22,}\b`),
	regexp.MustCompile(`\bglpat-[A-Za-z0-9_-]{20,}`),
	regexp.MustCompile(`\bnpm_[A-Za-z0-9]{36}\b`),
	regexp.MustCompile(`\bsk-(?:ant-|proj-)?[A-Za-z0-9_-]{20,}`),
	regexp.MustCompile(`\b[rs]k_(?:live|test)_[A-Za-z0-9]{16,}`),
	regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{35}\b`),
	regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{8,}\.eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}`),
	regexp.MustCompile(`\bxox[abposr]-[A-Za-z0-9-]{10,}`),
}

// labeled keeps a label and masks the value after it: an AWS secret key is 40
// characters with no prefix of its own, so only its name gives it away.
var labeled = regexp.MustCompile(`(?i)(\baws_?secret_?(?:access_?)?key\b["']?\s*[:=]\s*["']?)[A-Za-z0-9/+=]{40}`)

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
	for _, re := range []*regexp.Regexp{labeled, bearer} {
		s = re.ReplaceAllStringFunc(s, func(m string) string {
			r.count++
			return re.FindStringSubmatch(m)[1] + redacted
		})
	}
	return s
}
