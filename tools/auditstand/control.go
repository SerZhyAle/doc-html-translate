// control.go implements -Control (strategic section 6): the whole probe set on the idle desktop
// with no app started, same profile, same durations. It writes control.json - the probes' own
// jitter and the machine's own noise - and refuses (exit 2) when a probe's jitter alone exceeds
// the validity threshold, because a scene run on such a machine would measure the noise. A
// background load the operator declares by image name (-declared-load) is tolerated by the
// foreign-CPU rule and recorded as an advisory instead: a weaker, but named, baseline.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// jitterFraction is the invalid-run rule's threshold: a probe's own p95 above this fraction of
// its budget's absolute number makes the machine too noisy for any scene.
const jitterFraction = 0.2

// absoluteBudgetMs maps budget ids to their absolute millisecond bar, where one exists. Only
// probes feeding a budget with an absolute side are jitter-gated; the relative-only budgets
// (H9-H12) have no absolute number to breach.
var absoluteBudgetMs = map[string]float64{
	"H1": 50, // idle p95; under load the ceiling is 500 but the control machine is idle
	"H2": 50,
	"H3": 250,
}

// controlResult is what main.go turns into the verdict.
type controlResult struct {
	verdict     verdict
	detail      string
	runID       string
	unavailable []string // named probes that could not run, with reasons
}

// runControl executes every available probe for its duration with no app running, records the
// distributions, and applies the jitter gate.
func runControl(env *runEnv, perProbe time.Duration) (controlResult, error) {
	if runtime.GOOS != "windows" {
		return controlResult{}, cannotVerify(fmt.Errorf("desktop control needs Windows"))
	}
	if perProbe <= 0 {
		return controlResult{}, cannotVerify(fmt.Errorf("probe duration must be positive"))
	}
	if err := containedPath(filepath.Join(repoRoot(), "temp"), env.artifactsDir); err != nil {
		return controlResult{}, cannotVerify(err)
	}
	for _, name := range []string{"doc-html-ui.exe", "doc-html-translate.exe", filepath.Base(env.appPath), filepath.Base(env.uiPath)} {
		if name != "." && len(findProcessesByName(name)) > 0 {
			return controlResult{}, cannotVerify(fmt.Errorf("control requires no app running: %s is active", name))
		}
	}
	res := controlResult{}
	rec := &record{
		RunID:        newRunID("control"),
		Kind:         "control",
		CreatedAt:    time.Now(),
		Isolation:    "n/a",
		Valid:        true,
		DeclaredLoad: env.declaredLoad,
	}
	res.runID = rec.RunID
	watch := startValidityWatch(env.declaredLoad)
	for _, p := range probeRegistry() {
		if ok, reason := p.available(); !ok {
			res.unavailable = append(res.unavailable, fmt.Sprintf("%s: %s", p.name, reason))
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), perProbe)
		var ms []float64
		err := p.run(ctx, perProbe, func(v float64) { ms = append(ms, v) })
		cancel()
		if err != nil {
			rec.Valid = false
			rec.InvalidReasons = append(rec.InvalidReasons, fmt.Sprintf("probe %s failed: %v", p.name, err))
			continue
		}
		if len(ms) == 0 {
			res.unavailable = append(res.unavailable, fmt.Sprintf("%s: no samples in %s", p.name, perProbe))
			continue
		}
		m := summarize(p.name, ms)
		rec.Measurements = append(rec.Measurements, m)

		// The jitter gate: 20% of the tightest absolute budget this probe feeds.
		limit, budget := jitterLimit(p)
		if limit > 0 && m.P95Ms > limit {
			rec.Valid = false
			rec.InvalidReasons = append(rec.InvalidReasons,
				fmt.Sprintf("%s p95 %.1f ms above %.0f%% of %s (%.1f ms): the machine is too noisy for scenes today",
					p.name, m.P95Ms, jitterFraction*100, budget, limit))
		}
	}
	dir := env.runDir(rec.RunID)
	if reasons := watch.finish(); len(reasons) > 0 {
		rec.Valid = false
		rec.InvalidReasons = append(rec.InvalidReasons, reasons...)
		if summary := watch.offenderSummary(); summary != "" {
			appendNote(rec, "%s", summary)
		}
	}
	declared := watch.declaredLoadSummary()
	if declared != "" {
		appendNote(rec, "advisory: %s", declared)
	}
	if len(rec.Measurements) == 0 {
		rec.Valid = false
		rec.InvalidReasons = append(rec.InvalidReasons, "no probe supplied any samples")
	}
	for _, u := range res.unavailable {
		appendNote(rec, "unavailable: %s", u)
	}
	if _, err := saveRecord(dir, rec); err != nil {
		return res, err
	}
	b, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return res, err
	}
	if err := os.WriteFile(filepath.Join(dir, "control.json"), b, 0o600); err != nil {
		return res, err
	}

	fmt.Printf("control run %s (%d probe(s) measured, %d unavailable)\n", rec.RunID, len(rec.Measurements), len(res.unavailable))
	for _, m := range rec.Measurements {
		fmt.Printf("  %-22s n=%-5d p50=%8.2f ms  p95=%8.2f ms  max=%8.2f ms\n", m.Probe, m.N, m.P50Ms, m.P95Ms, m.MaxMs)
	}
	for _, u := range res.unavailable {
		fmt.Printf("  unavailable: %s\n", u)
	}
	for _, r := range rec.InvalidReasons {
		fmt.Printf("  INVALID: %s\n", r)
	}
	if summary := watch.offenderSummary(); summary != "" {
		fmt.Printf("  %s\n", summary)
	}
	if declared != "" {
		fmt.Printf("  advisory: %s\n", declared)
	}
	fmt.Printf("control.json: %s\n", dir)

	if !rec.Valid {
		res.verdict = verdictCouldNotVerify
		res.detail = fmt.Sprintf("%d validity violation(s); no scene may run until the machine is quiet", len(rec.InvalidReasons))
		return res, nil
	}
	res.verdict, res.detail = controlAdvisories(len(res.unavailable), rec.DeclaredLoad)
	res.runID = rec.RunID
	return res, nil
}

// controlAdvisories folds a valid control's advisories into its verdict: unavailable probes and
// a declared background load each keep it at PASS WITH ADVISORIES; neither invalidates it.
func controlAdvisories(unavailable int, declared []string) (verdict, string) {
	var parts []string
	if unavailable > 0 {
		parts = append(parts, fmt.Sprintf("%d unavailable probe(s)", unavailable))
	}
	if len(declared) > 0 {
		parts = append(parts, declaredLoadAdvisory(declared))
	}
	if len(parts) == 0 {
		return verdictPass, ""
	}
	return verdictAdvisory, strings.Join(parts, "; ")
}

// jitterLimit returns the smallest absolute jitter ceiling among the probe's budgets.
func jitterLimit(p latencyProbe) (limit float64, budget string) {
	for _, b := range p.budgetNames {
		if abs, ok := absoluteBudgetMs[b]; ok {
			frac := abs * jitterFraction
			if limit == 0 || frac < limit {
				limit = frac
				budget = b
			}
		}
	}
	return limit, budget
}

// latestControl returns today's newest valid control record, or nil. Scenes pass only against
// a same-day control (strategic section 6: the control runs before any scene).
func latestControl(env *runEnv) *record {
	recs, _ := loadAllRecords(env.artifactsDir)
	var best *record
	today := time.Now().Format("20060102")
	for _, r := range recs {
		if r.Kind != "control" {
			continue
		}
		if len(r.RunID) < 8 || r.RunID[:8] != today {
			continue
		}
		if best == nil || r.CreatedAt.After(best.CreatedAt) {
			best = r
		}
	}
	if best != nil && (!best.Valid || len(best.InvalidReasons) > 0 || len(best.Measurements) == 0) {
		return nil
	}
	return best
}

// controlP95 returns a probe's p95 from the control record, or false.
func controlP95(ctrl *record, probe string) (float64, bool) {
	if ctrl == nil {
		return 0, false
	}
	for _, m := range ctrl.Measurements {
		if m.Probe == probe && m.N > 0 && m.P95Ms > 0 {
			return m.P95Ms, true
		}
	}
	return 0, false
}
