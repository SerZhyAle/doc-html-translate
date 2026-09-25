package dialog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// When the GUI runs the converter, a native box would open behind the GUI's window, owned by
// a console nobody sees (APP-BEHAVIOUR rule 1). The GUI therefore sets HostEnv to HostStdio,
// and every question and notice travels over the child's own pipes instead: one marker line on
// stdout, which the GUI turns into an in-page dialog, and for a question one answer line back
// on stdin. The markers start with an ASCII record separator, which no log line begins with.
const (
	HostEnv    = "DOCHT_DIALOG_HOST"
	HostStdio  = "stdio"
	AskPrefix  = "\x1edht:ask "
	NotePrefix = "\x1edht:note "
	AnswerYes  = "yes"
)

// hostedByGUI reports whether a GUI is answering the dialogs.
func hostedByGUI() bool { return os.Getenv(HostEnv) == HostStdio }

// hostLine is the marker payload: what the dialog says, in the process language.
type hostLine struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

func writeHostLine(w io.Writer, prefix, title, message string) {
	b, _ := json.Marshal(hostLine{Title: title, Message: message})
	fmt.Fprintf(w, "%s%s\n", prefix, b)
}

// askHost puts the question to the GUI and waits for its answer. Anything but an explicit yes -
// "no", a closed pipe, a GUI that went away - is a no: an unanswered question about money must
// never read as consent.
func askHost(title, message string) bool {
	writeHostLine(os.Stdout, AskPrefix, title, message)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(line) == AnswerYes
}

// noteHost hands a notice to the GUI. Nothing waits for it to be read.
func noteHost(title, message string) {
	writeHostLine(os.Stdout, NotePrefix, title, message)
}
