package logging

import (
	"bytes"
	"strings"
	"testing"
)

// RunLogf writes to the same on-disk log as every other level, so it gets the same redaction:
// a recovered panic's value can carry whatever the failing call was holding.
func TestRunLogfAppliesRunLogFilter(t *testing.T) {
	var sink bytes.Buffer
	SetRunLogFilter(func(s string) string { return strings.ReplaceAll(s, "SECRET", "<redacted>") })
	defer SetRunLogFilter(nil)
	StartRunLog(&sink)
	defer StopRunLog()

	RunLogf("internal error: key=%s\n", "SECRET")

	if strings.Contains(sink.String(), "SECRET") || !strings.Contains(sink.String(), "key=<redacted>") {
		t.Fatalf("run log = %q", sink.String())
	}
}
