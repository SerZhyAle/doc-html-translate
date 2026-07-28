// Package htmlgen generates index.html from parsed EPUB metadata.
package htmlgen

import (
	"bytes"
	"fmt"
	"html"
	"os"
	"path" // hrefs are URLs, not OS paths
	"path/filepath"
	"strings"
	"unicode"

	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/i18n"

	gohtml "golang.org/x/net/html"
)

// GenerateIndex creates index.html in outputDir with a table of contents
// based on the book's spine order. CSS files from the manifest are linked.
func GenerateIndex(book *epub.Book, outputDir string) (string, error) {
	return GenerateIndexWithSnippetsDepth(book, outputDir, nil, 0)
}

// GenerateIndexWithSnippets creates index.html in outputDir with a table of contents.
// Precomputed snippets are used when available. TOC nesting is unlimited.
func GenerateIndexWithSnippets(book *epub.Book, outputDir string, snippets map[string]string) (string, error) {
	return GenerateIndexWithSnippetsDepth(book, outputDir, snippets, 0)
}

// GenerateIndexWithSnippetsDepth writes index.html. When book.TOC is populated
// (authored NCX/nav, PDF outline, or the heading-scan fallback) it renders a
// multi-level collapsible TOC limited to `depth` levels (0 = unlimited).
// Otherwise it falls back to the flat, one-entry-per-spine-page list.
func GenerateIndexWithSnippetsDepth(book *epub.Book, outputDir string, snippets map[string]string, depth int) (string, error) {
	indexPath := filepath.Join(outputDir, "index.html")
	WriteFavicon(outputDir)

	// Collect CSS files from manifest
	var cssLinks []string
	for _, item := range book.Manifest {
		if item.MediaType == "text/css" {
			href := item.Href
			if book.BasePath != "" && book.BasePath != "." {
				href = book.BasePath + "/" + href
			}
			cssLinks = append(cssLinks, fmt.Sprintf(`    <link rel="stylesheet" href="%s">`, html.EscapeString(href)))
		}
	}

	spineHrefs := book.SpineHrefs()

	// Build the <nav> body: a multi-level collapsible TOC when one is available,
	// otherwise the flat spine list (preserves behaviour for callers that do not
	// populate book.TOC).
	var navBody string
	if len(book.TOC) > 0 {
		navBody = renderTOCTree(book.TOC, book.BasePath, depth)
	} else {
		navBody = renderFlatSpineTOC(book, outputDir, snippets, spineHrefs)
	}

	title := book.Title
	if title == "" {
		title = "EPUB Book"
	}

	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n")
	sb.WriteString("<html lang=\"en\">\n")
	sb.WriteString("<head>\n")
	sb.WriteString("  <meta charset=\"UTF-8\">\n")
	sb.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	sb.WriteString(fmt.Sprintf("  <title>%s</title>\n", html.EscapeString(title)))
	sb.WriteString("  " + faviconLink("")) // the TOC index sits at the output root
	if len(cssLinks) > 0 {
		sb.WriteString(strings.Join(cssLinks, "\n") + "\n")
	}
	sb.WriteString("  <style>\n")
	sb.WriteString("    body { font-family: Georgia, 'Times New Roman', serif; width: 95%; max-width: 1400px; margin: 2em auto; padding: 0 1em; }\n")
	sb.WriteString("    h1 { border-bottom: 1px solid #ccc; padding-bottom: 0.3em; }\n")
	sb.WriteString("    nav ul { list-style: none; padding-left: 1.3em; }\n")
	sb.WriteString("    nav > ul { padding-left: 0; }\n")
	sb.WriteString("    nav li { margin: 0.4em 0; }\n")
	sb.WriteString("    nav a { text-decoration: none; color: #1a0dab; }\n")
	sb.WriteString("    nav a:hover { text-decoration: underline; }\n")
	sb.WriteString("    nav details { margin: 0.2em 0; }\n")
	sb.WriteString("    nav summary { cursor: pointer; padding: 0.2em 0; color: #1a0dab; }\n")
	sb.WriteString("    nav summary a { display: inline; }\n")
	sb.WriteString("    .toc-label { font-weight: bold; margin-right: 0.4em; }\n")
	sb.WriteString("    .toc-snippet { font-size: 0.9em; color: #333; font-style: italic; }\n")
	sb.WriteString("    .toc-section { color: #333; }\n")
	sb.WriteString("    .meta { color: #666; font-size: 0.9em; margin-bottom: 2em; }\n")
	sb.WriteString("  </style>\n")
	sb.WriteString(readerCSS)
	sb.WriteString("</head>\n")
	sb.WriteString("<body>\n")
	sb.WriteString(fmt.Sprintf("  <h1>%s</h1>\n", html.EscapeString(title)))
	sb.WriteString(fmt.Sprintf("  <p class=\"meta\">%s</p>\n", html.EscapeString(i18n.S("Chapters: %d", len(spineHrefs)))))
	continueLabel := i18n.S("Continue reading")
	sb.WriteString(fmt.Sprintf("  <div class=\"dht-toolbar\" lang=\"%s\"%s>", i18n.Language(), chromeDirAttr()))
	sb.WriteString(readerControlsHTML())
	sb.WriteString(fmt.Sprintf(`<a id="dht-continue" class="dht-continue" href="#">&#9656; %s</a>`, continueLabel))
	sb.WriteString("</div>\n")
	sb.WriteString("  <nav>\n")
	sb.WriteString(navBody)
	sb.WriteString("  </nav>\n")
	sb.WriteString(readerScript(bookStorageKey(book.Title, len(spineHrefs)), "", 0, len(spineHrefs)))
	sb.WriteString("</body>\n")
	sb.WriteString("</html>\n")

	if err := os.WriteFile(indexPath, []byte(sb.String()), 0o644); err != nil {
		return "", fmt.Errorf("write index.html: %w", err)
	}

	return indexPath, nil
}

// renderFlatSpineTOC renders the legacy one-entry-per-spine-page list, with a
// first-sentence snippet (translated when available) as each label. Returns the
// <ul>…</ul> block for insertion into <nav>.
func renderFlatSpineTOC(book *epub.Book, outputDir string, snippets map[string]string, spineHrefs []string) string {
	var tocEntries []string
	for i, href := range spineHrefs {
		fullHref := href
		if book.BasePath != "" && book.BasePath != "." {
			fullHref = book.BasePath + "/" + href
		}
		label := chapterLabel(href, i+1)

		snippet := ""
		if snippets != nil {
			snippet = snippets[href]
		}
		if snippet == "" {
			pagePath := bookPath(outputDir, book.BasePath, href)
			snippet = extractSnippet(pagePath)
		}

		if snippet != "" {
			tocEntries = append(tocEntries, fmt.Sprintf(
				`      <li><a href="%s"><span class="toc-label">%d.</span><span class="toc-snippet">%s</span></a></li>`,
				html.EscapeString(fullHref), i+1, html.EscapeString(snippet)))
		} else {
			tocEntries = append(tocEntries, fmt.Sprintf(
				`      <li><a href="%s"><span class="toc-label">%s</span></a></li>`,
				html.EscapeString(fullHref), html.EscapeString(label)))
		}
	}

	var sb strings.Builder
	sb.WriteString("    <ul>\n")
	if len(tocEntries) > 0 {
		sb.WriteString(strings.Join(tocEntries, "\n") + "\n")
	}
	sb.WriteString("    </ul>\n")
	return sb.String()
}

// renderTOCTree renders a nested TOC as collapsible <details> sections (for
// entries with children) and plain links (for leaves), honouring depth
// (0 = unlimited). Returns the top-level <ul>…</ul> block.
func renderTOCTree(entries []epub.TOCEntry, basePath string, depth int) string {
	var sb strings.Builder
	renderTOCList(&sb, entries, basePath, depth, 1, 2)
	return sb.String()
}

func renderTOCList(sb *strings.Builder, entries []epub.TOCEntry, basePath string, depth, level, indent int) {
	pad := strings.Repeat("  ", indent)
	sb.WriteString(pad + "<ul>\n")
	for _, e := range entries {
		renderTOCEntry(sb, e, basePath, depth, level, indent+1)
	}
	sb.WriteString(pad + "</ul>\n")
}

func renderTOCEntry(sb *strings.Builder, e epub.TOCEntry, basePath string, depth, level, indent int) {
	pad := strings.Repeat("  ", indent)

	label := html.EscapeString(e.Title)
	if strings.TrimSpace(e.Title) == "" {
		label = "&#8230;" // ellipsis placeholder for an untitled grouping node
	}

	var labelHTML string
	if e.Href != "" {
		labelHTML = fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(prefixBase(basePath, e.Href)), label)
	} else {
		labelHTML = fmt.Sprintf(`<span class="toc-section">%s</span>`, label)
	}

	showChildren := len(e.Children) > 0 && (depth <= 0 || level < depth)
	if showChildren {
		sb.WriteString(pad + "<li><details open><summary>" + labelHTML + "</summary>\n")
		renderTOCList(sb, e.Children, basePath, depth, level+1, indent+1)
		sb.WriteString(pad + "</details></li>\n")
	} else {
		sb.WriteString(pad + "<li>" + labelHTML + "</li>\n")
	}
}

// prefixBase joins the OPF base directory to a book-relative href (which may
// carry a #fragment), matching the spine-href convention.
func prefixBase(basePath, href string) string {
	if basePath != "" && basePath != "." {
		return basePath + "/" + href
	}
	return href
}

func bookPath(outputDir, basePath, href string) string {
	if basePath != "" && basePath != "." {
		return filepath.Join(outputDir, filepath.FromSlash(basePath), filepath.FromSlash(href))
	}
	return filepath.Join(outputDir, filepath.FromSlash(href))
}

// extractSnippet reads an HTML page and returns the first meaningful sentence
// (up to ~150 chars) from its visible text content. Returns "" on any error.
func extractSnippet(htmlPath string) string {
	data, err := os.ReadFile(htmlPath)
	if err != nil {
		return ""
	}
	doc, err := gohtml.Parse(bytes.NewReader(data))
	if err != nil {
		return ""
	}
	return ExtractSnippetFromDoc(doc)
}

// ExtractSnippetFromDoc returns the first meaningful visible-text snippet from a parsed HTML document.
func ExtractSnippetFromDoc(doc *gohtml.Node) string {
	// Collect visible text from <body>, skipping nav/script/style/structural elements.
	var buf strings.Builder
	var walk func(*gohtml.Node)
	walk = func(n *gohtml.Node) {
		if n.Type == gohtml.ElementNode {
			switch n.Data {
			case "script", "style", "head", "nav", "button":
				return // skip non-content nodes
			case "div", "header", "footer":
				// Skip our structural divs (page-header, dht-navbar, etc.)
				if nodeHasClass(n, "page-header") || nodeHasClass(n, "dht-navbar") ||
					nodeAttr(n, "id") == "dht-nav" || nodeAttr(n, "id") == "dht-zoom-sync" {
					return
				}
			}
		}
		if n.Type == gohtml.TextNode {
			t := strings.TrimSpace(n.Data)
			if t != "" {
				if buf.Len() > 0 {
					buf.WriteByte(' ')
				}
				buf.WriteString(t)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if buf.Len() > 300 {
				return // collected enough
			}
			walk(c)
		}
	}
	// Start from body if available.
	body := findBodyNode(doc)
	if body != nil {
		walk(body)
	} else {
		walk(doc)
	}

	text := strings.TrimSpace(buf.String())
	if text == "" {
		return ""
	}

	const maxLen = 250
	runes := []rune(text)

	// Find up to 2 sentence boundaries (., !, ?) within maxLen.
	limit := len(runes)
	if limit > maxLen {
		limit = maxLen
	}
	sentences := 0
	for i, r := range runes[:limit] {
		if r == '.' || r == '!' || r == '?' {
			sentences++
			if sentences == 2 {
				return strings.TrimRightFunc(string(runes[:i+1]), unicode.IsSpace)
			}
		}
	}
	// Fewer than 2 sentences found within limit — return what we have.
	if len(runes) > maxLen {
		return strings.TrimRightFunc(string(runes[:maxLen]), unicode.IsSpace) + ".."
	}
	return text
}

// findBodyNode walks the parse tree to find the <body> element.
func findBodyNode(n *gohtml.Node) *gohtml.Node {
	if n.Type == gohtml.ElementNode && n.Data == "body" {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findBodyNode(c); found != nil {
			return found
		}
	}
	return nil
}

// nodeAttr returns the value of the named attribute on an element node, or "".
func nodeAttr(n *gohtml.Node, name string) string {
	for _, a := range n.Attr {
		if a.Key == name {
			return a.Val
		}
	}
	return ""
}

// nodeHasClass reports whether an element node has the given CSS class.
func nodeHasClass(n *gohtml.Node, class string) bool {
	for _, word := range strings.Fields(nodeAttr(n, "class")) {
		if word == class {
			return true
		}
	}
	return false
}

// GenerateSinglePageIndex creates an index.html that instantly redirects
// to the single content page. Used when the book fits on one page —
// no TOC or navigation bars are needed.
func GenerateSinglePageIndex(book *epub.Book, outputDir string) (string, error) {
	indexPath := filepath.Join(outputDir, "index.html")

	spineHrefs := book.SpineHrefs()
	if len(spineHrefs) == 0 {
		return "", fmt.Errorf("book has no spine entries")
	}

	target := spineHrefs[0]
	if book.BasePath != "" && book.BasePath != "." {
		target = book.BasePath + "/" + target
	}

	// This path skips the navbar injection, so the icon has to be put on the content page
	// directly. The redirect page below is replaced before it ever paints, so it needs no
	// icon of its own - the page the reader lands on does.
	WriteFavicon(outputDir)
	injectFavicon(filepath.Join(outputDir, filepath.FromSlash(target)), path.Dir(target))

	// JS redirect (instant, preserves browser history correctly vs meta-refresh)
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <script>location.replace(%q);</script>
</head>
<body></body>
</html>
`, target)

	if err := os.WriteFile(indexPath, []byte(html), 0o644); err != nil {
		return "", fmt.Errorf("write single-page index: %w", err)
	}
	return indexPath, nil
}

// chapterLabel creates a human-readable label for a spine entry.
func chapterLabel(href string, index int) string {
	base := filepath.Base(href)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	// Clean up common naming patterns
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")

	if name == "" {
		return fmt.Sprintf("Chapter %d", index)
	}
	return fmt.Sprintf("%d. %s", index, name)
}
