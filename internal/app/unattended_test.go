package app

import "testing"

// P34: a -noopen batch or a piped run is unattended, so its warnings are logged instead of
// shown as a modal box; a console run with a human in front of it keeps the box.
func TestUnattendedRun(t *testing.T) {
	for _, c := range []struct {
		noOpen, terminal, want bool
	}{
		{false, true, false},
		{true, true, true},
		{false, false, true},
		{true, false, true},
	} {
		if got := unattendedRun(c.noOpen, c.terminal); got != c.want {
			t.Errorf("unattendedRun(noOpen=%v, terminal=%v) = %v, want %v", c.noOpen, c.terminal, got, c.want)
		}
	}
}
