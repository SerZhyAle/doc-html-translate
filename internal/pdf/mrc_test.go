package pdf

import (
	"encoding/json"
	"image"
	"image/color"
	"os"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func TestMRCCompositeFixture(t *testing.T) {
	var fixture struct {
		Background struct {
			Width, Height int
			RGB           [3]uint8
		}
		Foreground struct {
			Width, Height int
			RGB           [3]uint8
		}
		Selected [2]int
		Ordinary [2]int
	}
	data, err := os.ReadFile("../../tests/testdata/pdf_mrc_pair.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	bg := image.NewRGBA(image.Rect(0, 0, fixture.Background.Width, fixture.Background.Height))
	fg := image.NewRGBA(image.Rect(0, 0, fixture.Foreground.Width, fixture.Foreground.Height))
	for y := 0; y < bg.Bounds().Dy(); y++ {
		for x := 0; x < bg.Bounds().Dx(); x++ {
			bg.SetRGBA(x, y, color.RGBA{fixture.Background.RGB[0], fixture.Background.RGB[1], fixture.Background.RGB[2], 255})
		}
	}
	for y := 0; y < fg.Bounds().Dy(); y++ {
		for x := 0; x < fg.Bounds().Dx(); x++ {
			fg.SetRGBA(x, y, color.RGBA{fixture.Foreground.RGB[0], fixture.Foreground.RGB[1], fixture.Foreground.RGB[2], 255})
		}
	}
	mask := image.NewGray(fg.Bounds())
	for i := range mask.Pix {
		mask.Pix[i] = 255
	}
	mask.SetGray(fixture.Selected[0], fixture.Selected[1], color.Gray{Y: 0})
	got, err := composeMRCPixels(bg, fg, mask)
	if err != nil {
		t.Fatal(err)
	}
	selected := got.RGBAAt(fixture.Selected[0], fixture.Selected[1])
	ordinary := got.RGBAAt(fixture.Ordinary[0], fixture.Ordinary[1])
	if selected.R != fixture.Foreground.RGB[0] || selected.G != fixture.Foreground.RGB[1] || selected.B != fixture.Foreground.RGB[2] {
		t.Errorf("selected pixel = %v, want foreground %v", selected, fixture.Foreground.RGB)
	}
	if ordinary.R != fixture.Background.RGB[0] || ordinary.G != fixture.Background.RGB[1] || ordinary.B != fixture.Background.RGB[2] {
		t.Errorf("ordinary pixel = %v, want background %v", ordinary, fixture.Background.RGB)
	}
	background := model.Image{ObjNr: 1, Width: fixture.Background.Width, Height: fixture.Background.Height}
	foreground := model.Image{ObjNr: 2, Width: fixture.Foreground.Width, Height: fixture.Foreground.Height, HasImgMask: true}
	if b, f, ok := mrcPair(map[int]model.Image{1: background, 2: foreground}); !ok || b.ObjNr != 1 || f.ObjNr != 2 {
		t.Fatal("masked same-shape pair was not recognized")
	}
	foreground.HasImgMask = false
	foreground.HasSMask = true
	if _, _, ok := mrcPair(map[int]model.Image{1: background, 2: foreground}); ok {
		t.Fatal("soft-masked illustration mistaken for MRC")
	}
}
