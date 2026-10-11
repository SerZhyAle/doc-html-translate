//go:build windows

package main

import (
	"context"
	"testing"
	"time"
)

// The listener times its own marker notifications; it never sends SHCNE_ASSOCCHANGED (that is a
// system-wide refresh the audit measures in the app and must not cause on a desktop).
func TestShellNotifyProbeDeliversItsMarkers(t *testing.T) {
	_ = takeShellEvents() // start from empty counters
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var ms []float64
	if err := runShellNotifyProbe(ctx, 1500*time.Millisecond, func(v float64) { ms = append(ms, v) }); err != nil {
		t.Fatalf("probe failed: %v", err)
	}
	if len(ms) < 3 {
		t.Fatalf("only %d marker observation(s) in 1.5 s", len(ms))
	}
	for _, v := range ms {
		if v <= 0 || v > 2000 {
			t.Errorf("latency %.3f ms is outside (0, 2000]", v)
		}
	}
	ev := takeShellEvents()
	if ev.MarkersSent < len(ms) {
		t.Errorf("markers sent %d < observations %d", ev.MarkersSent, len(ms))
	}
	if ev.Counts[shcneUpdateItem] < len(ms) {
		t.Errorf("listener counted %d update-item events for %d delivered markers", ev.Counts[shcneUpdateItem], len(ms))
	}
	if ev.associationBroadcasts() != 0 {
		t.Logf("note: %d association broadcast(s) came from another program during the test", ev.associationBroadcasts())
	}
}
