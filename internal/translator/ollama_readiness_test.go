package translator

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOllamaReadinessOnlyReadsLocalCatalog(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/tags" {
			t.Errorf("unexpected Ollama request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"models":[{"name":"sample:latest"}]}`))
	}))
	defer server.Close()
	got, err := ollamaModelAvailableAt(context.Background(), "sample", server.URL+"/api/tags")
	if err != nil || !got {
		t.Fatalf("installed model: available=%v, err=%v", got, err)
	}
	got, err = ollamaModelAvailableAt(context.Background(), "other", server.URL+"/api/tags")
	if err != nil || got {
		t.Fatalf("missing model: available=%v, err=%v", got, err)
	}
}
