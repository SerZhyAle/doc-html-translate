package evidence

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
)

// Folders of a run directory that hold one sub-folder per scene. The desktop runner fills both;
// the extension runner has no pages.
const (
	PagesDir = "pages"
	ShotsDir = "shots"
)

// FilesManifest names the file that records, per scene, every collected file and its size. It is
// written when a run finishes so a later run can tell that the files it would copy are still the
// ones that run produced.
const FilesManifest = "files.json"

// sceneSizes maps a scene id to its files (slash paths relative to the run) and their sizes.
type sceneSizes map[string]map[string]int64

// sceneFiles lists the files a scene owns in a run directory.
func sceneFiles(dir, sceneID string) (map[string]int64, error) {
	files := map[string]int64{}
	for _, sub := range []string{PagesDir, ShotsDir} {
		root := filepath.Join(dir, sub, sceneID)
		if _, err := os.Stat(root); err != nil {
			continue
		}
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.Type().IsRegular() {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			files[filepath.ToSlash(rel)] = info.Size()
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}

// WriteFilesManifest records the size of every file the listed scenes own in the run directory.
func WriteFilesManifest(dir string, sceneIDs []string) error {
	out := sceneSizes{}
	for _, id := range sceneIDs {
		files, err := sceneFiles(dir, id)
		if err != nil {
			return err
		}
		out[id] = files
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, FilesManifest), append(data, '\n'), 0o644)
}

// loadFilesManifest reads a run's manifest. A run that wrote none is (nil, false, nil); one that
// wrote an unreadable manifest is an error, never a silent downgrade to the weaker check.
func loadFilesManifest(dir string) (m sceneSizes, present bool, err error) {
	data, err := os.ReadFile(filepath.Join(dir, FilesManifest))
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, true, err
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, true, err
	}
	return m, true, nil
}
