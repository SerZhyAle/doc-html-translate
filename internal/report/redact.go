package report

import (
	"net/url"
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

// redactions are the rules that do not depend on the machine.
// Follows DIAGNOSTIC-REPORT 0.12 (rules 3, 7, 8 C) using the contract marker [REDACTED].
var redactions = []redaction{
	// 1. Google API key shapes (recognisable with or without label).
	{
		re:   regexp.MustCompile(`AIza[A-Za-z0-9_\-]{35,}`),
		with: "[REDACTED]",
	},
	// 2. Xtream-style credential-in-path: /live|movie|series/<user>/<pass>/<id>.
	{
		re:   regexp.MustCompile(`(?i)(/(?:live|movie|series)/)[^/\s\[\]]+/[^/\s\[\]]+(/[^/\s]+)`),
		with: "${1}[REDACTED]/[REDACTED]${2}",
	},
	// 3. URL userinfo (including passwords with /, ?, # before @host).
	{
		re:   regexp.MustCompile(`(?i)(https?://)[^/\s@:]+:[^@\s]+@`),
		with: "${1}[REDACTED]:[REDACTED]@",
	},
	{
		re:   regexp.MustCompile(`(?i)(https?://)[^/\s@]+@`),
		with: "${1}[REDACTED]@",
	},
	// 4. Secret query parameters (signed CDN signatures, tokens, keys, passwords), supporting &amp;.
	{
		re:   regexp.MustCompile(`(?i)((?:[?&]|&amp;)(?:X-Amz-Signature|X-Amz-Credential|X-Amz-Security-Token|wmsAuthSign|hdnts|hdnea|Policy|Key-Pair-Id|api_key|apiKey|client_secret|clientSecret|refresh_token|refreshToken|access_token|accessToken|auth_token|authToken|googleKey|google_key|token|password|pass|accessPin|pin|secret|auth|sig|signature|key)=)([^&\s"'#\[\]]+)`),
		with: "${1}[REDACTED]",
	},
	// 5. JSON structured fields: string values under secret keys.
	{
		re:   regexp.MustCompile(`(?i)("(?:accessPin|pin|password|pass|token|accessToken|refreshToken|authToken|access_token|refresh_token|auth_token|secret|clientSecret|client_secret|apiKey|api_key|googleKey|google_key|auth|authorization|signature|sig|X-Amz-Signature|X-Amz-Credential|X-Amz-Security-Token|wmsAuthSign|hdnts|hdnea|Policy|Key-Pair-Id)"\s*:\s*)"(?:[^"\\[\]]|\\.)*"`),
		with: `${1}"[REDACTED]"`,
	},
	// 6. JSON structured fields: numeric/primitive values under secret keys.
	{
		re:   regexp.MustCompile(`(?i)("(?:accessPin|pin|password|pass|token|secret|key)"\s*:\s*)\d+`),
		with: `${1}"[REDACTED]"`,
	},
	// 7. Connection string / semicolon-separated key-value pairs.
	{
		re:   regexp.MustCompile(`(?i)((?:^|[;\s])(?:password|pass|userpass|secret|token|auth|accesspin|api_key|apikey)\s*=\s*)([^;\s"'\[\]]+)(;|\s|$)`),
		with: "${1}[REDACTED]${3}",
	},
	// 8. Bearer tokens.
	{
		re:   regexp.MustCompile(`(?i)(Bearer\s+)[A-Za-z0-9_\-\.]{8,}`),
		with: "${1}[REDACTED]",
	},
	// 9. Generic key=value or key: value pairs (short and long secrets).
	{
		re:   regexp.MustCompile(`(?i)((?:key|token|secret|password|accesspin|pin|apikey|api_key|client_secret)["']?\s*[=:]\s*["']?)([^"'\s,;{}\[\]]+)`),
		with: "${1}[REDACTED]",
	},
	// 10. Free-text password labels (e.g. "Password hunter2").
	{
		re:   regexp.MustCompile(`(?i)((?:password|pass|secret)\s+)(["']?[^"'\s,;{}\[\]]+["']?)`),
		with: "${1}[REDACTED]",
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
		// Settings JSON doubles Windows separators; logs and settings can also carry
		// slash paths and percent-escaped file URLs. Keep the location order above
		// so a profile rule cannot consume the start of LOCALAPPDATA first.
		slash := strings.ReplaceAll(dir, `\`, `/`)
		urlPath := slash
		if !strings.HasPrefix(urlPath, "/") {
			urlPath = "/" + urlPath // file:///C:/Users/... on Windows
		}
		variants := []string{
			(&url.URL{Scheme: "file", Path: urlPath}).String(),
			strings.ReplaceAll(dir, `\`, `\\`),
			slash,
			dir,
		}
		seen := make(map[string]bool, len(variants))
		for _, variant := range variants {
			if seen[variant] {
				continue
			}
			seen[variant] = true
			rules = append(rules, redaction{
				re:   regexp.MustCompile(`(?i)` + regexp.QuoteMeta(variant)),
				with: r.with,
			})
		}
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
