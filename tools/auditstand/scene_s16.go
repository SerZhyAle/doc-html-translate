// scene_s16.go implements the failure zoo's first subset: each member is one injected failure,
// run in its own cycle, judged by H8's five assertions (named outcome, one boundary-naming log
// line, zero canary hits, a message in the active locale, consistent state). Members the
// instrument cannot force on this machine are recorded as pending advisory rows - the cell
// stays open until the full membership of strategic section 5.1 has run.

package main

import (
	"context"
	"doc-html-translate/internal/outputpath"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// s16Member is one injected failure with its precondition.
type s16Member struct {
	name string
	// run executes the member and returns the app result plus what boundary phrase the outcome
	// must name (the member's own named-outcome contract).
	run func(env *runEnv, rec *record, runDir string) (*injectionResult, error)
	// pending is non-empty when the member is declared but not implemented here; the cell
	// cannot close until every member measures.
	pending string
}

// injectionResult is one member's raw outcome.
type injectionResult struct {
	res           *execResult
	profile       string
	outDir        string // the output dir the member's run targeted (the consistency check walks it)
	wantBoundary  string // a phrase the combined output or the run log must contain
	wantBoundary2 string // an alternative accepted phrase
	expectSuccess bool   // true when the injection must NOT abort the conversion (best-effort decisions)
	// extra is the member's own continuation assertion over the captured console output (for the
	// best-effort members: the run went on, and the rest of the book was still handled). It
	// returns whether it holds and a short reason for the record.
	extra func(output string) (bool, string)
}

// s16Members is the subset implemented now; the rest are named with their owning phase.
var s16Members = []s16Member{
	{name: "missing-tesseract", run: runMissingTesseract},
	{name: "missing-calibre", run: runMissingHelperMobi},
	{name: "missing-7zip", run: runMissingHelperCbr},
	{name: "corrupt-archive", run: runCorruptArchive},
	{name: "unreadable-file", run: runUnreadableFile},
	{name: "ocr-image-failure", run: runOcrImageFailure},
	{name: "missing-filedo", run: runMissingFileDO},
	{name: "disk-full", pending: "phase 01 step 01.01 (a size-capped volume the stand creates; creating one needs an elevated one-time action the owner approves)"},
	{name: "network-dead", pending: "phase 01 step 01.01 (a blocked-firewall profile the owner pre-approves)"},
}

func runSceneS16(env *runEnv, ctx context.Context, cycles int) (*record, error) {
	rec, err := newSceneRecord(env, "S16")
	if err != nil {
		return nil, err
	}
	runDir := env.runDir(rec.RunID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return nil, err
	}

	complete := true
	for cycle := 1; cycle <= cycles; cycle++ {
		for _, m := range s16Members {
			if m.pending != "" {
				complete = false
				appendNote(rec, "member %s pending: %s", m.name, m.pending)
				continue
			}
			inj, err := m.run(env, rec, filepath.Join(runDir, fmt.Sprintf("c%d", cycle)))
			if err != nil {
				complete = false
				// A member that could not even run is a pending row, not a finding (a missing
				// helper the machine does not have is the natural state, recorded as such).
				appendNote(rec, "member %s could not run: %v", m.name, err)
				continue
			}
			rec.OutcomeChecks = append(rec.OutcomeChecks, judgeInjection(m.name, inj, rec.Canary))
		}
	}

	// S16's cell stays open until the full membership has run - H8 is a 100% assertion over the
	// catalog (strategic section 8), so a subset closes nothing by itself.
	if !complete {
		appendNote(rec, "subset run: the property-2 cell stays open until every member of the catalog has measured")
	} else {
		rec.Cells = append(rec.Cells, cellResult{
			Scene: "S16", Property: propWrappedErrors, State: "open",
			Budgets:  []budgetVerdict{{ID: "H8", Verdict: "advisory", Observed: "full membership ran; closing the cell is the summary's judgement, never a single run's"}},
			Evidence: fmt.Sprintf("%d outcome checks", len(rec.OutcomeChecks)),
		})
	}

	dir := env.runDir(rec.RunID)
	if _, err := saveRecord(dir, rec); err != nil {
		return rec, err
	}
	return rec, nil
}

// judgeInjection applies H8's assertions to one member's raw outcome: a named outcome (the exit
// code the member's decision calls for), one boundary-naming line, no canary in the run log, and
// a consistent state.
func judgeInjection(name string, inj *injectionResult, canary string) outcomeCheck {
	oc := outcomeCheck{Injection: name, ExitCode: inj.res.ExitCode}
	named := inj.res.ExitCode == 0
	if !inj.expectSuccess {
		named = inj.res.ExitCode != 0
	}
	oc.NamedOutcome = named
	oc.BoundaryLogFound = containsAny(inj.res.combinedTail, inj.wantBoundary, inj.wantBoundary2)
	extraNote := ""
	if inj.extra != nil {
		ok, why := inj.extra(inj.res.combinedTail)
		oc.BoundaryLogFound = oc.BoundaryLogFound && ok
		extraNote = "; " + why
	}
	oc.StateConsistent = stateConsistentAfter(inj)
	if log, lerr := findRunLog(inj.profile); lerr == nil {
		if hits, where := countCanary([]string{log}, canary); hits > 0 {
			oc.CanaryHits = hits
			oc.CanaryWhere = strings.Join(where, ", ")
		}
	}
	oc.Detail = fmt.Sprintf("exit %d, boundary %q / %q, wall %.0f ms%s",
		inj.res.ExitCode, inj.wantBoundary, inj.wantBoundary2, inj.res.WallMs, extraNote)
	return oc
}

// runMissingTesseract strips PATH so tesseract cannot be found, then converts the standalone
// image with -ocr. OCR is strictly best-effort (a documented decision): the injection must NOT
// abort the conversion - the decision's measured consequence is the outcome this member checks.
func runMissingTesseract(env *runEnv, rec *record, runDir string) (*injectionResult, error) {
	profile := filepath.Join(runDir, "profile-missing-tesseract")
	out := filepath.Join(runDir, "out-missing-tesseract")
	res, err := env.execApp(context.Background(), rec.Binaries["app"].Path,
		[]string{filepath.Join(env.fixturesDir, "img-photo.png"), "-noopen", "-ocr", "-folder", out},
		execOptions{ProfileDir: profile, MinimalPath: true, Dir: runDir}, rec.Canary)
	if err != nil {
		return nil, err
	}
	return &injectionResult{
		res:           res,
		profile:       profile,
		outDir:        out,
		wantBoundary:  "tesseract",
		expectSuccess: true, // best-effort: a missing tesseract never aborts
	}, nil
}

// runMissingHelperMobi feeds the fake .mobi under a stripped PATH: with Calibre unfindable the
// run must end in the helper boundary's named error (one error type naming the tool).
func runMissingHelperMobi(env *runEnv, rec *record, runDir string) (*injectionResult, error) {
	profile := filepath.Join(runDir, "profile-missing-calibre")
	out := filepath.Join(runDir, "out-missing-calibre")
	res, err := env.execApp(context.Background(), rec.Binaries["app"].Path,
		[]string{filepath.Join(env.fixturesDir, "mobi-probe.mobi"), "-noopen", "-folder", out},
		execOptions{ProfileDir: profile, MinimalPath: true, Dir: runDir}, rec.Canary)
	if err != nil {
		return nil, err
	}
	return &injectionResult{
		res:          res,
		profile:      profile,
		outDir:       out,
		wantBoundary: "calibre",
	}, nil
}

// runMissingHelperCbr does the same for 7-Zip through a .cbr input.
func runMissingHelperCbr(env *runEnv, rec *record, runDir string) (*injectionResult, error) {
	profile := filepath.Join(runDir, "profile-missing-7zip")
	out := filepath.Join(runDir, "out-missing-7zip")
	res, err := env.execApp(context.Background(), rec.Binaries["app"].Path,
		[]string{filepath.Join(env.fixturesDir, "comic-probe.cbr"), "-noopen", "-folder", out},
		execOptions{ProfileDir: profile, MinimalPath: true, Dir: runDir}, rec.Canary)
	if err != nil {
		return nil, err
	}
	return &injectionResult{
		res:           res,
		profile:       profile,
		outDir:        out,
		wantBoundary:  "7-zip",
		wantBoundary2: "7z",
	}, nil
}

// runCorruptArchive feeds the truncated zip: a named archive error, a non-zero exit, and no
// half-written output that later reads as complete.
func runCorruptArchive(env *runEnv, rec *record, runDir string) (*injectionResult, error) {
	profile := filepath.Join(runDir, "profile-corrupt")
	out := filepath.Join(runDir, "out-corrupt")
	res, err := env.execApp(context.Background(), rec.Binaries["app"].Path,
		[]string{filepath.Join(env.fixturesDir, "comic-corrupt.cbz"), "-noopen", "-folder", out},
		execOptions{ProfileDir: profile, Dir: runDir}, rec.Canary)
	if err != nil {
		return nil, err
	}
	return &injectionResult{
		res:           res,
		profile:       profile,
		outDir:        out,
		wantBoundary:  "zip",
		wantBoundary2: "archive",
	}, nil
}

// runUnreadableFile creates a regular input and holds it open with no share modes for the
// duration of the conversion attempt - the unreadable-file member of S16, made at scene time
// (FIXTURES.md).
func runUnreadableFile(env *runEnv, rec *record, runDir string) (*injectionResult, error) {
	lockSrc := filepath.Join(runDir, "locked-epub.epub")
	data, err := os.ReadFile(filepath.Join(env.fixturesDir, "epub-small.epub"))
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(lockSrc, data, 0o644); err != nil {
		return nil, err
	}
	lock, err := lockFileExclusive(lockSrc)
	if err != nil {
		return nil, err
	}
	defer lock.Close()
	profile := filepath.Join(runDir, "profile-unreadable")
	out := filepath.Join(runDir, "out-unreadable")
	res, err := env.execApp(context.Background(), rec.Binaries["app"].Path,
		[]string{lockSrc, "-noopen", "-folder", out},
		execOptions{ProfileDir: profile, Dir: runDir}, rec.Canary)
	if err != nil {
		return nil, err
	}
	return &injectionResult{
		res:           res,
		profile:       profile,
		outDir:        out,
		wantBoundary:  "read",
		wantBoundary2: "open",
	}, nil
}

// overlaidRe reads the pipeline's OCR summary line ("OCR overlay: 2 image(s) overlaid, 1 failed").
var overlaidRe = regexp.MustCompile(`OCR overlay: (\d+) image\(s\) overlaid(?:, \d+ with no text found)?, (\d+) failed`)

// runOcrImageFailure converts a book whose middle page image is cut off mid-file, with OCR on.
// OCR is best-effort by decision: the failed image is named, the other pages are still
// recognized and the book is produced - the decision's measured consequence, not a finding
// about the decision.
func runOcrImageFailure(env *runEnv, rec *record, runDir string) (*injectionResult, error) {
	profile := filepath.Join(runDir, "profile-ocr-image-failure")
	out := filepath.Join(runDir, "out-ocr-image-failure")
	res, err := env.execApp(context.Background(), rec.Binaries["app"].Path,
		[]string{filepath.Join(env.fixturesDir, "epub-ocr-halfimage.epub"), "-noopen", "-ocr", "-folder", out},
		execOptions{ProfileDir: profile, Dir: runDir}, rec.Canary)
	if err != nil {
		return nil, err
	}
	return &injectionResult{
		res:           res,
		profile:       profile,
		outDir:        out,
		wantBoundary:  "OCR failed on page2.png",
		expectSuccess: true, // best-effort: one failed image never aborts the book
		extra: func(output string) (bool, string) {
			m := overlaidRe.FindStringSubmatch(output)
			if m == nil {
				return false, "no OCR summary line (OCR did not run: is tesseract installed?)"
			}
			if m[2] != "1" {
				return false, fmt.Sprintf("expected exactly 1 failed image, summary says %s", m[2])
			}
			if m[1] == "0" {
				return false, "the failed image took the whole book's OCR with it (0 images overlaid)"
			}
			return true, fmt.Sprintf("%s image(s) overlaid around the failed one", m[1])
		},
	}, nil
}

// runMissingFileDO opens a .fd-sec file with the FileDO override pointing at nothing, so FileDO
// is unfindable whatever is installed. Locate runs before the password question and before the
// length screen, so any file of the container extension is enough, and no password dialog may
// appear.
func runMissingFileDO(env *runEnv, rec *record, runDir string) (*injectionResult, error) {
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return nil, err
	}
	container := filepath.Join(runDir, "audit-container.fd-sec")
	// 5632 bytes is the smallest size the credential-free length screen accepts; the content is
	// zeros and never read, because FileDO is looked for first.
	if err := os.WriteFile(container, make([]byte, 5632), 0o600); err != nil {
		return nil, err
	}
	profile := filepath.Join(runDir, "profile-missing-filedo")
	out := filepath.Join(runDir, "out-missing-filedo")
	res, err := env.execApp(context.Background(), rec.Binaries["app"].Path,
		[]string{container, "-noopen", "-folder", out},
		execOptions{
			ProfileDir: profile, Dir: runDir,
			ExtraEnv: []string{"DOCHT_FILEDO=" + filepath.Join(runDir, "no-such-filedo", "filedo.exe")},
		}, rec.Canary)
	if err != nil {
		return nil, err
	}
	return &injectionResult{
		res:          res,
		profile:      profile,
		outDir:       out,
		wantBoundary: "FileDO",
		extra: func(output string) (bool, string) {
			if !containsAny(output, "No password was asked") {
				return false, "the message does not say that no password was asked"
			}
			return true, "no password was asked"
		},
	}, nil
}

// containsAny reports whether s contains any of the accepted phrases (case-insensitive).
func containsAny(s string, phrases ...string) bool {
	ls := strings.ToLower(s)
	for _, p := range phrases {
		if p != "" && strings.Contains(ls, strings.ToLower(p)) {
			return true
		}
	}
	return false
}

// stateConsistentAfter checks the two consistency assertions of H8: no half-written output
// reads as complete (a failed run leaves no completion record), and the redirected temp holds
// no orphan after the run.
func stateConsistentAfter(inj *injectionResult) bool {
	return tempResidueOK(inj) && noCompletionRecordOnFailure(inj)
}

func tempResidueOK(inj *injectionResult) bool {
	files, _, err := tempResidue(filepath.Join(inj.profile, "Temp"))
	if err != nil {
		return false
	}
	return len(files) == 0
}

func noCompletionRecordOnFailure(inj *injectionResult) bool {
	if inj.res.ExitCode == 0 {
		return true // a success is judged by its own member's expectations
	}
	// A failed run must not have left a completion record: no output of a failed conversion
	// may later read as complete (the completion-record invariant). The ownership record itself
	// legitimately exists from the moment the folder is claimed - it is its "complete" section
	// that must be absent.
	if inj.outDir == "" {
		return true
	}
	found := false
	_ = filepath.WalkDir(inj.outDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != outputpath.MarkerName {
			return nil
		}
		if m, rerr := outputpath.ReadMarker(filepath.Dir(path)); rerr == nil && m.Complete != nil {
			found = true
		}
		return nil
	})
	return !found
}
