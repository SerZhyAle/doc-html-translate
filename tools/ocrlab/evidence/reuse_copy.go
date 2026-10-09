package evidence

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"doc-html-translate/tools/ocrlab/exchange"
)

// DiagFile is the diagnostics sidecar both editions write into a run directory.
const DiagFile = "ocr-diag.jsonl"

// Apply copies a reusable scene into the run directory dst - its pages and screenshots, and its
// lines of the diagnostics sidecar - and returns the scene record to put in the new evidence, with
// ReusedFrom set so the copy never passes for a fresh measurement. Both producers call it, which
// is what keeps the provenance field identical. On any failure nothing is left behind and the
// caller collects the scene normally.
func Apply(r *Reusable, dst string) (Scene, error) {
	id := r.Scene.SceneID
	var made []string
	undo := func() {
		for _, p := range made {
			_ = os.RemoveAll(p)
		}
	}
	for _, sub := range []string{PagesDir, ShotsDir} {
		from := filepath.Join(r.Bundle, sub, id)
		if _, err := os.Stat(from); err != nil {
			continue
		}
		to := filepath.Join(dst, sub, id)
		if _, err := os.Stat(to); err == nil {
			undo()
			return Scene{}, fmt.Errorf("%s already exists in the new run", filepath.Join(sub, id))
		}
		made = append(made, to)
		if err := copyTree(from, to); err != nil {
			undo()
			return Scene{}, err
		}
	}
	if err := copySidecarLines(r.Bundle, dst, id); err != nil {
		undo()
		return Scene{}, err
	}
	sc := r.Scene
	from := r.From
	from.ReusedAt = time.Now().UTC().Format(time.RFC3339)
	sc.ReusedFrom = &from
	return sc, nil
}

// copyTree copies the regular files of a directory tree and checks each landed at its full size.
func copyTree(from, to string) error {
	return filepath.WalkDir(from, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(from, path)
		if err != nil {
			return err
		}
		target := filepath.Join(to, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !d.Type().IsRegular() {
			return nil
		}
		return copyFileChecked(path, target)
	})
}

func copyFileChecked(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	n, err := io.Copy(out, in)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	if info, err := in.Stat(); err != nil || info.Size() != n {
		return fmt.Errorf("%s: copied %d bytes, expected the file's size", src, n)
	}
	return nil
}

// copySidecarLines appends the scene's lines of the earlier run's diagnostics sidecar to the new
// run's. A line names the page image it describes; the desktop pages moved into the new run, so
// that name is rewritten to follow them, and every other byte of the line is kept. The extension's
// lines name the corpus file, which has not moved, and are copied as they are.
func copySidecarLines(srcBundle, dst, sceneID string) error {
	raw, err := os.ReadFile(filepath.Join(srcBundle, DiagFile))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	pageDir := filepath.Join(dst, PagesDir, sceneID)
	_, pagesMoved := os.Stat(pageDir)
	var out bytes.Buffer
	for _, line := range bytes.Split(raw, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var head struct {
			File string `json:"file"`
		}
		if json.Unmarshal(line, &head) != nil || head.File == "" || exchange.SceneID(head.File) != sceneID {
			continue
		}
		if pagesMoved == nil {
			oldName, _ := json.Marshal(head.File)
			// Either separator may appear: the line was written on whichever platform ran the scene.
			newName, _ := json.Marshal(filepath.Join(pageDir, path.Base(strings.ReplaceAll(head.File, `\`, "/"))))
			line = bytes.Replace(line, append([]byte(`"file":`), oldName...), append([]byte(`"file":`), newName...), 1)
		}
		out.Write(line)
		out.WriteByte('\n')
	}
	if out.Len() == 0 {
		return nil
	}
	f, err := os.OpenFile(filepath.Join(dst, DiagFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, err = f.Write(out.Bytes())
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}
