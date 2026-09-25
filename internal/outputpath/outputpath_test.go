package outputpath

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
		{name: "reserved_stem_with_dot", in: "CON.tar", out: "CON_.tar"},
		{name: "reserved_conin", in: "conin$", out: "conin$_"},
		{name: "reserved_superscript_port", in: "COM¹", out: "COM¹_"},
		{name: "com_with_letters_kept", in: "COMA", out: "COMA"},
		{name: "invalid_chars_replaced", in: "a:b|c?", out: "a_b_c_"},
		{name: "control_chars_replaced", in: "a\tb", out: "a_b"},
		{name: "dots_only_fall_back", in: "....", out: "document"},
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
	got := OutputDirFor(input, folder)
	want := filepath.Join(folder, "Auntie")
	if got != want {
		t.Fatalf("OutputDirFor() = %q, want %q", got, want)
	}
}
