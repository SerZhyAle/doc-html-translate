// slope.go implements probe P5 (strategic section 6): ordinary least squares over a sample
// series, with the slope reported together with its 95% confidence band. A leak is a slope whose
// band excludes zero in the growth direction after documented caches are subtracted - never a
// snapshot difference.

package main

import "math"

// slopeResult is the leak statistic for one resource series.
type slopeResult struct {
	N      int     // samples used
	Slope  float64 // units per sample (bytes, handles, ...)
	StdErr float64 // standard error of the slope
	CILo   float64 // 95% confidence band, lower
	CIHi   float64 // 95% confidence band, upper
}

// isLeak reports the spec's rule: a leak finding requires the 95% band to exclude zero in the
// growth direction.
func (s slopeResult) isLeak() bool {
	return s.N >= 30 && s.CILo > 0
}

// olsSlope fits y = a + b*x over the samples and returns b with its 95% confidence band. With
// fewer than 3 points the band is not defined; the result carries N and a zero band.
func olsSlope(y []float64) slopeResult {
	n := len(y)
	if n < 3 {
		return slopeResult{N: n}
	}
	var sx, sy, sxx, sxy float64
	for i, v := range y {
		fx := float64(i)
		sx += fx
		sy += v
		sxx += fx * fx
		sxy += fx * v
	}
	den := float64(n)*sxx - sx*sx
	if den == 0 {
		return slopeResult{N: n}
	}
	b := (float64(n)*sxy - sx*sy) / den
	a := (sy - b*sx) / float64(n)
	var sse float64
	for i, v := range y {
		r := v - (a + b*float64(i))
		sse += r * r
	}
	sxxDev := sxx - sx*sx/float64(n)
	if sxxDev == 0 || n <= 2 {
		return slopeResult{N: n, Slope: b}
	}
	se := math.Sqrt(sse / float64(n-2) / sxxDev)
	tc := tCrit95(n - 2)
	return slopeResult{N: n, Slope: b, StdErr: se, CILo: b - tc*se, CIHi: b + tc*se}
}

// tCrit95 returns the two-sided 95% critical value of Student's t for the given degrees of
// freedom, from a small table for the sample sizes the stand actually produces and the normal
// approximation beyond it. A stand that needs better than this is overfitting its own noise.
func tCrit95(df int) float64 {
	switch {
	case df <= 0:
		return 0
	case df == 1:
		return 12.706
	case df == 2:
		return 4.303
	case df == 3:
		return 3.182
	case df == 4:
		return 2.776
	case df == 5:
		return 2.571
	case df == 6:
		return 2.447
	case df == 7:
		return 2.365
	case df == 8:
		return 2.306
	case df == 9:
		return 2.262
	case df == 10:
		return 2.228
	case df == 11:
		return 2.201
	case df == 12:
		return 2.179
	case df == 13:
		return 2.160
	case df == 14:
		return 2.145
	case df == 15:
		return 2.145
	case df <= 20:
		return 2.145 // use the lower-df bound conservatively between table entries
	case df <= 25:
		return 2.086
	case df <= 30:
		return 2.060
	case df <= 40:
		return 2.042
	case df <= 60:
		return 2.021
	case df <= 120:
		return 2.000
	default:
		return 1.980
	}
}
