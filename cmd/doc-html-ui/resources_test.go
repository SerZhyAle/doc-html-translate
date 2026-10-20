package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
)

// The resource index answers from the conversion's own probes, so a missing helper is reported missing
// and a found one found, and the endpoint needs no network (SETTINGS-NAVIGATION 3.4).
func TestResourcesFollowTheReadinessProbes(t *testing.T) {
	saved := readinessProbes
	t.Cleanup(func() { readinessProbes = saved })
	readinessProbes.calibre = func() bool { return true }
	readinessProbes.sevenZip = func() bool { return false }
	readinessProbes.filedo = func() bool { return false }
	readinessProbes.ocrLocate = func() (string, error) { return "", errors.New("no tesseract") }
	readinessProbes.ocrMissing = func(context.Context, string, string) []string { return nil }

	rec := httptest.NewRecorder()
	handleResources(rec, httptest.NewRequest("GET", "/api/resources", nil))
	var got resourceFacts
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("not JSON: %v: %s", err, rec.Body.String())
	}
	want := resourceFacts{Tesseract: false, Calibre: true, SevenZip: false, FileDO: false}
	if got != want {
		t.Fatalf("resources = %+v, want %+v", got, want)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
}
