package ocr

import "testing"

// The measure and the comparator decide which rung of the rescue ladder a reader ends up seeing,
// so the cases that matter are the shapes the corpus actually produced: nothing at all, the one
// stray word a poster came back with, and a sparse but correct caption that must not be treated as
// a failure because it is short.
func TestResultStrengthCountsWhatAReaderWouldSee(t *testing.T) {
	tests := []struct {
		name string
		res  Result
		want int
	}{
		{
			name: "an empty result found nothing",
			res:  Result{Width: 960, Height: 1280},
			want: 0,
		},
		{
			name: "one stray word on a large poster",
			res: Result{Width: 960, Height: 1280, Blocks: []Block{
				{Text: "| МОЖЕМ", X0: 0, Y0: 659, X1: 269, Y1: 736, LineH: 77},
			}},
			want: 2,
		},
		{
			name: "a sparse but correct caption is not weak for being short",
			res: Result{Width: 600, Height: 300, Blocks: []Block{
				{Text: "THE END", X0: 30, Y0: 240, X1: 260, Y1: 275, LineH: 35},
			}},
			want: 2,
		},
		{
			name: "several plates count together",
			res: Result{Width: 960, Height: 1280, Blocks: []Block{
				{Text: "ТРАХАТЬСЯ: МЫ ЖЕ ЛЮДИ,"},
				{Text: "МОЖЕМ ПРОСТО ПОГОВОРИТЬ"},
			}},
			want: 7,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := resultStrength(tc.res); got != tc.want {
				t.Errorf("resultStrength = %d, want %d", got, tc.want)
			}
		})
	}
}

// A retry may only replace what is already on the page, never merely equal it: the two attempts
// read the same pixels, so an equal count is one piece of evidence and not two.
func TestStrictlyBetterKeepsTheIncumbentOnATie(t *testing.T) {
	poor := Result{Blocks: []Block{{Text: "| МОЖЕМ"}}}
	rich := Result{Blocks: []Block{{Text: "ТРАХАТЬСЯ: МЫ ЖЕ ЛЮДИ, МОЖЕМ ПРОСТО ПОГОВОРИТЬ"}}}
	tie := Result{Blocks: []Block{{Text: "ЕЩЁ ДВА"}}}
	empty := Result{}

	tests := []struct {
		name               string
		candidate, current Result
		want               bool
	}{
		{"a richer rung replaces a poorer one", rich, poor, true},
		{"a poorer rung does not replace a richer one", poor, rich, false},
		{"an equal count keeps the incumbent", tie, poor, false},
		{"anything replaces nothing", poor, empty, true},
		{"nothing replaces nothing", empty, empty, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := strictlyBetter(tc.candidate, tc.current); got != tc.want {
				t.Errorf("strictlyBetter = %v, want %v", got, tc.want)
			}
		})
	}
}
