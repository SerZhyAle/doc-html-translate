package ocr

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// TestGreyRenditionIsSingleChannel pins the property the rescue pass exists for. Tesseract's
// default thresholder runs Otsu per colour channel and takes ink only where every channel agrees,
// which erases the lettering on saturated artwork; the rescue works only because the image it
// hands over has one channel to threshold. An RGB copy whose three channels merely hold equal
// values would still go down the per-channel path, so "it looks grey" is not the invariant -
// 8-bit depth is.
func TestGreyRenditionIsSingleChannel(t *testing.T) {
	src := filepath.Join(t.TempDir(), "colour.png")
	writePNGFile(t, src, colourScene(60, 40))

	grey := greyRendition(src)
	if grey == nil {
		t.Fatal("greyRendition failed on a decodable image")
	}
	path, cleanup, ok := writeTempPNG(grey)
	if !ok {
		t.Fatal("could not stage the grey rendition")
	}
	defer cleanup()

	im := decodeImage(path)
	if im == nil {
		t.Fatal("grey rendition not decodable")
	}
	if _, isGrey := im.(*image.Gray); !isGrey {
		t.Errorf("grey rendition is %T, want *image.Gray (8-bit, one channel)", im)
	}
	if got, want := im.Bounds().Size(), image.Pt(60, 40); got != want {
		t.Errorf("size = %v, want %v", got, want)
	}
	// Luminance, not a channel drop: a saturated ground and white lettering must stay far apart.
	ground := color.GrayModel.Convert(im.At(2, 2)).(color.Gray).Y
	letter := color.GrayModel.Convert(im.At(30, 20)).(color.Gray).Y
	if int(letter)-int(ground) < 64 {
		t.Errorf("ground %d and lettering %d collapsed together; the rescue would read nothing", ground, letter)
	}
}

func TestGreyRenditionRejectsNonImage(t *testing.T) {
	p := filepath.Join(t.TempDir(), "not-an-image.png")
	if err := os.WriteFile(p, []byte("nonsense"), 0o644); err != nil {
		t.Fatal(err)
	}
	if greyRendition(p) != nil {
		t.Error("greyRendition accepted a file that is not an image")
	}
}

// TestOrdinaryPassCommandLineUnchanged: the rescue ladder must not alter the pass every readable
// scene already goes through. The default thresholding method is the engine's own and is left
// unnamed, so a scene that works today produces the identical command line - which is what makes
// the ladder a pure addition rather than a re-tune.
func TestOrdinaryPassCommandLineUnchanged(t *testing.T) {
	args := tesseractArgs("img.png", "eng", "", 150, thresholdEngineDefault)
	if slices.Contains(args, "thresholding_method=0") {
		t.Errorf("ordinary pass names the default thresholder: %v", args)
	}
	for _, want := range []string{"stdout", "--psm", "3", "-l", "eng", "tessedit_create_tsv=1", "user_defined_dpi=150"} {
		if !slices.Contains(args, want) {
			t.Errorf("ordinary pass lost %q: %v", want, args)
		}
	}
}

// TestRescuePassNamesItsThresholder: the second rung differs from the first only in who decides
// where ink ends, so the flag has to reach tesseract.
func TestRescuePassNamesItsThresholder(t *testing.T) {
	args := tesseractArgs("img.png", "eng", "", 0, thresholdLeptonicaOtsu)
	if !slices.Contains(args, "thresholding_method=1") {
		t.Errorf("rescue pass did not request Leptonica's tiled Otsu: %v", args)
	}
	if slices.Contains(args, "user_defined_dpi=0") {
		t.Errorf("dpi 0 must stay unnamed so tesseract estimates it: %v", args)
	}
}

// TestGreyRescueLadderOrder: the ordinary thresholder is tried first on the grey copy, because on
// the diagnostic balloon scenes greyscale alone is enough; the tiled thresholder is the deeper
// rung that a varying background (a gradient) needs. Trying them the other way round would spend
// the more expensive pass on scenes the cheap one already reads.
func TestGreyRescueLadderOrder(t *testing.T) {
	if want := []int{thresholdEngineDefault, thresholdLeptonicaOtsu}; !slices.Equal(greyRescuePasses, want) {
		t.Errorf("greyRescuePasses = %v, want %v", greyRescuePasses, want)
	}
}

// TestRescueConfidenceFloorIsStricter: a rescue is a second guess after the ordinary pass found
// nothing, so it has to clear a higher bar - otherwise the ladder turns "no text here" into a plate
// of invented words painted over artwork. Measured on the lab's corpus: genuine rescued lettering
// scored 93.1-97.0, hallucinated lettering 50.8.
func TestRescueConfidenceFloorIsStricter(t *testing.T) {
	if ocrRescueLineConf <= ocrMinLineConf {
		t.Errorf("rescue floor %v does not exceed the ordinary floor %v", ocrRescueLineConf, ocrMinLineConf)
	}
	// The measured bands either side of the floor must both stay on their own side of it.
	const worstGenuineRescue, bestHallucination = 93.1, 50.8
	if ocrRescueLineConf > worstGenuineRescue {
		t.Errorf("rescue floor %v rejects genuine lettering measured at %v", ocrRescueLineConf, worstGenuineRescue)
	}
	if ocrRescueLineConf <= bestHallucination {
		t.Errorf("rescue floor %v admits hallucinated lettering measured at %v", ocrRescueLineConf, bestHallucination)
	}
}

// TestClusterLinesHonoursItsFloor: the floor is a parameter, not the package constant, or the
// rescue passes would silently reuse the ordinary gate.
func TestClusterLinesHonoursItsFloor(t *testing.T) {
	line := func(conf float64, y int) *ocrLine {
		l := &ocrLine{x0: 10, y0: y, x1: 200, y1: y + 20, confSum: conf, confN: 1}
		l.text.WriteString("Readable enough words here")
		return l
	}
	lines := []*ocrLine{line(60, 10), line(95, 40)}
	if got := len(clusterLines(lines, ocrMinLineConf)); got != 1 {
		t.Errorf("ordinary floor kept %d plates, want 1 (both lines clustered)", got)
	}
	if got := len(clusterLines(lines, ocrRescueLineConf)); got != 1 {
		t.Errorf("rescue floor kept %d plates, want 1 (only the confident line)", got)
	}
	if got := clusterLines(lines, ocrRescueLineConf); len(got) == 1 && got[0].Y0 != 40 {
		t.Errorf("rescue floor kept the line at y=%d, want the confident one at y=40", got[0].Y0)
	}
	if got := len(clusterLines(lines, 99)); got != 0 {
		t.Errorf("a floor above every line kept %d plates, want 0", got)
	}
}

// colourScene draws white lettering on a saturated ground - the shape of the comic-balloon case
// that the colour pass cannot read.
func colourScene(w, h int) image.Image {
	im := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			im.SetRGBA(x, y, color.RGBA{188, 92, 64, 255})
		}
	}
	for y := 15; y < 25; y++ {
		for x := 20; x < 45; x++ {
			im.SetRGBA(x, y, color.RGBA{255, 255, 255, 255})
		}
	}
	return im
}

func writePNGFile(t *testing.T, path string, im image.Image) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, im); err != nil {
		t.Fatal(err)
	}
}
