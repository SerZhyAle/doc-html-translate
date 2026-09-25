package assets

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	cssImport = regexp.MustCompile(`(?i)@import\s+(?:url\(\s*("[^"]*"|'[^']*'|[^)\s]*)\s*\)|("[^"]*"|'[^']*'))([^;]*);?`)
	cssURL    = regexp.MustCompile(`(?i)url\(\s*("[^"]*"|'[^']*'|[^)\s'"]*)\s*\)`)
	// cssRef finds both in one pass, so the target of an @import is never also treated as
	// a plain url() asset.
	cssRef = regexp.MustCompile(cssImport.String() + `|` + cssURL.String())
)

// RewriteCSS copies the local files CSS text (from a stylesheet or style attribute living
// in baseDir) references and points its url()s at the copies. A remote @import is removed -
// the output is read offline - and a local one is copied as a stylesheet in turn. A
// reference refused for leaving the source tree becomes "none" so it loads nothing.
func (c *Copier) RewriteCSS(css, baseDir string) string {
	return cssRef.ReplaceAllStringFunc(css, func(m string) string {
		if m[0] == '@' {
			return c.rewriteImport(m, baseDir)
		}
		sub := cssURL.FindStringSubmatch(m)
		name, out := c.place(unquote(sub[1]), baseDir, false)
		switch out {
		case copied:
			return `url("` + name + `")`
		case refused:
			return "none"
		}
		return m
	})
}

func (c *Copier) rewriteImport(m, baseDir string) string {
	sub := cssImport.FindStringSubmatch(m)
	target := sub[1]
	if target == "" {
		target = sub[2]
	}
	ref := unquote(target)
	if isRemote(ref) {
		return ""
	}
	name, out := c.place(ref, baseDir, true)
	switch out {
	case copied:
		return `@import url("` + name + `")` + sub[3] + ";"
	case refused:
		return ""
	}
	return m
}

// LinkedStylesheet copies a stylesheet the document links (href as written, resolved from
// the document's directory) and returns the output name to link, or ok=false when there is
// nothing to link: a remote sheet (never fetched), one outside the source tree, or a
// missing file. media is the link's media attribute.
func (c *Copier) LinkedStylesheet(href, media string) (string, bool) {
	if isRemote(href) || strings.HasPrefix(strings.ToLower(strings.TrimSpace(href)), "data:") {
		return "", false
	}
	name, out := c.place(href, c.root, true)
	if out != copied {
		return "", false
	}
	return c.scopeToMedia(name, media)
}

// InlineStylesheet writes the text of a document's <style> block to its own stylesheet
// file and returns the output name to link. A file rather than an inline block because
// the single-page merge carries only the body and the book's linked stylesheets.
func (c *Copier) InlineStylesheet(css, media string) (string, bool) {
	name := c.claim("style.css")
	if err := os.WriteFile(filepath.Join(c.outDir, name), []byte(c.RewriteCSS(css, c.root)), 0o644); err != nil {
		return "", false
	}
	c.sheets++
	return c.scopeToMedia(name, media)
}

// scopeToMedia returns name when the sheet applies everywhere; otherwise it writes a
// wrapper that imports the sheet under the media query and returns the wrapper. The
// stylesheet list a page can carry has no media slot, and a print-only sheet applied on
// screen would restyle the page.
func (c *Copier) scopeToMedia(name, media string) (string, bool) {
	media = strings.Map(func(r rune) rune {
		if strings.ContainsRune(";{}\"'\\", r) {
			return -1
		}
		return r
	}, strings.TrimSpace(media))
	if media == "" || strings.EqualFold(media, "all") {
		return name, true
	}
	wrapper := c.claim(strings.TrimSuffix(name, filepath.Ext(name)) + "_media.css")
	body := fmt.Sprintf("@import url(%q) %s;\n", name, media)
	if err := os.WriteFile(filepath.Join(c.outDir, wrapper), []byte(body), 0o644); err != nil {
		return "", false
	}
	c.sheets++
	return wrapper, true
}

// writeSheetFile copies the stylesheet at src to the output name, rewriting its references
// relative to its own directory.
func (c *Copier) writeSheetFile(src, name string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(c.outDir, name), []byte(c.RewriteCSS(string(data), filepath.Dir(src))), 0o644)
}

func unquote(s string) string {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return s
}
