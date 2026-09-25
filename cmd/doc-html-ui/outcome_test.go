package main

import (
	"errors"
	"testing"
)

// A cancel that lands after the converter already exited cleanly kills nothing, so the run
// is reported as done; a cancel that stopped the converter (a non-nil exit) stays cancelled.
func TestOutcomeCleanExitWinsOverLateCancel(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir()) // a failed outcome writes the GUI log
	for _, tc := range []struct {
		cancelled bool
		err       error
		want      runEnd
	}{
		{cancelled: true, err: nil, want: runEnd{State: "done"}},
		{cancelled: false, err: nil, want: runEnd{State: "done"}},
		{cancelled: true, err: errors.New("killed"), want: runEnd{State: "cancelled"}},
		{cancelled: false, err: errors.New("start failed"), want: runEnd{State: "failed"}},
	} {
		if got := outcome(tc.cancelled, tc.err); got != tc.want {
			t.Errorf("outcome(%v, %v) = %+v, want %+v", tc.cancelled, tc.err, got, tc.want)
		}
	}
}
