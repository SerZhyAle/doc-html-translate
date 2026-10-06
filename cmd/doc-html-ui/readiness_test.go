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
	readinessProbes.filedo = func() bool { return true }
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

// A FileDO secret file is screened before Convert: FileDO must be installed and the size must
// pass the format's length rules. Both are action items, neither asks for a password.
func TestReadinessForAFileDOSecretFile(t *testing.T) {
	saved := readinessProbes
	t.Cleanup(func() { readinessProbes = saved })
	readinessProbes.filedo = func() bool { return true }

	container := func(size int) string {
		t.Helper()
		p := filepath.Join(t.TempDir(), "notes.fd-sec")
		if err := os.WriteFile(p, make([]byte, size), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	codes := func(r readinessResult) []string {
		var out []string
		for _, i := range r.Issues {
			out = append(out, i.Code)
		}
		return out
	}

	if got := checkReadiness(context.Background(), runRequest{Input: container(12288)}); got.State != "ready" {
		t.Errorf("a container that passes both screens is ready, got %+v", got)
	}
	got := checkReadiness(context.Background(), runRequest{Input: container(100)})
	if got.State != "action" || !slices.Equal(codes(got), []string{"fdsecScreen"}) {
		t.Errorf("a size that rules the file out = %+v", got)
	}
	readinessProbes.filedo = func() bool { return false }
	got = checkReadiness(context.Background(), runRequest{Input: container(12288)})
	if got.State != "action" || !slices.Equal(codes(got), []string{"filedo"}) {
		t.Errorf("a missing FileDO = %+v", got)
	}
	// Missing FileDO is reported ahead of the size: install first, then the size can matter.
	got = checkReadiness(context.Background(), runRequest{Input: container(100)})
	if !slices.Equal(codes(got), []string{"filedo"}) {
		t.Errorf("missing FileDO must come first, got %+v", got)
	}
}
