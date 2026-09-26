package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"doc-html-translate/internal/logging"
)

// P36: the error that ends a run must reach the run log (and so a -report bundle), not only
// stderr.
func TestReportFailureRecordsRunLog(t *testing.T) {
	var runLog, stderr bytes.Buffer
	logging.StartRunLog(&runLog)
	t.Cleanup(logging.StopRunLog)

	reportFailure(&stderr, 2, errors.New("parse: broken xref table"))

	if got := stderr.String(); got != "Error: parse: broken xref table\n" {
		t.Errorf("stderr = %q", got)
	}
	if got := runLog.String(); !strings.Contains(got, "Run failed (exit code 2): parse: broken xref table") {
		t.Errorf("run log lacks the final error: %q", got)
	}
}
