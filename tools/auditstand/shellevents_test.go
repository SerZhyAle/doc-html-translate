package main

import "testing"

func TestShellEventsSeparateTheStandsOwnMarkers(t *testing.T) {
	const updateItem = 0x2000
	e := shellEvents{Counts: map[uint32]int{updateItem: 10, assocChangedEvent: 2, 0x2: 5}, MarkersSent: 8}
	if e.associationBroadcasts() != 2 {
		t.Errorf("associationBroadcasts = %d", e.associationBroadcasts())
	}
	// 17 total - 2 association - 8 markers = 7 events that are not the stand's.
	if e.otherEvents() != 7 {
		t.Errorf("otherEvents = %d, want 7", e.otherEvents())
	}
	if (shellEvents{Counts: map[uint32]int{}, MarkersSent: 3}).otherEvents() != 0 {
		t.Error("otherEvents must never go negative")
	}
}
