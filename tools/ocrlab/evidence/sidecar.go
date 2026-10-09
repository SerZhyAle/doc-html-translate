package evidence

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"doc-html-translate/tools/ocrlab/exchange"
)

// SidecarBlock is a recognized block as the diagnostics sidecar records it: the text the pipeline
// kept and drew a plate for.
type SidecarBlock struct {
	Text string  `json:"text"`
	Conf float64 `json:"conf"`
}

// SidecarDropped is a line a gate rejected: text the engine read that the reader never gets. Gate
// names the test it failed (OCR-OVERLAY rule 12) and Floor the confidence floor of the pass.
type SidecarDropped struct {
	Text  string  `json:"text"`
	Conf  float64 `json:"conf"`
	Floor float64 `json:"floor"`
	Gate  string  `json:"gate"`
}

// SidecarRecord is one image's line of the diagnostics sidecar, written identically by both
// editions (internal/ocr diagImage, extension makeDiagRecord). Blocks and Dropped are always
// arrays on the writing side, so "the engine read nothing" and "everything was thrown away" stay
// distinguishable here.
type SidecarRecord struct {
	File    string           `json:"file"`
	Blocks  []SidecarBlock   `json:"blocks"`
	Dropped []SidecarDropped `json:"dropped"`
}

// LoadSidecar reads a run's diagnostics sidecar, keyed by scene id. A scene with more than one line
// (a re-run appended to the same file) keeps the last. A missing sidecar is an empty result, not an
// error: it only exists when the producer was asked to write it, and a scene without a record is
// reported as having an unknown loss point rather than refused.
func LoadSidecar(path string) (map[string]*SidecarRecord, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]*SidecarRecord{}, nil
		}
		return nil, err
	}
	out := map[string]*SidecarRecord{}
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 0, 1<<20), 64<<20)
	for n := 1; sc.Scan(); n++ {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var rec SidecarRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			return nil, fmt.Errorf("%s line %d: %w", path, n, err)
		}
		if rec.File == "" {
			return nil, fmt.Errorf("%s line %d: names no file", path, n)
		}
		out[exchange.SceneID(rec.File)] = &rec
	}
	return out, sc.Err()
}
