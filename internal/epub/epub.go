// Package epub handles EPUB archive extraction and metadata parsing.
// Supports both EPUB2 (.ncx navigation) and EPUB3 (nav.xhtml).
package epub

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"doc-html-translate/internal/i18n"
	"doc-html-translate/internal/logging"
)

// Book represents a parsed EPUB structure.
type Book struct {
	Title    string
	Manifest []ManifestItem
	Spine    []SpineItem
	BasePath string     // directory within EPUB where content.opf resides (slash path, "." at the root)
	TOC      []TOCEntry // authored table of contents (NCX navMap / nav.xhtml), nil if none

	// hrefRewrites maps an original content href to its final href after
	// normalization (e.g. "chapter1.xhtml" -> "chapter1.html", or an
	// index.* -> "_content_index.html" reserved-name rename). It lets TOC
	// resolution follow the same renames the content files underwent.
	hrefRewrites map[string]string

	// spineTocID is the manifest id referenced by <spine toc="...">, used to
	// locate the EPUB2 NCX when no manifest media-type identifies it.
	spineTocID string
}

// TOCEntry is one node of a (possibly nested) table of contents.
// Href is a URL relative to the OPF base directory and may carry a
// #fragment: the file part is URLPath of a manifest href, so unlike
// ManifestItem.Href it is already escaped for generated HTML.
type TOCEntry struct {
	Title    string
	Href     string
	Children []TOCEntry
}

// ManifestItem represents a single item in the OPF manifest.
//
// For an EPUB, Href is not the raw OPF attribute: it has passed resolveBookPath,
// so it is a decoded, cleaned slash path relative to BasePath that is known to
// stay inside the book. It is a file path, not a URL - pass it through URLPath
// before writing it into generated HTML.
type ManifestItem struct {
	ID         string
	Href       string
	MediaType  string
	Properties string // OPF properties attribute (e.g. "nav", "cover-image")
}

// SpineItem represents an entry in the OPF spine (reading order).
type SpineItem struct {
	IDRef string
}

// container.xml structures
type containerXML struct {
	XMLName   xml.Name   `xml:"container"`
	RootFiles []rootFile `xml:"rootfiles>rootfile"`
}

type rootFile struct {
	FullPath  string `xml:"full-path,attr"`
	MediaType string `xml:"media-type,attr"`
}

// content.opf structures
type packageOPF struct {
	XMLName  xml.Name    `xml:"package"`
	Metadata opfMetadata `xml:"metadata"`
	Manifest opfManifest `xml:"manifest"`
	Spine    opfSpine    `xml:"spine"`
}

type opfMetadata struct {
	Title []string `xml:"title"`
}

type opfManifest struct {
	Items []opfItem `xml:"item"`
}

type opfItem struct {
	ID         string `xml:"id,attr"`
	Href       string `xml:"href,attr"`
	MediaType  string `xml:"media-type,attr"`
	Properties string `xml:"properties,attr"`
}

type opfSpine struct {
	Toc      string       `xml:"toc,attr"`
	ItemRefs []opfItemRef `xml:"itemref"`
}

type opfItemRef struct {
	IDRef string `xml:"idref,attr"`
}

// Extract unpacks the EPUB (ZIP) into outputDir.
// Returns the Book metadata parsed from container.xml and content.opf.
// Uses best-effort: problematic files are skipped with warnings printed to stderr.
func Extract(epubPath, outputDir string) (*Book, error) {
	r, err := zip.OpenReader(epubPath)
	if err != nil {
		return nil, fmt.Errorf("open epub: %w", err)
	}
	defer r.Close()

	// Extract all files
	foldedNames := make(map[string]string, len(r.File))
	for _, f := range r.File {
		if err := extractFile(f, outputDir); err != nil {
			// best-effort: warn and continue
			fmt.Fprintf(os.Stderr, "WARNING: skip %s: %v\n", f.Name, err)
			continue
		}
		warnCaseCollision(foldedNames, f.Name)
	}

	// Parse container.xml to find content.opf path
	opfPath, err := parseContainer(outputDir)
	if err != nil {
		return nil, fmt.Errorf("parse container.xml: %w", err)
	}

	// Parse content.opf
	book, err := parseOPF(outputDir, opfPath)
	if err != nil {
		return nil, fmt.Errorf("parse content.opf: %w", err)
	}

	dropMissingContent(book, outputDir)

	// Re-serialize XHTML as HTML, transcode to UTF-8, unwrap SVG covers and
	// resolve the reserved index.html name, rewriting in-book links to match.
	// Must run before parseTOC so the TOC follows the recorded renames.
	normalizeContent(book, outputDir)

	// Parse the authored table of contents (EPUB3 nav.xhtml or EPUB2 toc.ncx)
	// into book.TOC. Best-effort: a missing/malformed TOC just leaves it nil
	// and the generator falls back to a spine/heading-based TOC.
	if err := parseTOC(book, outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: table-of-contents parse skipped: %v\n", err)
	}

	return book, nil
}

// warnCaseCollision reports an archive entry whose name differs from an
// earlier one only in letter case: on a case-insensitive file system (NTFS)
// the later entry has just overwritten the earlier one.
func warnCaseCollision(seen map[string]string, name string) {
	name = path.Clean(strings.ReplaceAll(name, `\`, "/"))
	key := strings.ToLower(name)
	if prev, ok := seen[key]; ok {
		if prev != name {
			logging.Errorf("WARNING: %s\n", i18n.S("Book files %s and %s differ only in letter case; on Windows one replaces the other", prev, name))
		}
		return
	}
	seen[key] = name
}

// dropMissingContent removes HTML manifest items whose file the archive did
// not ship, so one broken reference costs that chapter and not the whole book:
// every later stage (merge, navbar, split, translation) may then assume each
// content item exists.
func dropMissingContent(book *Book, outputDir string) {
	kept := book.Manifest[:0]
	for _, item := range book.Manifest {
		if isHTMLMediaType(item.MediaType) {
			st, err := os.Stat(bookPath(outputDir, book.BasePath, item.Href))
			if err != nil || !st.Mode().IsRegular() {
				logging.Errorf("WARNING: %s\n", i18n.S("Skipped book item %s: the file is not in the book", item.Href))
				continue
			}
		}
		kept = append(kept, item)
	}
	book.Manifest = kept
}

// recordHrefRewrite notes that content addressed as `from` by an earlier
// reference is now served from `to`. Each normalization stage records its own
// direct mapping; resolveHrefChain follows them transitively, so an
// "x.xhtml -> x.html" rewrite and a later "x.html -> _content_x.html" rename
// compose without either stage having to know about the other.
func (b *Book) recordHrefRewrite(from, to string) {
	if from == to {
		return
	}
	if b.hrefRewrites == nil {
		b.hrefRewrites = make(map[string]string)
	}
	b.hrefRewrites[from] = to
}

// resolveHrefChain follows hrefRewrites transitively from href to its final
// form, guarding against cycles.
func (b *Book) resolveHrefChain(href string) string {
	seen := make(map[string]bool)
	for {
		next, ok := b.hrefRewrites[href]
		if !ok || next == href || seen[href] {
			return href
		}
		seen[href] = true
		href = next
	}
}

func toHTMLExt(href string) string {
	low := strings.ToLower(href)
	if strings.HasSuffix(low, ".xhtml") {
		return href[:len(href)-len(".xhtml")] + ".html"
	}
	if strings.HasSuffix(low, ".xhtm") {
		return href[:len(href)-len(".xhtm")] + ".html"
	}
	return href
}

func bookPath(outputDir, basePath, href string) string {
	if basePath != "" && basePath != "." {
		return filepath.Join(outputDir, filepath.FromSlash(basePath), filepath.FromSlash(href))
	}
	return filepath.Join(outputDir, filepath.FromSlash(href))
}

// extractFile safely extracts a single file from the ZIP, protecting against path traversal.
func extractFile(f *zip.File, destDir string) error {
	// Normalize path separators
	name := filepath.FromSlash(f.Name)

	// Path traversal protection
	target := filepath.Join(destDir, name)
	if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)+string(os.PathSeparator)) {
		return fmt.Errorf("path traversal attempt: %s", f.Name)
	}

	if f.FileInfo().IsDir() {
		return os.MkdirAll(target, 0o755)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.Create(target)
	if err != nil {
		return err
	}
	defer out.Close()

	// Limit extraction to 100 MB per file as a safety measure
	_, err = io.Copy(out, io.LimitReader(rc, 100*1024*1024))
	return err
}

// parseContainer reads META-INF/container.xml and returns the full-path to the .opf file.
func parseContainer(baseDir string) (string, error) {
	containerPath := filepath.Join(baseDir, "META-INF", "container.xml")
	data, err := os.ReadFile(containerPath)
	if err != nil {
		return "", fmt.Errorf("read container.xml: %w", err)
	}

	var c containerXML
	if err := xml.Unmarshal(data, &c); err != nil {
		return "", fmt.Errorf("unmarshal container.xml: %w", err)
	}

	for _, rf := range c.RootFiles {
		if rf.MediaType == "application/oebps-package+xml" || strings.HasSuffix(rf.FullPath, ".opf") {
			opfPath, err := resolveBookPath("", rf.FullPath)
			if err != nil {
				return "", fmt.Errorf("rootfile %q: %w", rf.FullPath, err)
			}
			return opfPath, nil
		}
	}

	return "", fmt.Errorf("no rootfile found in container.xml")
}

// parseOPF reads content.opf and returns a populated Book. opfRelPath has
// already passed resolveBookPath. Every manifest href is resolved here, once
// (ADR-1 of hotfix-epub-href-containment): an item that is malformed or points
// outside the book is dropped with a warning, and its spine entries with it,
// so no later stage ever turns a book-supplied name into a path on its own.
func parseOPF(baseDir, opfRelPath string) (*Book, error) {
	opfFullPath := filepath.Join(baseDir, filepath.FromSlash(opfRelPath))
	data, err := os.ReadFile(opfFullPath)
	if err != nil {
		return nil, fmt.Errorf("read opf: %w", err)
	}

	var pkg packageOPF
	if err := xml.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("unmarshal opf: %w", err)
	}

	book := &Book{
		BasePath: path.Dir(opfRelPath),
	}

	if len(pkg.Metadata.Title) > 0 {
		book.Title = pkg.Metadata.Title[0]
	}

	for _, item := range pkg.Manifest.Items {
		resolved, err := resolveBookPath(book.BasePath, item.Href)
		if err != nil {
			logging.Errorf("WARNING: %s\n", i18n.S("Skipped book item %s: %v", item.Href, err))
			continue
		}
		item.Href = relToBase(book.BasePath, resolved)
		book.Manifest = append(book.Manifest, ManifestItem(item))
	}

	book.spineTocID = pkg.Spine.Toc

	for _, ref := range pkg.Spine.ItemRefs {
		book.Spine = append(book.Spine, SpineItem(ref))
	}

	return book, nil
}

// SpineHrefs returns ordered list of content file hrefs based on the spine reading order.
// Paths are relative to the OPF base directory.
func (b *Book) SpineHrefs() []string {
	// Build manifest lookup: id -> href
	idToHref := make(map[string]string, len(b.Manifest))
	for _, item := range b.Manifest {
		idToHref[item.ID] = item.Href
	}

	var hrefs []string
	for _, ref := range b.Spine {
		if href, ok := idToHref[ref.IDRef]; ok {
			hrefs = append(hrefs, href)
		}
	}
	return hrefs
}

// ContentFiles returns all manifest items that are XHTML/HTML content.
func (b *Book) ContentFiles() []ManifestItem {
	var result []ManifestItem
	for _, item := range b.Manifest {
		if isHTMLMediaType(item.MediaType) {
			result = append(result, item)
		}
	}
	return result
}

func isHTMLMediaType(mt string) bool {
	return mt == "application/xhtml+xml" || mt == "text/html"
}
