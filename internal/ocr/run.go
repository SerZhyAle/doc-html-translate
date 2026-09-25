package ocr

import (
	"context"
	"fmt"
	"runtime/debug"

	"doc-html-translate/internal/logging"
	"doc-html-translate/internal/procrun"
)

// tesseractEnv is the environment every Tesseract process gets on top of ours. The worker pool
// already runs one process per spare core (ocrWorkers), so OpenMP's own threads inside each
// process would multiply that by the core count again and oversubscribe the machine.
var tesseractEnv = []string{"OMP_THREAD_LIMIT=1"}

// runTesseract runs one Tesseract call through the shared runner, with a deadline taken from
// budget and the size of input (the image being read, or "" for a call that reads none).
func runTesseract(budget procrun.Budget, bin, input string, args []string) (procrun.Result, error) {
	timeout := budget.For(0)
	if input != "" {
		timeout = budget.ForFile(input)
	}
	return procrun.Run(context.Background(), procrun.Cmd{
		Tool:    "tesseract",
		Path:    bin,
		Args:    args,
		Env:     tesseractEnv,
		Timeout: timeout,
	})
}

// recognizeImage is what a worker calls for one image; a variable only so a test can stand in
// a recognizer that panics.
var recognizeImage = Recognize

// recognizeSafe runs one image's recognition behind a panic guard. The image decoders and the
// staging code run on whatever a book embeds, and OCR is best-effort (OCR-PIPELINE): a picture
// that panics a decoder is one failed image in the report, not the end of the conversion. A
// panic in a worker goroutine cannot be recovered anywhere else, so the guard has to be here.
func recognizeSafe(bin, imgPath, lang, dataDir string) (res Result, err error) {
	defer func() {
		if r := recover(); r != nil {
			// The console hears about it from the overlay report, which lists each failed
			// image with its reason; the stack is for the run log only.
			logging.RunLogf("OCR of %s panicked: %v\n%s\n", imgPath, r, debug.Stack())
			res, err = Result{}, fmt.Errorf("internal error while reading the image: %v", r)
		}
	}()
	return recognizeImage(bin, imgPath, lang, dataDir)
}
