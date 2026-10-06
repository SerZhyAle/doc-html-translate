package fdsec

import (
	"os"
	"os/exec"
	"path/filepath"
)

// OverrideEnv names an explicit full path to FileDO. It is used as given: no identity check.
const OverrideEnv = "DOCHT_FILEDO"

// lookPath is replaceable so a test does not depend on what is installed on the machine.
var lookPath = exec.LookPath

// Locate finds the installed FileDO: an explicit override variable, then PATH, then the folders
// the install channels use when PATH has not been refreshed for this process yet - the setup
// installer's folder, the winget link folder and the Store execution-alias folder. Where the
// executable carries a version resource its product name must be FileDO; the Store alias has
// none and is accepted. There is deliberately no version gate: FileDO's exit class decides.
func Locate() (string, error) {
	if p := os.Getenv(OverrideEnv); p != "" {
		if _, err := os.Stat(p); err != nil {
			return "", &Error{Class: NotFound}
		}
		return p, nil
	}
	if p, err := lookPath("filedo"); err == nil && isFileDO(p) {
		return p, nil
	}
	for _, p := range knownPaths() {
		if _, err := os.Lstat(p); err == nil && isFileDO(p) {
			return p, nil
		}
	}
	return "", &Error{Class: NotFound}
}

// Available reports whether Locate finds FileDO.
func Available() bool {
	_, err := Locate()
	return err == nil
}

// knownPaths lists the per-channel install locations, in lookup order.
func knownPaths() []string {
	var out []string
	if pf := os.Getenv("ProgramFiles"); pf != "" {
		out = append(out, filepath.Join(pf, "FileDO", "filedo.exe"))
	}
	if la := os.Getenv("LOCALAPPDATA"); la != "" {
		out = append(out,
			filepath.Join(la, `Microsoft\WinGet\Links`, "filedo.exe"),
			filepath.Join(la, `Microsoft\WindowsApps`, "filedo.exe"))
	}
	return out
}
