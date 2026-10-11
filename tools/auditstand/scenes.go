// scenes.go is the runner side of the scene catalog: which scene steps exist, how a run is
// orchestrated, and what closing a matrix cell means. A scene run that cannot judge anything
// exits 2; a run that measures but leaves budgets unevaluated exits 3 and closes no cell by
// itself (strategic section 5.4).

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// sceneFunc runs one scene and returns its record. Implementations live beside this file
// (scene_s03.go, scene_s16.go).
type sceneFunc func(env *runEnv, ctx context.Context, cycles int) (*record, error)

// implementedScenes maps scene ids to their implementation. A scene absent here is declared but
// not implemented: the runner says so with exit 2 and names the owning phase - recorded, never
// silent.
var implementedScenes = map[sceneID]sceneFunc{
	"S03": runSceneS03,
	"S08": runSceneS08,
	"S15": runSceneS15,
	"S16": runSceneS16,
}

// sceneResult is what main.go turns into the verdict.
type sceneResult struct {
	verdict verdict
	detail  string
	runID   string
}

// runScene performs the pre-flight discipline (control present, fixtures present, isolation
// honest) and then hands over to the scene's own steps.
func runScene(env *runEnv, id sceneID, cycles int) (res sceneResult, resultErr error) {
	s, ok := sceneByID(id)
	if !ok {
		return res, cannotVerify(fmt.Errorf("unknown scene %s (see -Plan for the catalog)", id))
	}
	fn, implemented := implementedScenes[id]
	if !implemented {
		return res, cannotVerify(fmt.Errorf("scene %s (%s) has no implemented steps yet; it is owned by phase %02d of DEV/plan/113_2026-10-11_dynamic-behaviour-audit/", id, s.title, s.phase))
	}

	// Registry-touching scenes refuse to start until the Level-B key backup exists (Phase 0
	// step 00.05). No backup, no run - the release gate's discipline.
	if env.isolation != "B-env" && env.isolation != "B-full" {
		return res, cannotVerify(fmt.Errorf("isolation %q is not implemented; a label alone cannot prove a disposable Windows user", env.isolation))
	}
	if s.touchesRegistry && env.isolation != "B-full" {
		return res, cannotVerify(fmt.Errorf("scene %s can write the registry; the named-key backup of strategic section 7 is not configured yet (phase 0 step 00.05)", id))
	}

	// Scenes pass only against a same-day control (strategic section 6).
	ctrl := latestControl(env)
	if ctrl == nil {
		return res, cannotVerify(fmt.Errorf("no valid control run for today; run -Control first - a scene passes only when the relative form (2x control p95) holds too"))
	}

	missing, corrupt, err := checkFixtures(env.fixturesDir)
	if err != nil {
		return res, cannotVerify(err)
	}
	if len(missing) > 0 {
		return res, cannotVerify(fmt.Errorf("fixtures missing (%s); run -Fixtures first", strings.Join(missing, ", ")))
	}
	if len(corrupt) > 0 {
		return res, cannotVerify(fmt.Errorf("fixture hashes failed: %s", strings.Join(corrupt, ", ")))
	}
	if err := containedPath(filepath.Join(repoRoot(), "temp"), env.artifactsDir); err != nil {
		return res, cannotVerify(err)
	}
	if err := containedPath(filepath.Join(repoRoot(), "temp"), env.fixturesDir); err != nil {
		return res, cannotVerify(err)
	}
	if fileExists(filepath.Join(env.artifactsDir, "HALTED")) {
		return res, cannotVerify(fmt.Errorf("campaign halted after failed profile recovery; inspect HALTED and the recovery manifest"))
	}
	env.sceneRunID = newRunID(string(id))
	defer func() { env.sceneRunID = ""; env.recovery = nil; env.profiles = nil }()
	roots, err := registrationRoots(repoRoot(), env.appPath, env.uiPath)
	if err != nil {
		return res, cannotVerify(err)
	}
	if env.isolation == "B-full" {
		store, err := newRegistryStore()
		if err != nil {
			return res, cannotVerify(err)
		}
		env.recovery, err = backupProfile(store, env.runDir(env.sceneRunID), roots, nil)
		if err != nil {
			return res, cannotVerify(fmt.Errorf("registry backup: %w", err))
		}
	}
	if _, err := env.resolveApp(); err != nil {
		return res, cannotVerify(err)
	}
	before, err := treeFingerprint()
	if err != nil {
		return res, cannotVerify(err)
	}
	rec := &record{RunID: env.sceneRunID, Kind: "scene", Scene: string(id), SceneTitle: s.title, CreatedAt: time.Now(), Isolation: env.isolation, Valid: true}
	env.watch = startValidityWatch()
	// This finalizer covers scene errors as well as successes. Failed restoration is a P0
	// finding, preserves the profile and exports, and prevents another scene from starting.
	defer func() {
		if reasons := env.watch.finish(); len(reasons) > 0 {
			rec.Valid = false
			rec.InvalidReasons = append(rec.InvalidReasons, reasons...)
			if summary := env.watch.offenderSummary(); summary != "" {
				appendNote(rec, "%s", summary)
			}
		}
		env.watch = nil
		rec.ControlRef = ctrl.RunID
		rec.Cycles = cycles
		rec.TreeBefore = before
		if resultErr != nil {
			rec.Valid = false
			rec.InvalidReasons = append(rec.InvalidReasons, resultErr.Error())
		}
		if env.recovery != nil {
			rec.RecoveryManifest = env.recovery.path
			rec.RecoverySHA256 = env.recovery.hash
			if err := env.recovery.restore(); err != nil {
				rec.Valid = false
				rec.Findings = append(rec.Findings, runFinding{"P0", "T", "H13", err.Error()})
				_ = os.WriteFile(filepath.Join(env.artifactsDir, "HALTED"), []byte(rec.RunID+"\n"+err.Error()), 0o600)
				res.verdict = verdictFail
				res.detail = err.Error()
				resultErr = nil
			} else {
				rec.RestoreVerified = true
			}
		}
		if env.recovery == nil || rec.RestoreVerified {
			if err := env.cleanupProfiles(); err != nil {
				rec.Valid = false
				rec.InvalidReasons = append(rec.InvalidReasons, err.Error())
			}
		}
		after, err := treeFingerprint()
		rec.TreeAfter = after
		if err != nil || after != before {
			rec.Valid = false
			rec.InvalidReasons = append(rec.InvalidReasons, "repository tree changed or could not be verified after the run")
		}
		for _, bin := range rec.Binaries {
			hash, err := sha256File(bin.Path)
			if err != nil || hash != bin.SHA256 {
				rec.Valid = false
				rec.InvalidReasons = append(rec.InvalidReasons, "binary changed mid-run")
			}
		}
		if !rec.Valid && res.verdict != verdictFail {
			res.verdict = verdictCouldNotVerify
			res.detail = strings.Join(rec.InvalidReasons, "; ")
		}
		res.runID = rec.RunID
		if _, err := saveRecord(env.runDir(rec.RunID), rec); err != nil {
			resultErr = cannotVerify(err)
		}
	}()

	if cycles < 1 {
		cycles = 1
	}
	sceneRec, err := fn(env, context.Background(), cycles)
	if sceneRec != nil {
		rec = sceneRec
	}
	if err != nil {
		return res, err
	}
	rec.ControlRef = ctrl.RunID
	res.runID = rec.RunID

	// Verdict from what the scene recorded: any failed budget or violated outcome check is a
	// finding (exit 1); fully measured cells close, partial ones keep the run at exit 3.
	var fails []string
	var advisories []string
	allMeasured := len(rec.Cells) == 5
	for _, c := range rec.Cells {
		for _, b := range c.Budgets {
			switch b.Verdict {
			case "fail":
				fails = append(fails, fmt.Sprintf("%s: %s breached (%s)", b.ID, b.Limit, b.Observed))
			case "advisory":
				advisories = append(advisories, fmt.Sprintf("%s advisory: %s", b.ID, b.Observed))
			}
		}
		if c.State != "measured" {
			allMeasured = false
		}
	}
	for _, oc := range rec.OutcomeChecks {
		if !oc.NamedOutcome || !oc.BoundaryLogFound || oc.CanaryHits > 0 || !oc.StateConsistent || (oc.LocaleChecked && !oc.MessageInLocale) {
			fails = append(fails, fmt.Sprintf("outcome check %s violated (named=%v boundary_log=%v canary=%d consistent=%v locale_checked=%v in_locale=%v)",
				oc.Injection, oc.NamedOutcome, oc.BoundaryLogFound, oc.CanaryHits, oc.StateConsistent, oc.LocaleChecked, oc.MessageInLocale))
		}
	}
	for _, sc := range rec.StateChecks {
		if !sc.ok() {
			fails = append(fails, fmt.Sprintf("toggle %s: observable=%v reverted=%v (%s)", sc.Toggle, sc.Observable, sc.Reverted, sc.Detail))
		}
	}
	switch {
	case len(fails) > 0:
		res.verdict = verdictFail
		res.detail = strings.Join(fails, "; ")
	case !allMeasured || len(advisories) > 0:
		res.verdict = verdictAdvisory
		res.detail = strings.Join(advisories, "; ")
		if !allMeasured {
			res.detail = "cells remain open; " + res.detail
		}
	default:
		res.verdict = verdictPass
	}
	return res, nil
}

// budgetCheck is the two-form evaluation (RUN_RECORD.md): the absolute hypothesis and, when the
// probe has a same-day control p95, the relative 2x form. The hard bar (H1's watchdog tick) is
// absolute-only and P1 by definition.
func budgetCheck(b budget, observed string, ms float64, ctrl *record, probe string) budgetVerdict {
	v := budgetVerdict{ID: b.id, Observed: observed, Limit: b.limit}
	absolutePass := ms <= absoluteBudgetMsOf(b)
	if b.id == "H1" && ms > 5000 {
		v.Verdict = "fail"
		return v
	}
	if p95, ok := controlP95(ctrl, probe); ok {
		v.ControlP95Ms = p95
		within := ms <= 2*p95
		v.Within2xControl = &within
		v.Verdict = passFail(absolutePass && within)
		return v
	}
	// No control number for the probe's relative form: an advisory, never a pass.
	if !absolutePass {
		v.Verdict = "fail"
	} else {
		v.Verdict = "advisory"
	}
	return v
}

// absoluteBudgetMsOf returns the hypothesis's absolute bar in ms for latency budgets that have
// one (the idle p95 form; scene runs compare their scene-form number against it).
func absoluteBudgetMsOf(b budget) float64 {
	switch b.id {
	case "H1":
		return 500 // the under-load ceiling a conversion-sharing scene is judged against
	case "H2":
		return 50 // only S14 may use the 100 ms storm ceiling
	case "H3":
		return 700 // under-load ceiling
	case "H4":
		return 2000
	case "H5":
		return 1500
	case "H6":
		return 1500
	case "H7":
		return 5000
	default:
		return 1e18 // non-latency budgets (H8-H14) are scored by their own assertions
	}
}

func passFail(ok bool) string {
	if ok {
		return "pass"
	}
	return "fail"
}

// note records a free-form observation into the record's notes column.
func appendNote(rec *record, format string, args ...interface{}) {
	line := fmt.Sprintf(format, args...)
	if rec.Notes != "" {
		rec.Notes += "\n"
	}
	rec.Notes += line
}

// sortedFixtureNames is a deterministic artifact listing helper for consistency checks.
func sortedFixtureNames(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names
}

// newSceneRecord starts a scene record with the common fields filled.
func newSceneRecord(env *runEnv, id sceneID) (*record, error) {
	s, ok := sceneByID(id)
	if !ok {
		return nil, cannotVerify(fmt.Errorf("unknown scene %s", id))
	}
	appBin, err := env.resolveApp()
	if err != nil {
		return nil, cannotVerify(err)
	}
	rec := &record{
		RunID:      newRunID(string(id)),
		Kind:       "scene",
		CreatedAt:  time.Now(),
		Scene:      string(id),
		SceneTitle: s.title,
		Isolation:  env.isolation,
		Valid:      true,
		Binaries:   map[string]binaryID{"app": appBin},
		Canary:     fmt.Sprintf("dht-audit-canary-%d", time.Now().UnixNano()),
		Profile:    map[string]string{},
	}
	if env.sceneRunID != "" {
		rec.RunID = env.sceneRunID
	}
	if env.recovery != nil {
		rec.RecoveryManifest = env.recovery.path
		rec.RecoverySHA256 = env.recovery.hash
		if _, err := saveRecord(env.runDir(rec.RunID), rec); err != nil {
			return nil, err
		}
	}
	return rec, nil
}
