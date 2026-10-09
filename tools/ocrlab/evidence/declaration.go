package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/truth"
)

// Procedure changes whenever a measurement changes meaning. Historical scores are not migrated.
//
// ocrlab-110-v1: residual lettering is judged on the text-hidden capture and is unmeasured where
// the background is not a single tone and no lettering mask is declared; "readability" is now the
// glyphs actually drawn and the plate-rectangle luma separation is BackgroundContrast; hard
// protected-area damage counts painted pixels, with the rectangle geometry kept as
// RectangleIntrusion; each scene is read with its own declared language unless the operator forces
// one (SceneLang), and a scene whose language data is missing is unmeasured, not scored.
const Procedure = "ocrlab-110-v1"

// Declaration is a versioned local lab envelope, separate from OCR-EXCHANGE records.
// Snapshots are written before conversion; missing human approval stays missing.
type Declaration struct {
	Version        int        `json:"version"`
	Purpose        string     `json:"purpose"`
	SceneIDs       []string   `json:"sceneIds"`
	StressCases    []string   `json:"stressCases"`
	Viewports      []Viewport `json:"viewports"`
	Procedure      string     `json:"procedure"`
	SourceRevision string     `json:"sourceRevision"`
	SourceDigest   string     `json:"sourceDigest"`
	// ProducerDigest is the digest of the source bytes a collection depends on (ProducerEntries).
	// A declaration without one predates scene reuse and its evidence is never reused.
	ProducerDigest  string            `json:"producerDigest,omitempty"`
	ScorerDigest    string            `json:"scorerDigest"`
	ThresholdDigest string            `json:"thresholdDigest"`
	Environment     string            `json:"environment"`
	Settings        map[string]string `json:"settings"`
	Scenes          map[string]Input  `json:"scenes"`
	FamilyDigest    string            `json:"familyDigest,omitempty"`
}

type Input struct {
	MediaSHA256      string `json:"mediaSha256"`
	AnnotationSHA256 string `json:"annotationSha256"`
	SceneSHA256      string `json:"sceneSha256"`
	// LetteringSHA256 identifies the declared lettering mask, when the scene has one.
	LetteringSHA256 string `json:"letteringSha256,omitempty"`
	// Lang is the OCR language the scene is declared to be read with (SceneLang), so a run that
	// read it with another one is caught by Issues rather than trusted.
	Lang string `json:"lang,omitempty"`
}

func Digest(data []byte) string { h := sha256.Sum256(data); return hex.EncodeToString(h[:]) }

// TreeDigest hashes actual source bytes, including local changes and untracked source files.
func TreeDigest(root string, dirs ...string) (string, error) {
	var paths []string
	for _, dir := range dirs {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".mjs") || strings.HasSuffix(path, ".css") || strings.HasSuffix(path, ".json") || strings.HasSuffix(path, ".html") || filepath.Base(path) == "go.mod" || filepath.Base(path) == "go.sum" {
				paths = append(paths, path)
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
		rel, _ := filepath.Rel(root, path)
		fmt.Fprintf(h, "%s\x00%s\x00", filepath.ToSlash(rel), Digest(data))
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Freeze writes the declaration. lang is the operator's explicit -lang, or "" when each scene reads
// with its own declared language.
func Freeze(dir, purpose, root, annotations, lang string, scenes []*corpus.Scene, viewports []Viewport, stresses []string, family *corpus.Manifest) (*Declaration, error) {
	if purpose != "exploratory" && purpose != "selected-dev" && purpose != "full-benchmark" {
		return nil, fmt.Errorf("unknown purpose %q", purpose)
	}
	if len(scenes) == 0 {
		return nil, fmt.Errorf("cannot declare an empty selection")
	}
	seen := map[string]bool{}
	for _, s := range scenes {
		if s.ID == "" || s.ID == "." || s.ID == ".." || strings.ContainsAny(s.ID, "/\\:") || seen[s.ID] {
			return nil, fmt.Errorf("unsafe or duplicate scene id %q", s.ID)
		}
		seen[s.ID] = true
	}
	if _, err := os.Stat(filepath.Join(dir, "declaration.json")); err == nil {
		return nil, fmt.Errorf("run directory already declared; choose a new directory")
	}
	settingsLang := lang
	if settingsLang == "" {
		settingsLang = LangPerScene
	}
	d := &Declaration{Version: 1, Purpose: purpose, Procedure: Procedure, Viewports: viewports, StressCases: stresses, Environment: fmt.Sprintf("%s/%s; %s; CPUs=%d; GOMAXPROCS=%d", runtime.GOOS, runtime.GOARCH, runtime.Version(), runtime.NumCPU(), runtime.GOMAXPROCS(0)), Settings: map[string]string{"lang": settingsLang}, Scenes: map[string]Input{}}
	rev, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err == nil {
		d.SourceRevision = strings.TrimSpace(string(rev))
	}
	d.SourceDigest, err = TreeDigest(".", "internal", "tools/ocrlab", "extension/src", "extension/scripts", "go.mod", "go.sum", "extension/package.json", "extension/package-lock.json")
	if err != nil {
		return nil, err
	}
	d.ProducerDigest, err = ProducerDigest(".")
	if err != nil {
		return nil, err
	}
	d.ScorerDigest, err = TreeDigest(".", "tools/ocrlab/metrics", "tools/ocrlab/truth", "tools/ocrlab/report")
	if err != nil {
		return nil, err
	}
	if data, err := os.ReadFile("DEV/ocrlab/thresholds.json"); err == nil {
		d.ThresholdDigest = Digest(data)
	}
	if err := os.MkdirAll(filepath.Join(dir, "truth"), 0755); err != nil {
		return nil, err
	}
	if family == nil {
		return nil, fmt.Errorf("full source-family manifest required")
	}
	if err := corpus.Save(filepath.Join(dir, "family-corpus.json"), family); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "family-corpus.json"))
	if err != nil {
		return nil, err
	}
	d.FamilyDigest = Digest(data)
	manifest := corpus.Manifest{SchemaVersion: corpus.SchemaVersion}
	for _, s := range scenes {
		d.SceneIDs = append(d.SceneIDs, s.ID)
		manifest.Scenes = append(manifest.Scenes, *s)
		data, _ := json.Marshal(s)
		input := Input{SceneSHA256: Digest(data), Lang: SceneLang(annotations, lang, s).Lang}
		input.MediaSHA256, err = corpus.HashFile(s.Path(root))
		if err != nil {
			return nil, err
		}
		if data, err := os.ReadFile(truth.FinalPath(annotations, s.ID)); err == nil {
			input.AnnotationSHA256 = Digest(data)
			if err := os.WriteFile(truth.FinalPath(filepath.Join(dir, "truth"), s.ID), data, 0644); err != nil {
				return nil, err
			}
		}
		if data, err := os.ReadFile(truth.LetteringPath(annotations, s.ID)); err == nil {
			input.LetteringSHA256 = Digest(data)
			if err := os.WriteFile(truth.LetteringPath(filepath.Join(dir, "truth"), s.ID), data, 0644); err != nil {
				return nil, err
			}
		}
		d.Scenes[s.ID] = input
	}
	sort.Strings(d.SceneIDs)
	if err := corpus.Save(filepath.Join(dir, "corpus.json"), &manifest); err != nil {
		return nil, err
	}
	data, err = json.MarshalIndent(d, "", "  ")
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(filepath.Join(dir, "declaration.json"), append(data, '\n'), 0644)
	return d, err
}

func LoadDeclaration(dir string) (*Declaration, error) {
	data, err := os.ReadFile(filepath.Join(dir, "declaration.json"))
	if err != nil {
		return nil, err
	}
	var d Declaration
	if err = json.Unmarshal(data, &d); err != nil {
		return nil, err
	}
	if d.Version != 1 {
		return nil, fmt.Errorf("unsupported declaration version %d", d.Version)
	}
	return &d, nil
}

// Finish records the tree at the end of execution; edits while a browser runs void acceptance.
func Finish(dir string) error {
	d, err := LoadDeclaration(dir)
	if err != nil {
		return err
	}
	digest, err := TreeDigest(".", "internal", "tools/ocrlab", "extension/src", "extension/scripts", "go.mod", "go.sum", "extension/package.json", "extension/package-lock.json")
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(struct {
		SourceDigest string `json:"sourceDigest"`
		Unchanged    bool   `json:"unchanged"`
	}{digest, digest == d.SourceDigest}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "execution.json"), append(data, '\n'), 0644); err != nil {
		return err
	}
	return WriteFilesManifest(dir, d.SceneIDs)
}
