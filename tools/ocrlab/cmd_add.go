package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"

	"doc-html-translate/tools/ocrlab/corpus"
)

// LocalDir is where `ocrlab add` puts copies of local images. Copied rather than referenced in
// place so the corpus root stays self-contained and a hash means something: a file that moves or
// changes under the lab would otherwise silently invalidate every past measurement.
const LocalDir = "local"

// cmdAdd registers images the owner already has - their own scans, a screenshot, anything on
// disk - so the measure-look-fix loop can start on real content without hand-editing JSON.
//
// The licence gate is not weakened by this, it is applied correctly: OWN means the person
// running the command is the source, which is exactly the case the gate was written to allow.
// Third-party material still needs its own licence and its own human verification.
func cmdAdd(args []string) error {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	p := addCommonFlags(fs)
	cats := fs.String("categories", "document", "comma-separated categories: "+categoryList())
	split := fs.String("split", "dev", "dev or holdout")
	lic := fs.String("licence", string(corpus.LicenceOwn), "licence: OWN for your own material, or PD/PDM/CC0/CC-BY-3.0/CC-BY-4.0")
	by := fs.String("by", "", "who verified the licence (required for anything but OWN)")
	sourceURL := fs.String("source", "", "where it came from (required for anything but OWN)")
	licURL := fs.String("licence-url", "", "the asset's own licence page")
	note := fs.String("note", "", "free-text note")
	derivedFrom := fs.String("derived-from", "", "scene id this one was cut, blurred or downscaled from")
	inPlace := fs.Bool("in-place", false, "reference the file where it is instead of copying it into the corpus root")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return errors.New("usage: ocrlab add [flags] <image>..")
	}

	categories, err := parseCategories(*cats)
	if err != nil {
		return err
	}
	licence := corpus.Licence(*lic)
	if !licence.Valid() {
		return fmt.Errorf("licence %q is not one of the accepted values", licence)
	}
	verifier := *by
	if licence == corpus.LicenceOwn && verifier == "" {
		verifier = "owner (own material)"
	}
	if verifier == "" {
		return errors.New("-by is required: someone has to have read the asset's own licence page")
	}
	if licence.NeedsDownload() && *sourceURL == "" {
		return errors.New("-source is required for third-party material, so the scene can be re-fetched")
	}

	m, err := corpus.Load(p.manifest)
	if err != nil {
		return err
	}
	today := todayStamp()

	for _, src := range fs.Args() {
		scene, err := describeImage(src, p.root, *inPlace)
		if err != nil {
			return fmt.Errorf("%s: %w", src, err)
		}
		scene.Licence = licence
		scene.LicenceVerifiedBy = verifier
		scene.LicenceVerifiedOn = today
		scene.RetrievedOn = today
		scene.SourceURL = *sourceURL
		scene.LicenceURL = *licURL
		scene.Categories = categories
		scene.Split = corpus.Split(*split)
		scene.Note = *note
		// A crop, blur or downscale stays tied to its source entry rather than pretending to be an
		// independent find: the manifest's own rule, and what lets a reader see that two scenes are
		// not independent evidence. Whether the parent exists is verify's business, not add's.
		scene.DerivedFrom = *derivedFrom
		if !scene.Split.Valid() {
			return fmt.Errorf("split %q is neither dev nor holdout", *split)
		}

		existing := m.Find(scene.ID)
		m.Upsert(scene)
		verb := "added"
		if existing != nil {
			verb = "updated"
		}
		fmt.Printf("%-8s %-40s %dx%d  %s\n", verb, scene.ID, scene.Width, scene.Height, scene.File)
	}
	if err := corpus.Save(p.manifest, m); err != nil {
		return err
	}
	fmt.Printf("\nNext: annotate what the image says, then run it.\n")
	fmt.Printf("  go run ./tools/ocrlab seed <id>     draft the annotation from OCR, then correct it by hand\n")
	fmt.Printf("  go run ./tools/ocrlab run -scene <id> -out temp/ocrlab/loop\n")
	return nil
}

// describeImage measures the file and places it under the corpus root.
func describeImage(src, root string, inPlace bool) (corpus.Scene, error) {
	var s corpus.Scene
	f, err := os.Open(src)
	if err != nil {
		return s, err
	}
	cfg, _, decErr := image.DecodeConfig(f)
	f.Close()
	if decErr != nil {
		return s, fmt.Errorf("not a readable image: %w", decErr)
	}

	base := filepath.Base(src)
	s.ID = slug(strings.TrimSuffix(base, filepath.Ext(base)))
	s.Width, s.Height = cfg.Width, cfg.Height

	dst := src
	if !inPlace {
		dst = filepath.Join(root, LocalDir, base)
		if err := copyInto(src, dst); err != nil {
			return s, err
		}
		s.File = filepath.ToSlash(filepath.Join(LocalDir, base))
	} else {
		rel, err := filepath.Rel(root, src)
		if err != nil || strings.HasPrefix(rel, "..") {
			return s, fmt.Errorf("-in-place needs a file under the corpus root %s", root)
		}
		s.File = filepath.ToSlash(rel)
	}

	info, err := os.Stat(dst)
	if err != nil {
		return s, err
	}
	s.Bytes = info.Size()
	sum, err := corpus.HashFile(dst)
	if err != nil {
		return s, err
	}
	s.SHA256 = sum
	return s, nil
}

func copyInto(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

var slugBad = regexp.MustCompile(`[^a-z0-9]+`)

// slug makes a stable, ASCII scene id out of a file name. Non-ASCII names are common here (the
// corpus holds Cyrillic-named scans) and a scene id ends up in file paths, URLs and report
// anchors, so it is folded rather than passed through.
func slug(s string) string {
	out := slugBad.ReplaceAllString(strings.ToLower(s), "-")
	out = strings.Trim(out, "-")
	if out == "" {
		out = "scene"
	}
	return out
}

func parseCategories(list string) ([]corpus.Category, error) {
	var out []corpus.Category
	for _, part := range strings.Split(list, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		c := corpus.Category(part)
		if !c.Valid() {
			return nil, fmt.Errorf("category %q is not one of: %s", part, categoryList())
		}
		out = append(out, c)
	}
	if len(out) == 0 {
		return nil, errors.New("name at least one category")
	}
	return out, nil
}

func categoryList() string {
	var parts []string
	for _, c := range corpus.Categories() {
		parts = append(parts, string(c))
	}
	return strings.Join(parts, ", ")
}
