package ocr

import (
	"strings"
	"testing"
)

// Chrome returns a percentage min-height unresolved, and plates carry exactly that
// (percentStyle), so a fit that only parseFloat()s it pins the box to N px instead of N% of the
// figure - past the bottom of a figure under 100 px. The structural check below is the Go-side
// guard; the rendered behaviour is covered by tools/ocrlab/runner's phone-figure browser test.
func TestOcrScriptResolvesPercentMinHeightAgainstFigure(t *testing.T) {
	for _, want := range []string{
		`/%$/.test(mh)`,
		`parseFloat(mh)/100*b.parentNode.getBoundingClientRect().height`,
		`:(parseFloat(mh)||0)`,
	} {
		if !strings.Contains(ocrScript, want) {
			t.Errorf("ocrScript lost %q: a percentage min-height must resolve against the figure height", want)
		}
	}
	if strings.Contains(ocrScript, `parseFloat(getComputedStyle(b).minHeight)`) {
		t.Error("ocrScript reads the computed min-height with a bare parseFloat again")
	}
}
