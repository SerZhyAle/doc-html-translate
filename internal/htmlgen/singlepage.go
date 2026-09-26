package htmlgen

import (
	"bytes"
	"fmt"
	"html"
	"os"
	"path"
	"path/filepath"
	"strings"

	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/fsutil"
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
  /* An image-page book (comic, scan) is merged page by page, so the separator marks a
     page break rather than a chapter: no rule, just breathing room between the plates. */
  hr.dht-page-sep { border: 0; margin: 1.5em 0; }
</style>
`

// GenerateSinglePage merges every spine content file, in reading order, into a single
// index.html carrying the unified reader header (theme/font controls, progress bar) but no
// per-chapter navigation and no separate table-of-contents page. The merged file is written
// inside the book's base directory, each chapter's references rebased from its own folder and
// its in-book links turned into in-page anchors (prepareMerge); when that base directory is
// nested, a redirecting index.html is also written at the output root.
// On success the book is collapsed to a single merged spine entry so the downstream OCR and
// translation steps operate on the one file (and the post-translation TOC step, keyed off
// len(Spine) > 1, is skipped). Returns the entry-point path to open in the browser.
func GenerateSinglePage(book *epub.Book, outputDir, sourceName string) (string, error) {
	spineHrefs := book.SpineHrefs()
	if len(spineHrefs) == 0 {
		return "", fmt.Errorf("book has no spine entries")
	}

	// Merge chapter bodies in spine order; take the source language and direction from the
	// first page. Dropping dir turned an RTL book LTR in the default single-page flow.
	var body strings.Builder
	lang, dir := "en", ""
	chapters := make([]*mergeChapter, 0, len(spineHrefs))
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
			dir = htmlDir(doc)
		}
		chapters = append(chapters, &mergeChapter{href: path.Clean(href), doc: doc})
	}
	prepareMerge(chapters)
	inners := make([]string, 0, len(chapters))
	var allScopedCSS []string
	for i, ch := range chapters {
		inner, err := chapterInnerHTML(ch, i+1)
		if err != nil {
			return "", fmt.Errorf("extract body %s: %w", ch.href, err)
		}
		inners = append(inners, inner)
		if len(ch.scopedCSS) > 0 {
			allScopedCSS = append(allScopedCSS, strings.Join(ch.scopedCSS, "\n"))
		}
	}

	// A book whose spine entries are image pages (comic, scanned image, multi-frame TIFF)
	// gets the page separator; a text book keeps the chapter rule. The wrapper the
	// extractors emit is the signal, so no extra field has to be threaded through the
	// pipeline just to say "these are pages, not chapters".
	sep := "\n  <hr class=\"dht-chapter-sep\">\n"
	if isPagedBook(inners) {
		sep = "\n  <hr class=\"dht-page-sep\">\n"
	}
	for i, inner := range inners {
		if i > 0 {
			body.WriteString(sep)
		}
		body.WriteString(inner)
	}

	// CSS from the manifest, relative to the merged file's directory (the base dir), so the
	// hrefs are used as authored - no BasePath prefix needed.
	var cssLinks []string
	for _, item := range book.Manifest {
		if item.MediaType == "text/css" {
			cssLinks = append(cssLinks, fmt.Sprintf(`  <link rel="stylesheet" href="%s">`, html.EscapeString(epub.URLPath(item.Href))))
		}
	}

	title := book.Title
	if title == "" {
		title = "Document"
	}

	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n")
	// The book's language stays on <html>: it is what makes Chrome offer "Translate page".
	if dir == "rtl" || dir == "ltr" || dir == "auto" {
		sb.WriteString(fmt.Sprintf("<html lang=\"%s\" dir=\"%s\">\n", html.EscapeString(lang), dir))
	} else {
		sb.WriteString(fmt.Sprintf("<html lang=\"%s\">\n", html.EscapeString(lang)))
	}
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
	if len(allScopedCSS) > 0 {
		sb.WriteString("<style id=\"dht-scoped-css\">\n")
		sb.WriteString(strings.Join(allScopedCSS, "\n\n"))
		sb.WriteString("\n</style>\n")
	}
	sb.WriteString("</head>\n")
	sb.WriteString("<body>\n")
	pageCount := 0
	if isPagedBook(inners) {
		pageCount = len(inners)
	}
	sb.WriteString(buildSinglePageHeader(sourceName, title, pageCount))
	sb.WriteString("\n<main class=\"dht-single\">\n")
	sb.WriteString(body.String())
	sb.WriteString("\n</main>\n")
	sb.WriteString(navBarScript)
	sb.WriteString(readerScript(readerKey(book), "index.html", 1, 1))
	sb.WriteString("</body>\n</html>\n")

	WriteFavicon(outputDir)

	// Write the merged file inside the base directory so relative refs resolve.
	mergedPath := bookPath(outputDir, book.BasePath, "index.html")
	if err := os.MkdirAll(filepath.Dir(mergedPath), 0o755); err != nil {
		return "", fmt.Errorf("create merged dir: %w", err)
	}
	if err := fsutil.WriteFile(mergedPath, []byte(sb.String()), 0o644); err != nil {
		return "", fmt.Errorf("write single page: %w", err)
	}

	// The merged file has absorbed every spine page, so the originals are dead weight: they
	// are unreachable (prepareMerge pointed every in-book link into the merged page), they carry no navbar, no theme and no OCR
	// plates - the overlay step runs on the merged file only - so anyone who does reach one
	// by guessing a filename gets a worse page than the book they asked for. Removal is
	// best-effort and happens only after the merge is safely on disk; a file we cannot
	// delete is left alone rather than failing a conversion that already succeeded.
	for _, href := range spineHrefs {
		// Never the merged file itself - compared ignoring case, because on NTFS a chapter
		// named Index.html is that file, and removing it would delete the whole book.
		if strings.EqualFold(path.Clean(href), "index.html") {
			continue
		}
		_ = os.Remove(bookPath(outputDir, book.BasePath, href))
	}

	entry := filepath.Join(outputDir, "index.html")
	if book.BasePath != "" && book.BasePath != "." {
		// Redirect entry at the output root -> merged page in the base dir.
		target := epub.URLPath(book.BasePath) + "/index.html"
		redirect := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <script>location.replace(%s);</script>
</head>
<body></body>
</html>
`, jsString(target))
		if err := fsutil.WriteFile(entry, []byte(redirect), 0o644); err != nil {
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
// navigation (no prev/next, no TOC link) - only the reader controls and the version link,
// matching the shared chrome used elsewhere.
//
// pageCount > 0 adds a page jump box for an image-page book (comic, scan). The merged
// document is one long scroll of dozens of pages, so without it a reader cannot reach page
// 30 except by dragging the scrollbar, and cannot link to it at all.
func buildSinglePageHeader(sourceName, title string, pageCount int) string {
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

	var pageSel string
	if pageCount > 1 {
		var opts strings.Builder
		for i := 1; i <= pageCount; i++ {
			opts.WriteString(fmt.Sprintf(`<option value="#page_%03d">%s</option>`,
				i, html.EscapeString(fmt.Sprintf("%d / %d", i, pageCount))))
		}
		// nav.go-to-page, not nav.go-to: "Go to" alone opens another screen (ICON-SET rule 3).
		pageSel = fmt.Sprintf(`<span class="dht-sel">%[1]s<select id="dht-page-sel" title="%[2]s" aria-label="%[2]s">%[3]s</select></span>`,
			glyphSVG("nav.go-to-page"), html.EscapeString(i18n.S("Go to page")), opts.String())
	}

	// lang/dir on the bar itself, never on <html> - see buildNavBarHTML for why.
	return fmt.Sprintf(`<div class="dht-navbar" lang="%s"%s>%s%s<div class="nav-actions">%s%s%s</div><div id="dht-progress" class="dht-progress"></div></div>`,
		i18n.Language(), chromeDirAttr(), fileEl, titleEl, pageSel, readerControlsHTML(), versionLink)
}

// isPagedBook reports whether the merged bodies are image pages rather than text
// chapters, by looking for the section.dht-page wrapper that internal/comic and
// internal/img emit around a page image. Every entry must carry it: one text chapter
// in the spine makes the book a text book, and the chapter rule is then the honest
// separator.
func isPagedBook(inners []string) bool {
	if len(inners) == 0 {
		return false
	}
	for _, inner := range inners {
		if !strings.Contains(inner, `class="dht-page"`) {
			return false
		}
	}
	return true
}

// htmlLang returns the language of the document's <html> element: lang, or xml:lang when
// lang is absent (XHTML-born pages often carry only that), or "".
func htmlLang(doc *gohtml.Node) string {
	var find func(*gohtml.Node) string
	find = func(n *gohtml.Node) string {
		if n.Type == gohtml.ElementNode && n.Data == "html" {
			if l := strings.TrimSpace(nodeAttr(n, "lang")); l != "" {
				return l
			}
			return strings.TrimSpace(nodeAttr(n, "xml:lang"))
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

// htmlDir returns the lower-cased dir of the document's <html> element, falling back to
// <body>, because a converted HTML page may declare its direction on either.
func htmlDir(doc *gohtml.Node) string {
	var d string
	var find func(*gohtml.Node)
	find = func(n *gohtml.Node) {
		if n.Type == gohtml.ElementNode && (n.Data == "html" || n.Data == "body") && d == "" {
			d = strings.ToLower(strings.TrimSpace(nodeAttr(n, "dir")))
		}
		for c := n.FirstChild; c != nil && d == ""; c = c.NextSibling {
			find(c)
		}
	}
	find(doc)
	return d
}

// chapterInnerHTML renders the chapter's content wrapped in a container div with
// class "dht-chapter dht-ch-<pos>" and any attributes from the original <body>.
func chapterInnerHTML(ch *mergeChapter, pos int) (string, error) {
	body := findBodyNode(ch.doc)
	if body == nil {
		return "", fmt.Errorf("no <body> element")
	}
	scopeClass := fmt.Sprintf("dht-ch-%d", pos)
	classes := []string{"dht-chapter", scopeClass}
	var otherAttrs []string
	for _, a := range body.Attr {
		if a.Namespace != "" {
			continue
		}
		switch a.Key {
		case "class":
			if val := strings.TrimSpace(a.Val); val != "" {
				classes = append(classes, val)
			}
		case "id":
			// Root body id is mapped to rootIDs / anchor if linked.
		default:
			otherAttrs = append(otherAttrs, fmt.Sprintf(`%s="%s"`, a.Key, html.EscapeString(a.Val)))
		}
	}
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf(`<div class="%s"`, html.EscapeString(strings.Join(classes, " "))))
	for _, attr := range otherAttrs {
		buf.WriteString(" ")
		buf.WriteString(attr)
	}
	buf.WriteString(">\n")
	for c := body.FirstChild; c != nil; c = c.NextSibling {
		if err := gohtml.Render(&buf, c); err != nil {
			return "", err
		}
	}
	buf.WriteString("\n</div>")
	return buf.String(), nil
}
