package ocr

import (
	"encoding/json"
	"image"
	"os"
	"sync"
)

// diagEnvVar names the file the overlay writes its per-image diagnostics to. Unset - the normal
// case, and the only case a user ever sees - means the whole feature is off.
//
// This exists so the visual-fidelity lab (tools/ocrlab) can read the plate geometry the app
// actually produced instead of re-implementing percentStyle and blockColors and then measuring
// its own reimplementation. The strategic spec's edition table asks for exactly this: enough
// diagnostic output for the corpus runner, without changing normal user output. An
// output-identity test in diag_test.go is what keeps the second half of that promise true.
const diagEnvVar = "DOCHT_OCR_DIAG"

// DiagnosticsPath returns the diagnostics file path, or "" when diagnostics are off. Read on
// every call rather than cached at init, so a test can turn it on and off within one process.
func DiagnosticsPath() string { return os.Getenv(diagEnvVar) }

// diagBlock is one recognized block as it was rendered.
type diagBlock struct {
	Text       string `json:"text"`
	X0         int    `json:"x0"`
	Y0         int    `json:"y0"`
	X1         int    `json:"x1"`
	Y1         int    `json:"y1"`
	LineH      int    `json:"lineH"`
	Style      string `json:"style"`
	Background string `json:"background,omitempty"`
	Ink        string `json:"ink,omitempty"`
}

// diagDropped is one line the confidence floor rejected: text the engine read that the reader
// never gets. Recorded because the floor is otherwise an invisible decision - a scene where a
// correctly read word was discarded looks exactly like a scene where nothing was recognized, and
// the lab cannot score a decision it cannot see.
type diagDropped struct {
	Text  string  `json:"text"`
	Conf  float64 `json:"conf"`
	Floor float64 `json:"floor"`
	X0    int     `json:"x0"`
	Y0    int     `json:"y0"`
	X1    int     `json:"x1"`
	Y1    int     `json:"y1"`
}

// diagImage is one overlaid image's record, written as a single JSON line.
type diagImage struct {
	File    string        `json:"file"`
	Width   int           `json:"width"`
	Height  int           `json:"height"`
	Blocks  []diagBlock   `json:"blocks"`
	Dropped []diagDropped `json:"dropped,omitempty"`
}

// diagMu serializes appends. Phase 3 of OverlayBook is sequential today, but a diagnostics
// writer that corrupts its own file the day that changes would be a nasty thing to debug.
var diagMu sync.Mutex

// recordDiagnostics appends one line describing what wrapImage just drew. A no-op when
// diagnostics are off, and best-effort when they are on: a failure to write a developer's debug
// file must never affect a conversion.
//
// The style and colours are recomputed here with the same pure functions wrapImage used, rather
// than threaded back out of it. That keeps the rendering path byte-identical whether or not
// diagnostics are on - which is the property the lab depends on and the test asserts.
func recordDiagnostics(file string, res Result, srcImg image.Image) {
	path := DiagnosticsPath()
	if path == "" {
		return
	}
	rec := diagImage{File: file, Width: res.Width, Height: res.Height}
	for _, b := range res.Blocks {
		db := diagBlock{
			Text: b.Text, X0: b.X0, Y0: b.Y0, X1: b.X1, Y1: b.Y1, LineH: b.LineH,
			Style: percentStyle(b, res.Width, res.Height),
		}
		if srcImg != nil {
			if bg, ink, ok := blockColors(srcImg, b); ok {
				db.Background, db.Ink = bg, ink
			}
		}
		rec.Blocks = append(rec.Blocks, db)
	}
	// A conversion rather than a field-by-field copy: the two structs are the same record in two
	// vocabularies, and a field added to DroppedLine and forgotten here becomes a compile error
	// instead of a value silently missing from the diagnostics.
	for _, d := range res.Dropped {
		rec.Dropped = append(rec.Dropped, diagDropped(d))
	}
	line, err := json.Marshal(rec)
	if err != nil {
		return
	}

	diagMu.Lock()
	defer diagMu.Unlock()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(line, '\n'))
}
