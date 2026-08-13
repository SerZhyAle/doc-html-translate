package report

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// entries reads an archive and returns its entry names mapped to their contents.
func entries(t *testing.T, path string) map[string]string {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("zip.OpenReader(%q): %v", path, err)
	}
	defer func() { _ = zr.Close() }()

	out := make(map[string]string, len(zr.File))
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open entry %q: %v", f.Name, err)
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatalf("read entry %q: %v", f.Name, err)
		}
		out[f.Name] = string(data)
	}
	return out
}

func buildOpts() BuildOptions {
	return BuildOptions{
		AppVersion:   "26.0811.1400",
		SettingsJSON: []byte(`{"tocDepth":3}`),
		At:           time.Date(2026, 8, 11, 14, 0, 0, 0, time.UTC),
	}
}

func TestBuildProducesReadableArchive(t *testing.T) {
	setHome(t)
	writeLogs(t, 3, 64)

	path, dropped, err := Build(buildOpts())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if dropped != 0 {
		t.Errorf("dropped %d small logs, want 0", dropped)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("archive not written: %v", err)
	}
	if !strings.Contains(filepath.Base(path), "26.0811.1400") {
		t.Errorf("archive name %q does not carry the version", filepath.Base(path))
	}

	got := entries(t, path)
	if _, ok := got["environment.txt"]; !ok {
		t.Error("archive has no environment.txt")
	}
	if _, ok := got["settings.json"]; !ok {
		t.Error("archive has no settings.json")
	}
	logs := 0
	for name := range got {
		switch {
		case name == "environment.txt" || name == "settings.json":
		case strings.HasPrefix(name, "logs/"):
			logs++
		default:
			t.Errorf("unexpected archive entry %q", name)
		}
	}
	if logs == 0 {
		t.Error("archive carries no run log")
	}
}

func TestBuildDropsOldestLogsOverCap(t *testing.T) {
	setHome(t)
	// Four logs of 6 MB each: only two can fit under the 15 MB cap.
	names := writeLogs(t, 4, 6<<20)

	path, dropped, err := Build(buildOpts())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if dropped == 0 {
		t.Fatal("nothing was dropped although the logs exceed the cap")
	}

	got := entries(t, path)
	newest, oldest := "logs/"+names[len(names)-1], "logs/"+names[0]
	if _, ok := got[newest]; !ok {
		t.Errorf("the newest log %q was dropped", newest)
	}
	if _, ok := got[oldest]; ok {
		t.Errorf("the oldest log %q survived the cap", oldest)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat archive: %v", err)
	}
	if info.Size() > MaxArchiveBytes {
		t.Errorf("archive is %d bytes, over the %d cap", info.Size(), MaxArchiveBytes)
	}
}

func TestBuildRedactsSettings(t *testing.T) {
	setHome(t)
	writeLogs(t, 1, 32)

	const secret = "AIzaSyA1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q7r"
	opts := buildOpts()
	opts.SettingsJSON = []byte(`{"googleKey":"` + secret + `"}`)

	path, _, err := Build(opts)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	got := entries(t, path)["settings.json"]
	if strings.Contains(got, secret) {
		t.Fatalf("the API key reached the archive: %q", got)
	}
	if !strings.Contains(got, "<redacted>") {
		t.Errorf("settings.json was not redacted: %q", got)
	}
}

func TestBuildNeverArchivesTheAPIKeyFile(t *testing.T) {
	setHome(t)
	writeLogs(t, 1, 32)
	if err := os.WriteFile(filepath.Join(Dir(), "google_api.key"), []byte("AIzaSyA1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q7r"), 0o600); err != nil {
		t.Fatalf("write key file: %v", err)
	}

	path, _, err := Build(buildOpts())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	for name, body := range entries(t, path) {
		if strings.Contains(name, "google_api.key") {
			t.Errorf("the key file was archived as %q", name)
		}
		if strings.Contains(body, "AIzaSyA1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q7r") {
			t.Errorf("the key's value reached entry %q", name)
		}
	}
}
