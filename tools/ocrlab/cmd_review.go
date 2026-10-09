package main

import (
	"errors"
	"flag"
	"fmt"

	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/review"
	"doc-html-translate/tools/ocrlab/truth"
)

func cmdReview(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: ocrlab review export|import [flags] [draft.json]")
	}
	f := flag.NewFlagSet("review", flag.ContinueOnError)
	p := addCommonFlags(f)
	out := f.String("out", "temp/ocrlab/review.html", "offline editor")
	if err := f.Parse(args[1:]); err != nil {
		return err
	}
	m, err := corpus.Load(p.manifest)
	if err != nil {
		return err
	}
	switch args[0] {
	case "export":
		anns, err := truth.LoadDir(p.annotations)
		if err != nil {
			return err
		}
		if err := review.Write(*out, p.root, m, anns); err != nil {
			return err
		}
		fmt.Println("review workspace:", *out)
	case "import":
		if f.NArg() != 1 {
			return errors.New("review import requires one draft file")
		}
		dest, err := review.Import(f.Arg(0), p.annotations, m)
		if err != nil {
			return err
		}
		fmt.Println("validated draft awaiting human annotation and independent review:", dest)
	default:
		return errors.New("review requires export or import")
	}
	return nil
}
