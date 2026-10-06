// Package fdsectest is a stand-in for FileDO's restore verb, so the tests of this package and of
// the pipeline can exercise every outcome class without a real FileDO or a real container.
//
// A test binary calls RunFakeIfAsked first thing in TestMain and points DOCHT_FILEDO at its own
// executable; setting DOCHT_FAKE_FILEDO to a mode then makes that executable behave as FileDO.
package fdsectest

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

// Env selects the mode; BuildEnv overrides the build printed in the banner.
const (
	Env      = "DOCHT_FAKE_FILEDO"
	BuildEnv = "DOCHT_FAKE_FILEDO_BUILD"
	// RightPassword is the one password the "wrong" mode accepts.
	RightPassword = "right-pw"
	// TrueName is the name the fake seals into every container: tests assert it never appears.
	TrueName = "Sealed True Name"
	// SecretVar is the variable FileDO's pe: source reads.
	SecretVar = "DOCHT_FDSEC_SECRET"
)

// RunFakeIfAsked behaves as FileDO and exits when Env is set; otherwise it returns.
func RunFakeIfAsked() {
	mode := os.Getenv(Env)
	if mode == "" {
		return
	}
	build := os.Getenv(BuildEnv)
	if build == "" {
		build = "2610040306"
	}
	// FileDO's banner is its first output line; a successful restore then prints the true name.
	fmt.Printf("\n2026-10-06 12:00:00 sza@ukr.net %s\n", build)
	os.Exit(run(mode, os.Args[1:]))
}

func run(mode string, args []string) int {
	secret := os.Getenv(SecretVar)
	// The converter passes FileDO's global --no-history first, so no history.json lands in the cwd.
	if len(args) != 7 || args[0] != "--no-history" {
		return 2
	}
	args = args[1:]
	if args[1] != "unsecure" || args[2] != "to" || args[5] != "-y" {
		return 2
	}
	switch args[4] {
	case "pe:" + SecretVar:
		if secret == "" {
			return 2
		}
	case "p:":
		if secret != "" {
			return 2
		}
	default:
		return 2
	}
	for _, a := range args {
		if secret != "" && strings.Contains(a, secret) {
			return 2
		}
	}
	container, dir := args[0], args[3]
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return 5
	}

	// What real FileDO leaks into its error stream; the converter must never forward it.
	leak := func(code int) int {
		fmt.Fprintf(os.Stderr, "fake filedo: %s failed (%s) %s\n", TrueName, secret, container)
		return code
	}
	switch mode {
	case "usage":
		return leak(2)
	case "wrong":
		if secret != RightPassword {
			return leak(3)
		}
		return write(container, dir, TrueName+".txt", txtBody)
	case "damaged":
		return leak(4)
	case "io":
		return leak(5)
	case "unsupported":
		return leak(6)
	case "crash":
		return leak(1)
	case "ok-txt":
		return write(container, dir, TrueName+".txt", txtBody)
	case "ok-png":
		return write(container, dir, TrueName+".png", pngBody())
	case "ok-docx":
		return write(container, dir, TrueName+".docx", []byte("PK\x03\x04 not a converted type"))
	case "ok-nested":
		return write(container, dir, TrueName+".fd-sec", make([]byte, 5632))
	case "ok-folder":
		sub := filepath.Join(dir, TrueName)
		if os.Mkdir(sub, 0o755) != nil {
			return 5
		}
		return write(container, sub, "inside.txt", txtBody)
	}
	return 2
}

var txtBody = []byte("Chapter one\n\nHello from the secret file.\n")

func write(container, dir, name string, body []byte) int {
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, body, 0o600); err != nil {
		return 5
	}
	fmt.Printf("OK %s -> %s (%s, %d B)\n", container, p, name, len(body))
	return 0
}

func pngBody() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 8), G: uint8(y * 8), B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
