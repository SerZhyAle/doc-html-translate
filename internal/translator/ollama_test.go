package translator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

// ollamaStub answers the numbered-batch prompt with "N. XX:<text>" lines; reply decides per
// request, and a nil return means "answer normally".
func ollamaStub(t *testing.T, reply func(prompt string) (status int, body string, ok bool)) *OllamaClient {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ollamaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode: %v", err)
			return
		}
		if reply != nil {
			if status, body, ok := reply(req.Prompt); ok {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(body))
				return
			}
		}
		_ = json.NewEncoder(w).Encode(ollamaResponse{Response: echoNumbered(req.Prompt)})
	}))
	t.Cleanup(srv.Close)
	c := NewOllamaClient("test-model")
	c.baseURL = srv.URL
	return c
}

var promptLine = regexp.MustCompile(`(?m)^(\d+)\. (.*)$`)

func echoNumbered(prompt string) string {
	var sb strings.Builder
	for _, m := range promptLine.FindAllStringSubmatch(prompt, -1) {
		fmt.Fprintf(&sb, "%s. XX:%s\n", m[1], m[2])
	}
	return sb.String()
}

// A failed batch used to throw away every batch of the page that had already come back.
func TestOllamaKeepsSuccessfulBatches(t *testing.T) {
	c := ollamaStub(t, func(prompt string) (int, string, bool) {
		if strings.Contains(prompt, "text 25") {
			return http.StatusInternalServerError, "boom", true
		}
		return 0, "", false
	})
	texts := make([]string, 30)
	for i := range texts {
		texts[i] = fmt.Sprintf("text %d", i)
	}
	got, err := c.Translate(context.Background(), texts, "en", "de")
	var partial *PartialError
	if !errors.As(err, &partial) {
		t.Fatalf("err = %v, want *PartialError", err)
	}
	if len(got) != len(texts) {
		t.Fatalf("len = %d", len(got))
	}
	if got[0] != "XX:text 0" || got[19] != "XX:text 19" {
		t.Fatalf("first batch lost: %q %q", got[0], got[19])
	}
	if len(partial.Missing) != 10 || partial.Missing[0] != 20 || got[20] != "" {
		t.Fatalf("missing = %v, got[20] = %q", partial.Missing, got[20])
	}
}

func TestOllamaCancelled(t *testing.T) {
	c := ollamaStub(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Translate(ctx, []string{"a"}, "en", "de"); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}
