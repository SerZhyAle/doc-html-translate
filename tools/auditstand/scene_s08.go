// scene_s08.go implements S08 translate-guards: the refusal paths of the translation engines,
// none of which spends money or sends the document anywhere (register row N2).
//
//   - google-cost-guard      -google -max-cost <below the estimate> with a planted fake key: the
//     estimate guard refuses before any request is built.
//   - google-cost-guard-ru   the same under -ui-lang ru: H8's "message present in the active
//     locale" is asserted on the refusal line.
//   - google-no-key          -google with no key anywhere: the canonical wrapped error.
//   - ollama-unavailable     -ollama against a local service that is down (or, when one is
//     running, against a model that does not exist - the stand never stops a service the owner
//     runs).
//
// The fake key carries the run's canary, so a leak of the key into the log, the console, an output
// page or the profile is a canary hit - a secret in an artifact, P1 by the severity table.

package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

// exitAPI is the pipeline's code for "translation failed or stopped part-way; the book is still
// produced". A test pins it to internal/pipeline so a renumbering fails there, not on a live run.
const exitAPI = 4

// s08MemberTimeout bounds one member: a hung app must end the member, not the campaign.
const s08MemberTimeout = 2 * time.Minute

// fakeGoogleKey is a syntactically plausible API key carrying the canary.
func fakeGoogleKey(canary string) string { return "AIzaSy" + canary }

// ollamaListening reports whether something already accepts connections on Ollama's fixed local
// port (the app hard-codes localhost:11434, so the stand cannot point it elsewhere).
func ollamaListening() bool {
	for _, addr := range []string{"127.0.0.1:11434", "[::1]:11434"} {
		c, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
		if err == nil {
			_ = c.Close()
			return true
		}
	}
	return false
}

// s08Member is one refusal path and what the stand asserts about it.
type s08Member struct {
	name     string
	args     func(src, out string) []string
	uiLang   string // "" = the default; set, the refusal line must carry that language's script
	plantKey bool
	// wantExit judges the exit code; label names the expectation in the record.
	wantExit func(code int) bool
	label    string
	// boundary phrases: any one must be on a console line and in the persisted run log.
	boundary []string
	// translations the completion record may carry. Never "full": a refusal that left a record
	// claiming a finished translation would let a re-run reopen an untranslated book.
	translations []string
}

func s08Members() []s08Member {
	ollamaName, ollamaArgs := "ollama-unavailable", func(src, out string) []string {
		return []string{src, "-noopen", "-ollama", "-folder", out}
	}
	if ollamaListening() {
		// A running service is the owner's; the member then injects the other engine failure the
		// guard family covers - a model that is not installed. Ollama answers 404, it does not
		// pull, so nothing is downloaded and no model is loaded.
		ollamaName, ollamaArgs = "ollama-model-missing", func(src, out string) []string {
			return []string{src, "-noopen", "-ollama", "-ollama-model", "audit-no-such-model:0", "-folder", out}
		}
	}
	return []s08Member{
		{
			name: "google-cost-guard",
			args: func(src, out string) []string {
				return []string{src, "-noopen", "-google", "-max-cost", "0.01", "-folder", out}
			},
			plantKey:     true,
			wantExit:     func(c int) bool { return c == 0 },
			label:        "exit 0 (the user's own limit was honored; the book is converted untranslated)",
			boundary:     []string{"exceeds -max-cost"},
			translations: []string{"none"},
		},
		{
			name: "google-cost-guard-ru",
			args: func(src, out string) []string {
				return []string{src, "-noopen", "-google", "-max-cost", "0.01", "-ui-lang", "ru", "-folder", out}
			},
			uiLang:       "ru",
			plantKey:     true,
			wantExit:     func(c int) bool { return c == 0 },
			label:        "exit 0 (the user's own limit was honored; the book is converted untranslated)",
			boundary:     []string{"-max-cost"},
			translations: []string{"none"},
		},
		{
			name: "google-no-key",
			args: func(src, out string) []string {
				return []string{src, "-noopen", "-google", "-folder", out}
			},
			wantExit:     func(c int) bool { return c == exitAPI },
			label:        fmt.Sprintf("exit %d (translation unavailable; failure must not read as success)", exitAPI),
			boundary:     []string{"API key not available", "Google Translate skipped"},
			translations: []string{"none"},
		},
		{
			name:         ollamaName,
			args:         ollamaArgs,
			wantExit:     func(c int) bool { return c == exitAPI },
			label:        fmt.Sprintf("exit %d (engine failed; the book is still produced)", exitAPI),
			boundary:     []string{"ollama"},
			translations: []string{"none", "partial"},
		},
	}
}

func runSceneS08(env *runEnv, ctx context.Context, cycles int) (*record, error) {
	rec, err := newSceneRecord(env, "S08")
	if err != nil {
		return nil, err
	}
	runDir := env.runDir(rec.RunID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return nil, err
	}
	// The app reads a plaintext key next to its own executable first - and migrates, then
	// deletes, what it reads. That file belongs to whoever put it there; the stand never lets
	// the app consume it.
	exeKey := filepath.Join(filepath.Dir(rec.Binaries["app"].Path), "google_api.key")
	if fileExists(exeKey) {
		return rec, cannotVerify(fmt.Errorf("a google_api.key sits next to the app binary under test; the app would read and delete it - move it away or test a copy of the binaries"))
	}

	src := filepath.Join(env.fixturesDir, "epub-small.epub")
	complete := true
	for cycle := 1; cycle <= cycles; cycle++ {
		for _, m := range s08Members() {
			oc, err := runS08Member(env, ctx, rec, m, src, filepath.Join(runDir, fmt.Sprintf("c%d", cycle)))
			if err != nil {
				complete = false
				appendNote(rec, "member %s could not run: %v", m.name, err)
				continue
			}
			rec.OutcomeChecks = append(rec.OutcomeChecks, oc)
		}
	}
	if complete {
		rec.Cells = append(rec.Cells, cellResult{
			Scene: "S08", Property: propWrappedErrors, State: "open",
			Budgets:  []budgetVerdict{{ID: "H8", Verdict: "advisory", Observed: "every refusal path ran; closing the cell is the summary's judgement, never a single run's"}},
			Evidence: fmt.Sprintf("%d outcome checks", len(rec.OutcomeChecks)),
		})
	} else {
		appendNote(rec, "subset run: the property-2 cell stays open until every refusal path has measured")
	}
	if _, err := saveRecord(env.runDir(rec.RunID), rec); err != nil {
		return rec, err
	}
	return rec, nil
}

func runS08Member(env *runEnv, ctx context.Context, rec *record, m s08Member, src, cycleDir string) (outcomeCheck, error) {
	profile := filepath.Join(cycleDir, "profile-"+m.name)
	out := filepath.Join(cycleDir, "out-"+m.name)
	keyFile := filepath.Join(profile, "LocalAppData", "doc-html-translate", "google_api.key")
	if m.plantKey {
		if err := env.prepareProfile(profile); err != nil {
			return outcomeCheck{}, cannotVerify(err)
		}
		if err := os.MkdirAll(filepath.Dir(keyFile), 0o700); err != nil {
			return outcomeCheck{}, err
		}
		if err := os.WriteFile(keyFile, []byte(fakeGoogleKey(rec.Canary)), 0o600); err != nil {
			return outcomeCheck{}, err
		}
	}
	mctx, cancel := context.WithTimeout(ctx, s08MemberTimeout)
	defer cancel()
	res, err := env.execApp(mctx, rec.Binaries["app"].Path, m.args(src, out),
		execOptions{ProfileDir: profile, Dir: cycleDir}, rec.Canary)
	if err != nil {
		return outcomeCheck{}, err
	}

	oc := outcomeCheck{Injection: "S08-" + m.name, ExitCode: res.ExitCode}
	oc.NamedOutcome = m.wantExit(res.ExitCode)

	// One log line naming the boundary: on the console and in the persisted run log.
	logText := ""
	if log, lerr := findRunLog(profile); lerr == nil {
		if b, rerr := os.ReadFile(log); rerr == nil {
			logText = string(b)
		}
	}
	oc.BoundaryLogFound = containsAny(res.combinedTail, m.boundary...) && containsAny(logText, m.boundary...)

	// No secret anywhere the app wrote or printed: the console capture and every file under the
	// profile and the output folder.
	files := append(listFiles(profile), listFiles(out)...)
	hits, where := countCanary(files, rec.Canary)
	if n := strings.Count(res.combinedTail, rec.Canary); n > 0 {
		hits += n
		where = append(where, "console output")
	}
	oc.CanaryHits = hits
	oc.CanaryWhere = strings.Join(where, ", ")

	// Consistent state: the record says what really happened (never a finished translation), the
	// plaintext key the stand planted was migrated away, and the redirected temp is empty.
	translation, complete, cerr := completionOf(out)
	okTranslation := cerr == nil && complete && stringIn(translation, m.translations)
	keyGone := !m.plantKey || !fileExists(keyFile)
	tempFiles, _, terr := tempResidue(filepath.Join(profile, "Temp"))
	oc.StateConsistent = okTranslation && keyGone && terr == nil && len(tempFiles) == 0

	if m.uiLang != "" {
		oc.LocaleChecked = true
		oc.MessageInLocale = lineInScript(res.combinedTail, "-max-cost", m.uiLang)
	}
	oc.Detail = fmt.Sprintf("%s; wall %.0f ms; record translation=%q complete=%v; plaintext key removed=%v; temp files=%d",
		m.label, res.WallMs, translation, complete, keyGone, len(tempFiles))
	return oc, nil
}

// listFiles returns every regular file under root; a missing root is an empty list.
func listFiles(root string) []string {
	var out []string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			out = append(out, path)
		}
		return nil
	})
	return out
}

// lineInScript reports whether a console line naming marker carries a letter of the script the
// interface language is written in. It is deliberately a script test, not a translation test: it
// separates "the line was localized" from "the line is the English literal", which is all H8's
// locale part asks of the stand; whether the wording is right is the owner's ru/uk human pass.
func lineInScript(text, marker, lang string) bool {
	table := scriptOf(lang)
	if table == nil {
		return false
	}
	for _, line := range strings.Split(text, "\n") {
		if !strings.Contains(line, marker) {
			continue
		}
		for _, r := range line {
			if unicode.Is(table, r) {
				return true
			}
		}
	}
	return false
}

// scriptOf maps the interface languages whose script differs from the English literals' to their
// Unicode script. Latin-script languages cannot be told from English by script, so the stand
// does not claim a verdict for them (nil).
func scriptOf(lang string) *unicode.RangeTable {
	switch lang {
	case "ru", "uk":
		return unicode.Cyrillic
	case "ar", "ur":
		return unicode.Arabic
	case "hi":
		return unicode.Devanagari
	case "bn":
		return unicode.Bengali
	case "zh":
		return unicode.Han
	}
	return nil
}
