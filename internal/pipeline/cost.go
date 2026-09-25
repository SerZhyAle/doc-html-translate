package pipeline

import (
	"fmt"
	"unicode/utf8"

	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/i18n"
	"doc-html-translate/internal/logging"
)

// googleUSDPerMillionChars is the Cloud Translation v2 list price the estimate uses.
const googleUSDPerMillionChars = 20

// confirmThreshold keeps a book this small from raising the cost dialog when no -max-cost is
// set: the question would be about a fraction of a cent.
const confirmThreshold = 1000

// approveGoogleCost applies the -max-cost guard and the confirmation dialog.
//
// A -max-cost limit is a pre-approval: an estimate within it translates without asking, so a
// scripted or unattended run never stops at a dialog, and one over it is refused at any size.
// Without a limit the dialog asks, as it always did, for anything above confirmThreshold.
func (r Runner) approveGoogleCost(book *epub.Book, pages []contentPage) bool {
	chars := billableChars(book, pages)
	estCost := float64(chars) / 1_000_000 * googleUSDPerMillionChars
	if r.cfg.MaxCost > 0 {
		if estCost > r.cfg.MaxCost {
			logging.Printf("[3/4] Translation skipped - estimated cost $%s USD (%s characters) exceeds -max-cost $%s limit\n",
				formatUSD(estCost), formatInt(chars), formatUSD(r.cfg.MaxCost))
			return false
		}
		logging.Println(i18n.S("[3/4] Estimated cost $%s USD (%s characters) is within -max-cost $%s - translating without asking.",
			formatUSD(estCost), formatInt(chars), formatUSD(r.cfg.MaxCost)))
		return true
	}
	if chars <= confirmThreshold {
		return true
	}
	msg := fmt.Sprintf(
		"Characters to send: %s\nEstimated cost: $%s USD\n\nProceed with Google Translate?",
		formatInt(chars), formatUSD(estCost),
	)
	if !r.engines.confirm("Google Translate - Cost Warning", msg) {
		logging.Println("[3/4] Translation cancelled by user")
		return false
	}
	return true
}

// billableChars counts every character the engine will be sent - the pages, the book title and
// the authored TOC labels - in characters, which is what Google bills, not in UTF-8 bytes, which
// doubled the count for Cyrillic and tripled it for CJK.
func billableChars(book *epub.Book, pages []contentPage) int {
	total := 0
	for _, page := range pages {
		if page.err == nil {
			total += page.charCount
		}
	}
	total += utf8.RuneCountInString(book.Title)
	var labels []string
	collectTOCTitles(book.TOC, &labels)
	for _, l := range labels {
		total += utf8.RuneCountInString(l)
	}
	return total
}

// formatUSD shows cents, or four decimals below a cent, so a small estimate does not read as
// $0.00 next to a limit it is being compared with.
func formatUSD(v float64) string {
	if v != 0 && v < 0.01 {
		return fmt.Sprintf("%.4f", v)
	}
	return fmt.Sprintf("%.2f", v)
}

// formatInt formats an integer with thousands separators.
func formatInt(n int) string {
	s := fmt.Sprintf("%d", n)
	out := make([]byte, 0, len(s)+len(s)/3)
	for i, c := range s {
		pos := len(s) - i
		if i > 0 && pos%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(c))
	}
	return string(out)
}
