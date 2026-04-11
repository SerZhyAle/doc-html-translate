package pipeline

import (
	"path/filepath"
	"testing"
)

func TestSanitizeOutputName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		out  string
	}{
		{name: "keeps_regular_name", in: "My Book", out: "My Book"},
		{name: "trims_trailing_dot", in: "Title..", out: "Title"},
		{name: "trims_trailing_space", in: "Title   ", out: "Title"},
		{name: "empty_after_trim_falls_back", in: "   ...   ", out: "document"},
		{name: "reserved_name_gets_suffix", in: "CON", out: "CON_"},
		{name: "reserved_name_com3_gets_suffix", in: "com3", out: "com3_"},
		{name: "non_reserved_name_kept", in: "Company", out: "Company"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := sanitizeOutputName(tt.in)
			if got != tt.out {
				t.Fatalf("sanitizeOutputName(%q) = %q, want %q", tt.in, got, tt.out)
			}
		})
	}
}

func TestOutputDirForUsesSanitizedBaseName(t *testing.T) {
	t.Parallel()

	input := filepath.Join("C:\\", "books", "Auntie...pdf")
	folder := filepath.Join("D:\\", "out")
	got := outputDirFor(input, folder)
	want := filepath.Join(folder, "Auntie")
	if got != want {
		t.Fatalf("outputDirFor() = %q, want %q", got, want)
	}
}
