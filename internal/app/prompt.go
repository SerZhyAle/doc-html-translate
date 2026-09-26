package app

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"doc-html-translate/internal/i18n"
)

// stdin is the one buffered reader over os.Stdin: a reader per prompt would strand whatever the
// previous one had buffered, and that text would answer nothing or the wrong question.
var stdin = bufio.NewReader(os.Stdin)

// askYes prints prompt and reads one answer line. Anything but an explicit yes is treated as no,
// so pressing Enter declines. Accepted are "y"/"yes" plus the affirmative of the interface
// language, because someone reading the prompt in Bengali will answer in Bengali.
func askYes(prompt string) bool {
	fmt.Print(prompt)
	return isYes(readLine(stdin))
}

// readLine consumes a whole line, so words left over after the first one ("n yy") belong to this
// answer rather than to the next prompt, which gates a registry write.
func readLine(r *bufio.Reader) string {
	line, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return ""
	}
	return strings.TrimRight(line, "\r\n")
}

// isYes reports whether a whole answer line is an explicit yes.
func isYes(line string) bool {
	answer := strings.ToLower(strings.TrimSpace(line))
	switch answer {
	case "y", "yes":
		return true
	}
	return answer == i18n.S("y") || answer == i18n.S("yes")
}
