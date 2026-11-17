package ocr

import (
	"fmt"
	"strings"
)

// LangsText is the listing `-ocr-langs` prints and `ocr langs` prints without --json: the tessdata folders the app reads,
// then one line per catalogue language saying whether it is installed or how to download it. It is English and is not
// part of a job (OCR-INVOCATION section F); both spellings call this one function, so they cannot differ.
func LangsText() string {
	installed := map[string]bool{}
	for _, c := range Installed() {
		installed[c] = true
	}
	var b strings.Builder
	fmt.Fprintln(&b, "OCR languages (tessdata:", strings.Join(DataDirs(), "; ")+")")
	for _, l := range Available {
		mark := "  available - download with: -ocr-download " + l.Code
		if installed[l.Code] {
			mark = "  installed"
		}
		fmt.Fprintf(&b, "  %-9s %-22s%s\n", l.Code, l.Name, mark)
	}
	return b.String()
}

// LangRow is one line of the data `ocr langs --json` answers with.
type LangRow struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
}

// LangRows is the catalogue with the installed languages marked: the data form of LangsText. It names no folder.
func LangRows() []LangRow {
	installed := map[string]bool{}
	for _, c := range Installed() {
		installed[c] = true
	}
	rows := make([]LangRow, 0, len(Available))
	for _, l := range Available {
		rows = append(rows, LangRow{Code: l.Code, Name: l.Name, Installed: installed[l.Code]})
	}
	return rows
}
