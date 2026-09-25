package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"doc-html-translate/internal/report"
)

// APP-BEHAVIOUR rule 6: a failure the user meets is a named cause and a set of actions; the raw
// error goes to the log, which "Send logs to the author" packs. The page never shows an error
// string - it words the cause itself - so the GUI keeps a log of its own for the raw half.

// guiLog is this launch's log file, created on the first failure only: a launch where nothing
// went wrong leaves no file behind. The name carries the launch time; the folder is resolved at
// write time, so it follows %LOCALAPPDATA% as the rest of the report store does.
var guiLog = struct {
	sync.Mutex
	started time.Time
}{started: time.Now()}

// logFailure records one failed action and its raw error. Writing the log is a side channel: its
// own failure is dropped rather than turned into a second failure the user has to read.
func logFailure(op string, err error) {
	if err == nil {
		return
	}
	guiLog.Lock()
	defer guiLog.Unlock()
	path := report.GUILogPath(guiLog.started, os.Getpid())
	if mkErr := os.MkdirAll(filepath.Dir(path), 0o755); mkErr != nil {
		return
	}
	f, openErr := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if openErr != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s gui %s: %v\n", time.Now().Format(time.RFC3339), op, err)
}
