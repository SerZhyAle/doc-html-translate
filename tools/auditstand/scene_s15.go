// scene_s15.go implements S15 big-inputs: the inputs that sit at or beyond the published budgets
// of ticket 12 (internal/limits) and of ticket 81, converted for real. The cap decisions are not
// findings - an input over a cap is scored as the refusal working. What the scene measures is the
// consequence: the refusal's exit code and message, the process's peak private bytes against
// ticket 12's criterion ("converting a 2 GB CBZ keeps peak memory well under 1 GB"; 1 GiB is the
// hard line here, the number is recorded either way) and whether anything is left behind.
//
// Two members read the multi-GiB fixtures built by `-Fixtures -Big`; without them they are
// recorded as pending, never skipped silently.

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// peakMemoryLine is ticket 12's criterion 3 as a hard ceiling: the 386 build has a 2 GB address
// space, and the criterion says "well under 1 GB". The recorded number is what the owner judges
// "well under" against; crossing the line itself is a fail.
const peakMemoryLine = 1 << 30

type s15Member struct {
	name    string
	fixture string // file name under the fixture root
	big     bool   // built only by -Fixtures -Big
	args    []string
	// wantExit judges the exit code; label names the expectation.
	wantExit func(code int) bool
	label    string
	// boundary: any phrase must be on the console. Success members name their own summary line.
	boundary []string
	// complete says whether the run must leave a completion record (a converted book) or none
	// (a refusal).
	complete bool
	timeout  time.Duration
}

var s15Members = []s15Member{
	{
		name: "rtf-nested-500", fixture: "rtf-nested.rtf",
		wantExit: func(c int) bool { return c == 0 }, label: "exit 0 (depth 500 is under the nesting cap)",
		boundary: []string{"Paragraphs:"}, complete: true, timeout: 2 * time.Minute,
	},
	{
		name: "tiff-multiframe", fixture: "tiff-multiframe.tif",
		wantExit: func(c int) bool { return c == 0 }, label: "exit 0 (every frame becomes a page)",
		boundary: []string{"3-page TIFF"}, complete: true, timeout: 2 * time.Minute,
	},
	{
		name: "tiff-over-cap", fixture: "tiff-over-cap.tif",
		wantExit: func(c int) bool { return c != 0 }, label: "non-zero (refused from the header, nothing decoded)",
		boundary: []string{"above the limit of"}, complete: false, timeout: 2 * time.Minute,
	},
	{
		name: "comic-entry-over-cap", fixture: "comic-big-2gb.cbz", big: true,
		wantExit: func(c int) bool { return c != 0 }, label: "non-zero (the one 2 GiB entry is over the page cap; no page is left)",
		boundary: []string{"the limit for one file"}, complete: false, timeout: 10 * time.Minute,
	},
	{
		name: "comic-2gb-pages", fixture: "comic-2gb-pages.cbz", big: true,
		wantExit: func(c int) bool { return c == 0 }, label: "exit 0 (2.0 GiB of valid pages is within every budget)",
		boundary: []string{fmt.Sprintf("Pages: %d", comicPagesCount)}, complete: true, timeout: 60 * time.Minute,
	},
}

func runSceneS15(env *runEnv, ctx context.Context, cycles int) (*record, error) {
	rec, err := newSceneRecord(env, "S15")
	if err != nil {
		return nil, err
	}
	runDir := env.runDir(rec.RunID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return nil, err
	}
	complete := true
	for cycle := 1; cycle <= cycles; cycle++ {
		for _, m := range s15Members {
			src := filepath.Join(env.fixturesDir, m.fixture)
			if !fileExists(src) {
				complete = false
				appendNote(rec, "member %s pending: fixture %s absent (build it with -Fixtures -Big)", m.name, m.fixture)
				continue
			}
			oc, peak, err := runS15Member(env, ctx, rec, m, src, filepath.Join(runDir, fmt.Sprintf("c%d", cycle)))
			if err != nil {
				complete = false
				appendNote(rec, "member %s could not run: %v", m.name, err)
				continue
			}
			rec.OutcomeChecks = append(rec.OutcomeChecks, oc)
			rec.Measurements = append(rec.Measurements, peak)
		}
	}

	// Property 4 evidence (ticket 12's criterion): the peak of every member against the line.
	if len(rec.Measurements) > 0 {
		worst := 0.0
		for _, m := range rec.Measurements {
			if m.MaxMs > worst {
				worst = m.MaxMs
			}
		}
		rec.Cells = append(rec.Cells, cellResult{
			Scene: "S15", Property: propLeaks, State: "open",
			Budgets:  []budgetVerdict{peakVerdict(worst)},
			Evidence: fmt.Sprintf("%d member(s); the slope half of H13 needs >= 10 cycles", len(rec.Measurements)),
		})
	}
	if complete {
		rec.Cells = append(rec.Cells, cellResult{
			Scene: "S15", Property: propWrappedErrors, State: "open",
			Budgets:  []budgetVerdict{{ID: "H8", Verdict: "advisory", Observed: "every big input ran; closing the cell is the summary's judgement, never a single run's"}},
			Evidence: fmt.Sprintf("%d outcome checks", len(rec.OutcomeChecks)),
		})
	} else {
		appendNote(rec, "subset run: the S15 cells stay open until every member has measured")
	}
	if _, err := saveRecord(env.runDir(rec.RunID), rec); err != nil {
		return rec, err
	}
	return rec, nil
}

func runS15Member(env *runEnv, ctx context.Context, rec *record, m s15Member, src, cycleDir string) (outcomeCheck, measurement, error) {
	profile := filepath.Join(cycleDir, "profile-"+m.name)
	out := filepath.Join(cycleDir, "out-"+m.name)
	mctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()
	args := append([]string{src, "-noopen", "-folder", out}, m.args...)
	res, err := env.execApp(mctx, rec.Binaries["app"].Path, args, execOptions{ProfileDir: profile, Dir: cycleDir}, rec.Canary)
	if err != nil {
		return outcomeCheck{}, measurement{}, err
	}

	oc := outcomeCheck{Injection: "S15-" + m.name, ExitCode: res.ExitCode, NamedOutcome: m.wantExit(res.ExitCode)}
	oc.BoundaryLogFound = containsAny(res.combinedTail, m.boundary...)
	if log, lerr := findRunLog(profile); lerr == nil {
		oc.BoundaryLogFound = oc.BoundaryLogFound && fileContainsAny(log, m.boundary)
		if hits, where := countCanary([]string{log}, rec.Canary); hits > 0 {
			oc.CanaryHits = hits
			oc.CanaryWhere = strings.Join(where, ", ")
		}
	} else {
		oc.BoundaryLogFound = false
	}

	_, hasRecord, _ := completionOf(out)
	tempFiles, _, terr := tempResidue(filepath.Join(profile, "Temp"))
	oc.StateConsistent = hasRecord == m.complete && terr == nil && len(tempFiles) == 0

	peak := peakPrivateBytes(res.P4Samples)
	pm := summarize("S15.peak-private-bytes."+m.name, []float64{peak})
	pm.Unit = "bytes"
	oc.Detail = fmt.Sprintf("%s; wall %.1f s; peak private %s; completion record=%v; temp files=%d",
		m.label, res.WallMs/1000, humanBytes(peak), hasRecord, len(tempFiles))
	return oc, pm, nil
}

// peakPrivateBytes is the highest private-byte reading of the app process across its samples.
// The sampler follows the app's own PID, which is the process whose address space the 386 build
// limits; helper processes (Tesseract, 7-Zip) have their own.
func peakPrivateBytes(samples []sample) float64 {
	peak := 0.0
	for _, s := range samples {
		if v, ok := s.Fields["private_bytes"].(float64); ok && v > peak {
			peak = v
		}
	}
	return peak
}

// peakVerdict judges a peak against ticket 12's line. A reading of zero means the sampler never
// saw the process, which is no evidence at all.
func peakVerdict(peak float64) budgetVerdict {
	v := budgetVerdict{ID: "H13", Limit: "peak private bytes of the app process below 1 GiB (ticket 12 criterion 3: well under 1 GB)"}
	switch {
	case peak <= 0:
		v.Verdict = "advisory"
		v.Observed = "no process sample was taken; the peak is unknown"
	case peak >= peakMemoryLine:
		v.Verdict = "fail"
		v.Observed = "peak private " + humanBytes(peak) + " reached the 1 GiB line"
	default:
		v.Verdict = "advisory" // the number is recorded; "well under" and the slope half stay the owner's call
		v.Observed = "peak private " + humanBytes(peak) + " (" + fmt.Sprintf("%.0f%%", 100*peak/peakMemoryLine) + " of the 1 GiB line)"
	}
	return v
}

func humanBytes(b float64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.2f GiB", b/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MiB", b/(1<<20))
	default:
		return fmt.Sprintf("%.0f KiB", b/(1<<10))
	}
}

// fileContainsAny reads a (log) file and reports whether it holds any phrase.
func fileContainsAny(path string, phrases []string) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return containsAny(string(b), phrases...)
}
