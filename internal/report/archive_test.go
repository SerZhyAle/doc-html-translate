package report

import (
	"archive/zip"
	"encoding/json"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
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
	env, ok := got["environment.txt"]
	if !ok {
		t.Error("archive has no environment.txt")
	}
	if strings.Contains(env, "logs_left_out") || strings.Contains(got["report.txt"], "logs left out") {
		t.Errorf("the summary claims logs were left out although none were:\n%s\n%s", env, got["report.txt"])
	}
	if _, ok := got["settings.json"]; !ok {
		t.Error("archive has no settings.json")
	}
	// DIAGNOSTIC-REPORT rule 2 and 8 B: environment.txt is what a program reads (KEY=value, the
	// schema first), report.txt is what the author reads (label: value), same facts.
	if !strings.HasPrefix(env, "schema_version=1\napp_version=26.0811.1400\n") {
		t.Errorf("environment.txt begins %q, want schema_version then app_version", strings.SplitN(env, "\n", 3)[:2])
	}
	for _, line := range strings.Split(strings.TrimRight(env, "\n"), "\n") {
		if k, _, ok := strings.Cut(line, "="); !ok || k == "" || strings.ContainsAny(k, " :") {
			t.Errorf("environment.txt line %q is not KEY=value with a spaceless key", line)
		}
	}
	if rep := got["report.txt"]; !strings.HasPrefix(rep, "version: 26.0811.1400\nedition: ") || !strings.Contains(rep, "\ninterface language: ") {
		t.Errorf("report.txt is not the labelled block:\n%s", rep)
	}
	logs := 0
	for name := range got {
		switch {
		case name == "environment.txt" || name == "report.txt" || name == "settings.json":
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
	if want := omittedMarker(6 << 20); got[oldest] != want {
		t.Errorf("the oldest log %q holds %d bytes, want only the omission marker %q", oldest, len(got[oldest]), want)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat archive: %v", err)
	}
	if info.Size() > MaxArchiveBytes {
		t.Errorf("archive is %d bytes, over the %d cap", info.Size(), MaxArchiveBytes)
	}
}

// The archive says on its own that its history is partial: the count the caller gets back is also
// a key of environment.txt, appended after the existing ones.
func TestBuildRecordsLogsLeftOutInTheEnvironment(t *testing.T) {
	setHome(t)
	writeLogs(t, 4, 6<<20)

	path, dropped, err := Build(buildOpts())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if dropped != 2 {
		t.Fatalf("dropped %d logs, want 2 (two 6 MB logs fit under the cap)", dropped)
	}
	got := entries(t, path)
	env := got["environment.txt"]
	lines := strings.Split(strings.TrimRight(env, "\n"), "\n")
	if last, want := lines[len(lines)-1], "logs_left_out=2"; last != want {
		t.Errorf("last environment.txt line = %q, want %q (an additive key after the existing ones)\n%s", last, want, env)
	}
	if !strings.HasPrefix(env, "schema_version=1\napp_version=26.0811.1400\n") {
		t.Errorf("environment.txt begins %q, want schema_version then app_version (DIAGNOSTIC-REPORT 8 B)", strings.SplitN(env, "\n", 3)[:2])
	}
	for _, line := range lines {
		if k, _, ok := strings.Cut(line, "="); !ok || k == "" || strings.ContainsAny(k, " :") {
			t.Errorf("environment.txt line %q is not KEY=value with a spaceless key", line)
		}
	}
	rep := got["report.txt"]
	rlines := strings.Split(strings.TrimRight(rep, "\n"), "\n")
	if last, want := rlines[len(rlines)-1], "logs left out: 2"; last != want {
		t.Errorf("last report.txt line = %q, want %q\n%s", last, want, rep)
	}
	if !strings.HasPrefix(rep, "version: 26.0811.1400\n") {
		t.Errorf("the labelled report moved: report.txt begins %q", strings.SplitN(rep, "\n", 2)[0])
	}
}

// DIAGNOSTIC-REPORT 0.15, amendment 2026-10-29 item 1: the frozen prefix, field order and separator,
// reason archive_cap, and the size of the source as read - never its redacted size.
func TestBuildLeavesTheOmissionMarkerInPlaceOfALogLeftOut(t *testing.T) {
	if got, want := omittedMarker(42), "[Diag] LOG OMITTED | reason=archive_cap | source_bytes=42\n"; got != want {
		t.Fatalf("omittedMarker(42) = %q, want %q", got, want)
	}
	setHome(t)
	names := writeLogs(t, 4, 64)
	logs, err := logFiles()
	if err != nil {
		t.Fatal(err)
	}

	// Room for two of the four logs: packLogs is driven with a small budget instead of 15 MiB of logs.
	kept, dropped := packLogs(logs, 2*64)
	if dropped != 2 {
		t.Fatalf("dropped %d logs, want 2", dropped)
	}
	got := make(map[string]string, len(kept))
	for _, e := range kept {
		got[e.name] = string(e.data)
	}
	for i, n := range names {
		entry, ok := got["logs/"+n]
		switch {
		case !ok:
			t.Errorf("log %q has no entry: neither the log nor its omission marker", n)
		case i < 2 && entry != omittedMarker(64):
			t.Errorf("older log %q holds %q, want the omission marker %q", n, entry, omittedMarker(64))
		case i >= 2 && entry != strings.Repeat("x", 64):
			t.Errorf("newer log %q was not kept whole: %q", n, entry)
		}
	}
}

// A log larger than the archive is decided from its size, not from its content: the run must not
// pay the log's size in memory to learn that it does not fit.
func TestBuildSkipsAnOversizedLogWithoutReadingIt(t *testing.T) {
	setHome(t)
	names := writeLogs(t, 2, 64)
	huge := RunLogPath(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), 1)
	f, err := os.Create(huge)
	if err != nil {
		t.Fatal(err)
	}
	const hugeBytes = 64 << 20
	if err := f.Truncate(hugeBytes); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	path, dropped, err := Build(buildOpts())
	runtime.ReadMemStats(&after)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if dropped != 1 {
		t.Errorf("dropped %d logs, want 1 (the oversized one)", dropped)
	}
	if grew := after.TotalAlloc - before.TotalAlloc; grew > MaxArchiveBytes {
		t.Errorf("Build allocated %d bytes: the %d byte log was read into memory", grew, int64(hugeBytes))
	}
	got := entries(t, path)
	if entry, want := got["logs/"+filepath.Base(huge)], omittedMarker(hugeBytes); entry != want {
		t.Errorf("the oversized log's entry holds %d bytes, want only the omission marker %q", len(entry), want)
	}
	for _, n := range names {
		if _, ok := got["logs/"+n]; !ok {
			t.Errorf("the small log %q was dropped along with the oversized one", n)
		}
	}
	if env := got["environment.txt"]; !strings.Contains(env, "logs left out: 1\n") {
		t.Errorf("environment.txt does not record the omission:\n%s", env)
	}
}

// DIAGNOSTIC-REPORT 8A: <Product>-logs[-<version>]-yyyyMMdd-HHmmss.zip, and a second report in the
// same second moves the stamp on instead of leaving that shape.
func TestBuildNamesTheArchiveByTheContract(t *testing.T) {
	setHome(t)
	writeLogs(t, 1, 16)

	opts := buildOpts()
	first, _, err := Build(opts)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	second, _, err := Build(opts)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts.AppVersion = ""
	unstamped, _, err := Build(opts)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	for _, c := range []struct{ path, want string }{
		{first, "doc-html-translate-logs-26.0811.1400-20260811-140000.zip"},
		{second, "doc-html-translate-logs-26.0811.1400-20260811-140001.zip"},
		{unstamped, "doc-html-translate-logs-20260811-140000.zip"},
	} {
		if got := filepath.Base(c.path); got != c.want {
			t.Errorf("archive named %q, want %q", got, c.want)
		}
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
	if !strings.Contains(got, "[REDACTED]") {
		t.Errorf("settings.json was not redacted: %q", got)
	}
}

func TestBuildRedactsRecentPathsInSettings(t *testing.T) {
	home := filepath.Join(t.TempDir(), "Audit User")
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	t.Setenv("LOCALAPPDATA", filepath.Join(home, "AppData", "Local"))
	slash := filepath.ToSlash(home)
	filePath := slash
	if !strings.HasPrefix(filePath, "/") {
		filePath = "/" + filePath
	}
	recent := []string{
		filepath.Join(home, "Books", "native.epub"),
		slash + "/Books/slash.epub",
		(&url.URL{Scheme: "file", Path: filePath + "/Books/url.epub"}).String(),
	}
	settings, err := json.Marshal(map[string]any{"recentDocuments": recent})
	if err != nil {
		t.Fatal(err)
	}
	opts := buildOpts()
	opts.SettingsJSON = settings
	path, _, err := Build(opts)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	got := entries(t, path)["settings.json"]
	if !json.Valid([]byte(got)) {
		t.Fatalf("redacted settings.json is invalid JSON: %q", got)
	}
	if strings.Contains(got, "Audit User") || strings.Contains(got, "Audit%20User") {
		t.Fatalf("profile name reached settings.json: %q", got)
	}
	if strings.Count(got, "%USERPROFILE%") != len(recent) {
		t.Errorf("settings.json did not redact all recent paths: %q", got)
	}
	// DIAGNOSTIC-REPORT 8 B: the paths keep their folders' variables and lose the document names.
	for _, name := range []string{"native", "slash", "url"} {
		if strings.Contains(got, name) {
			t.Errorf("document name %q reached settings.json: %q", name, got)
		}
	}
	if strings.Count(got, "<document>.epub") != len(recent) {
		t.Errorf("settings.json did not replace all document names: %q", got)
	}
}

// The settings copy is the author's view of how the app was configured, not of what the user
// converted: the output folder and the recent documents leave it, every other setting stays.
func TestBuildLeavesTheUsersNamesOutOfTheSettingsCopy(t *testing.T) {
	setHome(t)
	writeLogs(t, 1, 32)
	opts := buildOpts()
	opts.SettingsJSON = []byte(`{"tocDepth":3,"output":"D:\\Library\\Dune","recent":["D:\\Library\\Dune.epub","D:\\Library\\Emma.pdf"],"theme":"dark"}`)

	path, _, err := Build(opts)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	got := entries(t, path)["settings.json"]
	var settings map[string]any
	if err := json.Unmarshal([]byte(got), &settings); err != nil {
		t.Fatalf("settings.json is not JSON: %v\n%s", err, got)
	}
	for _, k := range []string{"output", "recent"} {
		if _, present := settings[k]; present {
			t.Errorf("settings.json still has %q: %s", k, got)
		}
	}
	for _, name := range []string{"Dune", "Emma", "Library"} {
		if strings.Contains(got, name) {
			t.Errorf("%q reached settings.json: %s", name, got)
		}
	}
	if settings["tocDepth"] != float64(3) || settings["theme"] != "dark" {
		t.Errorf("other settings changed: %s", got)
	}
}

func TestDropUserNamesLeavesNonObjectsAlone(t *testing.T) {
	for _, in := range []string{`not json`, `[1,2]`, `null`, ``} {
		if got := string(dropUserNames([]byte(in))); got != in {
			t.Errorf("dropUserNames(%q) = %q, want the input back", in, got)
		}
	}
}

func TestBuildNeverArchivesTheAPIKeyFile(t *testing.T) {
	setHome(t)
	writeLogs(t, 1, 32)
	if err := os.WriteFile(filepath.Join(Dir(), "google_api.key"), []byte("AIzaSyA1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q7r"), 0o600); err != nil {
		t.Fatalf("write key file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(Dir(), "google_api.key.dpapi"), []byte("protected-key-blob"), 0o600); err != nil {
		t.Fatalf("write protected key file: %v", err)
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

// A second report built in the same second gets its own file instead of replacing the first.
func TestBuildNeverOverwritesAReport(t *testing.T) {
	setHome(t)
	writeLogs(t, 1, 16)

	first, _, err := Build(buildOpts())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	second, _, err := Build(buildOpts())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if first == second {
		t.Fatalf("both reports written to %q", first)
	}
	for _, p := range []string{first, second} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("report %q missing: %v", p, err)
		}
	}
}
