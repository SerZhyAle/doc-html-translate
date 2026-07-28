package htmlgen

import (
	"bytes"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/i18n"

	gohtml "golang.org/x/net/html"
)

// singlePageCSS gives the merged document a single centred reading column and a light
// separator between the chapters that were concatenated into it.
const singlePageCSS = `
<style id="dht-single-css">
  body { margin: 0; }
  main.dht-single { width: 95%; max-width: 46em; margin: 1.5em auto 6em; padding: 0 1.2em; }
  main.dht-single img { max-width: 100%; height: auto; }
  hr.dht-chapter-sep { border: 0; border-top: 1px dashed var(--dht-border); margin: 2.5em 0; }
</style>
`

// GenerateSinglePage merges every spine content file, in reading order, into a single
// index.html carrying the unified reader header (theme/font controls, progress bar) but no
// per-chapter navigation and no separate table-of-contents page. The merged file is written
// inside the book's base directory so relative image/CSS references resolve unchanged; when
// that base directory is nested, a redirecting index.html is also written at the output root.
// On success the book is collapsed to a single merged spine entry so the downstream OCR and
// translation steps operate on the one file (and the post-translation TOC step, keyed off
// len(Spine) > 1, is skipped). Returns the entry-point path to open in the browser.
func GenerateSinglePage(book *epub.Book, outputDir, sourceName string) (string, error) {
	spineHrefs := book.SpineHrefs()
	if len(spineHrefs) == 0 {
		return "", fmt.Errorf("book has no spine entries")
	}

	// Merge chapter bodies in spine order; take the source language from the first page.
	var body strings.Builder
	lang := "en"
	for i, href := range spineHrefs {
		pagePath := bookPath(outputDir, book.BasePath, href)
		data, err := os.ReadFile(pagePath)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", href, err)
		}
		doc, err := gohtml.Parse(bytes.NewReader(data))
		if err != nil {
			return "", fmt.Errorf("parse %s: %w", href, err)
		}
		if i == 0 {
			if l := htmlLang(doc); l != "" {
				lang = l
			}
		} else {
			body.WriteString("\n  <hr class=\"dht-chapter-sep\">\n")
		}
		inner, err := bodyInnerHTML(doc)
		if err != nil {
			return "", fmt.Errorf("extract body %s: %w", href, err)
		}
		body.WriteString(inner)
	}

	// CSS from the manifest, relative to the merged file's directory (the base dir), so the
	// hrefs are used as authored - no BasePath prefix needed.
	var cssLinks []string
	for _, item := range book.Manifest {
		if item.MediaType == "text/css" {
			cssLinks = append(cssLinks, fmt.Sprintf(`  <link rel="stylesheet" href="%s">`, html.EscapeString(item.Href)))
		}
	}

	title := book.Title
	if title == "" {
		title = "Document"
	}

	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n")
	sb.WriteString(fmt.Sprintf("<html lang=%q>\n", lang))
	sb.WriteString("<head>\n")
	sb.WriteString("  <meta charset=\"UTF-8\">\n")
	sb.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	sb.WriteString(fmt.Sprintf("  <title>%s</title>\n", html.EscapeString(title)))
	// The icon is written to the output root, but the merged page can live one level down
	// inside the book's base dir, so link it relative to that.
	sb.WriteString("  " + faviconLink(book.BasePath))
	if len(cssLinks) > 0 {
		sb.WriteString(strings.Join(cssLinks, "\n") + "\n")
	}
	sb.WriteString(singlePageCSS)
	sb.WriteString(navBarCSS)
	sb.WriteString(readerCSS)
	sb.WriteString("</head>\n")
	sb.WriteString("<body>\n")
	sb.WriteString(buildSinglePageHeader(sourceName, title))
	sb.WriteString("\n<main class=\"dht-single\">\n")
	sb.WriteString(body.String())
	sb.WriteString("\n</main>\n")
	sb.WriteString(navBarScript)
	sb.WriteString(readerScript(bookStorageKey(book.Title, 1), "index.html", 1, 1))
	sb.WriteString("</body>\n</html>\n")

	WriteFavicon(outputDir)

	// Write the merged file inside the base directory so relative refs resolve.
	mergedPath := bookPath(outputDir, book.BasePath, "index.html")
	if err := os.MkdirAll(filepath.Dir(mergedPath), 0o755); err != nil {
		return "", fmt.Errorf("create merged dir: %w", err)
	}
	if err := os.WriteFile(mergedPath, []byte(sb.String()), 0o644); err != nil {
		return "", fmt.Errorf("write single page: %w", err)
	}

	entry := filepath.Join(outputDir, "index.html")
	if book.BasePath != "" && book.BasePath != "." {
		// Redirect entry at the output root -> merged page in the base dir.
		target := book.BasePath + "/index.html"
		redirect := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <script>location.replace(%q);</script>
</head>
<body></body>
</html>
`, target)
		if err := os.WriteFile(entry, []byte(redirect), 0o644); err != nil {
			return "", fmt.Errorf("write redirect index: %w", err)
		}
	}

	// Collapse the book to the single merged file for downstream OCR/translation.
	book.Manifest = []epub.ManifestItem{{ID: "dht-merged", Href: "index.html", MediaType: "application/xhtml+xml"}}
	book.Spine = []epub.SpineItem{{IDRef: "dht-merged"}}
	book.TOC = nil

	return entry, nil
}

// buildSinglePageHeader renders the unified reader header without the per-chapter
// navigation (no prev/next, no TOC link, no page counter) - only the reader controls and
// the version link, matching the shared chrome used elsewhere.
func buildSinglePageHeader(sourceName, title string) string {
	var fileEl, titleEl string
	if strings.TrimSpace(sourceName) != "" {
		fileEl = fmt.Sprintf(`<span class="nav-file" title="%s">%s</span>`,
			html.EscapeString(sourceName), html.EscapeString(sourceName))
	}
	if strings.TrimSpace(title) != "" {
		titleEl = fmt.Sprintf(`<span class="nav-title" title="%s">%s</span>`,
			html.EscapeString(title), html.EscapeString(title))
	}
	versionLink := fmt.Sprintf(
		`<a class="nav-version" href="%s" target="_blank" rel="noopener" title="%s">%s</a>`,
		projectURL, html.EscapeString(projectURL), html.EscapeString(versionLabel()))

	// lang/dir on the bar itself, never on <html> - see buildNavBarHTML for why.
	return fmt.Sprintf(`<div class="dht-navbar" lang="%s"%s>%s%s<div class="nav-actions">%s%s</div><div id="dht-progress" class="dht-progress"></div></div>`,
		i18n.Language(), chromeDirAttr(), fileEl, titleEl, readerControlsHTML(), versionLink)
}

// htmlLang returns the lang attribute of the document's <html> element, or "".
func htmlLang(doc *gohtml.Node) string {
	var find func(*gohtml.Node) string
	find = func(n *gohtml.Node) string {
		if n.Type == gohtml.ElementNode && n.Data == "html" {
			return nodeAttr(n, "lang")
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if v := find(c); v != "" {
				return v
			}
		}
		return ""
	}
	return find(doc)
}

// bodyInnerHTML renders the inner HTML of a parsed document's <body>.
func bodyInnerHTML(doc *gohtml.Node) (string, error) {
	body := findBodyNode(doc)
	if body == nil {
		return "", fmt.Errorf("no <body> element")
	}
	var buf bytes.Buffer
	for c := body.FirstChild; c != nil; c = c.NextSibling {
		if err := gohtml.Render(&buf, c); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}
