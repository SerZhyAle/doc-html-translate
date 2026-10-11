package main

import (
	"math"
	"math/rand"
	"testing"
)

func TestOLSSlopeDetectsGrowth(t *testing.T) {
	// A deterministic growth of 10 units per sample: the band must exclude zero. An exact
	// series has zero residual variance, so the band degenerates to the slope itself - still
	// excluding zero, which is the only thing the leak rule reads.
	y := make([]float64, 40)
	for i := range y {
		y[i] = 100 + 10*float64(i)
	}
	s := olsSlope(y)
	if !s.isLeak() {
		t.Fatalf("growth series scored as no-leak: %+v", s)
	}
	if math.Abs(s.Slope-10) > 1e-9 {
		t.Fatalf("slope = %v, want 10", s.Slope)
	}
	if s.CILo <= 0 {
		t.Fatalf("confidence band must exclude zero: %+v", s)
	}
}

func TestOLSSlopeFlatNoiseIsNoLeak(t *testing.T) {
	// Flat noise around a constant: the band must include zero with overwhelming regularity.
	rng := rand.New(rand.NewSource(42))
	leaks := 0
	for trial := 0; trial < 50; trial++ {
		y := make([]float64, 40)
		for i := range y {
			y[i] = 100 + rng.NormFloat64()*5
		}
		if olsSlope(y).isLeak() {
			leaks++
		}
	}
	if leaks > 2 {
		t.Fatalf("flat noise scored as leak in %d/50 trials - the band is too narrow", leaks)
	}
}

func TestOLSSlopeShortSeriesHasNoBand(t *testing.T) {
	s := olsSlope([]float64{1, 2})
	if s.isLeak() || s.N != 2 {
		t.Fatalf("two-point series must not produce a leak verdict: %+v", s)
	}
}
