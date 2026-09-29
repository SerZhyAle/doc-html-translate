package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type updateRoundTrip func(*http.Request) (*http.Response, error)

func (f updateRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestUpdateCheckVersionAndRequest(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	oldVersion, oldClient := Version, updateClient
	defer func() { Version, updateClient = oldVersion, oldClient }()
	Version = "26.0912.2026"
	requests := 0
	updateClient = &http.Client{Transport: updateRoundTrip(func(r *http.Request) (*http.Response, error) {
		requests++
		if r.URL.String() != latestReleaseAPI || r.URL.RawQuery != "" || r.Header.Get("X-Install-ID") != "" {
			t.Errorf("update request disclosed state or changed endpoint: %s", r.URL)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"tag_name":"v26.0929.1200","html_url":"https://evil.example/run.exe"}`)), Header: make(http.Header)}, nil
	})}
	request := func(auto bool) updateReply {
		t.Helper()
		body := `{"automatic":false}`
		if auto {
			body = `{"automatic":true}`
		}
		rec := httptest.NewRecorder()
		handleUpdateCheck(rec, httptest.NewRequest(http.MethodPost, "/api/update-check", strings.NewReader(body)))
		if rec.Code != 200 {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}
		var reply updateReply
		if err := json.NewDecoder(rec.Body).Decode(&reply); err != nil {
			t.Fatal(err)
		}
		return reply
	}
	if got := request(true); got.Status != "disabled" || requests != 0 {
		t.Fatalf("off made request: %+v, %d", got, requests)
	}
	if got := request(false); got.Status != "newer" || got.URL != releasePageBase+"v26.0929.1200" || requests != 1 {
		t.Fatalf("bad newer result: %+v, %d", got, requests)
	}
	if got := request(false); got.Status != "newer" || requests != 1 {
		t.Fatalf("cache missed: %+v, %d", got, requests)
	}
	if err := writeSettings([]byte(`{"autoUpdates":true}`)); err != nil {
		t.Fatal(err)
	}
	if got := request(true); got.Status != "newer" || requests != 1 {
		t.Fatalf("auto cache missed: %+v, %d", got, requests)
	}
	if got := request(true); got.Status != "newer" || requests != 1 {
		t.Fatalf("auto repeated: %+v, %d", got, requests)
	}
	Version = "26.0930.1200"
	if got := request(false); got.Status != "current" {
		t.Fatalf("newer local version: %+v", got)
	}
}

func TestUpdateVersionRejectsUntrustedTag(t *testing.T) {
	for _, tag := range []string{"v26.0929.1200.exe", "ext-cws-v26.0929.1200", "v26.929.1200", "v26.0929.1200/evil"} {
		if releaseVersion.MatchString(tag) {
			t.Errorf("accepted %q", tag)
		}
	}
}

func TestUpdateCheckOfflineIsQuiet(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	oldClient := updateClient
	defer func() { updateClient = oldClient }()
	requests := 0
	updateClient = &http.Client{Transport: updateRoundTrip(func(*http.Request) (*http.Response, error) {
		requests++
		return &http.Response{StatusCode: http.StatusServiceUnavailable, Body: io.NopCloser(strings.NewReader("offline")), Header: make(http.Header)}, nil
	})}
	if err := writeSettings([]byte(`{"autoUpdates":true}`)); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		rec := httptest.NewRecorder()
		handleUpdateCheck(rec, httptest.NewRequest(http.MethodPost, "/api/update-check", strings.NewReader(`{"automatic":true}`)))
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"unavailable"`) {
			t.Fatalf("offline result: %d %s", rec.Code, rec.Body.String())
		}
	}
	if requests != 1 {
		t.Fatalf("automatic offline check retried %d times", requests)
	}
}
