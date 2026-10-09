package report

import (
	"fmt"

	"doc-html-translate/tools/ocrlab/metrics"
)

// The pixel measurements can be absent, and an absent one must never print as a number: a "0.00"
// next to a missing capture reads as perfect concealment. These helpers are the one place the
// reports decide how each measurement shows.

func residualCell(r metrics.ResidualScore) string {
	if !r.IsMeasured() {
		return "unmeasured"
	}
	return fmt.Sprintf("%.2f", r.Residual)
}

func haloCell(r metrics.ResidualScore) string {
	if !r.IsMeasured() {
		return "unmeasured"
	}
	return fmt.Sprintf("%.2f", r.Halo)
}

// damageCell shows painted protected pixels. An unmeasured score keeps the pixels already proven
// painted, marked as a floor.
func damageCell(d metrics.DamageScore) string {
	switch {
	case d.IsMeasured():
		return fmt.Sprintf("%d", d.ProtectedHit)
	case d.ProtectedHit > 0:
		return fmt.Sprintf("%d+ (partial)", d.ProtectedHit)
	}
	return "unmeasured"
}

func readabilityCell(r metrics.ReadabilityScore) string {
	switch r.State {
	case metrics.StateMeasured:
		return fmt.Sprintf("%d glyph px, %d px tall, separation %.0f, %.0f%% of lines clipped",
			r.GlyphPx, r.MinGlyphHeightPx, r.MinSeparation, r.ClippedFraction*100)
	case metrics.StateNoGlyphs:
		return "no glyphs drawn"
	}
	return "unmeasured"
}
