package main

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"doc-html-translate/internal/dialog"
)

// readUntilSecretQuestion consumes the run stream up to the password marker and returns
// everything seen so far.
func readUntilSecretQuestion(t *testing.T, br *bufio.Reader) string {
	t.Helper()
	var seen strings.Builder
	for {
		line, err := br.ReadString('\n')
		seen.WriteString(line)
		if err != nil {
			t.Fatalf("the stream ended before the password question: %v", err)
		}
		if strings.HasPrefix(line, dialog.SecretPrefix) {
			return seen.String()
		}
	}
}

// The password question reaches the window as a marker line, the typed value goes back over
// /api/secret to the run's stdin, and the value is never in the stream: not in the command
// preview line, not in the log. A cancel ends the run without the value.
func TestSecretIsAskedInTheWindowAndNeverEchoed(t *testing.T) {
	for _, tc := range []struct {
		name   string
		answer func(body string) string
		want   runEnd
	}{
		{"right password", func(b string) string {
			v, _ := json.Marshal(fakeSecret)
			return strings.TrimSuffix(b, "}") + `,"value":` + string(v) + `}`
		}, runEnd{State: "done"}},
		{"another password", func(b string) string {
			return strings.TrimSuffix(b, "}") + `,"value":"not it"}`
		}, runEnd{State: "failed", Code: 4}},
		{"cancel", func(b string) string {
			return strings.TrimSuffix(b, "}") + `,"cancel":true}`
		}, runEnd{State: "failed", Code: 5}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv, tok, body := fakeRun(t, "secret")
			resp := startRun(context.Background(), t, srv, tok, body)
			defer resp.Body.Close()
			br := bufio.NewReader(resp.Body)
			before := readUntilSecretQuestion(t, br)
			if !strings.Contains(before, "> ") {
				t.Errorf("the command echo is missing: %q", before)
			}

			a := call(t, srv, http.MethodPost, "/api/secret", tc.answer(body), withToken(tok), withJSON())
			if a.StatusCode != http.StatusOK {
				t.Fatalf("answer = %d", a.StatusCode)
			}
			var ack struct{ OK bool }
			if err := json.NewDecoder(a.Body).Decode(&ack); err != nil || !ack.OK {
				t.Fatalf("the answer was not delivered: %v %+v", err, ack)
			}
			rest, _ := io.ReadAll(br)
			if !strings.HasSuffix(string(rest), endLine(tc.want)) {
				t.Errorf("run ended %q, want %q", tail(string(rest)), endLine(tc.want))
			}
			if all := before + string(rest); strings.Contains(all, fakeSecret) || strings.Contains(all, "not it") {
				t.Error("the password reached the stream")
			}
		})
	}
}

// A secret for a run that is not there is refused, not queued anywhere.
func TestSecretForAnUnclaimedRunIsRefused(t *testing.T) {
	srv, tok := guardedServer(t)
	a := call(t, srv, http.MethodPost, "/api/secret", `{"input":"C:\\nowhere\\x.fd-sec","output":"","value":"pw"}`, withToken(tok), withJSON())
	var ack struct{ OK bool }
	if err := json.NewDecoder(a.Body).Decode(&ack); err != nil || ack.OK {
		t.Errorf("an unclaimed run must refuse the secret: %v %+v", err, ack)
	}
}

// answerRunSecret writes exactly one JSON line to the claimed run's stdin.
func TestAnswerRunSecretWritesOneJSONLine(t *testing.T) {
	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	const key = "secret-test-key"
	if _, err := claimRun(key, cancel); err != nil {
		t.Fatal(err)
	}
	defer releaseRun(key)
	var sb strings.Builder
	setRunStdin(key, &sb)

	if !answerRunSecret(key, fakeSecret, false) {
		t.Fatal("not delivered")
	}
	line := sb.String()
	if !strings.HasSuffix(line, "\n") || strings.Count(line, "\n") != 1 {
		t.Fatalf("want one line, got %q", line)
	}
	var got struct {
		Cancel bool   `json:"cancel"`
		Value  string `json:"value"`
	}
	if err := json.Unmarshal([]byte(line), &got); err != nil || got.Value != fakeSecret || got.Cancel {
		t.Errorf("answer = %+v %v", got, err)
	}

	sb.Reset()
	if !answerRunSecret(key, "", true) {
		t.Fatal("cancel not delivered")
	}
	if !strings.Contains(sb.String(), `"cancel":true`) {
		t.Errorf("cancel line = %q", sb.String())
	}
	if answerRunSecret("no-such-run", "x", false) {
		t.Error("an unclaimed run must refuse")
	}
}

// The command line the window shows and runs never carries the password source: only the CLI's
// scripting flag names a variable, and the window does not use it.
func TestCommandLineNeverCarriesAPasswordSource(t *testing.T) {
	for _, in := range []string{`C:\docs\secret.fd-sec`, `C:\docs\book.epub`} {
		req := runRequest{Input: in, Output: `C:\out`, SrcLang: "en", DstLang: "ru", OCR: true, Force: true}
		line := formatCommandLine("doc-html-translate.exe", assembleArgs(req))
		for _, bad := range []string{"fdsec-password-env", "password"} {
			if strings.Contains(strings.ToLower(line), bad) {
				t.Errorf("command line %q contains %q", line, bad)
			}
		}
	}
}
