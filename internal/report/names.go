package report

import (
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// DIAGNOSTIC-REPORT 8 B keeps user-authored names out of every file of the bundle, and a document
// file name or title is one. The reader of a report is told the format and the failure, never which
// book it was; the user can name the book in the message if it matters.
const (
	documentPlaceholder = "<document>"
	titlePlaceholder    = "<document title>"
)

// documentExts are the formats a user converts, spelled the way a path ends. Image and HTML
// outputs are left out on purpose: `images/img-012.png` and `index.html` are the app's own names
// and the author needs them to read a failure.
const documentExts = `epub|pdf|mobi|azw3|azw|fb2|rtf|txt|md|markdown|docx|doc|odt|cbz|cbr|cb7|cbt`

var (
	// A path is a drive, UNC or %VARIABLE% root followed by folders; its last component is a
	// document file name when it ends in one of documentExts. Folders may hold spaces, so the
	// folder part is lazy: the first folder prefix that leaves a component ending in a document
	// extension is taken, and the scan resumes behind it for a second path on the same line.
	// `<` and `>` are not component characters, so a name already replaced is not matched again.
	pathedDocument = regexp.MustCompile(`(?i)((?:[A-Za-z]:[\\/]|\\\\|%[A-Za-z]+%[\\/])[^\r\n"<>|*?]*?[\\/])[^\\/\r\n"<>|*?]+?(\.(?:` + documentExts + `))\b`)
	// Every format's extractor logs `  Title: <title>`; the title is the book's, not the app's.
	titleLine = regexp.MustCompile(`(?m)^([ \t]*Title:[ \t]).*$`)
)

// hiddenNames is the set of names this process knows belong to the document it is converting:
// the names a path-shaped rule cannot see (a bare file name in a message, the output folder named
// after the book).
var hiddenNames struct {
	sync.Mutex
	names []string
	re    *regexp.Regexp
}

// HideDocument registers the name of the document at path, without its extension, so every later
// Redact replaces it with a placeholder and the extension (the format) stays readable. The run
// calls it for the input file before the first line that names it.
func HideDocument(path string) {
	base := baseName(path)
	hide(strings.TrimSuffix(base, filepath.Ext(base)))
}

// HideFolder is HideDocument for a folder, whose whole name is kept as the name: the output
// folder is named after the book, and a name like "Vol 1.2" has no extension to strip.
func HideFolder(path string) {
	hide(baseName(path))
}

func baseName(path string) string {
	base := filepath.Base(strings.TrimRight(path, `\/`))
	if base == "." || base == string(filepath.Separator) {
		return ""
	}
	return base
}

// hide registers name and its percent-encoded form. A name under three characters is not
// registered: it would be a word of the log, not a name.
func hide(name string) {
	if len([]rune(name)) < 3 {
		return
	}
	hiddenNames.Lock()
	defer hiddenNames.Unlock()
	for _, form := range []string{name, url.PathEscape(name)} {
		if !containsString(hiddenNames.names, form) {
			hiddenNames.names = append(hiddenNames.names, form)
		}
	}
	// The longest name goes first: the file name must be replaced whole, or its stem would be
	// replaced inside it and leave the extension behind.
	sort.Slice(hiddenNames.names, func(i, j int) bool { return len(hiddenNames.names[i]) > len(hiddenNames.names[j]) })
	quoted := make([]string, len(hiddenNames.names))
	for i, n := range hiddenNames.names {
		quoted[i] = regexp.QuoteMeta(n)
	}
	hiddenNames.re = regexp.MustCompile(`(?i)` + strings.Join(quoted, "|"))
}

func containsString(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// hideDocumentNames replaces document names and titles in s with placeholders.
func hideDocumentNames(s string) string {
	hiddenNames.Lock()
	re := hiddenNames.re
	hiddenNames.Unlock()
	if re != nil {
		s = re.ReplaceAllString(s, documentPlaceholder)
	}
	s = pathedDocument.ReplaceAllString(s, "${1}"+documentPlaceholder+"${2}")
	return titleLine.ReplaceAllString(s, "${1}"+titlePlaceholder)
}
