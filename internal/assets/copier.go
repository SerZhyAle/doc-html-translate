// Package assets copies the local files a source document references (images, fonts,
// stylesheets) into the flat output folder and rewrites the references to the copies.
// It is shared by the extractors whose input is a loose file next to its assets - HTML
// ("Save page as") and Markdown - so both apply one naming and containment policy.
package assets

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"doc-html-translate/internal/htmlgen"
	"doc-html-translate/internal/logging"
	"doc-html-translate/internal/outputpath"
)

// outcome says what became of one reference.
type outcome int

const (
	// kept: leave the reference as written - remote, inline (data:), a bare fragment, or a
	// local file that does not exist (it stays visibly broken rather than vanishing).
	kept outcome = iota
	// copied: the file now sits in the output folder under the returned name.
	copied
	// refused: the reference leaves the source tree (absolute path, file: URL, "../"
	// escape, or a symlink resolving outside). The caller must drop it: left as written it
	// would resolve against the output folder's surroundings instead of the source's.
	refused
)

// pageNamePattern matches the per-page files the extractors write (page_001.html ..).
var pageNamePattern = regexp.MustCompile(`^page_\d+\.html$`)

// Copier copies the assets of one source document. Output names are unique
// case-insensitively, because the output folder usually lives on a Windows or macOS
// filesystem where "A.png" and "a.png" are the same file.
type Copier struct {
	root    string // source tree root, symlinks resolved; nothing outside it is copied
	outDir  string
	byPath  map[string]string // resolved source path -> output name
	infos   []copiedFile      // same-file check for paths that differ only by case or link
	used    map[string]bool   // lower-cased output names taken
	refused map[string]bool   // references already reported, so each is logged once
	files   int
	sheets  int
}

type copiedFile struct {
	info os.FileInfo
	name string
}

// NewCopier returns a Copier for a document whose directory tree is srcDir.
func NewCopier(srcDir, outDir string) (*Copier, error) {
	abs, err := filepath.Abs(srcDir)
	if err != nil {
		return nil, fmt.Errorf("resolve source folder: %w", err)
	}
	root, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("resolve source folder: %w", err)
	}
	return &Copier{
		root:    root,
		outDir:  outDir,
		byPath:  make(map[string]string),
		used:    make(map[string]bool),
		refused: make(map[string]bool),
	}, nil
}

// Root is the resolved source tree root; references in the document itself resolve from it.
func (c *Copier) Root() string { return c.root }

// Files is the number of non-stylesheet files copied (images, fonts ..).
func (c *Copier) Files() int { return c.files }

// Stylesheets is the number of stylesheet files written.
func (c *Copier) Stylesheets() int { return c.sheets }

// place resolves ref (as written in a document or stylesheet living in baseDir) and copies
// the file into the output folder. A stylesheet is rewritten on the way so its own url()
// references are copied too.
func (c *Copier) place(ref, baseDir string, sheet bool) (string, outcome) {
	rel, out := classify(ref)
	if out != copied {
		if out == refused {
			c.noteRefused(ref)
		}
		return "", out
	}
	candidate := filepath.Join(baseDir, filepath.FromSlash(rel))
	if !within(c.root, candidate) {
		c.noteRefused(ref)
		return "", refused
	}
	real, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", kept // dangling: leave it visibly broken
	}
	if !within(c.root, real) {
		c.noteRefused(ref)
		return "", refused
	}
	info, err := os.Stat(real)
	if err != nil || info.IsDir() {
		return "", kept
	}
	if name, ok := c.lookup(real, info); ok {
		return name, copied
	}

	name := c.claim(filepath.Base(candidate))
	// Recorded before the write so a stylesheet that @imports itself (or a cycle of them)
	// resolves to the name instead of recursing.
	c.byPath[real] = name
	idx := len(c.infos)
	c.infos = append(c.infos, copiedFile{info: info, name: name})
	if sheet {
		err = c.writeSheetFile(real, name)
	} else {
		err = copyFile(real, filepath.Join(c.outDir, name))
	}
	if err != nil {
		logging.Errorf("WARNING: could not copy %s: %v\n", ref, err)
		delete(c.byPath, real)
		// By index, not the tail: a stylesheet's own references were appended after it.
		c.infos = append(c.infos[:idx], c.infos[idx+1:]...)
		return "", kept
	}
	if sheet {
		c.sheets++
	} else {
		c.files++
	}
	return name, copied
}

// lookup returns the output name of a file already copied, matching by resolved path
// first and then by identity, so one source file is copied once however it is spelled.
func (c *Copier) lookup(real string, info os.FileInfo) (string, bool) {
	if name, ok := c.byPath[real]; ok {
		return name, true
	}
	for _, f := range c.infos {
		if os.SameFile(f.info, info) {
			return f.name, true
		}
	}
	return "", false
}

// claim reserves a collision-free output name derived from base.
func (c *Copier) claim(base string) string {
	base = sanitizeName(base)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	name := base
	for i := 2; c.used[strings.ToLower(name)] || reserved(name); i++ {
		name = fmt.Sprintf("%s_%d%s", stem, i, ext)
	}
	c.used[strings.ToLower(name)] = true
	return name
}

func (c *Copier) noteRefused(ref string) {
	if c.refused[ref] {
		return
	}
	c.refused[ref] = true
	logging.Printf("  Skipped %s: it points outside the source folder\n", ref)
}

// reserved reports names the generator writes into the output folder itself.
func reserved(name string) bool {
	low := strings.ToLower(name)
	switch low {
	case "index.html", strings.ToLower(htmlgen.FaviconName), strings.ToLower(outputpath.MarkerName):
		return true
	}
	return pageNamePattern.MatchString(low)
}

// classify sorts a reference into kept (not a local file), refused (a local path that is
// absolute by construction) or copied (a relative path to resolve), returning the
// percent-decoded relative path for the last.
func classify(ref string) (string, outcome) {
	s := strings.TrimSpace(ref)
	if s == "" || strings.HasPrefix(s, "#") || strings.HasPrefix(s, "//") {
		return "", kept
	}
	if strings.HasPrefix(s, `\\`) {
		return "", refused // a UNC path: another machine's share
	}
	if scheme := urlScheme(s); scheme != "" {
		// A one-letter "scheme" is a Windows drive ("C:\pics\a.png").
		if scheme == "file" || len(scheme) == 1 {
			return "", refused
		}
		return "", kept
	}
	// Browsers read a backslash in a relative URL as a slash; pages saved on Windows use it.
	s = strings.ReplaceAll(s, `\`, "/")
	if q := strings.IndexAny(s, "?#"); q >= 0 {
		s = s[:q]
	}
	if dec, err := url.PathUnescape(s); err == nil {
		s = dec
	}
	if s == "" {
		return "", kept
	}
	if strings.HasPrefix(s, "/") || filepath.IsAbs(s) || filepath.VolumeName(s) != "" {
		return "", refused
	}
	return s, copied
}

// isRemote reports a reference fetched over the network.
func isRemote(ref string) bool {
	s := strings.TrimSpace(ref)
	if strings.HasPrefix(s, "//") {
		return true
	}
	scheme := urlScheme(s)
	return len(scheme) > 1 && scheme != "data" && scheme != "file"
}

// urlScheme returns the lower-cased scheme of an absolute URL, or "".
func urlScheme(s string) string {
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case ch >= 'a' && ch <= 'z', ch >= 'A' && ch <= 'Z':
		case i > 0 && (ch >= '0' && ch <= '9' || ch == '+' || ch == '-' || ch == '.'):
		case ch == ':' && i > 0:
			return strings.ToLower(s[:i])
		default:
			return ""
		}
	}
	return ""
}

// within reports whether p lies inside root (or is root).
func within(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	if err != nil || filepath.IsAbs(rel) {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// sanitizeName keeps a basename safe for the flat output directory.
func sanitizeName(base string) string {
	var sb strings.Builder
	for _, r := range base {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			sb.WriteRune(r)
		default:
			sb.WriteByte('_')
		}
	}
	name := sb.String()
	if strings.Trim(name, ".") == "" {
		return "file"
	}
	return name
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
