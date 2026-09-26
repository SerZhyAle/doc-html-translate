package tests

// Shared reading of the repository's Markdown for the documentation checks (DOC-INTERNAL-QUALITY
// rules 3, 5, 6, 7): which files are documents, which of their lines are prose rather than code,
// and which heading anchors GitHub gives each file.

import (
	"html"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode"
)

// repoFiles lists every file git would commit, as slash paths relative to the repo root: the
// tracked files plus untracked ones that are not ignored. That is the set scripts/doc-registry.ps1
// holds to the registry, and every .md in it must be claimed by a registry record, so "every .md
// here" is "every registry-claimed .md". A file deleted from the working tree is dropped: a link to
// it is broken whatever the index says.
func repoFiles(t *testing.T) map[string]bool {
	t.Helper()
	cmd := exec.Command("git", "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	cmd.Dir = ".."
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git ls-files: %v - the documentation checks need a git checkout", err)
	}
	files := map[string]bool{}
	for _, f := range strings.Split(string(out), "\x00") {
		if f == "" {
			continue
		}
		if _, serr := os.Stat(filepath.Join("..", filepath.FromSlash(f))); serr == nil {
			files[f] = true
		}
	}
	return files
}

// repoDirs is every directory that holds at least one of files, so a link to a folder resolves.
func repoDirs(files map[string]bool) map[string]bool {
	dirs := map[string]bool{".": true}
	for f := range files {
		for d := path.Dir(f); d != "." && !dirs[d]; d = path.Dir(d) {
			dirs[d] = true
		}
	}
	return dirs
}

func markdownFiles(files map[string]bool) []string {
	var out []string
	for f := range files {
		if strings.HasSuffix(strings.ToLower(f), ".md") {
			out = append(out, f)
		}
	}
	return out
}

var (
	fenceOpen   = regexp.MustCompile("^[ \t]*(`{3,}|~{3,})")
	quotePrefix = regexp.MustCompile(`^(?: {0,3}> ?)+`)
	htmlComment = regexp.MustCompile(`(?s)<!--.*?-->`)
)

// proseLines returns the file's lines with everything that is not prose blanked out: fenced code
// blocks, inline code spans and HTML comments. Line numbers are kept, so a finding points at the
// real line. Indented code blocks are not recognized - they cannot be told from a list item's
// continuation without a full parser, and the tree writes its code in fences.
func proseLines(src string) []string {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	src = htmlComment.ReplaceAllStringFunc(src, func(c string) string {
		return strings.Repeat("\n", strings.Count(c, "\n"))
	})
	lines := strings.Split(src, "\n")
	fence := ""
	for i, line := range lines {
		// A fence inside a blockquote opens and closes behind the same "> " markers.
		body := quotePrefix.ReplaceAllString(line, "")
		if fence != "" {
			trimmed := strings.TrimSpace(body)
			if strings.HasPrefix(trimmed, fence) && strings.Trim(trimmed, fence[:1]) == "" {
				fence = ""
			}
			lines[i] = ""
			continue
		}
		if m := fenceOpen.FindStringSubmatch(body); m != nil {
			fence = m[1]
			lines[i] = ""
			continue
		}
	}
	// A code span may run across the lines of one paragraph, so spans are stripped per paragraph.
	for start := 0; start < len(lines); {
		if strings.TrimSpace(lines[start]) == "" {
			start++
			continue
		}
		end := start
		for end < len(lines) && strings.TrimSpace(lines[end]) != "" {
			end++
		}
		para := strings.Split(stripCodeSpans(strings.Join(lines[start:end], "\n")), "\n")
		copy(lines[start:end], para)
		start = end
	}
	return lines
}

// stripCodeSpans removes inline code: a run of N backticks up to the next run of exactly N. An
// unmatched run is literal text, as CommonMark reads it. A removed span keeps its line breaks, so
// the text still splits back into the same lines.
func stripCodeSpans(line string) string {
	var b strings.Builder
	for i := 0; i < len(line); {
		if line[i] != '`' {
			b.WriteByte(line[i])
			i++
			continue
		}
		n := 0
		for i+n < len(line) && line[i+n] == '`' {
			n++
		}
		closeAt := -1
		for j := i + n; j < len(line); {
			if line[j] != '`' {
				j++
				continue
			}
			m := 0
			for j+m < len(line) && line[j+m] == '`' {
				m++
			}
			if m == n {
				closeAt = j
				break
			}
			j += m
		}
		if closeAt < 0 {
			b.WriteString(line[i : i+n])
			i += n
			continue
		}
		b.WriteString(" " + strings.Repeat("\n", strings.Count(line[i:closeAt], "\n")))
		i = closeAt + n
	}
	return b.String()
}

var (
	atxHeading    = regexp.MustCompile(`^ {0,3}#{1,6}(?:\s+(.*?))?\s*$`)
	setextLine    = regexp.MustCompile(`^ {0,3}(=+|-+)\s*$`)
	closingHashes = regexp.MustCompile(`\s+#+$`)
	mdImageOrLink = regexp.MustCompile(`!?\[([^\]]*)\]\([^)]*\)`)
	htmlTag       = regexp.MustCompile(`<[^>]+>`)
	underscoreEm  = regexp.MustCompile(`(^|[^\p{L}\p{N}_])_{1,2}([^_]+?)_{1,2}([^\p{L}\p{N}_]|$)`)
	explicitID    = regexp.MustCompile(`(?i)<a\s[^>]*\b(?:name|id)\s*=\s*["']([^"']+)["']|\bid\s*=\s*["']([^"']+)["']`)
)

// headingAnchors returns the anchors GitHub renders for a Markdown source: one slug per heading
// (a repeated slug gets -1, -2, ..) plus every explicit id or <a name>.
func headingAnchors(src string) map[string]bool {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	raw := strings.Split(src, "\n")
	prose := proseLines(src)
	anchors := map[string]bool{}
	seen := map[string]int{}
	add := func(text string) {
		slug := githubSlug(text)
		if n, dup := seen[slug]; dup {
			seen[slug] = n + 1
			slug = slug + "-" + strconv.Itoa(n+1)
		} else {
			seen[slug] = 0
		}
		anchors[slug] = true
	}
	for i, line := range prose {
		// A fenced line is blanked in prose; the raw line must agree it is not code.
		if line == "" {
			continue
		}
		for _, m := range explicitID.FindAllStringSubmatch(line, -1) {
			anchors[m[1]+m[2]] = true
		}
		if m := atxHeading.FindStringSubmatch(raw[i]); m != nil {
			add(closingHashes.ReplaceAllString(m[1], ""))
			continue
		}
		if i > 0 && setextLine.MatchString(line) && isParagraphLine(prose[i-1]) {
			add(raw[i-1])
		}
	}
	return anchors
}

// isParagraphLine reports whether a line can be the text of a setext heading: not blank and not a
// block of its own kind (list item, table row, quote, heading, rule).
func isParagraphLine(line string) bool {
	t := strings.TrimSpace(line)
	if t == "" || setextLine.MatchString(line) {
		return false
	}
	switch t[0] {
	case '|', '>', '#', '-', '*', '+':
		return false
	}
	return true
}

// githubSlug reproduces GitHub's heading id: the rendered text, lowercased, with every character
// that is not a letter, digit, mark, underscore, hyphen or space dropped and each space turned into
// a hyphen. The rendered text is approximated by unwrapping links, tags, emphasis and entities.
func githubSlug(heading string) string {
	s := strings.ReplaceAll(heading, "`", "")
	s = mdImageOrLink.ReplaceAllString(s, "$1")
	s = htmlTag.ReplaceAllString(s, "")
	for underscoreEm.MatchString(s) {
		s = underscoreEm.ReplaceAllString(s, "$1$2$3")
	}
	s = html.UnescapeString(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r == ' ':
			b.WriteByte('-')
		case r == '-' || r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.Is(unicode.M, r):
			b.WriteRune(r)
		}
	}
	return b.String()
}
