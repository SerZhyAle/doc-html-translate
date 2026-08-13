package report

import (
	"archive/zip"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// MaxArchiveBytes caps the archive at what a mail provider will still carry - the common
// limits are 20-25 MB, and the message body plus the provider's own encoding overhead eat
// into that, so the payload stays comfortably below the smallest of them.
const MaxArchiveBytes = 15 << 20

// BuildOptions is what the caller knows and the collector cannot find out for itself.
type BuildOptions struct {
	AppVersion   string
	Packaged     bool
	SettingsJSON []byte
	At           time.Time
}

// Build writes one archive - the environment summary, the redacted settings, and as many
// recent run logs as fit under MaxArchiveBytes, newest first - and reports how many logs had
// to be left out.
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
	name := fmt.Sprintf("report_doc-html-translate_%s_%s.zip", opts.AppVersion, opts.At.Format("20060102-1504"))
	path = filepath.Join(dir, name)

	f, err := os.Create(path)
	if err != nil {
		return "", 0, fmt.Errorf("create archive: %w", err)
	}
	zw := zip.NewWriter(f)

	written := 0
	add := func(entry string, data []byte) error {
		w, err := zw.Create(entry)
		if err != nil {
			return err
		}
		if _, err := w.Write(data); err != nil {
			return err
		}
		written += len(data)
		return nil
	}

	if err := add("environment.txt", []byte(Environment(opts.AppVersion, opts.Packaged))); err != nil {
		return "", 0, closeFailed(zw, f, path, err)
	}
	if len(opts.SettingsJSON) > 0 {
		if err := add("settings.json", RedactBytes(opts.SettingsJSON)); err != nil {
			return "", 0, closeFailed(zw, f, path, err)
		}
	}

	logs, err := logFiles()
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", 0, closeFailed(zw, f, path, err)
	}
	// Newest first: the run that just failed is the one the report is about, and it is the
	// last thing that may be dropped.
	sort.Slice(logs, func(i, j int) bool { return logs[i].name > logs[j].name })
	for _, l := range logs {
		data, readErr := os.ReadFile(filepath.Join(LogsDir(), l.name))
		if readErr != nil {
			continue
		}
		data = RedactBytes(data)
		if written+len(data) > MaxArchiveBytes {
			droppedLogs++
			continue
		}
		if err := add("logs/"+l.name, data); err != nil {
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

// closeFailed abandons a half-written archive rather than leaving one that looks complete.
func closeFailed(zw *zip.Writer, f *os.File, path string, cause error) error {
	_ = zw.Close()
	_ = f.Close()
	_ = os.Remove(path)
	return fmt.Errorf("write archive: %w", cause)
}
