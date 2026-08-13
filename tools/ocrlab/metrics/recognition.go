package metrics

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"doc-html-translate/tools/ocrlab/evidence"
	"doc-html-translate/tools/ocrlab/truth"
)

// DefaultDetectionIoU is the overlap at which a plate is said to have found a group. 0.5 is the
// detection convention; it is a parameter of the measurement rather than a quality bound, which
// is why it sits here and not in a thresholds file.
const DefaultDetectionIoU = 0.5

// Normalize reduces a string to what a reader would call "the same words". Whitespace collapses,
// NFC composes, case folds, and the punctuation Tesseract invents at glyph edges is dropped -
// counting a hallucinated comma as a character error would make every score about punctuation
// instead of about reading.
func Normalize(s string) string {
	s = norm.NFC.String(s)
	var b strings.Builder
	lastSpace := true
	for _, r := range s {
		switch {
		case unicode.IsSpace(r):
			if !lastSpace {
				b.WriteRune(' ')
				lastSpace = true
			}
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			// dropped
		default:
			b.WriteRune(unicode.ToLower(r))
			lastSpace = false
		}
	}
	return strings.TrimSpace(b.String())
}

// CER is the character error rate: Levenshtein distance over normalized runes, divided by the
// reference length. 0 when both are empty, 1 when the reference is empty and the output is not.
func CER(got, want string) float64 {
	g, w := []rune(Normalize(got)), []rune(Normalize(want))
	return errorRate(len(w), levenshtein(g, w))
}

// WER is the same over whitespace-separated tokens.
func WER(got, want string) float64 {
	g, w := strings.Fields(Normalize(got)), strings.Fields(Normalize(want))
	return errorRate(len(w), levenshteinStr(g, w))
}

func errorRate(refLen, dist int) float64 {
	switch {
	case refLen == 0 && dist == 0:
		return 0
	case refLen == 0:
		return 1
	default:
		return float64(dist) / float64(refLen)
	}
}

func levenshtein(a, b []rune) int {
	return levDistance(len(a), len(b), func(i, j int) bool { return a[i] == b[j] })
}

func levenshteinStr(a, b []string) int {
	return levDistance(len(a), len(b), func(i, j int) bool { return a[i] == b[j] })
}

// levDistance is the shared two-row Levenshtein, parameterized by an equality test so the rune
// and token versions share one implementation.
func levDistance(n, m int, eq func(i, j int) bool) int {
	if n == 0 {
		return m
	}
	if m == 0 {
		return n
	}
	prev := make([]int, m+1)
	cur := make([]int, m+1)
	for j := 0; j <= m; j++ {
		prev[j] = j
	}
	for i := 1; i <= n; i++ {
		cur[0] = i
		for j := 1; j <= m; j++ {
			cost := 1
			if eq(i-1, j-1) {
				cost = 0
			}
			cur[j] = min(min(cur[j-1]+1, prev[j]+1), prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[m]
}

// DetectionScore is how well the plates found the annotated text regions.
//
// Skipped is not a rounding detail. Groups the annotation marks as less than clear are excluded
// from precision and recall entirely rather than counted as misses: historical lettering a
// careful human cannot read must not masquerade as an OCR regression, and hiding those in the
// denominator would let a corpus of bad scans make any engine look broken.
type DetectionScore struct {
	Precision float64 `json:"precision"`
	Recall    float64 `json:"recall"`
	F1        float64 `json:"f1"`
	TP        int     `json:"tp"`
	FP        int     `json:"fp"`
	FN        int     `json:"fn"`
	Skipped   int     `json:"skipped"`
}

// Detection matches plates to groups at iouMin and returns the counts.
func Detection(plates []evidence.Plate, a *truth.Annotation, groups []truth.Group, w, h int, iouMin float64) DetectionScore {
	var score DetectionScore
	scored := make([]truth.Group, 0, len(groups))
	for _, g := range groups {
		if a.GroupAmbiguity(g) != truth.AmbiguityClear {
			score.Skipped++
			continue
		}
		scored = append(scored, g)
	}

	matchedPlate := make([]bool, len(plates))
	for _, g := range scored {
		best, bestIoU := -1, 0.0
		for i, p := range plates {
			if matchedPlate[i] {
				continue
			}
			if v := IoU(p.Rect.Region(), g.Bounds, w, h); v > bestIoU {
				best, bestIoU = i, v
			}
		}
		if best >= 0 && bestIoU >= iouMin {
			matchedPlate[best] = true
			score.TP++
		} else {
			score.FN++
		}
	}
	for i := range plates {
		if !matchedPlate[i] {
			score.FP++
		}
	}

	if d := score.TP + score.FP; d > 0 {
		score.Precision = float64(score.TP) / float64(d)
	}
	if d := score.TP + score.FN; d > 0 {
		score.Recall = float64(score.TP) / float64(d)
	}
	if score.Precision+score.Recall > 0 {
		score.F1 = 2 * score.Precision * score.Recall / (score.Precision + score.Recall)
	}
	return score
}

// TextScore is the reading accuracy over the groups that were both clear and matched.
type TextScore struct {
	MeanCER  float64 `json:"meanCer"`
	MeanWER  float64 `json:"meanWer"`
	WorstID  string  `json:"worstGroupId,omitempty"`
	WorstCER float64 `json:"worstCer"`
	Compared int     `json:"compared"`
	Skipped  int     `json:"skipped"`
}

// Text scores the recognized strings of matched pairs against their transcripts.
func Text(matches []Match, a *truth.Annotation) TextScore {
	var s TextScore
	var cers, wers []float64
	for _, m := range matches {
		if a.GroupAmbiguity(m.Group) != truth.AmbiguityClear || m.Group.Transcript == "" {
			s.Skipped++
			continue
		}
		c := CER(m.Plate.Text, m.Group.Transcript)
		cers = append(cers, c)
		wers = append(wers, WER(m.Plate.Text, m.Group.Transcript))
		if c > s.WorstCER {
			s.WorstCER, s.WorstID = c, m.Group.ID
		}
		s.Compared++
	}
	s.MeanCER, s.MeanWER = mean(cers), mean(wers)
	return s
}
