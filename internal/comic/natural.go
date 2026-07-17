package comic

import "strings"

// naturalLess reports whether name a sorts before name b under a natural,
// numeric-aware ordering: runs of ASCII digits compare by numeric value, so
// "page2.jpg" sorts before "page10.jpg" where a plain lexicographic sort would
// reverse them. This is load-bearing for comics - page order *is* archive entry
// order by filename - so the same rule must hold on both editions (see
// docs/PARITY.md). Comparison is case-insensitive on the non-numeric runs; two
// numerically equal runs ("07" vs "7") fall back to the shorter run first so the
// order is total and stable.
func naturalLess(a, b string) bool {
	la, lb := strings.ToLower(a), strings.ToLower(b)
	i, j := 0, 0
	for i < len(la) && j < len(lb) {
		if isDigit(la[i]) && isDigit(lb[j]) {
			si, ei := digitRun(la, i)
			sj, ej := digitRun(lb, j)
			if v := compareDigits(la[si:ei], lb[sj:ej]); v != 0 {
				return v < 0
			}
			// Equal numeric value: fewer leading zeros (shorter raw run) first.
			if (ei - i) != (ej - j) {
				return (ei - i) < (ej - j)
			}
			i, j = ei, ej
			continue
		}
		if la[i] != lb[j] {
			return la[i] < lb[j]
		}
		i++
		j++
	}
	return len(la)-i < len(lb)-j
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// digitRun returns [sigStart, end) for the digit run beginning at pos, with
// leading zeros stripped from sigStart (at least one digit is always kept, so an
// all-zero run yields the trailing "0"). end is the index just past the run.
func digitRun(s string, pos int) (sigStart, end int) {
	end = pos
	for end < len(s) && isDigit(s[end]) {
		end++
	}
	sigStart = pos
	for sigStart < end-1 && s[sigStart] == '0' {
		sigStart++
	}
	return sigStart, end
}

// compareDigits compares two zero-stripped digit runs by numeric value: the
// longer run is the larger number, and equal-length runs compare lexically.
// Returns -1, 0 or 1.
func compareDigits(x, y string) int {
	if len(x) != len(y) {
		if len(x) < len(y) {
			return -1
		}
		return 1
	}
	if x < y {
		return -1
	}
	if x > y {
		return 1
	}
	return 0
}
