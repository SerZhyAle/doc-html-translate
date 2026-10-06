package dialog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// When the GUI runs the converter, a native box would open behind the GUI's window, owned by
// a console nobody sees (APP-BEHAVIOUR rule 1). The GUI therefore sets HostEnv to HostStdio,
// and every question and notice travels over the child's own pipes instead: one marker line on
// stdout, which the GUI turns into an in-page dialog, and for a question one answer line back
// on stdin. The markers start with an ASCII record separator, which no log line begins with.
const (
	HostEnv        = "DOCHT_DIALOG_HOST"
	HostStdio      = "stdio"
	AskPrefix      = "\x1edht:ask "
	NotePrefix     = "\x1edht:note "
	ProgressPrefix = "\x1edht:progress "
	SecretPrefix   = "\x1edht:secret "
	AnswerYes      = "yes"
)

// HostedByGUI reports whether a GUI is answering the dialogs.
func HostedByGUI() bool { return os.Getenv(HostEnv) == HostStdio }

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

// secretLine is the marker payload of a masked-input question; retry says the previous answer
// did not open the file.
type secretLine struct {
	Title   string `json:"title"`
	Message string `json:"message"`
	Retry   bool   `json:"retry"`
}

// secretAnswer is the GUI's one-line reply: a cancel, or the typed value.
type secretAnswer struct {
	Cancel bool   `json:"cancel"`
	Value  string `json:"value"`
}

// AskSecret asks the GUI for a secret in its masked dialog and waits for the answer. The value
// travels only on the child's own stdin: it is never written to any writer, and a closed pipe,
// a malformed line or a cancel are all "not ok" - an unanswered question never reads as an
// empty password.
func AskSecret(title, message string, retry bool) (secret []byte, ok bool) {
	b, _ := json.Marshal(secretLine{Title: title, Message: message, Retry: retry})
	fmt.Fprintf(os.Stdout, "%s%s\n", SecretPrefix, b)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return nil, false
	}
	var a secretAnswer
	if json.Unmarshal([]byte(strings.TrimRight(line, "\r\n")), &a) != nil || a.Cancel {
		return nil, false
	}
	return []byte(a.Value), true
}

// noteHost hands a notice to the GUI. Nothing waits for it to be read.
func noteHost(title, message string) {
	writeHostLine(os.Stdout, NotePrefix, title, message)
}

var progressState struct {
	sync.Mutex
	stage string
	total int
	last  time.Time
}

// Progress reports a stable stage and, when known, completed units to the GUI.
// It is silent for console runs; their existing log remains the progress display.
func Progress(stage string, done, total int) {
	if !HostedByGUI() {
		return
	}
	progressState.Lock()
	defer progressState.Unlock()
	now := time.Now()
	if stage == progressState.stage && total == progressState.total && done < total && now.Sub(progressState.last) < 300*time.Millisecond {
		return
	}
	progressState.stage, progressState.total, progressState.last = stage, total, now
	b, _ := json.Marshal(struct {
		Stage string `json:"stage"`
		Done  int    `json:"done,omitempty"`
		Total int    `json:"total,omitempty"`
	}{stage, done, total})
	fmt.Fprintf(os.Stdout, "%s%s\n", ProgressPrefix, b)
}
