// Command helpcheck is the gate of the in-app help (ticket 118, CONTEXT-HELP rule 19): the surfaces of the GUI
// launcher against the help map, the pages against the locale set, the links, rule 12, the house text style, the two
// copies of the set and the freshness of the set against its inputs. It ends in one CHECK-VERDICT line.
//
//	go run ./tools/helpcheck              the check, from the repository root (scripts/help-check.ps1 runs it)
//
// Exit 0 PASS, 1 FAIL, 2 COULD NOT VERIFY, 3 PASS WITH ADVISORIES.
package main

import (
	"fmt"
	"os"

	"doc-html-translate/internal/helpcheck"
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		verdict(2, "no working directory")
	}
	fmt.Printf("help-check: subject = the in-app help of the GUI launcher (%s, %s) in %s\n", helpcheck.MapPath, helpcheck.MirrorDir+"/", root)
	r, err := helpcheck.Run(root)
	if err != nil {
		fmt.Println("help-check:", err)
		verdict(2, "the help set cannot be read")
	}
	for _, f := range r.Failures {
		fmt.Println("FAIL:", f)
	}
	for _, a := range r.Advisories {
		fmt.Println("advisory:", a)
	}
	verdict(r.Code(), r.Detail())
}

func verdict(code int, detail string) {
	word := map[int]string{0: "PASS", 1: "FAIL", 2: "COULD NOT VERIFY", 3: "PASS WITH ADVISORIES"}[code]
	fmt.Printf("help-check: %s (%s)\n", word, detail)
	os.Exit(code)
}
