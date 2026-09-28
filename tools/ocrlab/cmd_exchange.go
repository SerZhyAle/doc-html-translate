package main

import (
	"errors"
	"flag"
	"fmt"
	"path/filepath"

	"doc-html-translate/internal/ocr"
	"doc-html-translate/tools/ocrlab/exchange"
	"doc-html-translate/tools/ocrlab/runner"
)

// cmdExchange writes the OCR-OVERLAY section 7 comparison record for every scene of a saved run,
// from the run's diagnostics sidecar, into <run-dir>/exchange/<scene>.json. Offline, like score:
// no browser and no recognizer, so yesterday's run can be exchanged today. Either edition's run
// works - both write the same diagnostic line (OCR-PIPELINE amendment 1.4 D).
func cmdExchange(args []string) error {
	fs := flag.NewFlagSet("exchange", flag.ExitOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return cannotVerify(errors.New("usage: ocrlab exchange <run-dir>"))
	}
	dir := fs.Arg(0)
	ids, err := exchange.WriteDir(dir, runner.DiagFile, ocr.ExifOrientation)
	if err != nil {
		return err
	}
	fmt.Printf("exchange: %d scene record(s) in %s\n", len(ids), filepath.Join(dir, "exchange"))
	return nil
}
