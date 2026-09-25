package procrun

import (
	"math"
	"os"
	"strconv"
	"sync"
	"time"

	"doc-html-translate/internal/logging"
)

// ScaleEnv names the environment variable that multiplies every helper deadline, for a slow
// machine or an unusually heavy book. It is a float ("2", "0.5"); anything else is ignored.
const ScaleEnv = "DOCHT_TOOL_TIMEOUT_SCALE"

// Budget is one tool's deadline policy: a base, plus an allowance per megabyte of input,
// clamped to a ceiling. The scale from ScaleEnv applies after the clamp, so it can also lift
// the ceiling when a legitimate input needs more.
type Budget struct {
	Base  time.Duration
	PerMB time.Duration
	Max   time.Duration
}

// Per-tool policies (ticket bugfix-external-process-bounds, section 6.1). The per-megabyte
// allowances are generous on purpose: a deadline exists to end a hang, not to race a slow disk.
var (
	PDFToText = Budget{Base: 2 * time.Minute, PerMB: time.Minute / 50, Max: 30 * time.Minute}
	// Tesseract runs once per image, so its budget is per image.
	Tesseract = Budget{Base: 2 * time.Minute, PerMB: 10 * time.Second, Max: 10 * time.Minute}
	// TesseractProbe covers the short calls: --list-langs and the script-detection pass.
	TesseractProbe = Budget{Base: 30 * time.Second, PerMB: 10 * time.Second, Max: 5 * time.Minute}
	Calibre        = Budget{Base: 10 * time.Minute, PerMB: 30 * time.Second, Max: 60 * time.Minute}
	SevenZip       = Budget{Base: 2 * time.Minute, PerMB: 6 * time.Second, Max: 30 * time.Minute}
	ImageConvert   = Budget{Base: 2 * time.Minute, Max: 2 * time.Minute}
)

// For returns the deadline for an input of inputBytes.
func (b Budget) For(inputBytes int64) time.Duration {
	d := b.Base
	if inputBytes > 0 && b.PerMB > 0 {
		d += time.Duration(float64(b.PerMB) * float64(inputBytes) / float64(1<<20))
	}
	if b.Max > 0 && d > b.Max {
		d = b.Max
	}
	return time.Duration(float64(d) * timeoutScale())
}

// ForFile is For with the size of the file at path; an unreadable file counts as empty and
// gets the base deadline.
func (b Budget) ForFile(path string) time.Duration {
	var size int64
	if info, err := os.Stat(path); err == nil {
		size = info.Size()
	}
	return b.For(size)
}

var (
	scaleOnce  sync.Once
	scaleValue = 1.0
)

func timeoutScale() float64 {
	scaleOnce.Do(func() { scaleValue = parseScale(os.Getenv(ScaleEnv)) })
	return scaleValue
}

// parseScale reads ScaleEnv. A value that is not a positive number is reported once and
// ignored, rather than turning every deadline into zero or failing the run.
func parseScale(raw string) float64 {
	if raw == "" {
		return 1
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(v) || v <= 0 || v > 1000 {
		logging.Printf("  WARNING: ignoring %s=%q - expected a positive number such as 2 or 0.5\n", ScaleEnv, raw)
		return 1
	}
	return v
}
