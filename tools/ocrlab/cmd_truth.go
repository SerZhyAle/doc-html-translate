package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"doc-html-translate/internal/ocr"
	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/synth"
	"doc-html-translate/tools/ocrlab/truth"
)

// cmdSynth regenerates the diagnostic scenes, their exact annotations and their manifest
// entries. It is idempotent by construction: the drawings are pure functions of their own
// constants, so a second run rewrites identical bytes and leaves the diff empty.
func cmdSynth(args []string) error {
	fs := flag.NewFlagSet("synth", flag.ExitOnError)
	p := addCommonFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	m, err := corpus.Load(p.manifest)
	if err != nil {
		return err
	}
	scenes, anns, err := synth.Generate(p.root)
	if err != nil {
		return err
	}
	for _, s := range scenes {
		m.Upsert(s)
	}
	for _, a := range anns {
		// A generated annotation always overwrites: the drawing is its definition, so a stale
		// file on disk is wrong by construction rather than a human's work worth preserving.
		if err := truth.Save(truth.FinalPath(p.annotations, a.SceneID), a); err != nil {
			return err
		}
	}
	if err := corpus.Save(p.manifest, m); err != nil {
		return err
	}
	fmt.Printf("synth: %d scene(s) drawn into %s, %d annotation(s) written to %s\n",
		len(scenes), filepath.Join(p.root, synth.Dir), len(anns), p.annotations)
	return nil
}

// cmdSeed writes an OCR-seeded annotation draft to save a human the tedious part of the work:
// the boxes are roughly right and the transcript is roughly readable, so the reviewer corrects
// rather than types.
//
// Two safeguards make this safe to run. The draft is written to <id>.draft.json and refuses to
// touch an existing <id>.json, so a reviewed record is never clobbered by a re-seed. And the
// draft carries Origin=ocr-seed, which makes IsTruth false - the scorer will skip it and say
// why, rather than quietly grading the engine against its own output.
func cmdSeed(args []string) error {
	fs := flag.NewFlagSet("seed", flag.ExitOnError)
	p := addCommonFlags(fs)
	lang := fs.String("lang", "eng", "tesseract language for the seed pass")
	if err := fs.Parse(args); err != nil {
		return err
	}
	ids := fs.Args()
	if len(ids) == 0 {
		return errors.New("name at least one scene id to seed")
	}

	m, err := corpus.Load(p.manifest)
	if err != nil {
		return err
	}
	sel, missing := m.Select("all", ids)
	if len(missing) > 0 {
		return fmt.Errorf("unknown scene id(s): %v", missing)
	}
	bin, err := ocr.Locate()
	if err != nil {
		return err
	}

	for _, s := range sel {
		final := truth.FinalPath(p.annotations, s.ID)
		if _, err := os.Stat(final); err == nil {
			fmt.Printf("skip   %-40s %s already exists - delete it deliberately to re-seed\n", s.ID, final)
			continue
		}
		a, err := seedOne(bin, s, p.root, *lang)
		if err != nil {
			return fmt.Errorf("%s: %w", s.ID, err)
		}
		path := truth.DraftPath(p.annotations, s.ID)
		if err := truth.Save(path, a); err != nil {
			return err
		}
		fmt.Printf("draft  %-40s %d group(s) -> %s\n", s.ID, len(a.Groups), path)
	}
	fmt.Println("\nA draft is not truth. Correct the geometry and the transcript, set origin to")
	fmt.Println("\"human\", fill in review.annotatedBy, and rename the file to drop the .draft suffix.")
	return nil
}

// seedOne runs the recognizer over one scene and turns each recognized block into a draft
// group. Every group is typed "incidental" on purpose: the engine cannot tell a balloon from a
// caption, and guessing here would put an unearned label in front of the reviewer.
func seedOne(bin string, s *corpus.Scene, root, lang string) (*truth.Annotation, error) {
	res, err := ocr.Recognize(bin, s.Path(root), lang, "")
	if err != nil {
		return nil, err
	}
	a := &truth.Annotation{
		SchemaVersion: truth.SchemaVersion,
		SceneID:       s.ID,
		Origin:        truth.OriginOCRSeed,
		ImageWidth:    res.Width,
		ImageHeight:   res.Height,
		Ambiguity:     truth.AmbiguityClear,
		ReviewerNote:  "OCR-seeded draft. Boxes and transcript come from the engine under test and must be corrected before this scene can be scored.",
	}
	for i, b := range res.Blocks {
		id := fmt.Sprintf("g%d", i+1)
		box := truth.Box(id+"-bounds", b.X0, b.Y0, b.X1, b.Y1)
		line := truth.Box(id+"-line", b.X0, b.Y0, b.X1, b.Y1)
		a.Groups = append(a.Groups, truth.Group{
			ID:           id,
			Type:         truth.GroupIncidental,
			Transcript:   b.Text,
			Direction:    truth.DirLTR,
			ReadingOrder: i + 1,
			Lines:        []truth.Region{line},
			Bounds:       box,
			ReplaceArea:  truth.Box(id+"-replace", b.X0, b.Y0, b.X1, b.Y1),
		})
	}
	return a, nil
}
