package runner

// Translation-stress cases.
//
// These are literals, not translations. The strategic spec requires the visual runner to work
// offline and to call no paid service, and geometry does not care whether a string means
// anything - it cares how long it is, which script it is in and which way it runs. Fixing the
// strings is also what makes a clipping result reproducible: a real translator would return
// something different next week and every measurement would move for no reason.
//
// Ported verbatim into extension/scripts/ocrlab.mjs; a parity test compares the two lists.
type StressCase struct {
	Name string
	// Text is the phrase repeated until the replacement reaches Factor times the source
	// length. Empty Text means "leave the recognized text alone".
	Text   string
	Factor float64
	Dir    string // "ltr" or "rtl"
}

// StressCases is the fixed set every run applies, in this order.
var StressCases = []StressCase{
	{Name: "none", Text: "", Factor: 1, Dir: "ltr"},
	{Name: "short", Text: "Yes.", Factor: 0.2, Dir: "ltr"},
	{Name: "long-latin", Text: "Ausführliche Erläuterung der Sachlage ", Factor: 2.5, Dir: "ltr"},
	{Name: "long-cyrillic", Text: "Подробное разъяснение обстоятельств дела ", Factor: 2.5, Dir: "ltr"},
	{Name: "rtl-arabic", Text: "شرح مفصل للحالة ", Factor: 1.8, Dir: "rtl"},
	{Name: "cjk", Text: "详细说明情况 ", Factor: 1.2, Dir: "ltr"},
}

// StressNames is the case list as plain names, for the parity guard and the report headings.
func StressNames() []string {
	out := make([]string, 0, len(StressCases))
	for _, c := range StressCases {
		out = append(out, c.Name)
	}
	return out
}

// PrimaryStress is the case that represents "the page as recognized". Kept in step with
// metrics.PrimaryStressCase.
const PrimaryStress = "none"
