package pdf

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"

	"doc-html-translate/internal/limits"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/tannevaled/gobig2"
	"golang.org/x/image/draw"
)

// mrcPair recognizes a background and a foreground painted through a hard mask.
// A soft mask is ordinary transparency, and a lone masked image is left alone.
func mrcPair(imgs map[int]model.Image) (background, foreground model.Image, ok bool) {
	var list []model.Image
	for _, img := range imgs {
		if !img.Thumb {
			list = append(list, img)
		}
	}
	if len(list) != 2 {
		return background, foreground, false
	}
	a, b := list[0], list[1]
	if a.HasImgMask && !b.HasImgMask {
		a, b = b, a
	}
	if a.HasImgMask || !b.HasImgMask || a.HasSMask || b.HasSMask || !sameShapeRaster(a, b) ||
		imagePixels(b) < 4*imagePixels(a) {
		return background, foreground, false
	}
	return a, b, true
}

// maskForImage reads the separate /Mask XObject. pdfcpu exposes the flag on its
// image stub but does not apply the mask when extracting the foreground raster.
func maskForImage(ctx *model.Context, foreground model.Image) (*image.Gray, error) {
	if err := limits.CheckPixels(int64(foreground.Width), int64(foreground.Height)); err != nil {
		return nil, err
	}
	entry := ctx.Table[foreground.ObjNr]
	if entry == nil {
		return nil, fmt.Errorf("foreground object %d missing", foreground.ObjNr)
	}
	sd, _, err := ctx.DereferenceStreamDict(entry.Object)
	if err != nil || sd == nil {
		return nil, fmt.Errorf("foreground stream: %v", err)
	}
	maskObj := sd.Dict["Mask"]
	mask, _, err := ctx.DereferenceStreamDict(maskObj)
	if err != nil || mask == nil {
		return nil, fmt.Errorf("mask stream: %v", err)
	}
	if len(mask.FilterPipeline) != 1 || mask.FilterPipeline[0].Name != "JBIG2Decode" {
		return nil, fmt.Errorf("unsupported MRC mask filter")
	}
	if isMask := mask.BooleanEntry("ImageMask"); isMask == nil || !*isMask {
		return nil, fmt.Errorf("MRC stencil is not an image mask")
	}
	if mask.IntEntry("Width") == nil || mask.IntEntry("Height") == nil ||
		*mask.IntEntry("Width") != foreground.Width || *mask.IntEntry("Height") != foreground.Height {
		return nil, fmt.Errorf("MRC mask dimensions differ from foreground")
	}
	var globals []byte
	if globalsObj, hasGlobals := mask.FilterPipeline[0].DecodeParms["JBIG2Globals"]; hasGlobals {
		globalStream, _, err := ctx.DereferenceStreamDict(globalsObj)
		if err != nil || globalStream == nil {
			return nil, fmt.Errorf("JBIG2Globals stream: %v", err)
		}
		globals = globalStream.Raw
	}
	decoder, err := gobig2.NewDecoderEmbedded(bytes.NewReader(mask.Raw), globals)
	if err != nil {
		return nil, err
	}
	decoded, err := decoder.Decode()
	if err != nil {
		return nil, err
	}
	gray, ok := decoded.(*image.Gray)
	if !ok || gray.Bounds().Dx() != foreground.Width || gray.Bounds().Dy() != foreground.Height {
		return nil, fmt.Errorf("decoded mask has unexpected shape")
	}
	return gray, nil
}

func readMRCRaster(runCtx context.Context, img model.Image, dir string) (image.Image, error) {
	var err error
	img, err = tiffAsPNG(img)
	if err != nil {
		return nil, err
	}
	file, err := os.CreateTemp(dir, "mrc-*."+img.FileType)
	if err != nil {
		return nil, err
	}
	path := file.Name()
	_ = file.Close()
	defer func() { _ = os.Remove(path) }()
	if err := writeImageFile(runCtx, path, img); err != nil {
		return nil, err
	}
	if strings.EqualFold(img.FileType, "jpx") {
		path, err = convertJPXFile(runCtx, path)
		if err != nil {
			return nil, err
		}
		defer func() { _ = os.Remove(path) }()
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	decoded, _, err := image.Decode(f)
	return decoded, err
}

func composeMRCPixels(background, foreground image.Image, mask *image.Gray) (*image.RGBA, error) {
	w, h := foreground.Bounds().Dx(), foreground.Bounds().Dy()
	if w <= 0 || h <= 0 || mask.Bounds().Dx() != w || mask.Bounds().Dy() != h {
		return nil, fmt.Errorf("invalid MRC image dimensions")
	}
	if err := limits.CheckPixels(int64(w), int64(h)); err != nil {
		return nil, err
	}
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.ApproxBiLinear.Scale(out, out.Bounds(), background, background.Bounds(), draw.Src, nil)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// gobig2 represents selected stencil pixels as black. The vast
			// white area is the transparent part of the foreground image.
			if mask.GrayAt(x, y).Y < 128 {
				out.Set(x, y, color.NRGBAModel.Convert(foreground.At(foreground.Bounds().Min.X+x, foreground.Bounds().Min.Y+y)))
			}
		}
	}
	return out, nil
}

func writeMRCComposite(runCtx context.Context, pdfCtx *model.Context, bg, fg model.Image, dir, name string) error {
	mask, err := maskForImage(pdfCtx, fg)
	if err != nil {
		return err
	}
	background, err := readMRCRaster(runCtx, bg, dir)
	if err != nil {
		return err
	}
	foreground, err := readMRCRaster(runCtx, fg, dir)
	if err != nil {
		return err
	}
	composite, err := composeMRCPixels(background, foreground, mask)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	err = jpeg.Encode(f, composite, &jpeg.Options{Quality: 90})
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(path)
	}
	return err
}
