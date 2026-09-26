package tests

// DOC-INTERNAL-QUALITY rules 3, 6 and 7 for every Markdown document git would commit (all of
// them are registry-claimed - see repoFiles): a relative link resolves to a file or folder that
// is in the repository, a heading anchor names a heading or id the target renders, an image path
// resolves, no link uses http://, and no document pulls a script from elsewhere. Code spans and
// fences are examples, not links, and are not read.

import (
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var (
	// [text](dest) and ![alt](dest); a destination in <..> may hold spaces.
	mdInlineLink = regexp.MustCompile(`\]\(\s*(<[^>]*>|[^)\s]+)`)
	// [label]: dest
	mdRefDef = regexp.MustCompile(`^ {0,3}\[[^\]]+\]:\s*(<[^>]*>|\S+)`)
	// href/src attributes of raw HTML in the prose.
	htmlLinkAttr = regexp.MustCompile(`(?i)<(a|img|source|link|script)\b[^>]*?\b(?:href|src)\s*=\s*["']([^"']*)["']`)
	urlScheme    = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:`)
	// GitHub's line anchors on a rendered source file (#L10, #L10-L20).
	lineAnchor = regexp.MustCompile(`^L\d+(-L\d+)?$`)
)

type docLink struct {
	line int
	dest string
	tag  string // the HTML element, or "" for Markdown syntax
}

func docLinks(src string) []docLink {
	var out []docLink
	for i, line := range proseLines(src) {
		for _, m := range mdInlineLink.FindAllStringSubmatch(line, -1) {
			out = append(out, docLink{i + 1, strings.Trim(m[1], "<>"), ""})
		}
		if m := mdRefDef.FindStringSubmatch(line); m != nil {
			out = append(out, docLink{i + 1, strings.Trim(m[1], "<>"), ""})
		}
		for _, m := range htmlLinkAttr.FindAllStringSubmatch(line, -1) {
			out = append(out, docLink{i + 1, m[2], strings.ToLower(m[1])})
		}
	}
	return out
}

// linkProblem judges one destination found in file; files and dirs are the repository's content.
func linkProblem(file string, l docLink, files, dirs map[string]bool, anchorsOf func(string) map[string]bool) string {
	dest := strings.TrimSpace(l.dest)
	lower := strings.ToLower(dest)
	switch {
	case strings.HasPrefix(lower, "http://"):
		return "http:// link - use https:// (rule 7)"
	case l.tag == "script" && (urlScheme.MatchString(dest) || strings.HasPrefix(dest, "//")):
		return "a script loaded from elsewhere (rule 7)"
	case urlScheme.MatchString(dest) || strings.HasPrefix(dest, "//"):
		return ""
	}
	target, frag, _ := strings.Cut(dest, "#")
	if q := strings.IndexByte(target, '?'); q >= 0 {
		target = target[:q]
	}
	if u, err := url.PathUnescape(target); err == nil {
		target = u
	}
	resolved := file
	if target != "" {
		if strings.HasPrefix(target, "/") {
			resolved = path.Clean(strings.TrimPrefix(target, "/"))
		} else {
			resolved = path.Join(path.Dir(file), target)
		}
		resolved = strings.TrimSuffix(resolved, "/")
		if resolved == "" {
			resolved = "."
		}
		if !files[resolved] && !dirs[resolved] {
			kind := "link"
			if l.tag == "img" || l.tag == "source" {
				kind = "image"
			}
			return kind + " target " + resolved + " is not in the repository (rule 3/6)"
		}
	}
	if frag == "" || lineAnchor.MatchString(frag) || !strings.HasSuffix(strings.ToLower(resolved), ".md") {
		return ""
	}
	if f, err := url.PathUnescape(frag); err == nil {
		frag = f
	}
	anchors := anchorsOf(resolved)
	if anchors[frag] || anchors[strings.ToLower(frag)] {
		return ""
	}
	return "anchor #" + frag + " is not a heading or id in " + resolved + " (rule 3)"
}

func TestDocLinks(t *testing.T) {
	files := repoFiles(t)
	dirs := repoDirs(files)
	cache := map[string]map[string]bool{}
	anchorsOf := func(f string) map[string]bool {
		if a, ok := cache[f]; ok {
			return a
		}
		a := headingAnchors(readRepoFile(t, f))
		cache[f] = a
		return a
	}
	docs := markdownFiles(files)
	sort.Strings(docs)
	if len(docs) == 0 {
		t.Fatal("no Markdown document found - the git listing may have failed")
	}
	for _, f := range docs {
		for _, l := range docLinks(readRepoFile(t, f)) {
			if p := linkProblem(f, l, files, dirs, anchorsOf); p != "" {
				t.Errorf("%s:%d: %s (%s)", f, l.line, p, l.dest)
			}
		}
	}
}

// TestDocLinkRules pins the judgement on fixed inputs, so a regression in the scanner cannot pass
// silently by finding nothing.
func TestDocLinkRules(t *testing.T) {
	files := map[string]bool{"docs/A.md": true, "docs/img/x.png": true, "README.md": true}
	dirs := repoDirs(files)
	anchorsOf := func(f string) map[string]bool {
		return map[string]bool{"build-vs-release": true, "flags": true}
	}
	cases := []struct {
		dest, tag string
		bad       bool
	}{
		{"A.md", "", false},
		{"./A.md#flags", "", false},
		{"A.md#missing", "", true},
		{"../README.md", "", false},
		{"/README.md#build-vs-release", "", false},
		{"img/x.png", "img", false},
		{"img/y.png", "img", true},
		{"B.md", "", true},
		{"img/", "", false},
		{"http://example.com", "", true},
		{"https://example.com", "", false},
		{"https://cdn.example/x.js", "script", true},
		{"mailto:a@b", "", false},
		{"#flags", "", false},
		{"#nope", "", true},
		{"../internal/x.go#L10", "", true},
	}
	for _, c := range cases {
		got := linkProblem("docs/A.md", docLink{1, c.dest, c.tag}, files, dirs, anchorsOf)
		if (got != "") != c.bad {
			t.Errorf("%q (%s): problem %q, want bad=%v", c.dest, c.tag, got, c.bad)
		}
	}

	src := "# Build vs Release\n\n## Flags\n\n## Flags\n\n```\n# not a heading\n```\n\nSetext `code`\n---\n\n## `-ocr` and [links](x.md)\n\n## Что нового?\n"
	want := []string{"build-vs-release", "flags", "flags-1", "setext-code", "-ocr-and-links", "что-нового"}
	got := headingAnchors(src)
	for _, w := range want {
		if !got[w] {
			t.Errorf("headingAnchors: missing %q in %v", w, got)
		}
	}
	if got["not-a-heading"] {
		t.Error("headingAnchors: a line inside a fence became a heading")
	}

	links := docLinks("see `[x](gone.md)` and [y](here.md)\n```\n[z](fenced.md)\n```\n<img src=\"i.png\">\n[r]: ref.md\n")
	var dests []string
	for _, l := range links {
		dests = append(dests, l.dest)
	}
	if strings.Join(dests, ",") != "here.md,i.png,ref.md" {
		t.Errorf("docLinks: got %v, want here.md, i.png, ref.md", dests)
	}
}
