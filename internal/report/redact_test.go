package report

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setHome points both location variables at a temporary profile so the path rules are the
// same on every machine that runs the test.
func setHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	t.Setenv("LOCALAPPDATA", filepath.Join(home, "AppData", "Local"))
	return home
}

func TestRedactRemovesGoogleAPIKeyShapes(t *testing.T) {
	setHome(t)

	secrets := map[string]string{
		"bare":     "using AIzaSyA1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q7r for translation",
		"key=":     "google_key=AIzaSyA1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q7r",
		"token:":   "token: abcdefghijklmnopqrstuvwxyz012345",
		"password": "Password hunter2hunter2hunter2hunter2",
		"json":     `{"googleKey":"AIzaSyA1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q7r"}`,
	}
	for name, in := range secrets {
		got := Redact(in)
		if !strings.Contains(got, "[REDACTED]") {
			t.Errorf("%s: nothing was redacted in %q -> %q", name, in, got)
		}
		for _, leak := range []string{"AIzaSyA1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q7r", "abcdefghijklmnopqrstuvwxyz012345", "hunter2hunter2hunter2hunter2"} {
			if strings.Contains(got, leak) {
				t.Errorf("%s: secret survived redaction: %q", name, got)
			}
		}
	}
}

func TestRedactStructuredValueVectors(t *testing.T) {
	setHome(t)

	tests := []struct {
		name        string
		input       string
		expectField string
		expectDrop  string
	}{
		{
			name:        "json_access_pin",
			input:       `folderParams={"folders":[{"label":"Family","accessPin":"4821","slideshowInterval":15}]}`,
			expectField: `folderParams={"folders":[{"label":"Family","accessPin":"[REDACTED]","slideshowInterval":15}]}`,
			expectDrop:  `folderParams=[REDACTED]`,
		},
		{
			name:        "query_token",
			input:       `lastOpened=https://media.example.test/play?id=17&token=Zk93mQ&lang=en`,
			expectField: `lastOpened=https://media.example.test/play?id=17&token=[REDACTED]&lang=en`,
			expectDrop:  `lastOpened=[REDACTED]`,
		},
		{
			name:        "connection_string_password",
			input:       `source=Host=nas.example.test;Port=22;User=guest;Password=Hq7-pL2x;Timeout=15`,
			expectField: `source=Host=nas.example.test;Port=22;User=guest;Password=[REDACTED];Timeout=15`,
			expectDrop:  `source=[REDACTED]`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Redact(tc.input)
			if got != tc.expectField && got != tc.expectDrop {
				t.Errorf("Redact(%q) = %q; want field-redacted %q or dropped %q", tc.input, got, tc.expectField, tc.expectDrop)
			}
		})
	}
}

func TestRedactClarificationShapes(t *testing.T) {
	setHome(t)

	tests := []struct {
		name     string
		input    string
		expected string
		leak     string
	}{
		{
			name:     "xtream_live_path",
			input:    "GET /live/john_doe/secretPass123/45678.ts HTTP/1.1",
			expected: "GET /live/[REDACTED]/[REDACTED]/45678.ts HTTP/1.1",
			leak:     "secretPass123",
		},
		{
			name:     "xtream_movie_path",
			input:    "http://iptv.example.com/movie/alice/p@ss/movie.mp4",
			expected: "http://iptv.example.com/movie/[REDACTED]/[REDACTED]/movie.mp4",
			leak:     "alice",
		},
		{
			name:     "signed_cdn_amz",
			input:    "https://s3.example.com/doc.pdf?X-Amz-Signature=abcdef123456&X-Amz-Credential=AKIAIOSFODNN7EXAMPLE&foo=bar",
			expected: "https://s3.example.com/doc.pdf?X-Amz-Signature=[REDACTED]&X-Amz-Credential=[REDACTED]&foo=bar",
			leak:     "abcdef123456",
		},
		{
			name:     "signed_cdn_html_escaped_amp",
			input:    "https://stream.example.com/live.m3u8?wmsAuthSign=token123&amp;hdnts=exp=12345~acl=*&amp;quality=high",
			expected: "https://stream.example.com/live.m3u8?wmsAuthSign=[REDACTED]&amp;hdnts=[REDACTED]&amp;quality=high",
			leak:     "token123",
		},
		{
			name:     "cloudfront_policy_key_pair",
			input:    "https://d111.cloudfront.net/v.mp4?Policy=ey...&Key-Pair-Id=K2JCJMDEHXQW50",
			expected: "https://d111.cloudfront.net/v.mp4?Policy=[REDACTED]&Key-Pair-Id=[REDACTED]",
			leak:     "K2JCJMDEHXQW50",
		},
		{
			name:     "url_userinfo_standard",
			input:    "Connecting to http://admin:mypassword@nas.local/share",
			expected: "Connecting to http://[REDACTED]:[REDACTED]@nas.local/share",
			leak:     "mypassword",
		},
		{
			name:     "url_userinfo_special_chars_in_password",
			input:    "Connecting to http://admin:pass/word?extra#1@nas.local/share",
			expected: "Connecting to http://[REDACTED]@nas.local/share",
			leak:     "pass/word",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Redact(tc.input)
			if strings.Contains(got, tc.leak) {
				t.Errorf("%s: leak %q survived: %q", tc.name, tc.leak, got)
			}
			if !strings.Contains(got, "[REDACTED]") {
				t.Errorf("%s: expected [REDACTED] in %q", tc.name, got)
			}
		})
	}
}

func TestRedactShortensUserProfilePaths(t *testing.T) {
	home := setHome(t)

	in := filepath.Join(home, "Documents", "Books", "War and Peace.epub")
	got := Redact(in)
	if !strings.HasPrefix(got, "%USERPROFILE%") {
		t.Errorf("Redact(%q) = %q, want it to start with %%USERPROFILE%%", in, got)
	}
	if strings.Contains(got, home) {
		t.Errorf("the home directory survived: %q", got)
	}
	if !strings.Contains(got, "War and Peace.epub") {
		t.Errorf("the document's own name was lost: %q", got)
	}
}

func TestRedactKeepsDocumentFileNames(t *testing.T) {
	home := setHome(t)

	in := filepath.Join(home, "Загрузки", "Война и мир.fb2")
	got := Redact(in)
	if !strings.Contains(got, "Война и мир.fb2") {
		t.Errorf("a Cyrillic document name did not survive: %q", got)
	}
}

func TestRedactPrefersTheMoreSpecificLocation(t *testing.T) {
	home := setHome(t)

	in := filepath.Join(home, "AppData", "Local", "doc-html-translate", "logs", "run-20260729-130509.log")
	got := Redact(in)
	if !strings.HasPrefix(got, "%LOCALAPPDATA%") {
		t.Errorf("Redact(%q) = %q, want it to start with %%LOCALAPPDATA%%", in, got)
	}
}

func TestRedactProfilePathEncodings(t *testing.T) {
	home := filepath.Join(t.TempDir(), "Audit User")
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	t.Setenv("LOCALAPPDATA", filepath.Join(home, "AppData", "Local"))
	slash := filepath.ToSlash(home)
	filePath := slash
	if !strings.HasPrefix(filePath, "/") {
		filePath = "/" + filePath
	}
	paths := map[string]string{
		"native":        filepath.Join(home, "Books", "novel.epub"),
		"json escaped":  strings.ReplaceAll(home, `\`, `\\`) + `\\Books\\novel.epub`,
		"forward slash": slash + "/Books/novel.epub",
		"file URL":      (&url.URL{Scheme: "file", Path: filePath + "/Books/novel.epub"}).String(),
	}
	for name, path := range paths {
		t.Run(name, func(t *testing.T) {
			got := Redact(path)
			if !strings.Contains(got, "%USERPROFILE%") || !strings.Contains(got, "novel.epub") {
				t.Errorf("Redact(%q) = %q", path, got)
			}
			if strings.Contains(got, "Audit User") || strings.Contains(got, "Audit%20User") {
				t.Errorf("profile name survived redaction: %q", got)
			}
		})
	}
}

func TestRedactCatalogVectorFileIfAvailable(t *testing.T) {
	setHome(t)
	catalog := os.Getenv("SZA_CONTRACTS_ROOT")
	if catalog == "" {
		catalog = os.Getenv("SZA_CONTRACTS_CATALOG")
	}
	if catalog == "" {
		// Read catalog path if AGENTS.md mentions one
		agents, err := os.ReadFile(filepath.Join("..", "..", "AGENTS.md"))
		if err == nil {
			for _, line := range strings.Split(string(agents), "\n") {
				if strings.Contains(line, "The shared contracts catalog is at `") {
					parts := strings.Split(line, "`")
					if len(parts) >= 2 {
						catalog = parts[1]
						break
					}
				}
			}
		}
	}
	if catalog == "" {
		t.Skip("contracts catalog not configured")
	}
	vectorPath := filepath.Join(catalog, "diagnostic-report", "reference", "redaction-structured-value.txt")
	data, err := os.ReadFile(vectorPath)
	if err != nil {
		t.Skipf("catalog vector file not found at %q: %v", vectorPath, err)
	}

	var input, field, drop string
	total, matched := 0, 0
	for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "INPUT=") {
			input = strings.TrimPrefix(line, "INPUT=")
		}
		if strings.HasPrefix(line, "EXPECTED-FIELD=") {
			field = strings.TrimPrefix(line, "EXPECTED-FIELD=")
		}
		if strings.HasPrefix(line, "EXPECTED-DROP=") {
			drop = strings.TrimPrefix(line, "EXPECTED-DROP=")
			got := Redact(input)
			total++
			if got == field || got == drop {
				matched++
			} else {
				t.Errorf("Vector mismatch:\n  input:    %s\n  got:      %s\n  expected: %s (or %s)", input, got, field, drop)
			}
		}
	}
	if total == 0 {
		t.Fatal("no vectors found in catalog reference file")
	}
	if matched != total {
		t.Errorf("matched %d/%d vectors", matched, total)
	}
}
