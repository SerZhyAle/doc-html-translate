package main

import "testing"

func TestTimingSummaryPreservesSamplesAndDoesNotMutateInputs(t *testing.T) {
	samples := []float64{40, 10, 30, 20}
	s := summarizeTiming(samples)
	if s.Samples != 4 || s.Median != 25 || s.P95 != 40 || s.Min != 10 || s.Max != 40 || s.StdDev <= 0 {
		t.Fatalf("%+v", s)
	}
	if samples[0] != 40 {
		t.Fatal("input mutated")
	}
	if summarizeTiming(nil).Samples != 0 {
		t.Fatal("invented sample")
	}
}
