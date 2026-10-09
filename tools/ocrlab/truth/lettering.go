package truth

import (
	"fmt"
	"image"
	_ "image/png" // the sidecar is a PNG; registering the decoder is all this file needs
	"os"
	"path/filepath"
)

// LetteringPath is the optional sidecar that declares exactly which pixels of a scene are its
// original lettering: a PNG of the image's size, white where there is lettering.
//
// It exists because the lettering cannot be read back from the image itself on a gradient or a
// texture. The mask must come from something independent of the pixels being judged - the drawing
// code of a synthetic scene, or a person - and a mask derived from the same image would only
// reproduce the heuristic it is meant to calibrate. List core pixels only: an antialiased fringe
// that is within codec noise of the background cannot be told from background after removal.
func LetteringPath(dir, sceneID string) string {
	return filepath.Join(dir, sceneID+".lettering.png")
}

// LoadLettering reads a scene's lettering mask. A scene without a sidecar returns (nil, nil):
// absence is a legitimate state and means the lettering is unknown, not empty.
func LoadLettering(dir, sceneID string, w, h int) (*Mask, error) {
	f, err := os.Open(LetteringPath(dir, sceneID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("%s lettering mask: %w", sceneID, err)
	}
	b := img.Bounds()
	if b.Dx() != w || b.Dy() != h {
		return nil, fmt.Errorf("%s lettering mask is %dx%d, the annotation is %dx%d", sceneID, b.Dx(), b.Dy(), w, h)
	}
	m := NewMask(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bl, a := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			if a>>8 >= 128 && (r>>8+g>>8+bl>>8)/3 >= 128 {
				m.Set(x, y)
			}
		}
	}
	return m, nil
}
