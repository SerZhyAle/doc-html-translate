package txt

import (
	"unicode"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
)

// legacyCandidates are the pre-Unicode code pages this app's audience actually produces:
// Russian/Ukrainian DOS and early-web text. Ordered most-likely-first so an exact score tie
// resolves to the common case; in practice the frequency score below separates them.
var legacyCandidates = []struct {
	name string
	enc  encoding.Encoding
}{
	{"windows-1251", charmap.Windows1251},
	{"koi8-r", charmap.KOI8R},
	{"cp866", charmap.CodePage866},
	{"iso-8859-5", charmap.ISO8859_5},
}

// ruLetterFreq is the relative frequency (percent) of each lowercase Russian letter in
// ordinary text. It is what lets the detector tell the *right* code page from a wrong one
// that also yields Cyrillic: decoding cp1251 bytes as KOI8-R still produces Russian letters
// (the two swap the same byte range), but the *wrong* letters - common ones land on rare
// ones - so scoring each decoding by how natural its letter mix is picks the real encoding.
// Counting Cyrillic letters alone cannot: three of the four candidates score alike.
var ruLetterFreq = map[rune]float64{
	'о': 10.98, 'е': 8.45, 'а': 8.01, 'и': 7.35, 'н': 6.70, 'т': 6.26,
	'с': 5.47, 'р': 4.73, 'в': 4.54, 'л': 4.40, 'к': 3.49, 'м': 3.21,
	'д': 2.98, 'п': 2.81, 'у': 2.62, 'я': 2.01, 'ы': 1.90, 'ь': 1.74,
	'г': 1.70, 'з': 1.65, 'б': 1.59, 'ч': 1.44, 'й': 1.21, 'х': 0.97,
	'ж': 0.94, 'ш': 0.73, 'ю': 0.64, 'ц': 0.48, 'щ': 0.36, 'э': 0.32,
	'ф': 0.26, 'ъ': 0.04, 'ё': 0.04,
}

// cyrillicFit scores a decoded string two ways, because choosing the encoding and trusting
// the choice are different questions:
//   - freqWeight: the summed expected frequency of the Russian letters present. This picks
//     the best candidate. Decoding cp1251 Russian as KOI8-R yields about as *many* Cyrillic
//     letters (the two swap the same byte range), but the *wrong* ones - common letters land
//     on rare ones - so the natural decoding scores highest. Counting letters alone cannot
//     separate them; weighting by frequency does.
//   - fraction: Russian letters as a share of all runes. This is the confidence signal.
//     Real Russian is mostly Russian letters; a non-Russian legacy file (French Latin-1, a
//     stray binary) decodes to mostly ASCII and punctuation with only a few bytes landing on
//     Cyrillic, so its fraction stays low even when its freqWeight happens to top the others.
func cyrillicFit(s string) (freqWeight, fraction float64, letters, runes int) {
	for _, r := range s {
		runes++
		if f, ok := ruLetterFreq[unicode.ToLower(r)]; ok {
			freqWeight += f
			letters++
		}
	}
	if runes > 0 {
		fraction = float64(letters) / float64(runes)
	}
	return freqWeight, fraction, letters, runes
}

// minCyrillicFraction is the confidence floor. Measured: the real cp1251 corpus fixture is
// 0.76 Russian letters by rune, while French Latin-1 mis-read as KOI8-R (which otherwise wins
// on freqWeight) is 0.17 - only its handful of accented bytes land on Cyrillic. 0.30 sits well
// clear of both, so genuine Cyrillic legacy text is decoded and everything else passes through
// unchanged rather than being forced into a wrong alphabet.
const minCyrillicFraction = 0.30

// detectLegacy decodes bytes that carry no BOM and are not valid UTF-8 - a text file in a
// pre-Unicode Cyrillic code page. It tries each candidate, keeps the most Russian-looking
// result by freqWeight, and commits only when that result is confidently Cyrillic by fraction;
// otherwise it returns ok=false so the caller passes the raw bytes through, exactly the app's
// prior behaviour for this input. The chosen encoding's name lets the caller say which it used.
func detectLegacy(raw []byte) (text, encName string, ok bool) {
	bestWeight := -1.0
	var bestText, bestName string
	var bestFraction float64
	for _, c := range legacyCandidates {
		decoded, err := c.enc.NewDecoder().Bytes(raw)
		if err != nil {
			continue
		}
		weight, fraction, _, _ := cyrillicFit(string(decoded))
		if weight > bestWeight {
			bestWeight, bestText, bestName, bestFraction = weight, string(decoded), c.name, fraction
		}
	}
	if bestWeight < 0 || bestFraction < minCyrillicFraction {
		return "", "", false
	}
	return bestText, bestName, true
}
