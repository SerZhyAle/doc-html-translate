package ocr

import (
	"fmt"
	"image"
	"strings"
)

// Mode is how a plate conceals the source lettering under it. The names are the values of the
// plate's data-ocr-mode attribute and of the lab's evidence.Mode, and the extension's
// ocr-conceal.js uses the same strings (docs/PARITY.md "OCR", TestParityOCRConcealment).
//
// Every mode is CSS over the untouched <img>: the source image is never modified, and hiding the
// overlay (html.dht-ocr-off) shows it exactly as it was. The two modes besides the fill are what
// OCR-OVERLAY rule 7 admits since 1.1; the mechanism and its numbers are OCR-PIPELINE 1.3.
type Mode string

const (
	// ModeFill paints the block rectangle with the sampled paper colour - the plate as it has always
	// been. Chosen when the ring around the block is flat.
	ModeFill Mode = "fill"
	// ModeReconstruct paints the block with a linear gradient between the colours on two opposite
	// sides of its ring, so a caption on a sky or a vignette is not cut out as a flat patch.
	ModeReconstruct Mode = "reconstruct"
	// ModeMask paints only the block's line boxes, each padded a little, and leaves the space
	// between and beside them transparent. Chosen when the ring is busy - halftone, hatching, a
	// drawing or an outline running past the text - because there a block-wide patch erases more
	// artwork than the lettering it hides.
	ModeMask Mode = "mask"
)

// Concealment-mode decision, shared verbatim with ocr-conceal.js and held equal by
// TestParityOCRConcealment. The ring is the band ringNearerInk reads (a third of a line outside
// the block on each side), so the decision looks at the text's own surroundings and never at the
// median of the whole paragraph (strategic spec §7).
//
// Measured over all 162 blocks the desktop engine plates on the 38 of the 47 lab scenes that get one
// (DEV/research/RESEARCH_ocr-concealment-modes_2026-09-26.md):
//
//   - modeBusyMax: the share of ring pixels further than inkDeviationMin from their own side's
//     median. The share runs continuously from 0 to 0.80, so the bound is placed where it decides
//     something. On a one-line block the mask and the fill paint the same pixels - the padded line
//     box is clipped to the plate, which is that line box - so only the 47 blocks of two or more
//     lines are affected, and those split with a gap: 0.000-0.059 on paper, balloon interiors and
//     the gradient (the highest is a seven-line caption on a comic cover), 0.073 and up where the
//     block rectangle reaches past its short lines into a balloon outline, hatching or artwork
//     (samson-and-delilah-15 block 7, atomicwar0301), with the halftone scene at 0.25.
//   - modeFlatSpread: the channel-distance sum between two sides' medians above which they differ.
//     Scan noise and paper tone between two sides of a flat block reach 37 (a Hohlwein poster
//     title); the synthetic gradient caption measures 51. Whether a difference is a ramp is decided
//     by gradientAxis, not by this number alone.
//   - modeMaskPadDivisor: a masked line box grows by its line height over this on each side (at
//     least ringMinPad), so a glyph's antialiased edge and a descender the recognizer's box stops
//     short of are covered while most of the leading between two lines stays open.
const (
	// OCR-OVERLAY rule 13: derived - RESEARCH_ocr-concealment-modes_2026-09-26 (OCR-PIPELINE
	// amendment 1.3 D).
	modeBusyMax = 0.06
	// OCR-OVERLAY rule 13: derived - RESEARCH_ocr-concealment-modes_2026-09-26 (OCR-PIPELINE
	// amendment 1.3 D).
	modeFlatSpread = 40
	// OCR-OVERLAY rule 13: policy - covers an antialiased edge and a descender, leaves most of the
	// leading open; not measured.
	modeMaskPadDivisor = 6
)

// ringSide is one side of the band around a block: its median colour and its busy count.
type ringSide struct {
	med     [3]int
	busy, n int
	hasMed  bool
}

// ringStats is what the decision reads from the band around a block.
type ringStats struct {
	top, bottom, left, right ringSide
	busy, n                  int
}

func sideOf(img image.Image, x0, y0, x1, y1 int) ringSide {
	var s ringSide
	if x1-x0 < 1 || y1-y0 < 1 {
		return s
	}
	rs, gs, bs := samplePixels(img, x0, y0, x1, y1)
	if len(rs) == 0 {
		return s
	}
	s.med = [3]int{medianOf(rs), medianOf(gs), medianOf(bs)}
	s.hasMed = true
	s.n = len(rs)
	for i := range rs {
		if absInt(rs[i]-s.med[0])+absInt(gs[i]-s.med[1])+absInt(bs[i]-s.med[2]) > inkDeviationMin {
			s.busy++
		}
	}
	return s
}

// measureRing samples the four sides of the band just outside the block. The geometry is
// ringNearerInk's: the band is lh/ringPadDivisor (at least ringMinPad) wide, clamped to the image,
// with the corners belonging to the top and bottom sides.
func measureRing(img image.Image, b Block) ringStats {
	bnds := img.Bounds()
	x0 := clampInt(b.X0, bnds.Min.X, bnds.Max.X)
	y0 := clampInt(b.Y0, bnds.Min.Y, bnds.Max.Y)
	x1 := clampInt(b.X1, bnds.Min.X, bnds.Max.X)
	y1 := clampInt(b.Y1, bnds.Min.Y, bnds.Max.Y)
	lh := b.LineH
	if lh < 1 {
		lh = y1 - y0
	}
	pad := max(lh/ringPadDivisor, ringMinPad)
	ox0 := clampInt(x0-pad, bnds.Min.X, bnds.Max.X)
	oy0 := clampInt(y0-pad, bnds.Min.Y, bnds.Max.Y)
	ox1 := clampInt(x1+pad, bnds.Min.X, bnds.Max.X)
	oy1 := clampInt(y1+pad, bnds.Min.Y, bnds.Max.Y)

	var r ringStats
	r.top = sideOf(img, ox0, oy0, ox1, y0)
	r.bottom = sideOf(img, ox0, y1, ox1, oy1)
	r.left = sideOf(img, ox0, y0, x0, y1)
	r.right = sideOf(img, x1, y0, ox1, y1)
	for _, s := range []ringSide{r.top, r.bottom, r.left, r.right} {
		r.busy += s.busy
		r.n += s.n
	}
	return r
}

// spread is the channel-distance sum between two sides' medians, 0 when either side is missing
// (a block against the image edge has no band on that side and cannot show a gradient across it).
func spread(a, b ringSide) int {
	if !a.hasMed || !b.hasMed {
		return 0
	}
	return absInt(a.med[0]-b.med[0]) + absInt(a.med[1]-b.med[1]) + absInt(a.med[2]-b.med[2])
}

// decideMode turns the ring into a mode and a confidence in [0,1] - the concealment-mode decision
// (Phase 07 Step 07.1's chooseMode, split in two so the caller keeps the ring for the background).
//
// A ring too thin to judge (fewer than ringMinSamples pixels - a block filling the picture) is a
// mask at confidence 0: with no evidence about the surface, the mode that paints least is the one
// that cannot erase artwork. A busy ring is a mask whose confidence grows with how far past the
// bound it is. Otherwise the surface is paper or a gradient, and the confidence is how far the
// busy share sits under the bound - the risk that the patch lands on something that is not paper.
func decideMode(r ringStats) (Mode, float64) {
	if r.n < ringMinSamples {
		return ModeMask, 0
	}
	busy := float64(r.busy) / float64(r.n)
	if busy > modeBusyMax {
		return ModeMask, clamp01((busy - modeBusyMax) / modeBusyMax)
	}
	conf := clamp01(1 - busy/modeBusyMax)
	if _, _, _, ok := gradientAxis(r); ok {
		return ModeReconstruct, conf
	}
	return ModeFill, conf
}

// gradientAxis reports whether the ring reads as one smooth ramp across the block, and along
// which axis: the two sides on the axis with the larger spread must differ by more than
// modeFlatSpread, and both sides across it must sit in the middle third of the ramp between them
// (within a sixth of the spread of its midpoint) - a ramp passes through its middle colour halfway
// along, so the sides that span the block's length have to show that middle. The bound is relative
// because an absolute one passes an edge whose spread is only twice it: a paper side is exactly
// half the spread from the midpoint of paper and a dark rule.
//
// The second test is what separates a gradient from an edge. Two sides of a ring differ just as
// much when one of them lands on a panel border or a dark frame beside a white caption (measured:
// a 515 spread on samson-and-delilah-15's caption, whose right ring side is the panel rule), and a
// gradient painted from white to black across that caption would be the worst plate the program
// could draw. There the sides across the axis are paper, far from the white-black midpoint, so the
// block keeps the fill - which stays inside the block and never touches the edge beyond it.
func gradientAxis(r ringStats) (from, to ringSide, dir string, ok bool) {
	from, to, dir = r.top, r.bottom, "to bottom"
	a, b := r.left, r.right
	if spread(r.left, r.right) > spread(r.top, r.bottom) {
		from, to, dir = r.left, r.right, "to right"
		a, b = r.top, r.bottom
	}
	if spread(from, to) <= modeFlatSpread || !a.hasMed || !b.hasMed {
		return from, to, dir, false
	}
	var mid ringSide
	mid.hasMed = true
	for c := range 3 {
		mid.med[c] = (from.med[c] + to.med[c]) / 2
	}
	s := spread(from, to)
	return from, to, dir, 6*spread(a, mid) <= s && 6*spread(b, mid) <= s
}

func clamp01(v float64) float64 {
	return min(1, max(0, v))
}

// plateBackground is the CSS background value for a plate in the given mode, or "" to keep the
// fill (the caller's sampled paper colour). paper is the rgb() string blockColors sampled; w is
// the image's natural width, which the mask's cqw lengths are relative to.
//
// Reconstruct runs the gradient along the axis with the larger spread, from one side's median to
// the other's. Mask draws one paper-coloured layer per line box, positioned in cqw from the plate's
// own corner - the container is the image, so the stripes stay on the source lines at every
// viewport and do not stretch when the runtime fit lets a plate grow for a longer translation.
// Mirrors ocr-conceal.js plateBackground (docs/PARITY.md).
func plateBackground(mode Mode, r ringStats, b Block, paper string, w int) string {
	switch mode {
	case ModeReconstruct:
		from, to, dir, _ := gradientAxis(r)
		return fmt.Sprintf("linear-gradient(%s,%s,%s)", dir, rgb(from.med), rgb(to.med))
	case ModeMask:
		if paper == "" || w <= 0 || len(b.Lines) == 0 {
			return ""
		}
		lh := b.LineH
		if lh < 1 {
			lh = b.Y1 - b.Y0
		}
		pad := max(lh/modeMaskPadDivisor, ringMinPad)
		cq := func(v int) float64 { return float64(v) / float64(w) * 100 }
		layers := make([]string, 0, len(b.Lines))
		for _, l := range b.Lines {
			layers = append(layers, fmt.Sprintf("linear-gradient(%s,%s) %.3fcqw %.3fcqw/%.3fcqw %.3fcqw no-repeat",
				paper, paper, cq(l.X0-pad-b.X0), cq(l.Y0-pad-b.Y0), cq(l.X1-l.X0+2*pad), cq(l.Y1-l.Y0+2*pad)))
		}
		return strings.Join(layers, ",")
	}
	return ""
}

func rgb(c [3]int) string { return fmt.Sprintf("rgb(%d,%d,%d)", c[0], c[1], c[2]) }
