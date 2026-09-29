package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestReadinessUsesSelectedCapabilities(t *testing.T) {
	saved := readinessProbes
	t.Cleanup(func() { readinessProbes = saved })
	readinessProbes.calibre = func() bool { return false }
	readinessProbes.sevenZip = func() bool { return false }
	readinessProbes.ocrLocate = func() (string, error) { return "tesseract", nil }
	readinessProbes.ocrMissing = func(_ context.Context, _, _ string) []string { return []string{"rus"} }
	readinessProbes.googleKey = func() (string, error) { return "", errors.New("missing") }
	readinessProbes.ollama = func(_ context.Context, _ string) (bool, error) { return false, nil }

	file := func(ext string) string {
		t.Helper()
		p := filepath.Join(t.TempDir(), "book"+ext)
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	cases := []struct {
		name    string
		request runRequest
		want    []string
		state   string
	}{
		{"plain", runRequest{Input: file(".txt")}, nil, "ready"},
		{"text with OCR selected", runRequest{Input: file(".txt"), OCR: true}, nil, "ready"},
		{"mobi", runRequest{Input: file(".mobi")}, []string{"calibre"}, "action"},
		{"cbz", runRequest{Input: file(".cbz")}, []string{"ocrData"}, "optional"},
		{"cbr", runRequest{Input: file(".cbr")}, []string{"sevenzip", "ocrData"}, "action"},
		{"ocr", runRequest{Input: file(".epub"), OCR: true}, []string{"ocrData"}, "optional"},
		{"google", runRequest{Input: file(".pdf"), Google: true, MaxCost: "2"}, []string{"googleKey"}, "optional"},
		{"ollama", runRequest{Input: file(".pdf"), Ollama: true}, []string{"ollamaModel"}, "optional"},
		{"disabled", runRequest{Input: file(".pdf"), Google: true, NoTranslate: true}, nil, "ready"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := checkReadiness(context.Background(), tc.request)
			if got.State != tc.state {
				t.Fatalf("state = %q, want %q", got.State, tc.state)
			}
			var codes []string
			for _, issue := range got.Issues {
				codes = append(codes, issue.Code)
			}
			if !slices.Equal(codes, tc.want) {
				t.Fatalf("issues = %v, want %v", codes, tc.want)
			}
		})
	}
	readinessProbes.ocrLocate = func() (string, error) { return "", errors.New("missing") }
	got := checkReadiness(context.Background(), runRequest{Input: file(".png")})
	if len(got.Issues) != 1 || got.Issues[0].Code != "tesseract" || got.State != "optional" {
		t.Fatalf("missing Tesseract = %+v", got)
	}
	readinessProbes.ollama = func(_ context.Context, _ string) (bool, error) { return false, errors.New("offline") }
	got = checkReadiness(context.Background(), runRequest{Input: file(".pdf"), Ollama: true})
	if len(got.Issues) != 1 || got.Issues[0].Code != "ollamaService" {
		t.Fatalf("stopped Ollama = %+v", got)
	}
	got = checkReadiness(context.Background(), runRequest{Input: file(".pdf"), Google: true, Ollama: true, MaxCost: "invalid"})
	if got.GoogleLimit != "0" || len(got.Issues) != 1 || got.Issues[0].Code != "googleKey" {
		t.Fatalf("Google precedence and effective limit = %+v", got)
	}
}
