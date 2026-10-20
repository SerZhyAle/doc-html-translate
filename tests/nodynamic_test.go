package tests

// Ticket 115: github.com/gen2brain/jpegxl, built without the nodynamic tag, loads libjxl.dll from the DLL
// search path in its package init. internal/jxl/nodynamic_guard.go stops any local build that forgot the
// tag, but the CI release build is the one go build the local gate never runs: checked here, before a tag
// push pays for a run that would only stop on the guard.

import (
	"strings"
	"testing"
)

func TestReleaseWorkflowBuildsWithNodynamic(t *testing.T) {
	wf := readRepoFile(t, ".github", "workflows", "release.yml")
	env := strings.Index(wf, "\nenv:\n  GOFLAGS: -tags=nodynamic\n")
	build := strings.Index(wf, "go build")
	if env < 0 || build < 0 || env > build {
		t.Error(".github/workflows/release.yml must set GOFLAGS: -tags=nodynamic in its workflow-level env, ahead of the first go build")
	}
}
