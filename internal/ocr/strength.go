package ocr

import "strings"

// resultStrength measures how much a recognition attempt actually found: the number of words it
// placed on the image. The count is over the plates a pass produced, so the confidence floor that
// pass ran under is already applied - a word here is a word the app would show a reader.
//
// Word count rather than plated area or line count, and the reason is measured. Over the recorded
// desktop baseline (temp/ocrlab/base06-desktop) the scenes that read carry 2 to 23 words, and the
// weakest of them - synth-text-on-halftone, a caption of two words - is genuinely read. Area does
// not survive the comparison either: the same scene plates 3.9% of its image while a poster's one
// recovered word plates 1.7% of a much larger one, which is a difference in image size rather than
// in how much was found. Counting words is what the two ends of that range have in common.
func resultStrength(res Result) int {
	n := 0
	for _, b := range res.Blocks {
		n += len(strings.Fields(b.Text))
	}
	return n
}

// strictlyBetter reports whether a candidate result should replace the one already in hand.
//
// The rule is deliberately not symmetric: the candidate has to be ahead, and an equal result keeps
// the incumbent. A retry is a second look at the same pixels, so two attempts that find the same
// amount are not two pieces of evidence - they are one, and the earlier pass ran under the
// weaker prior. Ties therefore go to the result that was already there, and a rung that finds
// nothing new cannot displace one that found something.
func strictlyBetter(candidate, current Result) bool {
	return resultStrength(candidate) > resultStrength(current)
}
