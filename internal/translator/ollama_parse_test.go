package translator

import "testing"

// An empty numbered answer must not swallow the next line: "2." followed by "3. text" once
// put "3. text" into slot 2 and left slot 3 empty.
func TestParseNumberedEmptyAnswerDoesNotCrossLines(t *testing.T) {
	got := parseNumberedResponse("1. eins\n2.\n3. drei\n", 3)
	if got[0] != "eins" || got[1] != "" || got[2] != "drei" {
		t.Fatalf("got %q", got)
	}
}
