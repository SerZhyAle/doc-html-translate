package dialog

import (
	"bytes"
	"strings"
	"testing"

	"doc-html-translate/internal/logging"
)

// P34: an unattended run's warning goes to the run log, where a batch's operator finds it, and
// returns without waiting for anyone (dialog_windows_test.go proves no box opens).
func TestUnattendedWarningIsLogged(t *testing.T) {
	t.Setenv(HostEnv, "")
	SetUnattended(true)
	t.Cleanup(func() { SetUnattended(false) })
	var runLog bytes.Buffer
	logging.StartRunLog(&runLog)
	t.Cleanup(logging.StopRunLog)

	ShowWarning("PDF Images Not Displayed", "install ffmpeg")

	if got := runLog.String(); !strings.Contains(got, "WARNING: PDF Images Not Displayed") || !strings.Contains(got, "install ffmpeg") {
		t.Errorf("run log = %q, want the warning", got)
	}
}
