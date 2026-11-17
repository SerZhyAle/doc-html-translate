package report

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	"doc-html-translate/internal/fsutil"
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
	// Out is the file or existing folder the caller wants the archive in (ticket E0147, `report build --out`; folder
	// branch ticket E0149, APP-COMMANDS 0.3 item V): empty writes to the app's reports folder. A folder (existing, or
	// ending in a separator) receives the DIAGNOSTIC-REPORT 8A archive name; a missing folder is refused and never
	// created. An existing target file is refused with ErrOutExists unless Overwrite replaces it whole.
	Out       string
	Overwrite bool
}

// The refusals of a caller-named Out. They are checked before anything is collected or written.
var (
	ErrOutExists        = errors.New("report: the output file already exists")
	ErrOutFolderMissing = errors.New("report: the folder of the output file does not exist")
)

// IsNoSpace reports whether err is the system saying the disk is full: the command answers it as exit 13.
func IsNoSpace(err error) bool {
	var en syscall.Errno
	if !errors.As(err, &en) {
		return false
	}
	if runtime.GOOS == "windows" {
		return en == 39 || en == 112 // ERROR_HANDLE_DISK_FULL, ERROR_DISK_FULL
	}
	return en == syscall.ENOSPC
}

type archiveEntry struct {
	name string
	data []byte
}

// Build writes one archive - the environment summary, the redacted settings, and as many
// recent run logs as fit under MaxArchiveBytes, newest first - and reports how many logs had
// to be left out. When some were, environment.txt says so too (see leftOutLine) and each one leaves
// omittedMarker in its place, so a reader of the archive alone is never told that the history it
// holds is the whole of it.
//
// Every byte written here passes through Redact. This is the only route by which anything
// leaves the machine, so the gate lives at the gate: a caller must never assemble an archive
// of its own. Nothing from Dir() other than the log files is ever included - in particular
// none of the credential files that live beside them.
func Build(opts BuildOptions) (path string, droppedLogs int, err error) {
	// A caller-named target is judged before anything is collected, and never makes a folder.
	out := ""
	if opts.Out != "" {
		if out, err = checkOut(opts.Out, opts.AppVersion, opts.At, opts.Overwrite); err != nil {
			return "", 0, err
		}
	}
	dir := filepath.Join(Dir(), "reports")
	if out == "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", 0, fmt.Errorf("create report folder: %w", err)
		}
	}

	fields := environmentFields(opts.AppVersion, opts.Packaged)
	env, readable := renderEnvironment(fields), renderReport(fields)
	var settings []byte
	if len(opts.SettingsJSON) > 0 {
		settings = RedactBytes(dropUserNames(opts.SettingsJSON))
	}
	logs, err := logFiles()
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", 0, fmt.Errorf("list run logs: %w", err)
	}
	// The logs are chosen before anything is written because the count of those left out belongs
	// in environment.txt and report.txt, which come first. Their lines are reserved at their widest:
	// at most every log, and so is the omission marker each log left out leaves in its place.
	budget := MaxArchiveBytes - len(env) - len(readable) - len(settings) -
		len(leftOutLine(len(logs))) - len(leftOutReportLine(len(logs))) - len(logs)*len(omittedMarker(math.MaxInt64))
	kept, droppedLogs := packLogs(logs, budget)
	if droppedLogs > 0 {
		env += leftOutLine(droppedLogs)
		readable += leftOutReportLine(droppedLogs)
	}

	entries := []archiveEntry{{"environment.txt", []byte(env)}, {"report.txt", []byte(readable)}}
	if len(opts.SettingsJSON) > 0 {
		entries = append(entries, archiveEntry{"settings.json", settings})
	}
	entries = append(entries, kept...)

	if out != "" {
		// The archive is at most MaxArchiveBytes, so it is built in memory and handed over in one piece: created
		// exclusively when nothing is to be replaced, written through a temporary file that is renamed when something is.
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		if err := addEntries(zw, entries); err != nil {
			return "", 0, fmt.Errorf("write archive: %w", err)
		}
		if err := zw.Close(); err != nil {
			return "", 0, fmt.Errorf("finish archive: %w", err)
		}
		if err := deliver(out, buf.Bytes(), opts.Overwrite); err != nil {
			return "", 0, err
		}
		return out, droppedLogs, nil
	}

	f, path, err := createArchive(dir, opts.AppVersion, opts.At)
	if err != nil {
		return "", 0, fmt.Errorf("create archive: %w", err)
	}
	zw := zip.NewWriter(f)
	if err := addEntries(zw, entries); err != nil {
		return "", 0, closeFailed(zw, f, path, err)
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

// addEntries writes every entry into the archive, in order.
func addEntries(zw *zip.Writer, entries []archiveEntry) error {
	for _, e := range entries {
		w, err := zw.Create(e.name)
		if err == nil {
			_, err = w.Write(e.data)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// checkOut resolves a caller-named target (file path or existing folder, APP-COMMANDS item P.1 and 0.3 item V) and
// refuses what cannot be written: an existing file without overwrite, or a missing folder (including a path ending in
// a separator whose folder does not exist). It returns the resolved absolute file path.
func checkOut(out, version string, at time.Time, overwrite bool) (string, error) {
	isFolder := len(out) > 0 && os.IsPathSeparator(out[len(out)-1])
	abs, err := filepath.Abs(out)
	if err != nil {
		return "", fmt.Errorf("resolve output path: %w", err)
	}
	st, err := os.Stat(abs)
	if isFolder || (err == nil && st.IsDir()) {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return "", ErrOutFolderMissing
			}
			return "", fmt.Errorf("check output folder: %w", err)
		}
		if !st.IsDir() {
			return "", ErrOutFolderMissing
		}
		if at.IsZero() {
			at = time.Now()
		}
		target := filepath.Join(abs, archiveName(version, at))
		if _, terr := os.Stat(target); terr == nil {
			if !overwrite {
				return "", ErrOutExists
			}
			return target, nil
		} else if !errors.Is(terr, fs.ErrNotExist) {
			return "", fmt.Errorf("check output file: %w", terr)
		}
		return target, nil
	}
	switch {
	case err == nil && !overwrite:
		return "", ErrOutExists
	case err == nil:
		return abs, nil
	case !errors.Is(err, fs.ErrNotExist):
		return "", fmt.Errorf("check output file: %w", err)
	}
	if dst, derr := os.Stat(filepath.Dir(abs)); derr != nil || !dst.IsDir() {
		return "", ErrOutFolderMissing
	}
	return abs, nil
}

// deliver puts the archive at out. Without overwrite the file is created exclusively - a file that appeared since checkOut
// is still a refusal - and a half-written file is removed; with it the old file is replaced whole, or left as it was.
func deliver(out string, data []byte, overwrite bool) error {
	if overwrite {
		if err := fsutil.WriteFile(out, data, 0o644); err != nil {
			return fmt.Errorf("write archive: %w", err)
		}
		return nil
	}
	f, err := os.OpenFile(out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if errors.Is(err, fs.ErrExist) {
		return ErrOutExists
	}
	if err != nil {
		return fmt.Errorf("create archive: %w", err)
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(out)
		return fmt.Errorf("write archive: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(out)
		return fmt.Errorf("close archive: %w", err)
	}
	return nil
}

// leftOutLine is the key environment.txt gains when run logs did not make it into the archive.
// It is appended after the existing keys: a reader matches keys by name and ignores the ones it
// does not know (DIAGNOSTIC-REPORT 3), so adding one breaks none.
func leftOutLine(n int) string {
	return fmt.Sprintf("logs_left_out=%d\n", n)
}

// leftOutReportLine is the same fact in the labelled report.txt.
func leftOutReportLine(n int) string {
	return fmt.Sprintf("logs left out: %d\n", n)
}

// userNameKeys are the settings that hold a user's own names: the output folder (named after the
// user's work) and the list of recent documents. DIAGNOSTIC-REPORT 8 B keeps both out of the copy.
var userNameKeys = []string{"output", "recent"}

// dropUserNames removes userNameKeys from the settings object. Settings that are not a JSON object
// are returned as they are: Redact still runs over them, and nothing here guesses at their shape.
func dropUserNames(raw []byte) []byte {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return raw
	}
	for _, k := range userNameKeys {
		delete(fields, k)
	}
	out, err := json.MarshalIndent(fields, "", "  ")
	if err != nil {
		return raw
	}
	return out
}

// omittedMarker is DIAGNOSTIC-REPORT 0.15's omission marker for a whole log left out because the
// archive's size budget was reached. It stands under the log's own entry name, so a reader sees
// which run is missing; the line itself carries no path, name or content of the log.
func omittedMarker(sourceBytes int64) string {
	return fmt.Sprintf("[Diag] LOG OMITTED | reason=archive_cap | source_bytes=%d\n", sourceBytes)
}

// packLogs redacts the run logs newest first and keeps as many as fit in budget bytes, reporting
// how many it left out; each one left out is replaced by omittedMarker. The newest log is the run
// the report is about, so it is the last thing to go; an older, smaller log may still fit after a
// larger one was dropped.
func packLogs(logs []logFile, budget int) (kept []archiveEntry, leftOut int) {
	sort.Slice(logs, func(i, j int) bool { return logs[i].name > logs[j].name })
	omit := func(l logFile, sourceBytes int64) {
		leftOut++
		kept = append(kept, archiveEntry{"logs/" + l.name, []byte(omittedMarker(sourceBytes))})
	}
	for _, l := range logs {
		// A log larger than the whole archive can never fit. Deciding that from the listing keeps
		// it out of memory: reading it first and measuring afterwards would spend its full size,
		// twice with the redacted copy, to reach the same answer.
		if l.size > MaxArchiveBytes {
			omit(l, l.size)
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(LogsDir(), l.name))
		if readErr != nil {
			continue
		}
		sourceBytes := int64(len(data))
		data = RedactBytes(data)
		if len(data) > budget {
			omit(l, sourceBytes)
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
