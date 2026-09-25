package ocr

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
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
	args := tesseractArgs("img.png", "eng", "", 150, thresholdEngineDefault, ocrPageSegMode)
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
	args := tesseractArgs("img.png", "eng", "", 0, thresholdLeptonicaOtsu, ocrPageSegMode)
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
// the more expensive pass on scenes the cheap one already reads. The rung that gives up layout
// analysis comes after both, because layout analysis is what holds a real page together and only
// input that is not a page benefits from losing it.
//
// Asserted as the sequence of rungs rather than as a list of integers: a test that pinned the
// numbers would pass with the modes swapped between rungs.
func TestGreyRescueLadderOrder(t *testing.T) {
	want := []rescueRung{
		{thresholding: thresholdEngineDefault, psm: ocrPageSegMode},
		{thresholding: thresholdLeptonicaOtsu, psm: ocrPageSegMode},
		{thresholding: thresholdEngineDefault, psm: ocrSparsePageSegMode},
	}
	if !slices.Equal(greyRescuePasses, want) {
		t.Fatalf("greyRescuePasses = %v, want %v", greyRescuePasses, want)
	}
	last := greyRescuePasses[len(greyRescuePasses)-1]
	for _, rung := range greyRescuePasses[:len(greyRescuePasses)-1] {
		if rung.psm != ocrPageSegMode {
			t.Errorf("rung %v gives up layout analysis before the last rung does", rung)
		}
	}
	if last.psm != ocrSparsePageSegMode {
		t.Errorf("the last rung asks for %d, want the sparse mode %d", last.psm, ocrSparsePageSegMode)
	}
}

// TestSparseRungAsksForSparseSegmentation: the rung differs from the first only in what tesseract
// is told to look for, so the mode has to reach the command line - and the ordinary pass must keep
// asking for the page mode.
func TestSparseRungAsksForSparseSegmentation(t *testing.T) {
	ordinary := tesseractArgs("img.png", "rus", "", 232, thresholdEngineDefault, ocrPageSegMode)
	sparse := tesseractArgs("img.png", "rus", "", 232, thresholdEngineDefault, ocrSparsePageSegMode)
	if i := slices.Index(ordinary, "--psm"); i < 0 || ordinary[i+1] != "3" {
		t.Errorf("ordinary pass no longer asks for the page mode: %v", ordinary)
	}
	if i := slices.Index(sparse, "--psm"); i < 0 || sparse[i+1] != "11" {
		t.Errorf("sparse rung did not ask for sparse text: %v", sparse)
	}
}

// TestGreyRescueKeepsTheStrongestRung: the ladder used to stop at the first rung that produced any
// plate at all, which is how one recovered word ("| МОЖЕМ" on the reported poster) ended the search
// before the sparse rung could recover six lines. The comparator is what makes the later rung
// reachable, and it must never let a poorer rung displace a richer one.
func TestGreyRescueKeepsTheStrongestRung(t *testing.T) {
	oneWord := Result{Blocks: []Block{{Text: "| МОЖЕМ"}}}
	sixLines := Result{Blocks: []Block{{Text: "ТРАХАТЬСЯ: МЫ ЖЕ ЛЮДИ, МОЖЕМ ПРОСТО ПОГОВОРИТЬ"}}}
	if !strictlyBetter(sixLines, oneWord) {
		t.Error("the sparse rung's result cannot displace the first rung's one word")
	}
	if strictlyBetter(oneWord, sixLines) {
		t.Error("a later, poorer rung displaced a richer earlier one")
	}
	if strictlyBetter(Result{}, oneWord) {
		t.Error("a rung that found nothing displaced a rung that found something")
	}
}

// TestRescueConfidenceFloorIsStricter: a rescue is a second guess after the ordinary pass found
// nothing, so it has to clear a higher bar - otherwise the ladder turns "no text here" into a plate
// of invented words painted over artwork. The band is the 2026-08-15 corpus measurement
// (DEV/research/ocr_rescue_floor_2026-08-15.md): invented lettering reached 73.9 and genuine rescued
// lettering 69.2, so the populations overlap and the floor sits above both. It pins the side the
// corpus chose - no measured invention gets through - and states the price, rather than citing the
// older 93.1 / 50.8 pair that described a gap which no longer exists.
func TestRescueConfidenceFloorIsStricter(t *testing.T) {
	if ocrRescueLineConf <= ocrMinLineConf {
		t.Errorf("rescue floor %v does not exceed the ordinary floor %v", ocrRescueLineConf, ocrMinLineConf)
	}
	// Genuine rescued lettering was measured up to 69.2, under this bound: that loss is known and is
	// what ticket P49's follow-up reopens with a third axis, not by lowering this number.
	const bestInvention = 73.9
	if ocrRescueLineConf <= bestInvention {
		t.Errorf("rescue floor %v admits invented lettering measured at %v", ocrRescueLineConf, bestInvention)
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
	if got := len(clusterLines(lines, ocrMinLineConf, 400, 400)); got != 1 {
		t.Errorf("ordinary floor kept %d plates, want 1 (both lines clustered)", got)
	}
	if got := len(clusterLines(lines, ocrRescueLineConf, 400, 400)); got != 1 {
		t.Errorf("rescue floor kept %d plates, want 1 (only the confident line)", got)
	}
	if got := clusterLines(lines, ocrRescueLineConf, 400, 400); len(got) == 1 && got[0].Y0 != 40 {
		t.Errorf("rescue floor kept the line at y=%d, want the confident one at y=40", got[0].Y0)
	}
	if got := len(clusterLines(lines, 99, 400, 400)); got != 0 {
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

// The poster's sparse rung, as the probe of 2026-09-25 recorded it: mean confidence and ink height
// (upscaled space) of every line, read once with `rus` and once with the app's default `eng`
// (DEV/research/ocr_rescue_third_axis_2026-09-25.md). Only the heights and confidences matter to the
// rule, so the boxes are stacked at those heights.
func posterRung(rows []fixtureLine) []*ocrLine {
	y := 0
	for i := range rows {
		h := rows[i].y1
		rows[i].x0, rows[i].x1, rows[i].y0, rows[i].y1 = 40, 600, y, y+h
		y += h + 20
	}
	return fixtureLines(rows)
}

var posterRungRus = []fixtureLine{
	{y1: 287, conf: 69.2, text: "ЗАЧЕМ"},
	{y1: 281, conf: 80.7, text: "ТРАХАТЬСЯ:"},
	{y1: 155, conf: 96.1, text: "МЫ ЖЕ"},
	{y1: 154, conf: 41.2, text: "ВЗРОСЛЫЕ"},
	{y1: 171, conf: 92.6, text: "ЛЮДИ,"},
	{y1: 153, conf: 95.9, text: "МОЖЕМ"},
	{y1: 142, conf: 73.9, text: "ОБ ЗЛОМ"},
	{y1: 149, conf: 87.2, text: "ПРОСТО"},
	{y1: 137, conf: 95.0, text: "ПОГОВОРИТЬ"},
}

var posterRungEng = []fixtureLine{
	{y1: 287, conf: 0.0, text: "SAuEM"},
	{y1: 281, conf: 17.7, text: "TPAXATbGR:"},
	{y1: 155, conf: 34.8, text: "Mbl KE"},
	{y1: 153, conf: 34.1, text: "MODKEM"},
	{y1: 142, conf: 47.0, text: "Ob STOM"},
	{y1: 149, conf: 63.4, text: "NPOCTO"},
	{y1: 137, conf: 0.0, text: "NOTOBOPHTE"},
}

func admittedTexts(lines []*ocrLine) []string {
	var out []string
	for _, l := range lines {
		if l.admitted {
			out = append(out, l.text.String())
		}
	}
	return out
}

// TestAdmitBySizeKeepsTheHeadlineUnderTheRightLanguage is the ticket's case: ЗАЧЕМ at 69.2 is the
// size of ТРАХАТЬСЯ:, which the rung trusts at 80.7, so it is kept. ВЗРОСЛЫЕ at 41.2 is under the
// word rule's floor and stays out - the rule relieves the floor, it does not remove it.
func TestAdmitBySizeKeepsTheHeadlineUnderTheRightLanguage(t *testing.T) {
	lines := posterRung(slices.Clone(posterRungRus))
	admitBySize(lines, ocrRescueLineConf)
	if got, want := admittedTexts(lines), []string{"ЗАЧЕМ", "ОБ ЗЛОМ"}; !slices.Equal(got, want) {
		t.Errorf("admitted %q, want %q", got, want)
	}
	for _, l := range lines {
		if l.text.String() == "ЗАЧЕМ" && !keepLine(l, ocrRescueLineConf) {
			t.Error("keepLine drops an admitted line - the plates and the record would disagree")
		}
	}
}

// TestAdmitBySizeAdmitsNothingUnderTheWrongAlphabet is the regression that sank the 2026-08-15 rule:
// read with `eng`, NPOCTO carries a word and clears the word floor, but no line of the rung is
// trusted, so there is nothing to be the same size as and the poster gets no plate.
func TestAdmitBySizeAdmitsNothingUnderTheWrongAlphabet(t *testing.T) {
	lines := posterRung(slices.Clone(posterRungEng))
	admitBySize(lines, ocrRescueLineConf)
	if got := admittedTexts(lines); len(got) != 0 {
		t.Errorf("admitted %q with no trusted line on the page", got)
	}
	if got := clusterLines(lines, ocrRescueLineConf, 1920, 2560); len(got) != 0 {
		t.Errorf("the poster read with eng got %d plates, want none", len(got))
	}
}

// TestAdmitBySizeNeedsAWordAnchor: a stray glyph that clears the floor is not lettering the pass
// trusts. On the poster read with `eng`, rung 1 clears a `\` at 81.7 - an anchor made of it would
// hand the rule a size to match.
func TestAdmitBySizeNeedsAWordAnchor(t *testing.T) {
	lines := posterRung([]fixtureLine{
		{y1: 150, conf: 81.7, text: "\\"},
		{y1: 149, conf: 63.4, text: "NPOCTO"},
	})
	admitBySize(lines, ocrRescueLineConf)
	if got := admittedTexts(lines); len(got) != 0 {
		t.Errorf("admitted %q on a glyph anchor", got)
	}
}

// TestAdmitBySizeNeedsTheSameTypeSize: a word under the floor next to trusted lettering of another
// size is not relieved - a heading does not vouch for a caption, or the other way round.
func TestAdmitBySizeNeedsTheSameTypeSize(t *testing.T) {
	lines := posterRung([]fixtureLine{
		{y1: 281, conf: 80.7, text: "ТРАХАТЬСЯ:"},
		{y1: 142, conf: 73.9, text: "ОБ ЗЛОМ"},
	})
	admitBySize(lines, ocrRescueLineConf)
	if got := admittedTexts(lines); len(got) != 0 {
		t.Errorf("admitted %q across type sizes", got)
	}
}

// TestAdmitBySizeNeedsAWord: the word rule's run is part of the gate; a short line of the anchor's
// size - debris like `Cor` - stays out however confident it is.
func TestAdmitBySizeNeedsAWord(t *testing.T) {
	lines := posterRung([]fixtureLine{
		{y1: 150, conf: 95.9, text: "МОЖЕМ"},
		{y1: 150, conf: 69.0, text: "Cor"},
	})
	admitBySize(lines, ocrRescueLineConf)
	if got := admittedTexts(lines); len(got) != 0 {
		t.Errorf("admitted %q with a letter run under %d", got, ocrRescueWordRun)
	}
}

// TestParseTSVDoesNotAdmitBySize: the ordinary pass and the screen sweep go through parseTSV /
// parsePass(false) and must keep their floor exactly.
func TestParseTSVDoesNotAdmitBySize(t *testing.T) {
	tsv := "level\tpage_num\tblock_num\tpar_num\tline_num\tword_num\tleft\ttop\twidth\theight\tconf\ttext\n" +
		"1\t1\t0\t0\t0\t0\t0\t0\t1000\t1000\t-1\t\n" +
		"4\t1\t1\t1\t1\t0\t40\t10\t400\t281\t-1\t\n" +
		"5\t1\t1\t1\t1\t1\t40\t10\t400\t281\t80.7\tТРАХАТЬСЯ:\n" +
		"4\t1\t1\t1\t2\t0\t40\t320\t300\t287\t-1\t\n" +
		"5\t1\t1\t1\t2\t1\t40\t320\t300\t287\t69.2\tЗАЧЕМ\n"
	plain, err := parseTSV([]byte(tsv), ocrRescueLineConf, nil)
	if err != nil {
		t.Fatal(err)
	}
	relieved, err := parsePass([]byte(tsv), ocrRescueLineConf, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(plain.Dropped) != 1 || plain.Dropped[0].Text != "ЗАЧЕМ" {
		t.Errorf("parseTSV dropped %+v, want ЗАЧЕМ", plain.Dropped)
	}
	if len(relieved.Dropped) != 0 {
		t.Errorf("parsePass(bySize) still records %+v as dropped", relieved.Dropped)
	}
	if n := len(relieved.Blocks); n == 0 || !strings.Contains(relieved.Blocks[0].Text, "ЗАЧЕМ") {
		t.Errorf("parsePass(bySize) blocks = %+v, want ЗАЧЕМ on a plate", relieved.Blocks)
	}
}

func TestLongestLetterRun(t *testing.T) {
	for s, want := range map[string]int{"": 0, "4 y": 1, "TPAXATBCR: 4 y": 9, "ОБ ЗЛОМ": 4, "Cor!": 3, "$ЫСНТЕУ.": 6} {
		if got := longestLetterRun(s); got != want {
			t.Errorf("longestLetterRun(%q) = %d, want %d", s, got, want)
		}
	}
}
