package app

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

// P25: "n yy" is one answer (no), and it must not leave words behind to answer the next prompt,
// which is the one that writes the default handler to the registry.
func TestAskYesReadsWholeLine(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.WriteString("n yy\n"); err != nil {
		t.Fatal(err)
	}
	w.Close()
	oldStdin, oldReader := os.Stdin, stdin
	os.Stdin, stdin = r, bufio.NewReader(r)
	t.Cleanup(func() { os.Stdin, stdin = oldStdin, oldReader; r.Close() })

	if askYes("first? ") {
		t.Error(`"n yy" answered yes`)
	}
	if askYes("second? ") {
		t.Error("leftover words of the first answer answered the second prompt")
	}
}

func TestReadLineAnswers(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("y\n\r\nyes"))
	for i, want := range []bool{true, false, true, false} {
		if got := isYes(readLine(r)); got != want {
			t.Errorf("answer %d = %v, want %v", i, got, want)
		}
	}
}
