package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ProducerEntries are the source files and folders a collection depends on: the shipped converters
// and OCR code both editions run, the extension, and the runners and evidence code that drive them.
// The scorer packages (tools/ocrlab/metrics, truth, report) are deliberately absent - they only
// read evidence, so a scoring change must not discard a browser run.
//
// The list is conservative on purpose: an unrelated edit under one of these folders changes the
// digest and forces a fresh collection, because a missed dependency would reuse a stale result.
var ProducerEntries = []string{
	"internal",
	"cmd/doc-html-translate",
	"extension/src",
	"extension/scripts",
	"tools/ocrlab/runner",
	"tools/ocrlab/evidence",
	"tools/ocrlab/synth",
	"go.mod",
	"go.sum",
	"extension/package.json",
	"extension/package-lock.json",
}

// ProducerDigest hashes every non-test byte of ProducerEntries under root.
func ProducerDigest(root string) (string, error) { return ProducerDigestOf(root, ProducerEntries) }

// ProducerDigestOf is ProducerDigest over an explicit entry list. Every file counts whatever its
// extension (embedded assets and bundled helpers are product bytes), except test files and
// testdata folders, which no collection executes.
func ProducerDigestOf(root string, entries []string) (string, error) {
	seen := map[string]bool{}
	var paths []string
	add := func(p string) {
		if !seen[p] {
			seen[p] = true
			paths = append(paths, p)
		}
	}
	for _, entry := range entries {
		full := filepath.Join(root, filepath.FromSlash(entry))
		info, err := os.Stat(full)
		if err != nil {
			return "", err
		}
		if !info.IsDir() {
			add(full)
			continue
		}
		err = filepath.WalkDir(full, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "testdata" || d.Name() == "node_modules" {
					return filepath.SkipDir
				}
				return nil
			}
			if d.Type().IsRegular() && !isTestFile(d.Name()) {
				add(path)
			}
			return nil
		})
		if err != nil {
			return "", err
		}
	}
	sort.Strings(paths)
	h := sha256.New()
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path
		}
		fmt.Fprintf(h, "%s\x00%s\x00", filepath.ToSlash(rel), Digest(data))
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func isTestFile(name string) bool {
	return strings.HasSuffix(name, "_test.go") || strings.HasSuffix(name, ".test.mjs") || strings.HasSuffix(name, ".test.js")
}

// ScorerDigest is the identity `ocrlab score` stamps on a summary: the scoring packages plus the
// module files. Reuse compares it with the summary of an earlier run to know whether that run's
// per-scene verdict was produced by the scorer in force now.
func ScorerDigest(root string) (string, error) {
	return TreeDigest(root, "tools/ocrlab/metrics", "tools/ocrlab/truth", "tools/ocrlab/report", "go.mod", "go.sum")
}
