package textutil

import (
	"regexp"
	"slices"
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

// scriptBlocks are the Unicode blocks the extension's lang.js dominantScript counts, in the
// same tie-breaking order (first block holding the strictly largest count wins). Latin counts
// ASCII letters only: the 0x41-0x7a range also holds [ \ ] ^ _ `, and both editions skip those.
var scriptBlocks = []struct {
	name   string
	lo, hi rune
}{
	{"latin", 0x41, 0x7a},
	{"cyrillic", 0x0400, 0x04ff},
	{"greek", 0x0370, 0x03ff},
	{"arabic", 0x0600, 0x06ff},
	{"hebrew", 0x0590, 0x05ff},
	{"han", 0x4e00, 0x9fff},
	{"hiragana", 0x3040, 0x309f},
	{"katakana", 0x30a0, 0x30ff},
	{"hangul", 0xac00, 0xd7a3},
	{"devanagari", 0x0900, 0x097f},
	{"thai", 0x0e00, 0x0e7f},
}

// DominantScript counts a text sample's letters by Unicode block and returns the block most of
// them belong to ("unknown" when none) with the count it saw. It mirrors lang.js dominantScript.
func DominantScript(sample string) (script string, letters int) {
	counts := make([]int, len(scriptBlocks))
	for _, ch := range sample {
		for i, b := range scriptBlocks {
			if ch < b.lo || ch > b.hi {
				continue
			}
			if b.name == "latin" && !((ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')) {
				continue
			}
			counts[i]++
			letters++
			break
		}
	}
	if letters == 0 {
		return "unknown", 0
	}
	script = "unknown"
	max := -1
	for i, b := range scriptBlocks {
		if counts[i] > max {
			max = counts[i]
			script = b.name
		}
	}
	return script, letters
}

// declaredScripts is the script table shared with lang.js DECLARED_SCRIPTS: the scripts each
// known primary subtag is written in, over the blocks DominantScript counts. A subtag absent
// from the table (bn, ta, ka..) says nothing provable, so DeclarationContradicted never drops it.
var declaredScripts = map[string][]string{
	"en": {"latin"}, "fr": {"latin"}, "de": {"latin"}, "es": {"latin"}, "it": {"latin"},
	"pt": {"latin"}, "nl": {"latin"},
	"ru": {"cyrillic"}, "uk": {"cyrillic"}, "bg": {"cyrillic"}, "sr": {"cyrillic"},
	"mk": {"cyrillic"}, "be": {"cyrillic"},
	"el": {"greek"},
	"ar": {"arabic"}, "ur": {"arabic"}, "fa": {"arabic"}, "ps": {"arabic"},
	"he": {"hebrew"}, "yi": {"hebrew"},
	"zh": {"han"},
	"ja": {"han", "hiragana", "katakana"},
	"ko": {"hangul", "han"},
	"hi": {"devanagari"}, "mr": {"devanagari"}, "ne": {"devanagari"},
	"th": {"thai"},
}

// sampleMaxRunes is how far into a sample the heuristics read; lang.js SAMPLE_CHARS is the same
// window, so both editions decide on the same characters.
const sampleMaxRunes = 8000

// DeclarationContradicted says whether a text sample proves a declared language wrong (ticket
// 76): the sample's dominant script is known and is not a script the declared language is
// written in. The desktop does not read PDF /Lang yet; when it starts to, this guard is what
// its reader must apply first (docs/PARITY.md, "Declared source language"), deciding the same
// cases as the extension's declarationContradicted (tests/testdata/pdf_lang_cases.json).
// Without a sample, or for a language the table cannot speak for, nothing is provable and the
// declaration stands.
func DeclarationContradicted(declared, sample string) bool {
	expected := declaredScripts[strings.SplitN(NormalizeLangTag(declared), "-", 2)[0]]
	if expected == nil {
		return false
	}
	if strings.TrimSpace(sample) == "" {
		return false
	}
	if runes := []rune(sample); len(runes) > sampleMaxRunes {
		sample = string(runes[:sampleMaxRunes])
	}
	script, letters := DominantScript(sample)
	if letters == 0 || script == "unknown" {
		return false
	}
	return !slices.Contains(expected, script)
}
