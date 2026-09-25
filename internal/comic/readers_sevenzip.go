package comic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"doc-html-translate/internal/limits"
	"doc-html-translate/internal/logging"
	"doc-html-translate/internal/procrun"
)

// find7Zip locates the 7-Zip binary. It follows the MOBI/Calibre precedent
// (internal/mobi.findEbookConvert) of PATH *plus* a probe of known install
// paths - and here the probe is not redundant belt-and-suspenders: measured on
// the dev machine, 7-Zip's installer registers itself in neither the machine nor
// the user Path, so a bare LookPath fails on a machine that has 7-Zip installed.
// "7z" is the full CLI (handles RAR and 7z); "7za"/"7zr" are reduced builds kept
// only as a last resort.
func find7Zip() string {
	for _, name := range []string{"7z", "7za", "7zr"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	for _, p := range []string{
		`C:\Program Files\7-Zip\7z.exe`,
		`C:\Program Files (x86)\7-Zip\7z.exe`,
		`/usr/bin/7z`,
		`/usr/local/bin/7z`,
		`/opt/homebrew/bin/7z`,
		`/usr/bin/7za`,
		`/usr/local/bin/7za`,
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// sevenZipItem is one entry of a `7z l -slt` listing.
type sevenZipItem struct {
	path      string
	size      int64
	sizeKnown bool
	dir       bool
	link      bool
}

// openSevenZip lists a CBR (RAR) or CB7 (7z) archive with the 7-Zip CLI. Nothing
// is unpacked until the listing has passed the budget: fetch then unpacks only
// the chosen pages into a private temp directory. RAR and 7z have no pure-Go
// decoder, so this is the one comic path with a runtime dependency; when 7-Zip is
// absent it returns an actionable "install 7-Zip" notice (never a crash or a
// garbage conversion), the same contract MOBI keeps for Calibre.
func openSevenZip(path, ext, kind string) (*archive, error) {
	bin := find7Zip()
	if bin == "" {
		return nil, sevenZipMissing(path, ext, kind)
	}

	res, err := procrun.Run(context.Background(), procrun.Cmd{
		Tool:      "7-Zip",
		Path:      bin,
		Args:      sevenZipArgs("l", "-slt", path),
		Timeout:   procrun.SevenZip.ForFile(path),
		MaxStdout: sevenZipMaxListing,
		MaxStderr: 500,
	})
	if err != nil {
		if errors.Is(err, procrun.ErrTimeout) {
			return nil, err
		}
		return nil, fmt.Errorf("7-Zip failed to list comic (archive may be encrypted or corrupt): %w", err)
	}
	if res.StdoutTruncated {
		return nil, limits.UncheckableListing("")
	}

	items := parseSevenZipListing(string(res.Stdout))
	arc := &archive{count: len(items), close: func() {}}
	for _, it := range items {
		if it.dir || it.link {
			continue
		}
		if !it.sizeKnown && isPageEntry(it.path) {
			return nil, limits.UncheckableListing(it.path)
		}
		arc.entries = append(arc.entries, entry{name: it.path, size: it.size})
	}
	arc.fetch = func(pages []entry) ([]entry, error) {
		dir, err := extractSevenZip(bin, path, pages)
		if err != nil {
			return nil, err
		}
		arc.close = func() { _ = os.RemoveAll(dir) }
		return openExtracted(dir, pages), nil
	}
	return arc, nil
}

// sevenZipMaxListing bounds the captured listing. At about 250 bytes an entry it
// holds several times the entry budget; a listing past it cannot be checked, so
// the archive is refused rather than unpacked blind.
const sevenZipMaxListing = 16 << 20

// sevenZipArgs builds one command line. -bd drops the progress meter; on Windows
// -sccUTF-8 makes the listing UTF-8, so the names in it match the UTF-8 list file
// fetch writes (the console code page would mangle a Cyrillic page name).
// p7zip does not know -scc, and its output is UTF-8 already.
func sevenZipArgs(cmd string, rest ...string) []string {
	args := []string{cmd, "-bd"}
	if runtime.GOOS == "windows" {
		args = append(args, "-sccUTF-8")
	}
	return append(args, rest...)
}

// parseSevenZipListing reads the technical listing (`7z l -slt`). The archive's own
// properties come first and end at a "----------" line; after it every entry is a
// block of "Key = Value" lines separated by a blank line.
func parseSevenZipListing(out string) []sevenZipItem {
	var items []sevenZipItem
	var cur *sevenZipItem
	started := false
	for _, line := range strings.Split(strings.ReplaceAll(out, "\r\n", "\n"), "\n") {
		if !started {
			started = strings.TrimSpace(line) == "----------"
			continue
		}
		key, val, ok := strings.Cut(line, " = ")
		if !ok {
			key, ok = strings.CutSuffix(line, " =")
			if !ok {
				continue
			}
		}
		switch key {
		case "Path":
			items = append(items, sevenZipItem{path: val})
			cur = &items[len(items)-1]
		case "Size":
			if cur != nil {
				if n, err := strconv.ParseInt(val, 10, 64); err == nil && n >= 0 {
					cur.size, cur.sizeKnown = n, true
				}
			}
		case "Folder":
			if cur != nil && val == "+" {
				cur.dir = true
			}
		case "Attributes":
			if cur != nil {
				cur.dir = cur.dir || strings.HasPrefix(val, "D")
				cur.link = cur.link || isLinkMode(val)
			}
		case "Symbolic Link", "Hard Link", "Link":
			if cur != nil && val != "" {
				cur.link = true
			}
		}
	}
	return items
}

// isLinkMode reports the unix mode 7-Zip appends to Attributes ("A_ lrwxrwxrwx").
func isLinkMode(attrs string) bool {
	for _, f := range strings.Fields(attrs) {
		if len(f) == 10 && f[0] == 'l' {
			return true
		}
	}
	return false
}

// extractSevenZip unpacks exactly the chosen pages into a new temp directory, named
// through a UTF-8 list file with wildcard matching off (-spd), so a page called
// "*.jpg" cannot pull in the rest of the archive. The disk it can take is the
// listing's checked total.
func extractSevenZip(bin, path string, pages []entry) (string, error) {
	list, err := os.CreateTemp("", "doc-html-translate-comic-*.txt")
	if err != nil {
		return "", fmt.Errorf("create list file: %w", err)
	}
	defer func() { _ = os.Remove(list.Name()) }()
	for _, pg := range pages {
		if _, err := fmt.Fprintln(list, pg.name); err != nil {
			_ = list.Close()
			return "", fmt.Errorf("write list file: %w", err)
		}
	}
	if err := list.Close(); err != nil {
		return "", fmt.Errorf("write list file: %w", err)
	}

	dir, err := os.MkdirTemp("", "doc-html-translate-comic-")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	logging.Printf("  Extracting comic via 7-Zip (%s)..\n", bin)
	if _, err := procrun.Run(context.Background(), procrun.Cmd{
		Tool:      "7-Zip",
		Path:      bin,
		Args:      sevenZipArgs("x", path, "-y", "-spd", "-scsUTF-8", "-o"+dir, "@"+list.Name()),
		Timeout:   procrun.SevenZip.ForFile(path),
		MaxStdout: 64 << 10,
		MaxStderr: 500,
	}); err != nil {
		_ = os.RemoveAll(dir)
		if errors.Is(err, procrun.ErrTimeout) {
			return "", err
		}
		return "", fmt.Errorf("7-Zip failed to extract comic (archive may be encrypted or corrupt): %w", err)
	}
	return dir, nil
}

// openExtracted points each page at its unpacked file. Lstat, not Stat: a link that
// 7-Zip recreated is skipped rather than followed out of the temp directory, and a
// name that resolves outside it is skipped too.
func openExtracted(dir string, pages []entry) []entry {
	var out []entry
	for _, pg := range pages {
		p := filepath.Join(dir, filepath.FromSlash(pg.name))
		if rel, err := filepath.Rel(dir, p); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			logging.Printf("  WARNING: comic page %s is outside the archive, skipped\n", pg.name)
			continue
		}
		info, err := os.Lstat(p)
		if err != nil || !info.Mode().IsRegular() {
			logging.Printf("  WARNING: comic page %s was not unpacked as a regular file, skipped\n", pg.name)
			continue
		}
		pg.open = func() (io.ReadCloser, error) { return os.Open(p) }
		out = append(out, pg)
	}
	return out
}

// sevenZipMissing is the "install 7-Zip" notice. When the extension promised a
// container 7-Zip is not needed for, it says what the file really is, so the notice
// does not read as nonsense for a .cbz.
func sevenZipMissing(path, ext, kind string) error {
	name, want := "CBR", "cbr"
	if kind == container7z {
		name, want = "CB7", "cb7"
	}
	msg := fmt.Sprintf(
		"7-Zip not found - install 7-Zip from https://www.7-zip.org to open %s (.%s) comics;\n"+
			"CBZ and CBT comics work without it. After installing, re-open the file.",
		name, want,
	)
	if extKind[ext] != kind {
		msg = fmt.Sprintf("%s is a %s archive with a %s extension. %s", filepath.Base(path), kind, ext, msg)
	}
	return errors.New(msg)
}
