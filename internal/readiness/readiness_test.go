package readiness

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// swapProbes puts the whole probe set back when the test ends.
func swapProbes(t *testing.T) {
	t.Helper()
	saved := Probes
	t.Cleanup(func() { Probes = saved })
}

func TestReadinessUsesSelectedCapabilities(t *testing.T) {
	swapProbes(t)
	Probes.Calibre = func() bool { return false }
	Probes.SevenZip = func() bool { return false }
	Probes.FileDO = func() bool { return true }
	Probes.OCRLocate = func() (string, error) { return "tesseract", nil }
	Probes.OCRMissing = func(_ context.Context, _, _ string) []string { return []string{"rus"} }
	Probes.GoogleKey = func() (string, error) { return "", errors.New("missing") }
	Probes.Ollama = func(_ context.Context, _ string) (bool, error) { return false, nil }

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
		request Request
		want    []string
		state   string
	}{
		{"plain", Request{Input: file(".txt")}, nil, "ready"},
		{"text with OCR selected", Request{Input: file(".txt"), OCR: true}, nil, "ready"},
		{"mobi", Request{Input: file(".mobi")}, []string{"calibre"}, "action"},
		{"cbz", Request{Input: file(".cbz")}, []string{"ocrData"}, "optional"},
		{"cbr", Request{Input: file(".cbr")}, []string{"sevenzip", "ocrData"}, "action"},
		{"ocr", Request{Input: file(".epub"), OCR: true}, []string{"ocrData"}, "optional"},
		{"google", Request{Input: file(".pdf"), Google: true, MaxCost: "2"}, []string{"googleKey"}, "optional"},
		{"ollama", Request{Input: file(".pdf"), Ollama: true}, []string{"ollamaModel"}, "optional"},
		{"disabled", Request{Input: file(".pdf"), Google: true, NoTranslate: true}, nil, "ready"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Check(context.Background(), tc.request)
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
	Probes.OCRLocate = func() (string, error) { return "", errors.New("missing") }
	got := Check(context.Background(), Request{Input: file(".png")})
	if len(got.Issues) != 1 || got.Issues[0].Code != "tesseract" || got.State != "optional" {
		t.Fatalf("missing Tesseract = %+v", got)
	}
	Probes.Ollama = func(_ context.Context, _ string) (bool, error) { return false, errors.New("offline") }
	got = Check(context.Background(), Request{Input: file(".pdf"), Ollama: true})
	if len(got.Issues) != 1 || got.Issues[0].Code != "ollamaService" {
		t.Fatalf("stopped Ollama = %+v", got)
	}
	got = Check(context.Background(), Request{Input: file(".pdf"), Google: true, Ollama: true, MaxCost: "invalid"})
	if got.GoogleLimit != "0" || len(got.Issues) != 1 || got.Issues[0].Code != "googleKey" {
		t.Fatalf("Google precedence and effective limit = %+v", got)
	}
}

// A FileDO secret file is screened before Convert: FileDO must be installed and the size must
// pass the format's length rules. Both are action items, neither asks for a password.
func TestReadinessForAFileDOSecretFile(t *testing.T) {
	swapProbes(t)
	Probes.FileDO = func() bool { return true }

	container := func(size int) string {
		t.Helper()
		p := filepath.Join(t.TempDir(), "notes.fd-sec")
		if err := os.WriteFile(p, make([]byte, size), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	codes := func(r Result) []string {
		var out []string
		for _, i := range r.Issues {
			out = append(out, i.Code)
		}
		return out
	}

	if got := Check(context.Background(), Request{Input: container(12288)}); got.State != "ready" {
		t.Errorf("a container that passes both screens is ready, got %+v", got)
	}
	got := Check(context.Background(), Request{Input: container(100)})
	if got.State != "action" || !slices.Equal(codes(got), []string{"fdsecScreen"}) {
		t.Errorf("a size that rules the file out = %+v", got)
	}
	Probes.FileDO = func() bool { return false }
	got = Check(context.Background(), Request{Input: container(12288)})
	if got.State != "action" || !slices.Equal(codes(got), []string{"filedo"}) {
		t.Errorf("a missing FileDO = %+v", got)
	}
	// Missing FileDO is reported ahead of the size: install first, then the size can matter.
	got = Check(context.Background(), Request{Input: container(100)})
	if !slices.Equal(codes(got), []string{"filedo"}) {
		t.Errorf("missing FileDO must come first, got %+v", got)
	}
}

// What the verb `input check` answers for a path that is not a document: the window's own answer, as data.
func TestReadinessForAPathThatIsNotADocument(t *testing.T) {
	swapProbes(t)
	dir := t.TempDir()
	empty := filepath.Join(dir, "empty.epub")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for name, in := range map[string]string{"a folder": dir, "an empty file": empty, "no such path": filepath.Join(dir, "none.epub")} {
		got := Check(context.Background(), Request{Input: in})
		if got.State != "action" || len(got.Issues) != 1 || got.Issues[0].Code != "input" {
			t.Errorf("%s = %+v", name, got)
		}
	}
	if got := Check(context.Background(), Request{Input: "  "}); got.State != "ready" || len(got.Issues) != 0 {
		t.Errorf("nothing asked about = %+v", got)
	}
}

func TestPackagedFollowsTheInstallFolder(t *testing.T) {
	for exe, want := range map[string]bool{
		`C:\Program Files\WindowsApps\SZA.Doc-HTML-Translate_1.0_x64__abc\doc-html-ui.exe`: true,
		`c:\PROGRAM FILES\WINDOWSAPPS\x\doc-html-ui.exe`:                                   true,
		`C:\Tools\doc-html-translate.exe`:                                                  false,
		`D:\windowsapps-copy\doc-html-translate.exe`:                                       false,
	} {
		if got := packagedPath(exe); got != want {
			t.Errorf("packagedPath(%q) = %v, want %v", exe, got, want)
		}
	}
}
