// record.go holds the run-record format of DEV/plan/113_2026-10-11_dynamic-behaviour-audit/
// RUN_RECORD.md. The stand is the only writer; the summary is a reader. Field names are the
// contract - change them together with that document and bump Format.

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// recordFormat is the version the reader refuses unknown values of.
const recordFormat = 2

// sample is one observation of a probe at a moment inside a run.
type sample struct {
	TMs    int64                  `json:"t_ms"`
	Probe  string                 `json:"probe"`
	PID    int                    `json:"pid,omitempty"`
	Fields map[string]interface{} `json:"fields"`
}

// measurement is one named timing observation with its distribution summary.
type measurement struct {
	Probe  string     `json:"probe"`
	N      int        `json:"n"`
	P50Ms  float64    `json:"p50_ms"`
	P95Ms  float64    `json:"p95_ms"`
	MaxMs  float64    `json:"max_ms"`
	Unit   string     `json:"unit,omitempty"` // "ms" unless said otherwise
	Slopes *slopeJSON `json:"slopes,omitempty"`
}

// slopeJSON carries the P5 leak statistic for one resource inside a record.
type slopeJSON struct {
	Resource string  `json:"resource"`
	N        int     `json:"n"`
	Slope    float64 `json:"slope_per_sample"`
	CILo     float64 `json:"ci95_lo"`
	CIHi     float64 `json:"ci95_hi"`
	Leak     bool    `json:"leak"`
}

// budgetVerdict is one budget's evaluation inside a closed cell.
type budgetVerdict struct {
	ID              string  `json:"id"`
	Verdict         string  `json:"verdict"` // "pass" | "fail" | "advisory"
	Observed        string  `json:"observed"`
	Limit           string  `json:"limit"`
	ControlP95Ms    float64 `json:"control_p95_ms,omitempty"`
	Within2xControl *bool   `json:"within_2x_control,omitempty"`
}

// cellResult is one matrix cell a run closes (or leaves open, recorded as partial).
type cellResult struct {
	Scene    string          `json:"scene"`
	Property int             `json:"property"`
	State    string          `json:"state"` // "measured" | "open" (a partial run never fakes measured)
	Budgets  []budgetVerdict `json:"budgets,omitempty"`
	Evidence string          `json:"evidence,omitempty"`
}

// outcomeCheck is one P7 record: an injected failure or a completed flow, with the five H8
// assertions.
type outcomeCheck struct {
	Injection        string `json:"injection"`
	ExitCode         int    `json:"exit_code"`
	NamedOutcome     bool   `json:"named_outcome"`
	BoundaryLogFound bool   `json:"boundary_log_found"`
	CanaryHits       int    `json:"canary_hits"`
	CanaryWhere      string `json:"canary_where,omitempty"`
	StateConsistent  bool   `json:"state_consistent"`
	// LocaleChecked is set when the run used a non-English interface language and the scene
	// asserted H8's fifth part: the outcome message must be present in that language.
	LocaleChecked   bool   `json:"locale_checked,omitempty"`
	MessageInLocale bool   `json:"message_in_locale,omitempty"`
	Detail          string `json:"detail,omitempty"`
}

// binaryID identifies one binary a run started, with its content hash.
type binaryID struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// record is one run's record: kind "scene" or "control".
type record struct {
	Format           int                 `json:"format"`
	RunID            string              `json:"run_id"`
	Kind             string              `json:"kind"`
	CreatedAt        time.Time           `json:"created_at"`
	Scene            string              `json:"scene,omitempty"`
	SceneTitle       string              `json:"scene_title,omitempty"`
	Cycles           int                 `json:"cycles,omitempty"`
	Isolation        string              `json:"isolation"`
	Valid            bool                `json:"valid"`
	InvalidReasons   []string            `json:"invalid_reasons"`
	DeclaredLoad     []string            `json:"declared_load,omitempty"` // tolerated named background load; a scene carries its control's
	ControlRef       string              `json:"control_ref,omitempty"`
	Binaries         map[string]binaryID `json:"binaries,omitempty"`
	Profile          map[string]string   `json:"profile,omitempty"`
	Canary           string              `json:"canary,omitempty"`
	Samples          []sample            `json:"samples"`
	Measurements     []measurement       `json:"measurements"`
	OutcomeChecks    []outcomeCheck      `json:"outcome_checks"`
	StateChecks      []toggleCheck       `json:"state_checks,omitempty"` // P8, one per exercised GUI toggle
	Cells            []cellResult        `json:"cells"`
	Notes            string              `json:"notes"`
	RecoveryManifest string              `json:"recovery_manifest,omitempty"`
	RecoverySHA256   string              `json:"recovery_sha256,omitempty"`
	RestoreVerified  bool                `json:"restore_verified"`
	TreeBefore       string              `json:"tree_before,omitempty"`
	TreeAfter        string              `json:"tree_after,omitempty"`
	Findings         []runFinding        `json:"findings,omitempty"`
	BaselineApproved bool                `json:"baseline_approved,omitempty"`
}

type runFinding struct {
	Severity string `json:"severity"`
	Proof    string `json:"proof"`
	Budget   string `json:"budget"`
	Detail   string `json:"detail"`
}

// couldNotVerifyError marks a run that stopped before it could judge anything: a missing
// fixture, an unimplemented scene, an isolation refusal. Exit 2, never 1 - "the input was not
// there" is not "the input was judged and found defective" (CHECK-VERDICT rule 1).
type couldNotVerifyError struct{ err error }

func (e couldNotVerifyError) Error() string { return e.err.Error() }
func (e couldNotVerifyError) Unwrap() error { return e.err }

func cannotVerify(err error) error {
	if err == nil {
		return nil
	}
	return couldNotVerifyError{err}
}

// newRunID names a run: timestamp, kind or scene.
func newRunID(kindOrScene string) string {
	return time.Now().Format("20060102-150405.000000000") + "-" + kindOrScene
}

// saveRecord writes record.json under the run's artifact directory.
func saveRecord(dir string, r *record) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	r.Format = recordFormat
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	p := filepath.Join(dir, "record.json")
	if err := os.WriteFile(p, b, 0o644); err != nil {
		return "", err
	}
	return p, nil
}

// loadRecord reads one record.json and refuses unknown formats.
func loadRecord(path string) (*record, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r record
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if r.Format != recordFormat {
		return nil, fmt.Errorf("%s: unknown record format %d (want %d)", path, r.Format, recordFormat)
	}
	return &r, nil
}

// loadAllRecords walks the artifact root and returns every readable record.
func loadAllRecords(root string) ([]*record, []error) {
	var recs []*record
	var errs []error
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, []error{err}
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p := filepath.Join(root, e.Name(), "record.json")
		r, err := loadRecord(p)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		recs = append(recs, r)
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].RunID < recs[j].RunID })
	return recs, errs
}

// percentile of a sorted copy of xs at the given fraction (nearest-rank).
func percentile(xs []float64, p float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	idx := int(math.Ceil(p*float64(len(s)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(s) {
		idx = len(s) - 1
	}
	return s[idx]
}

// summarize builds a measurement from raw millisecond observations.
func summarize(probe string, ms []float64) measurement {
	return measurement{
		Probe: probe,
		N:     len(ms),
		P50Ms: percentile(ms, 0.50),
		P95Ms: percentile(ms, 0.95),
		MaxMs: percentile(ms, 1.0),
		Unit:  "ms",
	}
}

// sha256File hashes a file for the record's binaries block.
func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
