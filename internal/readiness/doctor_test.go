package readiness

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"doc-html-translate/internal/fdsec"
	"doc-html-translate/internal/pipeline"
)

// fakeMachine installs probes for a machine with the given helpers and no real lookup behind them. Tools that are found
// answer with the version "<name>-1.2".
func fakeMachine(t *testing.T, found map[string]bool) {
	t.Helper()
	swapProbes(t)
	Probes.Calibre = func() bool { return found[ToolCalibre] }
	Probes.SevenZip = func() bool { return found[ToolSevenZip] }
	Probes.FileDO = func() bool { return found[ToolFileDO] }
	Probes.PDFToText = func() bool { return found[ToolPDFToText] }
	Probes.OCRLocate = func() (string, error) {
		if found[ToolTesseract] {
			return `C:\Tools\Tesseract\tesseract.exe`, nil // a path the report must never carry
		}
		return "", errors.New("missing")
	}
	Probes.OCRLangs = func() []string { return []string{"eng", "rus"} }
	Probes.GoogleKey = func() (string, error) { return "", errors.New("no key") }
	Probes.Ollama = func(context.Context, string) (bool, error) {
		t.Error("the Ollama probe was called")
		return false, errors.New("must not be called")
	}
	Probes.Version = func(_ context.Context, tool string) string { return tool + "-1.2" }
	Probes.Packaged = func() bool { return false }
}

func formatOf(t *testing.T, rep Report, ext string) FormatInfo {
	t.Helper()
	for _, f := range rep.Formats {
		if f.Ext == ext {
			return f
		}
	}
	t.Fatalf("no format %q in %+v", ext, rep.Formats)
	return FormatInfo{}
}

func TestDoctorReportsEveryMember(t *testing.T) {
	fakeMachine(t, map[string]bool{ToolTesseract: true, ToolPDFToText: true, ToolCalibre: true, ToolSevenZip: true, ToolFileDO: true})
	Probes.Packaged = func() bool { return true }
	Probes.GoogleKey = func() (string, error) { return "k", nil }

	rep := Diagnose(context.Background(), "26.1114.0001", DoctorOptions{})
	if rep.Version != "26.1114.0001" || !rep.Packaged || !rep.GoogleKeyPresent {
		t.Errorf("version, packaged, googleKeyPresent: %+v", rep)
	}
	for name, got := range map[string]ToolInfo{
		ToolTesseract: rep.Tools.Tesseract, ToolPDFToText: rep.Tools.PDFToText, ToolCalibre: rep.Tools.Calibre, ToolSevenZip: rep.Tools.SevenZip,
	} {
		if !got.Found || got.Version != name+"-1.2" {
			t.Errorf("%s = %+v", name, got)
		}
	}
	if !rep.Tools.FileDO.Found || rep.Tools.FileDO.Version != "" {
		t.Errorf("FileDO has no version query: %+v", rep.Tools.FileDO)
	}
	if strings.Join(rep.OCRLangs, ",") != "eng,rus" || rep.Ollama != nil {
		t.Errorf("ocrLangs %v, ollama %+v", rep.OCRLangs, rep.Ollama)
	}

	// The members, by name, as a script reads them.
	data, err := json.Marshal(rep)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"version", "packaged", "tools", "formats", "ocrLangs", "googleKeyPresent"} {
		if _, ok := m[key]; !ok {
			t.Errorf("member %q is missing from %s", key, data)
		}
	}
	if _, ok := m["ollama"]; ok {
		t.Errorf("the ollama member is present without --ollama: %s", data)
	}
	var tools map[string]map[string]any
	if err := json.Unmarshal(m["tools"], &tools); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"tesseract", "pdftotext", "calibre", "sevenzip", "filedo"} {
		if _, ok := tools[name]["found"].(bool); !ok {
			t.Errorf("tools.%s.found is not a boolean: %v", name, tools[name])
		}
		if _, ok := tools[name]["version"].(string); !ok {
			t.Errorf("tools.%s.version is not a string: %v", name, tools[name])
		}
	}
}

func TestDoctorWithoutCalibreNamesItForMobiAndAzw3(t *testing.T) {
	fakeMachine(t, map[string]bool{ToolTesseract: true})
	rep := Diagnose(context.Background(), "x", DoctorOptions{})
	if rep.Tools.Calibre.Found || rep.Tools.Calibre.Version != "" {
		t.Errorf("calibre = %+v", rep.Tools.Calibre)
	}
	for _, ext := range []string{".mobi", ".azw3"} {
		if f := formatOf(t, rep, ext); f.Tool != ToolCalibre || f.Ready {
			t.Errorf("%s = %+v, want the missing tool calibre and ready false", ext, f)
		}
	}
	for _, ext := range []string{".epub", ".txt", ".pdf", ".png", ".cbz"} {
		if f := formatOf(t, rep, ext); f.Tool != "" || !f.Ready {
			t.Errorf("%s needs no helper: %+v", ext, f)
		}
	}
	if f := formatOf(t, rep, ".cbr"); f.Tool != ToolSevenZip || f.Ready || f.OptionalTool != ToolTesseract {
		t.Errorf(".cbr = %+v", f)
	}
	if f := formatOf(t, rep, ".fd-sec"); f.Tool != ToolFileDO || f.Ready {
		t.Errorf(".fd-sec = %+v", f)
	}
	if f := formatOf(t, rep, ".pdf"); f.OptionalTool != ToolPDFToText {
		t.Errorf(".pdf = %+v", f)
	}
}

// No member of the report holds a path or the key: the fake machine's tesseract lives in a drive path and the fake
// store holds a key, and neither text may appear in the JSON.
func TestDoctorLeaksNoPathAndNoKey(t *testing.T) {
	const key = "AIzaSy-FAKE-KEY-9f3b2c"
	fakeMachine(t, map[string]bool{ToolTesseract: true, ToolCalibre: true, ToolSevenZip: true, ToolFileDO: true, ToolPDFToText: true})
	Probes.GoogleKey = func() (string, error) { return key, nil }
	Probes.Ollama = func(context.Context, string) (bool, error) { return true, nil }
	Probes.Version = func(context.Context, string) string { return "5.3.3" }

	for _, opt := range []DoctorOptions{{}, {Ollama: true, OllamaModel: "gemma3:12b"}} {
		data, err := json.Marshal(Diagnose(context.Background(), "26.1114.0001", opt))
		if err != nil {
			t.Fatal(err)
		}
		for _, bad := range []string{`:\`, `:\\`, key, "Tesseract", "/"} {
			if strings.Contains(string(data), bad) {
				t.Errorf("the report holds %q: %s", bad, data)
			}
		}
		if !strings.Contains(string(data), `"googleKeyPresent":true`) {
			t.Errorf("a stored key is reported as present: %s", data)
		}
	}
}

// Without --ollama nothing is requested from any service: the probe is never called, and a server on the Ollama port
// never sees a connection (the second half is skipped when a real server holds the port).
func TestDoctorAsksNoServiceWithoutOllama(t *testing.T) {
	fakeMachine(t, nil) // its Ollama probe fails the test when called

	ln, err := net.Listen("tcp", "127.0.0.1:11434")
	accepted := make(chan struct{}, 1)
	if err == nil {
		defer func() { _ = ln.Close() }()
		go func() {
			if c, err := ln.Accept(); err == nil {
				_ = c.Close()
				accepted <- struct{}{}
			}
		}()
	}
	rep := Diagnose(context.Background(), "x", DoctorOptions{Ollama: false, OllamaModel: "gemma3:12b"})
	if rep.Ollama != nil {
		t.Errorf("ollama member without --ollama: %+v", rep.Ollama)
	}
	if err != nil {
		t.Logf("port 11434 is taken (%v): the listener half is skipped, the probe half ran", err)
		return
	}
	select {
	case <-accepted:
		t.Error("something connected to 127.0.0.1:11434 although --ollama was not given")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestDoctorWithOllama(t *testing.T) {
	cases := []struct {
		name    string
		present bool
		err     error
		want    OllamaInfo
	}{
		{"has the model", true, nil, OllamaInfo{Model: "m1", Reachable: true, ModelPresent: true}},
		{"answers without the model", false, nil, OllamaInfo{Model: "m1", Reachable: true}},
		{"does not answer", true, errors.New("offline"), OllamaInfo{Model: "m1"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeMachine(t, nil)
			var asked string
			Probes.Ollama = func(_ context.Context, model string) (bool, error) { asked = model; return tc.present, tc.err }
			rep := Diagnose(context.Background(), "x", DoctorOptions{Ollama: true, OllamaModel: " m1 "})
			if rep.Ollama == nil || *rep.Ollama != tc.want || asked != "m1" {
				t.Errorf("ollama = %+v asked %q, want %+v for m1", rep.Ollama, asked, tc.want)
			}
		})
	}
	fakeMachine(t, nil)
	var asked string
	Probes.Ollama = func(_ context.Context, model string) (bool, error) { asked = model; return true, nil }
	if rep := Diagnose(context.Background(), "x", DoctorOptions{Ollama: true}); asked != DefaultOllamaModel || rep.Ollama.Model != DefaultOllamaModel {
		t.Errorf("no model named: asked %q", asked)
	}
}

func TestParseVersion(t *testing.T) {
	for in, want := range map[string]string{
		"tesseract 5.3.3\n leptonica-1.83.1\n  libgif 5.2.1":                       "5.3.3",
		"ebook-convert (calibre 7.5.1)":                                            "7.5.1",
		"\n7-Zip 23.01 (x64) : Copyright (c) 1999-2023 Igor Pavlov : 2023-06-20\n": "23.01",
		"p7zip Version 16.02 (locale=C.UTF-8)":                                     "16.02",
		"pdftotext version 4.00\nCopyright 1996-2017 Glyph & Cog, LLC":             "4.00",
		"Usage: tool [options]":                                                    "",
		"build 12":                                                                 "",
		"":                                                                         "",
		strings.Repeat("9.", 40) + "9":                                             "",
	} {
		if got := parseVersion(in); got != want {
			t.Errorf("parseVersion(%q) = %q, want %q", in, got, want)
		}
	}
}

// helperProcess is the test binary run as a stand-in tool: HELPER_MODE says what it prints or does.
func TestHelperProcess(t *testing.T) {
	switch os.Getenv("READINESS_HELPER_MODE") {
	case "":
		return
	case "version":
		_, _ = os.Stderr.WriteString("faketool version 7.8.9\n")
		os.Exit(3) // a banner followed by a non-zero exit still counts
	case "sleep":
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}
	os.Exit(0)
}

func TestReadVersionRunsTheTool(t *testing.T) {
	t.Setenv("READINESS_HELPER_MODE", "version")
	if got := readVersion(context.Background(), "faketool", os.Args[0], "-test.run=^TestHelperProcess$"); got != "7.8.9" {
		t.Errorf("readVersion = %q, want 7.8.9", got)
	}
	if got := readVersion(context.Background(), "faketool", `Z:\no\such\tool.exe`); got != "" {
		t.Errorf("a tool that cannot start gives %q", got)
	}
}

func TestReadVersionGivesUpAtTheDeadline(t *testing.T) {
	t.Setenv("READINESS_HELPER_MODE", "sleep")
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	start := time.Now()
	if got := readVersion(ctx, "faketool", os.Args[0], "-test.run=^TestHelperProcess$"); got != "" {
		t.Errorf("readVersion = %q, want empty", got)
	}
	if d := time.Since(start); d > 2500*time.Millisecond {
		t.Errorf("took %v: the version query is not bounded", d)
	}
}

// formatTable is held to the converter both ways: every row is an input the converter accepts, and every extension the
// converter accepts has a row. The converter's predicate is probed over every extension of up to four letters or digits.
func TestFormatTableMatchesTheConverter(t *testing.T) {
	inTable := map[string]bool{}
	for _, r := range formatTable {
		if inTable[r.ext] {
			t.Errorf("%s is listed twice", r.ext)
		}
		inTable[r.ext] = true
		if !pipeline.Convertible(r.ext) && !fdsec.IsContainer(r.ext) {
			t.Errorf("%s is in the table and the converter does not accept it", r.ext)
		}
	}
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	var probe func(prefix string, left int)
	probe = func(prefix string, left int) {
		if len(prefix) > 1 && pipeline.Convertible(prefix) && !inTable[prefix] {
			t.Errorf("the converter accepts %s and the table does not list it", prefix)
		}
		if left == 0 {
			return
		}
		for i := 0; i < len(alphabet); i++ {
			probe(prefix+string(alphabet[i]), left-1)
		}
	}
	probe(".", 4)
	if !inTable[fdsec.Ext] {
		t.Errorf("the FileDO container %s is not in the table", fdsec.Ext)
	}
}
