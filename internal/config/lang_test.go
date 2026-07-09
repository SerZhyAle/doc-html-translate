package config

import (
	"strings"
	"testing"
)

func TestSuspiciousLangCodes(t *testing.T) {
	cases := []struct {
		name     string
		src, dst string
		wantBad  []string // substrings expected in the labels (nil = expect none)
	}{
		{"both plain", "en", "ru", nil},
		{"three-letter and script", "fil", "sr-Latn", nil},
		{"region subtag", "zh", "zh-CN", nil},
		{"portuguese brazil", "pt", "pt-BR", nil},
		{"full name src", "russian", "en", []string{"-src russian"}},
		{"full name dst", "en", "english", []string{"-dst english"}},
		{"underscore form", "en_US", "ru", []string{"-src en_US"}},
		{"empty dst", "en", "", []string{"-dst "}},
		{"both bad", "deutsch", "franc_ais", []string{"-src deutsch", "-dst franc_ais"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := SuspiciousLangCodes(c.src, c.dst)
			if len(c.wantBad) == 0 {
				if len(got) != 0 {
					t.Fatalf("SuspiciousLangCodes(%q,%q) = %v, want none", c.src, c.dst, got)
				}
				return
			}
			if len(got) != len(c.wantBad) {
				t.Fatalf("SuspiciousLangCodes(%q,%q) = %v, want %d entries", c.src, c.dst, got, len(c.wantBad))
			}
			joined := strings.Join(got, "|")
			for _, want := range c.wantBad {
				if !strings.Contains(joined, want) {
					t.Errorf("SuspiciousLangCodes(%q,%q) = %v, missing label containing %q", c.src, c.dst, got, want)
				}
			}
		})
	}
}
