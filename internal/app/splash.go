package app

import (
	"embed"
	"fmt"
	"strings"

	"doc-html-translate/internal/i18n"
)

// splashFS holds the welcome screen, one file per interface language. The text is a resource
// rather than a wall of fmt.Println calls: thirteen printSplash<LANG> functions would be
// unreadable, and a translator can edit a .txt without touching Go.
//
//go:embed splash/*.txt
var splashFS embed.FS

// splashRule is the placeholder each file uses for the horizontal rule, so the rule's width
// lives in one place instead of being retyped as 62 '=' characters in every language.
const splashRule = "{{rule}}"

// printSplash prints the informational welcome screen (help, features, usage, links) in the
// process language, falling back to English when a language has no file yet.
//
// The registration result and the "Press Enter" pause are printed by the caller so the first-run
// flow can slot its default-handler opt-in prompt in between.
func printSplash() {
	rule := strings.Repeat("=", 62)
	text, err := splashFS.ReadFile("splash/" + i18n.Language() + ".txt")
	if err != nil {
		text, err = splashFS.ReadFile("splash/en.txt")
		if err != nil {
			return
		}
	}
	fmt.Print(strings.ReplaceAll(string(text), splashRule, rule))
}
