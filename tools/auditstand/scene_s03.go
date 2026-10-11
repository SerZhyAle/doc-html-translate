// scene_s03.go implements S03 reopen-reuse end to end: a first conversion over epub-small, a
// second run that must reopen the finished output through the completion record, a -force run
// that must rebuild - with the CLI-first-output bar of H5, citizen probes during every run
// (property 3, disk/network/message per register row N1) and the P7 consistency assertions.
//
// Cell note: H5's "indistinguishable from the ping round trip" form needs the GUI host's ping
// (S01's instrumented window), so this scene records it as an advisory and the cell closes in
// phase 02 - an advisory never closes a cell by itself (strategic section 5.4).

package main

import (
	"context"
	"doc-html-translate/internal/outputpath"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// The pipeline's reuse log lines (internal/pipeline/pipeline.go). A test pins this file's
// constants to the pipeline's, so a wording change fails here, not on the owner's machine.
const (
	logReused        = "Book already extracted:"
	logForcedRebuild = "Book already extracted, forcing rebuild:"
)

func runSceneS03(env *runEnv, ctx context.Context, cycles int) (*record, error) {
	rec, err := newSceneRecord(env, "S03")
	if err != nil {
		return nil, err
	}
	runDir := env.runDir(rec.RunID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return nil, err
	}
	ctrl := latestControl(env)

	src := filepath.Join(env.fixturesDir, "epub-small.epub")
	for cycle := 1; cycle <= cycles; cycle++ {
		out1 := filepath.Join(runDir, fmt.Sprintf("out-c%d", cycle))
		canaryDir := filepath.Join(runDir, fmt.Sprintf("canary-c%d", cycle))
		profile := filepath.Join(runDir, fmt.Sprintf("profile-c%d", cycle))
		if err := os.MkdirAll(canaryDir, 0o755); err != nil {
			return rec, err
		}

		// Run 1: the cold conversion. First output bar of H5 measured on the pipeline's own
		// first line.
		r1, err := env.execApp(ctx, rec.Binaries["app"].Path,
			[]string{src, "-noopen", "-folder", out1},
			execOptions{ProfileDir: profile, Dir: canaryDir}, rec.Canary)
		if err != nil {
			return rec, err
		}
		marker := filepath.Join(out1, outputpath.MarkerName)
		markerOK := fileExists(marker)
		rec.OutcomeChecks = append(rec.OutcomeChecks, outcomeCheck{
			Injection:        "S03-first-conversion",
			ExitCode:         r1.ExitCode,
			NamedOutcome:     r1.ExitCode == 0,
			BoundaryLogFound: strings.Contains(r1.combinedTail, "Done."),
			StateConsistent:  markerOK && r1.ExitCode == 0,
			Detail:           fmt.Sprintf("wall %.0f ms, first output %.0f ms, marker present: %v", r1.WallMs, r1.FirstOutputMs, markerOK),
		})
		rec.Samples = append(rec.Samples, r1.P4Samples...)

		// Run 2: reopen through the completion record. The log line names the reuse; the wall
		// time is the number H5's reuse form compares against the ping round trip.
		r2, err := env.execApp(ctx, rec.Binaries["app"].Path,
			[]string{src, "-noopen", "-folder", out1},
			execOptions{ProfileDir: profile, Dir: canaryDir}, rec.Canary)
		if err != nil {
			return rec, err
		}
		reused := strings.Contains(r2.combinedTail, logReused) && !strings.Contains(r2.combinedTail, logForcedRebuild)
		rec.OutcomeChecks = append(rec.OutcomeChecks, outcomeCheck{
			Injection:        "S03-reopen",
			ExitCode:         r2.ExitCode,
			NamedOutcome:     r2.ExitCode == 0 && reused,
			BoundaryLogFound: reused,
			StateConsistent:  markerOK,
			Detail:           fmt.Sprintf("reopen wall %.0f ms, reuse line present: %v", r2.WallMs, reused),
		})

		// Run 3: -force rebuilds, and says so.
		r3, err := env.execApp(ctx, rec.Binaries["app"].Path,
			[]string{src, "-noopen", "-force", "-folder", out1},
			execOptions{ProfileDir: profile, Dir: canaryDir}, rec.Canary)
		if err != nil {
			return rec, err
		}
		forced := strings.Contains(r3.combinedTail, logForcedRebuild)
		rec.OutcomeChecks = append(rec.OutcomeChecks, outcomeCheck{
			Injection:        "S03-force-rebuild",
			ExitCode:         r3.ExitCode,
			NamedOutcome:     r3.ExitCode == 0 && forced,
			BoundaryLogFound: forced,
			StateConsistent:  fileExists(marker),
			Detail:           fmt.Sprintf("force wall %.0f ms, forced-rebuild line present: %v", r3.WallMs, forced),
		})
		rec.Samples = append(rec.Samples, r3.P4Samples...)

		// Property 3 (N1's applicable sub-probes): foreign disk, network and message probes run
		// concurrently with a fourth, throwaway conversion so the comparison against control
		// measures app-adjacent machine work (the register row N1 rules out the clipboard,
		// hooks and broadcast sub-probes for the CLI path).
		citizenMs := map[string][]float64{}
		var citizenMu sync.Mutex
		var citizenErrors []string
		citizenCtx, cancel := context.WithCancel(ctx)
		done := make(chan struct{})
		var wg sync.WaitGroup
		for _, name := range []string{"P6.fsync-writer", "P6.http-fetch", "P6.message-latency"} {
			p, ok := probeByName(name)
			if !ok {
				continue
			}
			wg.Add(1)
			go func(name string, fn func(context.Context, time.Duration, func(float64)) error) {
				defer wg.Done()
				err := fn(citizenCtx, 0, func(ms float64) { citizenMu.Lock(); citizenMs[name] = append(citizenMs[name], ms); citizenMu.Unlock() })
				if err != nil {
					citizenMu.Lock()
					citizenErrors = append(citizenErrors, name+": "+err.Error())
					citizenMu.Unlock()
				}
			}(name, p.run)
		}
		// One conversion runs while the citizens measure.
		r4, err := env.execApp(ctx, rec.Binaries["app"].Path,
			[]string{src, "-noopen", "-force", "-folder", filepath.Join(runDir, fmt.Sprintf("out-c%d-citizen", cycle))},
			execOptions{ProfileDir: profile, Dir: canaryDir}, rec.Canary)
		cancel()
		go func() { wg.Wait(); close(done) }()
		<-done
		if len(citizenErrors) > 0 {
			return rec, cannotVerify(fmt.Errorf("citizen probe failed: %s", strings.Join(citizenErrors, "; ")))
		}
		_ = r4
		if err != nil {
			return rec, err
		}

		// Budget verdicts. H5's absolute CLI bar is measurable here; the reuse-vs-ping form
		// lands with the GUI scenes, so this cell stays open with an advisory row.
		h5 := budgetCheck(findBudget("H5"), fmt.Sprintf("first output %.0f ms (bar 1500 ms); reopen %.0f ms", r1.FirstOutputMs, r2.WallMs),
			r1.FirstOutputMs, ctrl, "P2.host")
		if h5.Verdict == "advisory" {
			h5.Observed += "; the reuse-vs-ping comparison needs the GUI host ping (phase 02)"
		}
		rec.Cells = append(rec.Cells, cellResult{
			Scene: "S03", Property: propResponsiveness, State: "open",
			Budgets:  []budgetVerdict{h5},
			Evidence: fmt.Sprintf("runs 1-4 of cycle %d; marker present: %v", cycle, markerOK),
		})
		// Property 3: disk/network/message citizen latencies against control.
		if len(citizenMs) > 0 {
			cell := cellResult{Scene: "S03", Property: propNoSystemBlocking, State: "open", Evidence: "citizen probes overlap a forced rebuild"}
			for _, name := range []string{"P6.fsync-writer", "P6.http-fetch", "P6.message-latency"} {
				m := summarize(name, citizenMs[name])
				rec.Measurements = append(rec.Measurements, m)
				b := findBudget("H10")
				verdict := budgetCheck(b, fmt.Sprintf("p95 %.2f ms", m.P95Ms), m.P95Ms, ctrl, name)
				cell.Budgets = append(cell.Budgets, verdict)
			}
			rec.Cells = append(rec.Cells, cell)
		}

		// Temp residue (the leak cell's consistency half): the redirected temp must be empty.
		residue, _, err := tempResidue(filepath.Join(profile, "Temp"))
		if err != nil {
			return rec, err
		}
		// The canary scan: the token travelled in the environment; logs and outputs must not
		// echo it (P7).
		var scan []string
		if log, err := findRunLog(profile); err == nil {
			scan = append(scan, log)
		}
		_ = filepath.Walk(out1, func(path string, info fs.FileInfo, err error) error {
			if err == nil && !info.IsDir() && !strings.HasSuffix(path, ".epub") {
				scan = append(scan, path)
			}
			return nil
		})
		hits, where := countCanary(scan, rec.Canary)
		appendNote(rec, "cycle %d: temp residue %d file(s) under redirected temp; canary hits %d %v",
			cycle, len(residue), hits, where)
	}

	dir := env.runDir(rec.RunID)
	if _, err := saveRecord(dir, rec); err != nil {
		return rec, err
	}
	return rec, nil
}

// findBudget returns a budget by id.
func findBudget(id string) budget {
	for _, b := range budgets {
		if b.id == id {
			return b
		}
	}
	return budget{id: id}
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
