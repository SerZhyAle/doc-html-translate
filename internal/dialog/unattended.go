package dialog

import (
	"sync/atomic"

	"doc-html-translate/internal/logging"
)

// unattended is set for a run nobody is watching (a -noopen batch, output piped into a script).
// A modal warning there halts the batch until someone clicks, so it is logged instead.
var unattended atomic.Bool

// SetUnattended says whether a human is watching this run. Questions are unaffected: they still
// default to the safe answer, and a paid batch sets -max-cost instead of being asked.
func SetUnattended(v bool) { unattended.Store(v) }

// logWarning is a warning's non-blocking form: the console and the run log, where a batch's
// operator reads it afterwards.
func logWarning(title, message string) {
	logging.Errorf("WARNING: %s\n%s\n", title, message)
}
