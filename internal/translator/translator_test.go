package translator

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBatchTexts(t *testing.T) {
	texts := []string{"hello", "world", "foo", "bar"}
	batches := batchTexts(texts, 10)

	// "hello" (5) + "world" (5) = 10, then "foo" (3) + "bar" (3) = 6
	if len(batches) != 2 {
		t.Errorf("expected 2 batches, got %d", len(batches))
	}
}

func TestBatchTextsOversized(t *testing.T) {
	texts := []string{"short", "this is a very long text that exceeds limit", "tiny"}
	batches := batchTexts(texts, 10)

	// "short" alone, then the long one alone, then "tiny" alone
	if len(batches) != 3 {
		t.Errorf("expected 3 batches, got %d: %v", len(batches), batches)
	}
}

func newTestClient(serverURL string) *GoogleClient {
	return &GoogleClient{
		apiKey:     "test-key",
		baseURL:    serverURL,
		httpClient: &http.Client{},
	}
}

func TestTranslateSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req translateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		if req.Source != "en" || req.Target != "ru" {
			t.Errorf("unexpected langs: src=%s, dst=%s", req.Source, req.Target)
		}

		resp := translateResponse{}
		for _, q := range req.Q {
			resp.Data.Translations = append(resp.Data.Translations, struct {
				TranslatedText string `json:"translatedText"`
			}{TranslatedText: "translated:" + q})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	results, err := client.Translate([]string{"hello", "world"}, "en", "ru")
	if err != nil {
		t.Fatalf("Translate failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0] != "translated:hello" {
		t.Errorf("unexpected result[0]: %s", results[0])
	}
	if results[1] != "translated:world" {
		t.Errorf("unexpected result[1]: %s", results[1])
	}
}

func TestTranslateRetryOn429(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 2 {
			w.WriteHeader(429)
			w.Write([]byte(`{"error": {"code": 429, "message": "rate limited"}}`))
			return
		}

		var req translateRequest
		json.NewDecoder(r.Body).Decode(&req)

		resp := translateResponse{}
		for range req.Q {
			resp.Data.Translations = append(resp.Data.Translations, struct {
				TranslatedText string `json:"translatedText"`
			}{TranslatedText: "ok"})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	results, err := client.Translate([]string{"test"}, "en", "ru")
	if err != nil {
		t.Fatalf("Translate should succeed after retries, got: %v", err)
	}
	if len(results) != 1 || results[0] != "ok" {
		t.Errorf("unexpected result: %v", results)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts (2 failed + 1 success), got %d", attempts)
	}
}

func TestTranslateNonRetryableError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		w.Write([]byte(`{"error": {"code": 403, "message": "forbidden"}}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.Translate([]string{"test"}, "en", "ru")
	if err == nil {
		t.Fatal("expected error on 403, got nil")
	}
}

func TestTranslateEmpty(t *testing.T) {
	client := NewGoogleClient("test-key")
	results, err := client.Translate(nil, "en", "ru")
	if err != nil {
		t.Fatalf("expected nil error for empty input, got: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil results for empty input, got: %v", results)
	}
}
