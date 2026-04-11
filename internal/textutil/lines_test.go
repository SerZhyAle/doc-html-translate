package textutil

import "testing"

func TestNormalizeLineSeparators(t *testing.T) {
	input := "A\r\nB\rC\u0085D\u2028E\u2029F\vG\fH"
	got := NormalizeLineSeparators(input)
	want := "A\nB\nC\nD\nE\nF\nG\nH"
	if got != want {
		t.Fatalf("NormalizeLineSeparators() = %q, want %q", got, want)
	}
}

func TestNormalizeLineSeparatorsPreserveFormFeed(t *testing.T) {
	input := "A\r\nB\fC\u2028D"
	got := NormalizeLineSeparatorsPreserveFormFeed(input)
	want := "A\nB\fC\nD"
	if got != want {
		t.Fatalf("NormalizeLineSeparatorsPreserveFormFeed() = %q, want %q", got, want)
	}
}
