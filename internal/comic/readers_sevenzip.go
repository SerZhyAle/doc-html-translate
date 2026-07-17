package comic

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	"doc-html-translate/internal/logging"
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

// readSevenZip extracts a CBR (RAR) or CB7 (7z) archive with the 7-Zip CLI and
// returns its page entries. RAR and 7z have no pure-Go decoder, so this is the one
// comic path with a runtime dependency; when 7-Zip is absent it returns an
// actionable "install 7-Zip" notice (never a crash or a garbage conversion), the
// same contract MOBI keeps for Calibre.
func readSevenZip(path, ext string) ([]page, error) {
	bin := find7Zip()
	if bin == "" {
		kind := "CBR"
		if ext == ".cb7" {
			kind = "CB7"
		}
		return nil, fmt.Errorf(
			"7-Zip not found - install 7-Zip from https://www.7-zip.org to open %s (.%s) comics;\n"+
				"CBZ and CBT comics work without it. After installing, re-open the file.",
			kind, ext[1:],
		)
	}

	tmpDir, err := os.MkdirTemp("", "doc-html-translate-comic-")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	logging.Printf("  Extracting comic via 7-Zip (%s)..\n", bin)
	// x = extract with full paths; -y assume yes; -bd no progress; -o output dir.
	cmd := exec.Command(bin, "x", path, "-y", "-bd", "-o"+tmpDir)
	if out, cmdErr := cmd.CombinedOutput(); cmdErr != nil {
		msg := string(out)
		if len(msg) > 500 {
			msg = msg[:500] + ".."
		}
		return nil, fmt.Errorf("7-Zip failed to extract comic (archive may be encrypted or corrupt): %w\n%s", cmdErr, msg)
	}

	var c collector
	walkErr := filepath.WalkDir(tmpDir, func(p string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		if d.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(tmpDir, p)
		if rerr != nil {
			rel = filepath.Base(p)
		}
		if !isPageEntry(rel) {
			return nil
		}
		info, ierr := d.Info()
		if ierr == nil && info.Size() > maxPageBytes {
			logging.Printf("  WARNING: comic page %s is larger than %d bytes, skipped\n", rel, maxPageBytes)
			return nil
		}
		data, derr := os.ReadFile(p)
		if derr != nil {
			logging.Printf("  WARNING: comic page %s could not be read, skipped: %v\n", rel, derr)
			return nil
		}
		if int64(len(data)) > maxPageBytes {
			logging.Printf("  WARNING: comic page %s exceeded %d bytes, skipped\n", rel, maxPageBytes)
			return nil
		}
		return c.add(filepath.ToSlash(rel), data)
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return c.pages, nil
}
