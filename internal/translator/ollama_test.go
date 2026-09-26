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

// Done criterion 6: a segment with two lines comes back with both lines, because it is sent on
// its own rather than as one line of the numbered list.
func TestOllamaMultiLineSegmentKeepsEveryLine(t *testing.T) {
	var numbered, single int
	c := ollamaStub(t, func(prompt string) (int, string, bool) {
		if strings.HasPrefix(prompt, "Translate the following text") {
			single++
			text := prompt[strings.LastIndex(prompt, "\n\n")+2:]
			body, _ := json.Marshal(ollamaResponse{Response: strings.ReplaceAll("XX:"+text, "\n", "\nXX:")})
			return http.StatusOK, string(body), true
		}
		numbered++
		if strings.Contains(prompt, "second line") {
			t.Errorf("multi-line text went into the numbered prompt:\n%s", prompt)
		}
		return 0, "", false
	})
	in := []string{"one line", "first line\nsecond line", "another line"}
	got, err := c.Translate(context.Background(), in, "en", "de")
	if err != nil {
		t.Fatal(err)
	}
	if got[1] != "XX:first line\nXX:second line" {
		t.Fatalf("multi-line segment = %q", got[1])
	}
	if got[0] != "XX:one line" || got[2] != "XX:another line" {
		t.Fatalf("neighbours = %q, %q", got[0], got[2])
	}
	if single != 1 || numbered != 1 {
		t.Fatalf("single = %d, numbered = %d", single, numbered)
	}
}

// A reply line that starts with a number must not land in another slot: the first answer per
// number wins and out-of-range numbers are ignored.
func TestParseNumberedFirstAnswerWins(t *testing.T) {
	reply := "1. Es war ein kalter Tag\n2. Zweite Zeile\n1984. Ein Jahr\n2. overwrite attempt\n"
	got := parseNumberedResponse(reply, 3)
	if got[0] != "Es war ein kalter Tag" || got[1] != "Zweite Zeile" || got[2] != "" {
		t.Fatalf("got %q", got)
	}
}

func TestOllama1984LineKeepsItsSlot(t *testing.T) {
	c := ollamaStub(t, nil)
	in := []string{"1984. It was a bright cold day", "Chapter 2"}
	got, err := c.Translate(context.Background(), in, "en", "de")
	if err != nil {
		t.Fatal(err)
	}
	if got[0] != "XX:1984. It was a bright cold day" || got[1] != "XX:Chapter 2" {
		t.Fatalf("got %q", got)
	}
}

// The first request waits for a cold model load under the long timeout; after the model has
// answered, requests use the ordinary one.
func TestOllamaLoadTimeoutOnlyUntilReady(t *testing.T) {
	c := ollamaStub(t, nil)
	if c.ready.Load() {
		t.Fatal("a new client claims its model is loaded")
	}
	if _, err := c.Translate(context.Background(), []string{"a"}, "en", "de"); err != nil {
		t.Fatal(err)
	}
	if !c.ready.Load() {
		t.Fatal("a successful answer did not mark the model loaded")
	}
	if c.httpClient.Timeout != 0 {
		t.Fatalf("a client-wide timeout %v would also cut the cold load", c.httpClient.Timeout)
	}
}

// skipSecondAnswer answers the numbered prompt without its "2." line.
func skipSecondAnswer(prompt string) (int, string, bool) {
	reply := strings.Replace(echoNumbered(prompt), "2. XX:second text\n", "", 1)
	body, _ := json.Marshal(ollamaResponse{Response: reply})
	return http.StatusOK, string(body), true
}

// A slot the model skips in the numbered reply and leaves empty on every echo retry is
// untranslated: the reply says so, and the cache does not keep "" as its translation.
func TestOllamaSkippedNumberIsPartialAndNotCached(t *testing.T) {
	c := ollamaStub(t, func(prompt string) (int, string, bool) {
		if strings.HasPrefix(prompt, "Translate the following text") {
			body, _ := json.Marshal(ollamaResponse{Response: ""})
			return http.StatusOK, string(body), true
		}
		return skipSecondAnswer(prompt)
	})
	cache := NewCachingClient(c)
	got, err := cache.Translate(context.Background(), []string{"first text", "second text", "third text"}, "en", "de")
	var partial *PartialError
	if !errors.As(err, &partial) || !errors.Is(err, ErrNoTranslation) {
		t.Fatalf("err = %v, want a PartialError caused by ErrNoTranslation", err)
	}
	if len(partial.Missing) != 1 || partial.Missing[0] != 1 {
		t.Fatalf("missing = %v", partial.Missing)
	}
	if got[0] != "XX:first text" || got[1] != "" || got[2] != "XX:third text" {
		t.Fatalf("got = %q", got)
	}
	if _, cached := cache.cache["en:de:second text"]; cached || cache.Stats() != 2 {
		t.Fatalf("cache holds %d entries, second text cached = %v", cache.Stats(), cached)
	}
}

// A retry request that fails is not dropped: its error is the cause of the partial result.
func TestOllamaRetryFailureIsReported(t *testing.T) {
	c := ollamaStub(t, func(prompt string) (int, string, bool) {
		if strings.HasPrefix(prompt, "Translate the following text") {
			return http.StatusInternalServerError, "boom", true
		}
		return skipSecondAnswer(prompt)
	})
	got, err := c.Translate(context.Background(), []string{"first text", "second text"}, "en", "de")
	var partial *PartialError
	if !errors.As(err, &partial) || errors.Is(err, ErrNoTranslation) || !strings.Contains(err.Error(), "HTTP 500") {
		t.Fatalf("err = %v, want a PartialError caused by the failed retry", err)
	}
	if got[0] != "XX:first text" || len(partial.Missing) != 1 || partial.Missing[0] != 1 {
		t.Fatalf("got = %q, missing = %v", got, partial.Missing)
	}
}
