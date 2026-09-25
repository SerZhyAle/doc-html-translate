// Command icongen draws the product's system-surface icons from the mark and the vendored
// glyphs (internal/iconart) and writes them.
//
//	go run ./tools/icongen               # the committed icons, under the repository root
//	go run ./tools/icongen -msix <dir>   # the MSIX visual assets, into <dir> (the package's Assets)
//	go run ./tools/icongen -preview <dir> # every image as a PNG, for looking at
package main

import (
	"flag"
	"fmt"
	"image/png"
	"os"
	"path/filepath"

	"doc-html-translate/internal/iconart"
)

func main() {
	root := flag.String("root", ".", "repository root")
	msix := flag.String("msix", "", "write the MSIX visual assets into this folder instead")
	preview := flag.String("preview", "", "also write every image (each ICO frame too) as a PNG into this folder")
	flag.Parse()
	if err := run(*root, *msix, *preview); err != nil {
		fmt.Fprintln(os.Stderr, "icongen:", err)
		os.Exit(1)
	}
}

func run(root, msix, preview string) error {
	outs, dest := []iconart.Output(nil), root
	var err error
	if msix != "" {
		outs, err = iconart.MSIXOutputs(root)
		dest = msix
	} else {
		outs, err = iconart.RepoOutputs(root)
	}
	if err != nil {
		return err
	}
	for _, o := range outs {
		raw, err := o.Encode()
		if err != nil {
			return fmt.Errorf("%s: %w", o.Path, err)
		}
		p := filepath.Join(dest, filepath.FromSlash(o.Path))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(p, raw, 0o644); err != nil {
			return err
		}
		fmt.Printf("wrote %s (%d bytes)\n", p, len(raw))
		if preview != "" {
			if err := writePreview(preview, o); err != nil {
				return err
			}
		}
	}
	return nil
}

func writePreview(dir string, o iconart.Output) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	base := filepath.Base(o.Path)
	for _, img := range o.Images {
		b := img.Bounds()
		f, err := os.Create(filepath.Join(dir, fmt.Sprintf("%s.%dx%d.png", base, b.Dx(), b.Dy())))
		if err != nil {
			return err
		}
		err = png.Encode(f, img)
		f.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
