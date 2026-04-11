// Package epub handles EPUB archive extraction and metadata parsing.
// Supports both EPUB2 (.ncx navigation) and EPUB3 (nav.xhtml).
package epub

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Book represents a parsed EPUB structure.
type Book struct {
	Title    string
	Manifest []ManifestItem
	Spine    []SpineItem
	BasePath string // directory within EPUB where content.opf resides
}

// ManifestItem represents a single item in the OPF manifest.
type ManifestItem struct {
	ID        string
	Href      string
	MediaType string
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
	ID        string `xml:"id,attr"`
	Href      string `xml:"href,attr"`
	MediaType string `xml:"media-type,attr"`
}

type opfSpine struct {
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
	for _, f := range r.File {
		if err := extractFile(f, outputDir); err != nil {
			// best-effort: warn and continue
			fmt.Fprintf(os.Stderr, "WARNING: skip %s: %v\n", f.Name, err)
		}
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

	// Browser translators (notably Chrome Translate) often fail on local XHTML/XML
	// documents. Prepare HTML copies and update hrefs for better compatibility.
	if err := normalizeXHTMLToHTML(book, outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: XHTML->HTML normalization skipped: %v\n", err)
	}

	// Rename any content file whose resolved path conflicts with the generated
	// nav file (index.html). Must run after normalizeXHTMLToHTML so that
	// index.xhtml → index.html renames are already reflected in the manifest.
	if err := resolveReservedNames(book, outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: reserved-name conflict resolution skipped: %v\n", err)
	}

	return book, nil
}

// resolveReservedNames renames content files whose disk path would conflict
// with the nav file generated later (always outputDir/index.html).
// Updates manifest hrefs and cross-references within all HTML content files.
func resolveReservedNames(book *Book, outputDir string) error {
	navPath := filepath.Clean(filepath.Join(outputDir, "index.html"))
	hrefMap := make(map[string]string)

	for i := range book.Manifest {
		item := &book.Manifest[i]
		if !isHTMLMediaType(item.MediaType) {
			continue
		}
		contentPath := filepath.Clean(bookPath(outputDir, book.BasePath, item.Href))
		if contentPath != navPath {
			continue
		}
		// Conflict: rename to _content_<original> (e.g. _content_index.html)
		newHref := "_content_" + item.Href
		newPath := bookPath(outputDir, book.BasePath, newHref)
		if err := os.Rename(contentPath, newPath); err != nil {
			return fmt.Errorf("rename conflicting %s: %w", item.Href, err)
		}
		hrefMap[item.Href] = newHref
		item.Href = newHref
	}

	if len(hrefMap) == 0 {
		return nil
	}

	// Update href references inside all HTML content files.
	for _, item := range book.Manifest {
		if !isHTMLMediaType(item.MediaType) {
			continue
		}
		path := bookPath(outputDir, book.BasePath, item.Href)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		content := string(data)
		changed := false
		for old, newHref := range hrefMap {
			if strings.Contains(content, old) {
				content = strings.ReplaceAll(content, old, newHref)
				changed = true
			}
		}
		if changed {
			_ = os.WriteFile(path, []byte(content), 0o644)
		}
	}

	return nil
}

// normalizeXHTMLToHTML creates .html copies for XHTML content files and rewrites
// manifest hrefs plus intra-book links to point to the new .html files.
func normalizeXHTMLToHTML(book *Book, outputDir string) error {
	hrefMap := make(map[string]string)

	for i := range book.Manifest {
		item := &book.Manifest[i]
		if !isHTMLMediaType(item.MediaType) {
			continue
		}

		oldHref := item.Href
		newHref := toHTMLExt(oldHref)
		if newHref == oldHref {
			continue
		}

		srcPath := bookPath(outputDir, book.BasePath, oldHref)
		dstPath := bookPath(outputDir, book.BasePath, newHref)

		data, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("read xhtml %s: %w", oldHref, err)
		}

		content := string(data)
		// Make links and references prefer HTML targets.
		content = strings.ReplaceAll(content, ".xhtml", ".html")
		content = strings.ReplaceAll(content, ".xhtm", ".html")

		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			return fmt.Errorf("mkdir for %s: %w", newHref, err)
		}
		if err := os.WriteFile(dstPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write html %s: %w", newHref, err)
		}

		hrefMap[oldHref] = newHref
		item.Href = newHref
		item.MediaType = "text/html"
	}

	if len(hrefMap) == 0 {
		return nil
	}

	// Second pass: rewrite cross-links in all HTML content files using exact href map.
	for _, item := range book.Manifest {
		if !isHTMLMediaType(item.MediaType) {
			continue
		}
		path := bookPath(outputDir, book.BasePath, item.Href)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		content := string(data)
		for oldHref, newHref := range hrefMap {
			content = strings.ReplaceAll(content, oldHref, newHref)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			continue
		}
	}

	return nil
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
			return rf.FullPath, nil
		}
	}

	return "", fmt.Errorf("no rootfile found in container.xml")
}

// parseOPF reads content.opf and returns a populated Book.
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
		BasePath: filepath.Dir(opfRelPath),
	}

	if len(pkg.Metadata.Title) > 0 {
		book.Title = pkg.Metadata.Title[0]
	}

	for _, item := range pkg.Manifest.Items {
		book.Manifest = append(book.Manifest, ManifestItem{
			ID:        item.ID,
			Href:      item.Href,
			MediaType: item.MediaType,
		})
	}

	for _, ref := range pkg.Spine.ItemRefs {
		book.Spine = append(book.Spine, SpineItem{
			IDRef: ref.IDRef,
		})
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
