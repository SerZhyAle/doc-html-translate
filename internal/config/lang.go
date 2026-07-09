package config

import "regexp"

// langCodeShape matches a plausible language code: a 2-3 letter primary subtag,
// optionally followed by a region or script subtag (e.g. en, ru, zh-CN, pt-BR,
// sr-Latn). This is a shape check, not a registry lookup - it flags obvious mistakes
// (a full name like "russian", an underscore form like "en_US", an empty value)
// without rejecting valid-but-uncommon ISO 639 codes we don't enumerate.
var langCodeShape = regexp.MustCompile(`^[a-zA-Z]{2,3}(-[a-zA-Z0-9]{2,8})?$`)

// SuspiciousLangCodes returns human-readable labels for any of the given src/dst
// language codes that do not look like a language code. It is advisory only - the
// caller should warn and continue, never fail, since the check is heuristic (Google
// wants ISO-639 codes; Ollama tolerates free-form names via langName's fallback).
func SuspiciousLangCodes(src, dst string) []string {
	var bad []string
	if !langCodeShape.MatchString(src) {
		bad = append(bad, "-src "+src)
	}
	if !langCodeShape.MatchString(dst) {
		bad = append(bad, "-dst "+dst)
	}
	return bad
}
