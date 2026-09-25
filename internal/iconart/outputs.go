package iconart

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
)

// Output is one file this package draws: a PNG (one image) or an ICO (one frame per image).
type Output struct {
	Path   string // slash-separated, relative to the root it is written under
	Images []image.Image
}

// ICO sizes of ICON-RENDER rule 9: the program icon and the document type from 16 to 256,
// the Explorer verb at 16 with its 20, 24 and 32 display-scaling forms.
var (
	appICOSizes  = []int{16, 20, 24, 32, 48, 64, 256}
	verbICOSizes = []int{16, 20, 24, 32}
	// ExtensionSizes are the action icon sizes extension/manifest.json names.
	ExtensionSizes = []int{16, 32, 48, 128}
	// msixTargetSizes are the Square44x44Logo targetsize-* set Windows picks from.
	msixTargetSizes = []int{16, 24, 32, 48, 256}
)

// Glyph ids the system surfaces draw (ICON-RENDER rule 9).
const (
	VerbGlyph         = "action.convert"   // the "Convert to HTML" Explorer verb
	DocumentTypeGlyph = "content.document" // "a text, PDF or other readable document"
)

// ExeIcons are the ICOs embedded in both executables, in resource-index order: each
// cmd/*/versioninfo.json IconPath lists them so, and internal/windowsreg names the index
// ("<exe>",1 for the verb, "<exe>",2 for the document type).
var ExeIcons = []string{"assets/doc-html-translate.ico", "assets/convert-verb.ico", "assets/document-type.ico"}

var pathD = regexp.MustCompile(`\bd="([^"]+)"`)

// GlyphPath reads the path data of a vendored glyph, assets/glyphs/<id>.svg under root.
func GlyphPath(root, id string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(root, "assets", "glyphs", id+".svg"))
	if err != nil {
		return "", err
	}
	m := pathD.FindSubmatch(raw)
	if m == nil {
		return "", fmt.Errorf("assets/glyphs/%s.svg has no path data", id)
	}
	return string(m[1]), nil
}

func markFrames(sizes []int) []image.Image {
	out := make([]image.Image, len(sizes))
	for i, s := range sizes {
		out[i] = Mark(s, s, float64(s), Plated)
	}
	return out
}

func glyphFrames(root, id string, sizes []int) ([]image.Image, error) {
	d, err := GlyphPath(root, id)
	if err != nil {
		return nil, err
	}
	out := make([]image.Image, len(sizes))
	for i, s := range sizes {
		if out[i], err = Glyph(d, s, OneTon); err != nil {
			return nil, fmt.Errorf("%s: %w", id, err)
		}
	}
	return out, nil
}

// RepoOutputs are the committed render targets, relative to the repository root.
func RepoOutputs(root string) ([]Output, error) {
	verb, err := glyphFrames(root, VerbGlyph, verbICOSizes)
	if err != nil {
		return nil, err
	}
	doc, err := glyphFrames(root, DocumentTypeGlyph, appICOSizes)
	if err != nil {
		return nil, err
	}
	app := markFrames(appICOSizes)
	outs := []Output{
		{Path: "assets/doc-html-translate.ico", Images: app},
		// The browser-tab icons of the GUI window and of a converted book are the same file,
		// embedded by those packages.
		{Path: "cmd/doc-html-ui/favicon.ico", Images: app},
		{Path: "internal/htmlgen/favicon.ico", Images: app},
		{Path: "assets/convert-verb.ico", Images: verb},
		{Path: "assets/document-type.ico", Images: doc},
	}
	for _, s := range ExtensionSizes {
		outs = append(outs, Output{
			Path:   fmt.Sprintf("extension/icons/icon%d.png", s),
			Images: []image.Image{Mark(s, s, float64(s), Plated)},
		})
	}
	return outs, nil
}

// MSIXOutputs are the package's visual assets, relative to its Assets folder. The names carry
// MRT qualifiers (scale-*, targetsize-*, altform-*), which Windows resolves only through the
// package's resources.pri - build-msix.ps1 runs makepri after writing them.
func MSIXOutputs(root string) ([]Output, error) {
	var outs []Output
	add := func(name string, img image.Image) {
		outs = append(outs, Output{Path: name, Images: []image.Image{img}})
	}
	// Square44x44Logo: the app list, the taskbar, Alt+Tab. The plated forms are drawn on the
	// manifest BackgroundColor, so they carry the sheet alone; the unplated forms, which the
	// taskbar and Start prefer, carry the mark on its own plate and hold on light and dark.
	for _, s := range msixTargetSizes {
		add(fmt.Sprintf("Square44x44Logo.targetsize-%d.png", s), Mark(s, s, float64(s), OnPlatform))
		add(fmt.Sprintf("Square44x44Logo.targetsize-%d_altform-unplated.png", s), Mark(s, s, float64(s), Plated))
		add(fmt.Sprintf("Square44x44Logo.targetsize-%d_altform-lightunplated.png", s), Mark(s, s, float64(s), Plated))
	}
	for _, sc := range []int{100, 200} {
		k := float64(sc) / 100
		px := func(v int) int { return int(float64(v) * k) }
		add(fmt.Sprintf("Square44x44Logo.scale-%d.png", sc), Mark(px(44), px(44), float64(px(44)), OnPlatform))
		add(fmt.Sprintf("StoreLogo.scale-%d.png", sc), Mark(px(50), px(50), float64(px(50)), Plated))
		// Tiles show the name along the bottom edge, so the mark sits in the upper-middle box.
		add(fmt.Sprintf("Square71x71Logo.scale-%d.png", sc), Mark(px(71), px(71), float64(px(71))*0.8, OnPlatform))
		add(fmt.Sprintf("Square150x150Logo.scale-%d.png", sc), Mark(px(150), px(150), float64(px(150))*0.6, OnPlatform))
		add(fmt.Sprintf("Wide310x150Logo.scale-%d.png", sc), Mark(px(310), px(150), float64(px(150))*0.6, OnPlatform))
	}
	// The registered document type: content.document in the one tone for Explorer (unplated),
	// in white for the plated form Windows draws on the navy BackgroundColor.
	d, err := GlyphPath(root, DocumentTypeGlyph)
	if err != nil {
		return nil, err
	}
	for _, s := range msixTargetSizes {
		plated, err := Glyph(d, s, Sheet)
		if err != nil {
			return nil, err
		}
		unplated, _ := Glyph(d, s, OneTon)
		add(fmt.Sprintf("DocumentType.targetsize-%d.png", s), plated)
		add(fmt.Sprintf("DocumentType.targetsize-%d_altform-unplated.png", s), unplated)
		add(fmt.Sprintf("DocumentType.targetsize-%d_altform-lightunplated.png", s), unplated)
	}
	for _, sc := range []int{100, 200} {
		s := 44 * sc / 100
		img, _ := Glyph(d, s, Sheet)
		add(fmt.Sprintf("DocumentType.scale-%d.png", sc), img)
	}
	return outs, nil
}

// Encode returns the file bytes: an ICO for a .ico path, else a PNG of the one image.
func (o Output) Encode() ([]byte, error) {
	if filepath.Ext(o.Path) == ".ico" {
		return encodeICO(o.Images)
	}
	var b bytes.Buffer
	err := png.Encode(&b, o.Images[0])
	return b.Bytes(), err
}

// encodeICO writes every frame PNG-compressed (Windows Vista and later read that at any
// size), in the order given.
func encodeICO(frames []image.Image) ([]byte, error) {
	pngs := make([][]byte, len(frames))
	for i, f := range frames {
		var b bytes.Buffer
		if err := png.Encode(&b, f); err != nil {
			return nil, err
		}
		pngs[i] = b.Bytes()
	}
	var out bytes.Buffer
	le := func(v any) { _ = binary.Write(&out, binary.LittleEndian, v) }
	le(uint16(0))           // reserved
	le(uint16(1))           // type: icon
	le(uint16(len(frames))) // count
	offset := 6 + 16*len(frames)
	for i, f := range frames {
		b := f.Bounds()
		dim := func(v int) uint8 {
			if v >= 256 {
				return 0 // 0 means 256 in an ICONDIRENTRY
			}
			return uint8(v)
		}
		le(dim(b.Dx()))
		le(dim(b.Dy()))
		le(uint8(0))  // palette size
		le(uint8(0))  // reserved
		le(uint16(1)) // planes
		le(uint16(32))
		le(uint32(len(pngs[i])))
		le(uint32(offset))
		offset += len(pngs[i])
	}
	for _, p := range pngs {
		out.Write(p)
	}
	return out.Bytes(), nil
}

// DecodeICO returns the frames of an ICO written by encodeICO (PNG frames only).
func DecodeICO(raw []byte) ([]image.Image, error) {
	if len(raw) < 6 || binary.LittleEndian.Uint16(raw[2:]) != 1 {
		return nil, fmt.Errorf("not an icon file")
	}
	n := int(binary.LittleEndian.Uint16(raw[4:]))
	if len(raw) < 6+16*n {
		return nil, fmt.Errorf("icon directory of %d entries runs past the end of the file", n)
	}
	out := make([]image.Image, 0, n)
	for i := 0; i < n; i++ {
		e := raw[6+16*i:]
		size := binary.LittleEndian.Uint32(e[8:])
		off := binary.LittleEndian.Uint32(e[12:])
		// Summed in 64 bits: off+size in uint32 can wrap and pass the check.
		if uint64(off)+uint64(size) > uint64(len(raw)) {
			return nil, fmt.Errorf("frame %d runs past the end of the file", i)
		}
		img, err := png.Decode(bytes.NewReader(raw[off : off+size]))
		if err != nil {
			return nil, fmt.Errorf("frame %d: %w", i, err)
		}
		out = append(out, img)
	}
	return out, nil
}
