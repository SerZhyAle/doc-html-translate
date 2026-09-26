package textutil

import (
	"regexp"
	"strings"
)

// langTagRe keeps a BCP-47 primary subtag and an optional two-letter region. The trailing group
// stops a longer word from passing as a tag: "russian" is not "rus", and "zh-Hans" is "zh", not
// "zh-HA". extension/src/lang.js normalizeLangTag applies the same rule with a lookahead.
var langTagRe = regexp.MustCompile(`^([A-Za-z]{2,3})(?:[-_]([A-Za-z]{2}))?(?:$|[^A-Za-z0-9])`)

// NormalizeLangTag returns a document's declared language as a tag fit for <html lang>
// ("ru", "en-US"), or "" when the value does not start with one.
func NormalizeLangTag(tag string) string {
	m := langTagRe.FindStringSubmatch(strings.TrimSpace(tag))
	if m == nil {
		return ""
	}
	if m[2] != "" {
		return strings.ToLower(m[1]) + "-" + strings.ToUpper(m[2])
	}
	return strings.ToLower(m[1])
}
