// Command helpcheck is the gate of the in-app help (ticket 118, CONTEXT-HELP rule 19): the surfaces of the GUI
// launcher against the help map, the pages against the locale set, the links, rule 12, the house text style, the two
// copies of the set, the baseline against the committed file and the freshness of the set against its inputs. It ends
// in one CHECK-VERDICT line.
//
//	go run ./tools/helpcheck                              the check, from the repository root (scripts/help-check.ps1 runs it)
//	go run ./tools/helpcheck -accept-baseline "<reason>"  the update path of configs/help-baseline.txt (CHECK-BASELINE
//	                                                      rule 6): accepts the GUI locales that have no help pages yet,
//	                                                      prints each one and records the reason beside it
//
// Exit 0 PASS, 1 FAIL, 2 COULD NOT VERIFY, 3 PASS WITH ADVISORIES.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"doc-html-translate/internal/helpcheck"
)

func main() {
	accept := flag.String("accept-baseline", "", "accept the GUI locales without help pages into the baseline, with this reason")
	flag.Parse()

	root, err := os.Getwd()
	if err != nil {
		verdict(2, "no working directory")
	}
	if acceptGiven() {
		acceptBaseline(root, *accept)
		return
	}
	fmt.Printf("help-check: subject = the in-app help of the GUI launcher (%s, %s) in %s\n", helpcheck.MapPath, helpcheck.MirrorDir+"/", root)
	r, err := helpcheck.RunWith(root, helpcheck.Options{Committed: gitHead(root)})
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

// acceptGiven reports whether -accept-baseline was passed at all, even with an empty reason.
func acceptGiven() bool {
	given := false
	flag.Visit(func(f *flag.Flag) { given = given || f.Name == "accept-baseline" })
	return given
}

func acceptBaseline(root, reason string) {
	ids, err := helpcheck.Accept(root, reason, time.Now())
	if err != nil {
		fmt.Println("help-check:", err)
		verdict(2, "the baseline was not changed")
	}
	if len(ids) == 0 {
		fmt.Println("help-check: nothing to accept - every GUI locale without help pages is in the baseline already")
		verdict(0, "the baseline was not changed")
	}
	for _, id := range ids {
		fmt.Println("accepted:", id)
	}
	fmt.Printf("help-check: %s now carries %d new line(s) and the reason; review the diff before committing\n", helpcheck.BaselinePath, len(ids))
	verdict(0, fmt.Sprintf("%d accepted", len(ids)))
}

// gitHead reads a file as HEAD holds it. A tree with no HEAD yet, or a file HEAD does not hold, is "not found"; no git
// at all, or no checkout, is an error (a missing git result is never a pass).
func gitHead(root string) func(rel string) ([]byte, bool, error) {
	return func(rel string) ([]byte, bool, error) {
		if _, err := exec.LookPath("git"); err != nil {
			return nil, false, errors.New("git is not on PATH")
		}
		cmd := exec.Command("git", "-C", root, "show", "HEAD:"+rel)
		var out, errb bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &errb
		if err := cmd.Run(); err != nil {
			msg := errb.String()
			switch {
			case strings.Contains(msg, "does not exist in 'HEAD'"), strings.Contains(msg, "exists on disk, but not in 'HEAD'"),
				strings.Contains(msg, "invalid object name 'HEAD'"), strings.Contains(msg, "bad revision 'HEAD"),
				strings.Contains(msg, "unknown revision or path"):
				return nil, false, nil
			}
			return nil, false, fmt.Errorf("git show HEAD:%s: %s", rel, strings.TrimSpace(msg))
		}
		return out.Bytes(), true, nil
	}
}

func verdict(code int, detail string) {
	word := map[int]string{0: "PASS", 1: "FAIL", 2: "COULD NOT VERIFY", 3: "PASS WITH ADVISORIES"}[code]
	fmt.Printf("help-check: %s (%s)\n", word, detail)
	os.Exit(code)
}
