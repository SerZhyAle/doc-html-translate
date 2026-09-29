package translator

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProtectedKeySaveMigrationAndCorruption(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("LOCALAPPDATA", tmp)
	plain := filepath.Join(tmp, appDataDirName, googleAPIKeyFile)
	if err := os.MkdirAll(filepath.Dir(plain), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(plain, []byte("  AIzaSyMIGRATED  \n"), 0o600); err != nil {
		t.Fatal(err)
	}
	key, err := LoadGoogleAPIKey()
	if err != nil || key != "AIzaSyMIGRATED" {
		t.Fatalf("migration: key=%q err=%v", key, err)
	}
	if _, err := os.Stat(plain); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("plaintext remains: %v", err)
	}
	blob, err := os.ReadFile(GoogleAPIKeyPath())
	if err != nil || strings.Contains(string(blob), key) {
		t.Fatalf("protected file contains plaintext or is absent: %v", err)
	}
	if got, err := LoadGoogleAPIKey(); err != nil || got != key {
		t.Fatalf("reload: key=%q err=%v", got, err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Goog-Api-Key") != key {
			t.Error("migrated key was not sent in header")
		}
		_, _ = w.Write([]byte(`{"data":{"translations":[{"translatedText":"hello"}]}}`))
	}))
	defer server.Close()
	client := NewGoogleClient(key)
	client.baseURL = server.URL
	if got, err := client.Translate(context.Background(), []string{"hi"}, "en", "fr"); err != nil || len(got) != 1 || got[0] != "hello" {
		t.Fatalf("translation with migrated key: %q, %v", got, err)
	}
	if err := os.WriteFile(GoogleAPIKeyPath(), []byte("damaged"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadGoogleAPIKey(); !errors.Is(err, ErrGoogleKeyUnusable) {
		t.Fatalf("corrupt blob: %v", err)
	}
	if err := SaveGoogleAPIKey("AIzaSyREPLACED"); err != nil {
		t.Fatal(err)
	}
	if got, err := LoadGoogleAPIKey(); err != nil || got != "AIzaSyREPLACED" {
		t.Fatalf("replacement: key=%q err=%v", got, err)
	}
}
