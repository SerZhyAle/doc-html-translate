package report

import (
	"os"
	"regexp"
	"strings"
)

// redaction is one rule. The list is ordered and open: a future secret or personal-data field
// is one more entry here and is inherited by every report, rather than a new pass somewhere.
type redaction struct {
	re   *regexp.Regexp
	with string
}

// redactions are the rules that do not depend on the machine. `${1}` keeps the label that
// introduced the value, so a reader still sees *what* was hidden.
var redactions = []redaction{
	// A value long enough to be a credential, introduced by a word that names one. The
	// separator class covers `key=..`, `token: ..`, and the `"key":"..."` of a settings blob.
	{
		re:   regexp.MustCompile(`(?i)((?:key|token|secret|password)["']?\s*[=:]?\s*["']?)([A-Za-z0-9_\-]{20,})`),
		with: "${1}<redacted>",
	},
	// A Google API key is recognisable on its own, with or without a label in front of it.
	{
		re:   regexp.MustCompile(`AIza[A-Za-z0-9_\-]{35}`),
		with: "<redacted>",
	},
}

// pathRedactions hide where the user lives without hiding what they converted. They are built
// per call because the locations come from the environment, which a test moves.
//
// The order matters: %LOCALAPPDATA% normally sits *inside* the profile directory, so the
// longer, more specific location has to match first or it would be half-rewritten.
func pathRedactions() []redaction {
	var rules []redaction
	for _, r := range []struct{ dir, with string }{
		{os.Getenv("LOCALAPPDATA"), "%LOCALAPPDATA%"},
		{userHome(), "%USERPROFILE%"},
	} {
		dir := strings.TrimRight(r.dir, `\/`)
		if dir == "" {
			continue
		}
		rules = append(rules, redaction{
			re:   regexp.MustCompile(`(?i)` + regexp.QuoteMeta(dir)),
			with: r.with,
		})
	}
	return rules
}

func userHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

// Redact removes from s everything that must not reach the author: credentials outright, and
// the user's own folder layout reduced to the variables that stand for it. Document file
// names survive - a log whose document cannot be identified cannot be acted on.
func Redact(s string) string {
	for _, r := range pathRedactions() {
		s = r.re.ReplaceAllString(s, r.with)
	}
	for _, r := range redactions {
		s = r.re.ReplaceAllString(s, r.with)
	}
	return s
}

// RedactBytes is Redact for file content.
func RedactBytes(b []byte) []byte {
	return []byte(Redact(string(b)))
}
