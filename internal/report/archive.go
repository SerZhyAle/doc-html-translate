package report

import (
	"archive/zip"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// MaxArchiveBytes caps the archive at what a mail provider will still carry - the common
// limits are 20-25 MB, and the message body plus the provider's own encoding overhead eat
// into that, so the payload stays comfortably below the smallest of them.
const MaxArchiveBytes = 15 << 20

// productToken is the <Product> of DIAGNOSTIC-REPORT 8A's archive name.
const productToken = "doc-html-translate"

// BuildOptions is what the caller knows and the collector cannot find out for itself.
type BuildOptions struct {
	AppVersion   string
	Packaged     bool
	SettingsJSON []byte
	At           time.Time
}

type archiveEntry struct {
	name string
	data []byte
}

// Build writes one archive - the environment summary, the redacted settings, and as many
// recent run logs as fit under MaxArchiveBytes, newest first - and reports how many logs had
// to be left out. When some were, environment.txt says so too (see leftOutLine), so a reader of
// the archive alone is never told that the history it holds is the whole of it.
//
// Every byte written here passes through Redact. This is the only route by which anything
// leaves the machine, so the gate lives at the gate: a caller must never assemble an archive
// of its own. Nothing from Dir() other than the log files is ever included - in particular
// none of the credential files that live beside them.
func Build(opts BuildOptions) (path string, droppedLogs int, err error) {
	dir := filepath.Join(Dir(), "reports")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", 0, fmt.Errorf("create report folder: %w", err)
	}

	env := Environment(opts.AppVersion, opts.Packaged)
	var settings []byte
	if len(opts.SettingsJSON) > 0 {
		settings = RedactBytes(opts.SettingsJSON)
	}
	logs, err := logFiles()
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", 0, fmt.Errorf("list run logs: %w", err)
	}
	// The logs are chosen before anything is written because the count of those left out belongs
	// in environment.txt, which comes first. Its line is reserved at its widest: at most every log.
	budget := MaxArchiveBytes - len(env) - len(settings) - len(leftOutLine(len(logs)))
	kept, droppedLogs := packLogs(logs, budget)
	if droppedLogs > 0 {
		env += leftOutLine(droppedLogs)
	}

	f, path, err := createArchive(dir, opts.AppVersion, opts.At)
	if err != nil {
		return "", 0, fmt.Errorf("create archive: %w", err)
	}
	zw := zip.NewWriter(f)
	entries := []archiveEntry{{"environment.txt", []byte(env)}}
	if len(opts.SettingsJSON) > 0 {
		entries = append(entries, archiveEntry{"settings.json", settings})
	}
	for _, e := range append(entries, kept...) {
		w, err := zw.Create(e.name)
		if err == nil {
			_, err = w.Write(e.data)
		}
		if err != nil {
			return "", 0, closeFailed(zw, f, path, err)
		}
	}

	if err := zw.Close(); err != nil {
		_ = f.Close()
		return "", 0, fmt.Errorf("finish archive: %w", err)
	}
	if err := f.Close(); err != nil {
		return "", 0, fmt.Errorf("close archive: %w", err)
	}
	return path, droppedLogs, nil
}

// leftOutLine is the key environment.txt gains when run logs did not make it into the archive.
// It is appended after the existing keys: a reader matches keys by name and ignores the ones it
// does not know (DIAGNOSTIC-REPORT 3), so adding one breaks none.
func leftOutLine(n int) string {
	return fmt.Sprintf("logs left out: %d\n", n)
}

// packLogs redacts the run logs newest first and keeps as many as fit in budget bytes, reporting
// how many it left out. The newest log is the run the report is about, so it is the last thing
// to go; an older, smaller log may still fit after a larger one was dropped.
func packLogs(logs []logFile, budget int) (kept []archiveEntry, leftOut int) {
	sort.Slice(logs, func(i, j int) bool { return logs[i].name > logs[j].name })
	for _, l := range logs {
		// A log larger than the whole archive can never fit. Deciding that from the listing keeps
		// it out of memory: reading it first and measuring afterwards would spend its full size,
		// twice with the redacted copy, to reach the same answer.
		if l.size > MaxArchiveBytes {
			leftOut++
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(LogsDir(), l.name))
		if readErr != nil {
			continue
		}
		data = RedactBytes(data)
		if len(data) > budget {
			leftOut++
			continue
		}
		budget -= len(data)
		kept = append(kept, archiveEntry{"logs/" + l.name, data})
	}
	return kept, leftOut
}

// archiveName is DIAGNOSTIC-REPORT 8A's <Product>-logs[-<version>]-yyyyMMdd-HHmmss.zip. The
// version segment is optional in the contract; a build without a stamp simply leaves it out.
func archiveName(version string, at time.Time) string {
	var b strings.Builder
	b.WriteString(productToken + "-logs")
	if version != "" {
		b.WriteString("-" + version)
	}
	return b.String() + "-" + at.Format("20060102-150405") + ".zip"
}

// createArchive creates the archive named for `at`, or for the next second, and the next, when
// that name is taken. A report is never overwritten: two built within the same second - the GUI
// button pressed twice - would otherwise leave the user holding one file while the app announced
// two. The name moves on by a second rather than gaining a suffix, so every name keeps the
// contract's shape and a reader that parses the timestamp off its end still can.
func createArchive(dir, version string, at time.Time) (*os.File, string, error) {
	for n := 0; ; n++ {
		path := filepath.Join(dir, archiveName(version, at.Add(time.Duration(n)*time.Second)))
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if errors.Is(err, fs.ErrExist) && n < 1000 {
			continue
		}
		return f, path, err
	}
}

// closeFailed abandons a half-written archive rather than leaving one that looks complete.
func closeFailed(zw *zip.Writer, f *os.File, path string, cause error) error {
	_ = zw.Close()
	_ = f.Close()
	_ = os.Remove(path)
	return fmt.Errorf("write archive: %w", cause)
}
