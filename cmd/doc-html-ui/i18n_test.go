package main

import (
	"regexp"
	"strings"
	"testing"

	"doc-html-translate/internal/i18n"
)

// guiLangObjects extracts the top-level language objects of the embedded dictionary as
// name -> set of keys. The parse is deliberately shallow: it walks the file counting braces so
// it sees exactly the two levels the dictionary has, without pulling in a JS parser.
func guiLangObjects(t *testing.T) map[string]map[string]bool {
	t.Helper()

	start := strings.Index(uiI18nJS, "const I18N = {")
	if start < 0 {
		t.Fatal("i18n.js does not declare const I18N")
	}
	body := uiI18nJS[start:]

	langLine := regexp.MustCompile(`(?m)^ {4}([a-z]{2}): \{`)
	keyLine := regexp.MustCompile(`(?m)^ {8}([A-Za-z0-9_]+):`)

	out := map[string]map[string]bool{}
	locs := langLine.FindAllStringSubmatchIndex(body, -1)
	for i, loc := range locs {
		name := body[loc[2]:loc[3]]
		end := len(body)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		keys := map[string]bool{}
		for _, m := range keyLine.FindAllStringSubmatch(body[loc[1]:end], -1) {
			keys[m[1]] = true
		}
		out[name] = keys
	}
	return out
}

// TestGUIDictionariesCoverEveryLanguage is the GUI's half of the 13-language invariant. A key
// missing from one language silently falls back to English at runtime and looks perfectly fine,
// so nothing but a static check catches a language that was left behind.
func TestGUIDictionariesCoverEveryLanguage(t *testing.T) {
	langs := guiLangObjects(t)

	if len(langs) != len(i18n.Codes) {
		t.Fatalf("i18n.js has %d languages, internal/i18n ships %d", len(langs), len(i18n.Codes))
	}
	for _, code := range i18n.Codes {
		if _, ok := langs[code]; !ok {
			t.Errorf("i18n.js has no %q dictionary", code)
		}
	}

	en := langs["en"]
	if len(en) == 0 {
		t.Fatal("the English dictionary is empty - the parse is wrong, not the file")
	}
	for code, keys := range langs {
		for k := range en {
			if !keys[k] {
				t.Errorf("%s: missing key %q", code, k)
			}
		}
		for k := range keys {
			if !en[k] {
				t.Errorf("%s: key %q does not exist in English", code, k)
			}
		}
	}
}

// TestGUIMarkupKeysExist catches the other direction: markup asking for a key nobody defined.
// Such a key renders as its own name ("btnConvert") in the window.
func TestGUIMarkupKeysExist(t *testing.T) {
	en := guiLangObjects(t)["en"]
	if len(en) == 0 {
		t.Fatal("no English dictionary parsed")
	}

	attr := regexp.MustCompile(`data-i18n(?:-html|-ph|-title)?="([A-Za-z0-9_]+)"`)
	found := 0
	for _, m := range attr.FindAllStringSubmatch(uiHTML, -1) {
		found++
		if !en[m[1]] {
			t.Errorf("ui.html uses data-i18n key %q, which no dictionary defines", m[1])
		}
	}
	if found == 0 {
		t.Fatal("no data-i18n attributes found in ui.html - the scan is wrong")
	}
}

// TestGUIAboutKeysPresent pins the About section's own strings. The general key-set check would
// catch a language left behind, but not a section shipped in English because nobody added its
// keys anywhere - and the attach hint is the one string that decides whether a report arrives
// with its evidence.
func TestGUIAboutKeysPresent(t *testing.T) {
	langs := guiLangObjects(t)
	for _, code := range i18n.Codes {
		keys := langs[code]
		for _, k := range []string{"secAbout", "btnSendLogs", "sendLogsAttachHint", "mailBody"} {
			if !keys[k] {
				t.Errorf("%s: missing About key %q", code, k)
			}
		}
	}

	// The hint must say something in every language: an empty one leaves the user with an
	// archive they were never told to attach.
	// The working tree is CRLF, so the line-end anchor has to tolerate the carriage return.
	hint := regexp.MustCompile(`(?m)^ {8}sendLogsAttachHint: "(.*)",\r?$`)
	found := hint.FindAllStringSubmatch(uiI18nJS, -1)
	if len(found) != len(i18n.Codes) {
		t.Fatalf("sendLogsAttachHint declared %d times, want %d", len(found), len(i18n.Codes))
	}
	for _, m := range found {
		if strings.TrimSpace(m[1]) == "" {
			t.Error("sendLogsAttachHint is empty in one of the dictionaries")
		}
	}
}

// TestGUIMailAddressComesFromEnv keeps the author's address defined in one place. The About
// section fills its mail link from /api/env; a second literal in the page is how the two drift.
func TestGUIMailAddressComesFromEnv(t *testing.T) {
	if n := strings.Count(uiHTML, "sza@ukr.net"); n != 1 {
		t.Errorf("ui.html names sza@ukr.net %d times, want 1 (the byline link) - the report flow reads it from /api/env", n)
	}
}

// TestGUIDictionaryValuesAreNotEmpty guards the failure mode a key-set check cannot see: the key
// is there, the string is not.
func TestGUIDictionaryValuesAreNotEmpty(t *testing.T) {
	empty := regexp.MustCompile(`(?m)^ {8}[A-Za-z0-9_]+: "",`)
	if m := empty.FindString(uiI18nJS); m != "" {
		t.Errorf("i18n.js has an empty translation: %s", strings.TrimSpace(m))
	}
}
