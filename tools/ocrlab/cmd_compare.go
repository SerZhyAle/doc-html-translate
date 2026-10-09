package main

import (
	"errors"
	"flag"
	"fmt"

	"doc-html-translate/tools/ocrlab/evidence"
	"doc-html-translate/tools/ocrlab/report"
)

func cmdCompare(args []string) error {
	f := flag.NewFlagSet("compare", flag.ContinueOnError)
	if err := f.Parse(args); err != nil {
		return err
	}
	if f.NArg() != 2 {
		return errors.New("usage: ocrlab compare <before-dir> <after-dir>")
	}
	b, err := report.LoadData(f.Arg(0))
	if err != nil {
		return err
	}
	a, err := report.LoadData(f.Arg(1))
	if err != nil {
		return err
	}
	bd, _ := evidence.LoadDeclaration(f.Arg(0))
	ad, _ := evidence.LoadDeclaration(f.Arg(1))
	c := report.Compare(b, a, bd, ad)
	if err := report.WriteComparison(f.Arg(1), c); err != nil {
		return err
	}
	if err := report.WriteComparisonHTML(f.Arg(0), f.Arg(1), b, a, c); err != nil {
		return err
	}
	fmt.Printf("comparison: %d scenes, compatible=%t; %d limitations\n", len(c.Scenes), c.Compatible, len(c.Limitations))
	return nil
}
