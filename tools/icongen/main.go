// Command icongen draws the product's pictures from the mark and the vendored glyphs
// (internal/iconart) and writes them.
//
//	go run ./tools/icongen               # the committed icons, store logos and vector forms, under the repository root
//	go run ./tools/icongen -msix <dir>   # the MSIX visual assets, into <dir> (the package's Assets)
//	go run ./tools/icongen -manifest     # measure every picture and write assets/identity/identity-manifest.json
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
	manifest := flag.Bool("manifest", false, "write the identity manifest (PRODUCT-IDENTITY section 5) from the files on disk instead")
	preview := flag.String("preview", "", "also write every image (each ICO frame too) as a PNG into this folder")
	flag.Parse()
	var err error
	if *manifest {
		err = writeManifest(*root)
	} else {
		err = run(*root, *msix, *preview)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "icongen:", err)
		os.Exit(1)
	}
}

func writeManifest(root string) error {
	raw, err := iconart.Manifest(root)
	if err != nil {
		return err
	}
	return write(filepath.Join(root, filepath.FromSlash(iconart.ManifestPath)), raw)
}

func write(p string, raw []byte) error {
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %s (%d bytes)\n", p, len(raw))
	return nil
}

func run(root, msix, preview string) error {
	outs, dest := []iconart.Output(nil), root
	var err error
	if msix != "" {
		outs, err = iconart.MSIXOutputs(root)
		dest = msix
	} else {
		outs, err = iconart.RepoOutputs(root)
		if err == nil {
			// the MSIX logos again, kept in the repository so the identity manifest can name them
			var msixCopy []iconart.Output
			msixCopy, err = iconart.MSIXIdentityOutputs(root)
			outs = append(outs, msixCopy...)
		}
	}
	if err != nil {
		return err
	}
	for _, o := range outs {
		raw, err := o.Encode()
		if err != nil {
			return fmt.Errorf("%s: %w", o.Path, err)
		}
		if err := write(filepath.Join(dest, filepath.FromSlash(o.Path)), raw); err != nil {
			return err
		}
		if preview != "" {
			if err := writePreview(preview, o); err != nil {
				return err
			}
		}
	}
	if msix == "" {
		for _, s := range iconart.SVGOutputs() {
			if err := write(filepath.Join(root, filepath.FromSlash(s.Path)), s.Data); err != nil {
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
