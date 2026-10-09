package evidence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// ReuseKey is everything a scene's collection depends on. Two collections with equal keys are
// interchangeable; one differing component forces a fresh collection. Scoring, gating and
// reporting are not part of it - they always rerun on whatever evidence a run ends up with.
type ReuseKey struct {
	Edition   Edition
	Producer  string
	Procedure string
	// Environment is the OS/arch and toolchain of the declaration, without the CPU counts.
	Environment string

	// Input carries the scene's media, annotation, lettering mask and corpus-record hashes and the
	// language it is read with.
	Input       Input
	Viewports   []Viewport
	StressCases []string

	// The engine and browser identity, restricted to the language packs this scene reads.
	Tesseract string
	Tessdata  string
	Browser   string
}

// NewKey builds the key of one scene of a declared run from that run's declaration and the engine
// and browser it used. It fails when an identity the key needs was not recorded: two unknowns
// must never compare equal.
func NewKey(edition Edition, d *Declaration, sceneID string, engine Engine, browser Browser) (ReuseKey, error) {
	in, ok := d.Scenes[sceneID]
	if !ok {
		return ReuseKey{}, fmt.Errorf("scene %s is not in the declaration", sceneID)
	}
	if d.ProducerDigest == "" {
		return ReuseKey{}, fmt.Errorf("the declaration records no producer digest")
	}
	if engine.Tesseract == "" || strings.Contains(engine.Tesseract, "unknown") {
		return ReuseKey{}, fmt.Errorf("the OCR engine version was not recorded")
	}
	if browser.Version == "" {
		return ReuseKey{}, fmt.Errorf("the browser version was not recorded")
	}
	packs, err := scenePacks(engine.TessdataVersion, in.Lang)
	if err != nil {
		return ReuseKey{}, err
	}
	return ReuseKey{
		Edition:     edition,
		Producer:    d.ProducerDigest,
		Procedure:   d.Procedure,
		Environment: reuseEnvironment(d.Environment),
		Input:       in,
		Viewports:   d.Viewports,
		StressCases: d.StressCases,
		Tesseract:   engine.Tesseract,
		Tessdata:    packs,
		Browser:     browser.Name + "|" + browser.Version,
	}, nil
}

// Mismatch names the first component in which o differs from k, or "" when they are equal.
func (k ReuseKey) Mismatch(o ReuseKey) string {
	switch {
	case k.Edition != o.Edition:
		return "edition"
	case k.Producer != o.Producer:
		return "producer digest"
	case k.Procedure != o.Procedure:
		return "procedure"
	case k.Environment != o.Environment:
		return "environment"
	case k.Input.MediaSHA256 != o.Input.MediaSHA256:
		return "scene media"
	case k.Input.AnnotationSHA256 != o.Input.AnnotationSHA256:
		return "annotation"
	case k.Input.LetteringSHA256 != o.Input.LetteringSHA256:
		return "lettering mask"
	case k.Input.SceneSHA256 != o.Input.SceneSHA256:
		return "corpus record"
	case k.Input.Lang != o.Input.Lang:
		return "language"
	case !slices.Equal(k.Viewports, o.Viewports):
		return "viewports"
	case !slices.Equal(k.StressCases, o.StressCases):
		return "stress cases"
	case k.Tesseract != o.Tesseract:
		return "OCR engine version"
	case k.Tessdata != o.Tessdata:
		return "language data"
	case k.Browser != o.Browser:
		return "browser"
	}
	return ""
}

// reuseEnvironment keeps what changes how a scene renders (OS, architecture, toolchain) and drops
// the CPU counts, which only change how fast it does.
func reuseEnvironment(env string) string {
	var keep []string
	for _, part := range strings.Split(env, "; ") {
		if !strings.HasPrefix(part, "CPUs=") && !strings.HasPrefix(part, "GOMAXPROCS=") {
			keep = append(keep, part)
		}
	}
	return strings.Join(keep, "; ")
}

// scenePacks extracts, from an engine's "code=sha256:..;code=.." identity, the entries for the
// codes of a scene's language ("rus+eng"), sorted. A missing code is an error: a scene read with a
// pack nobody can name is not provably the same.
func scenePacks(tessdata, lang string) (string, error) {
	have := map[string]string{}
	for _, item := range strings.Split(tessdata, ";") {
		if code, hash, ok := strings.Cut(item, "="); ok && code != "" && hash != "" {
			have[code] = item
		}
	}
	var codes []string
	for _, code := range strings.Split(lang, "+") {
		if code = strings.TrimSpace(code); code != "" && !slices.Contains(codes, code) {
			codes = append(codes, code)
		}
	}
	if len(codes) == 0 {
		return "", fmt.Errorf("the scene records no language")
	}
	sort.Strings(codes)
	var out []string
	for _, code := range codes {
		item, ok := have[code]
		if !ok {
			return "", fmt.Errorf("no recorded identity for language data %q", code)
		}
		out = append(out, item)
	}
	return strings.Join(out, ";"), nil
}

// Reusable is a complete earlier scene that may stand in for a fresh collection.
type Reusable struct {
	Bundle string
	Scene  Scene
	From   ReusedFrom
}

// bundle is one earlier run directory, loaded lazily and at most once per Finder.
type bundle struct {
	dir     string
	decl    *Declaration
	declErr error
	declSHA string

	loaded bool
	run    *Run
	runErr error

	scoresLoaded bool
	scorerDigest string
	failing      map[string]bool
}

// Finder searches earlier run directories for reusable scenes. One Finder serves a whole run, so
// each earlier bundle is read once however many scenes ask.
type Finder struct {
	bundles []*bundle
	scorer  string
}

// NewFinder lists the candidate bundles under searchRoot, which is either one run directory or a
// folder of them, newest first. exclude is the run being written, which is never a candidate.
// scorer is the current scorer digest (ScorerDigest): an earlier run scored by the same scorer
// shows whether the scene had a hard failure; "" disables that rule.
func NewFinder(searchRoot, exclude, scorer string) (*Finder, error) {
	var dirs []string
	if hasDeclaration(searchRoot) {
		dirs = []string{searchRoot}
	} else {
		entries, err := os.ReadDir(searchRoot)
		if err != nil {
			if os.IsNotExist(err) {
				return &Finder{scorer: scorer}, nil
			}
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() && hasDeclaration(filepath.Join(searchRoot, e.Name())) {
				dirs = append(dirs, filepath.Join(searchRoot, e.Name()))
			}
		}
	}
	type dated struct {
		dir string
		at  int64
	}
	var list []dated
	skip, _ := os.Stat(exclude)
	for _, dir := range dirs {
		if info, err := os.Stat(dir); err == nil && skip != nil && os.SameFile(info, skip) {
			continue
		}
		at := int64(0)
		if info, err := os.Stat(filepath.Join(dir, "declaration.json")); err == nil {
			at = info.ModTime().UnixNano()
		}
		list = append(list, dated{dir, at})
	}
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].at != list[j].at {
			return list[i].at > list[j].at
		}
		return list[i].dir > list[j].dir
	})
	f := &Finder{scorer: scorer}
	for _, e := range list {
		f.bundles = append(f.bundles, &bundle{dir: e.dir})
	}
	return f, nil
}

func hasDeclaration(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "declaration.json"))
	return err == nil
}

// FindReusable is a one-shot search: the newest bundle under searchRoot whose scene sceneID is
// complete, passed, and was collected under exactly key.
func FindReusable(searchRoot, exclude, scorer string, key ReuseKey, sceneID string) (*Reusable, string, error) {
	f, err := NewFinder(searchRoot, exclude, scorer)
	if err != nil {
		return nil, "", err
	}
	r, why := f.Find(key, sceneID)
	return r, why, nil
}

// Find returns the newest bundle's record that may stand in for collecting sceneID under key, or
// nil and the reason the closest candidate was refused. A failed, partial, unmeasured or errored
// scene is never returned, so such a scene always gets a fresh attempt.
func (f *Finder) Find(key ReuseKey, sceneID string) (*Reusable, string) {
	closest, rank := "", -1
	note := func(r int, dir, why string) {
		if r > rank {
			rank, closest = r, filepath.ToSlash(dir)+": "+why
		}
	}
	for _, b := range f.bundles {
		d := b.declaration()
		if b.declErr != nil {
			note(0, b.dir, "declaration unreadable: "+b.declErr.Error())
			continue
		}
		if d.ProducerDigest == "" {
			note(0, b.dir, "predates scene reuse (no producer digest)")
			continue
		}
		if d.ProducerDigest != key.Producer {
			note(1, b.dir, "producer digest differs (a product file changed)")
			continue
		}
		if _, ok := d.Scenes[sceneID]; !ok {
			note(2, b.dir, "scene not in that run")
			continue
		}
		run := b.evidence()
		if b.runErr != nil {
			note(2, b.dir, "evidence unreadable: "+b.runErr.Error())
			continue
		}
		sc := run.Find(sceneID)
		if sc == nil {
			note(2, b.dir, "scene absent from that run's evidence")
			continue
		}
		have, err := NewKey(run.Edition, d, sceneID, run.Engine, run.Browser)
		if err != nil {
			note(3, b.dir, err.Error())
			continue
		}
		if why := key.Mismatch(have); why != "" {
			note(3, b.dir, why+" differs")
			continue
		}
		integrity, why := f.usable(b, sc, key.Input)
		if why != "" {
			note(4, b.dir, why)
			continue
		}
		collected := run.StartedAt
		if sc.ReusedFrom != nil && sc.ReusedFrom.CollectedAt != "" {
			collected = sc.ReusedFrom.CollectedAt
		}
		return &Reusable{
			Bundle: b.dir,
			Scene:  *sc,
			From: ReusedFrom{
				Bundle:            filepath.ToSlash(b.dir),
				DeclarationSHA256: b.declSHA,
				ProducerDigest:    d.ProducerDigest,
				CollectedAt:       collected,
				Integrity:         integrity,
			},
		}, ""
	}
	if closest == "" {
		return nil, "no earlier run found"
	}
	return nil, closest
}

func (b *bundle) declaration() *Declaration {
	if b.decl != nil || b.declErr != nil {
		return b.decl
	}
	data, err := os.ReadFile(filepath.Join(b.dir, "declaration.json"))
	if err != nil {
		b.declErr = err
		return nil
	}
	var d Declaration
	if err := json.Unmarshal(data, &d); err != nil {
		b.declErr = err
		return nil
	}
	if d.Version != 1 {
		b.declErr = fmt.Errorf("unsupported declaration version %d", d.Version)
		return nil
	}
	b.decl, b.declSHA = &d, Digest(data)
	return b.decl
}

func (b *bundle) evidence() *Run {
	if !b.loaded {
		b.loaded = true
		b.run, b.runErr = LoadRun(filepath.Join(b.dir, "evidence.json"))
	}
	return b.run
}

// scoreFacts reads what the earlier run's scoring said: the scorer that produced it and which
// scenes had a hard failure. Absent or unreadable scoring leaves failing nil, which disables the
// hard-failure rule rather than guessing.
func (b *bundle) scoreFacts() {
	if b.scoresLoaded {
		return
	}
	b.scoresLoaded = true
	var summary struct {
		ScorerDigest string `json:"scorerDigest"`
	}
	var scores []struct {
		SceneID  string   `json:"sceneId"`
		Failures []string `json:"failures"`
	}
	if readJSONFile(filepath.Join(b.dir, "summary.json"), &summary) != nil ||
		readJSONFile(filepath.Join(b.dir, "scores.json"), &scores) != nil {
		return
	}
	b.scorerDigest = summary.ScorerDigest
	b.failing = map[string]bool{}
	for _, s := range scores {
		if len(s.Failures) > 0 {
			b.failing[s.SceneID] = true
		}
	}
}

func readJSONFile(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// usable applies the "passed OK" rule to a scene whose key already matches: the run finished with
// its source unchanged, the record is complete, the files are the ones the run wrote and, when the
// same scorer already judged it, it had no hard failure. It returns how the files were checked.
func (f *Finder) usable(b *bundle, sc *Scene, in Input) (integrity, why string) {
	var execution struct {
		SourceDigest string `json:"sourceDigest"`
		Unchanged    bool   `json:"unchanged"`
	}
	if readJSONFile(filepath.Join(b.dir, "execution.json"), &execution) != nil ||
		!execution.Unchanged || execution.SourceDigest != b.decl.SourceDigest {
		return "", "that run did not finish with its source unchanged"
	}
	if why := completeScene(b.dir, b.decl, b.run.Edition, sc, in); why != "" {
		return "", why
	}
	integrity, why = checkManifest(b.dir, sc.SceneID)
	if why != "" {
		return "", why
	}
	if f.scorer != "" {
		b.scoreFacts()
		if b.scorerDigest == f.scorer && b.failing[sc.SceneID] {
			return "", "the scene had a hard failure under the current scorer"
		}
	}
	return integrity, ""
}
