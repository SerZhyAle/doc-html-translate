// Package fdsec opens FileDO secret files (.fd-sec) for the converter. It never reads the
// container format: the installed FileDO decrypts (ticket 94, ADR-1), and this package screens
// the file by length, finds FileDO, keeps one private plain copy at a time under the user's
// temp folder, hands the password to FileDO through the child's environment only and maps
// FileDO's exit classes onto typed errors.
//
// Hygiene (FDSEC-BEHAVIOUR section 8.2): nothing in this package forwards FileDO's output,
// and neither the password nor the name sealed inside the container reaches an error, a log
// line or a path another program can see.
package fdsec

import (
	"strings"

	"doc-html-translate/internal/i18n"
)

// Ext is the container extension. It is the only signal the format allows: nothing in the
// bytes identifies a container, so a match means "probably a FileDO secret file".
const Ext = ".fd-sec"

// MinFullBuild is the first released FileDO build that reads every kind of container (suites
// 1, 2 and 3). An older build reports a suite-2 or suite-3 container as damaged or unsupported.
const MinFullBuild = "2609260512"

// IsContainer reports whether ext (with its dot) names a FileDO secret file.
func IsContainer(ext string) bool { return strings.EqualFold(ext, Ext) }

// ScreenLength is the credential-free length screen of the format (FDSEC-FORMAT section 4.5
// rule 5 for suites 1 and 3, section 18.4 for suite 2). A size that fails both cannot be a
// container; a size that passes says nothing more.
func ScreenLength(size int64) bool {
	if size >= 5632 && size%512 == 0 {
		return true
	}
	const frame = 65552
	if size >= 9308 {
		body := size - 40
		k := (body + frame - 1) / frame
		return body-(k-1)*frame >= 17
	}
	return false
}

// Class is the kind of outcome. The three classes FDSEC-BEHAVIOUR section 7 forbids merging
// (Credential, Damaged, Unsupported) stay three here and in every message.
type Class int

const (
	NotFound    Class = iota // FileDO is not installed
	Screened                 // the length screen rules the file out
	Credential               // wrong password, or not a secret file, or altered (one class)
	Damaged                  // FileDO reports damage
	Unsupported              // a kind of container the installed FileDO does not support
	IO                       // FileDO could not read or write
	RefusedCall              // FileDO did not accept the call
	Failed                   // FileDO stopped unexpectedly, or the result was not one file
	Folder                   // the container holds a folder (suite 3)
	Nested                   // the container holds another container
	InnerType                // the document inside is of a type this app does not convert
	NoSource                 // no password source and nobody to ask
	EmptyEnv                 // the named environment variable is unset or empty
	Cancelled                // the user cancelled the password question
)

// Error is a typed failure. Name is the container's visible base name - never a path and
// never a sealed name. Build is the FileDO build that answered, when known. Inner is the
// extension of the document inside, for InnerType only.
type Error struct {
	Class Class
	Name  string
	Build string
	Inner string
}

func (e *Error) Error() string {
	var msg string
	switch e.Class {
	case NotFound:
		msg = i18n.S("%s looks like a FileDO secret file. Opening it needs FileDO, which was not found on this computer. Install it with: winget install SerZhyAle.FileDO - or get FileDO from the Microsoft Store or https://serzhyale.github.io/FileDO/ - then try again. No password was asked and nothing was changed.", e.Name)
	case Screened:
		msg = i18n.S("%s cannot be a FileDO secret file: its size rules it out. It is not a secret file, or it is damaged or cut short.", e.Name)
	case Credential:
		msg = i18n.S("FileDO could not open %s: the password is wrong, or the file is not a FileDO secret file, or it was altered.", e.Name)
	case Damaged:
		msg = i18n.S("FileDO reports %s as damaged or cut short.", e.Name) + e.outdatedSuffix()
	case Unsupported:
		msg = i18n.S("FileDO reports %s as a kind of secret file it does not support.", e.Name) + e.outdatedSuffix()
	case IO:
		msg = i18n.S("FileDO could not read or write a file while opening %s (disk full, no access, or a file in use).", e.Name)
	case RefusedCall:
		msg = i18n.S("FileDO did not accept the call to open %s - update FileDO and try again.", e.Name)
	case Folder:
		msg = i18n.S("%s holds a folder, not one document - open it with FileDO.", e.Name)
	case Nested:
		msg = i18n.S("%s holds another secret file - open the outer one with FileDO first.", e.Name)
	case InnerType:
		msg = i18n.S("%s holds a %s file, which this app does not convert.", e.Name, e.Inner)
	case NoSource:
		msg = i18n.S("No password was given for %s and nobody can be asked here. Run it in a console or from the app window, or name an environment variable that holds the password: -fdsec-password-env NAME", e.Name)
	case EmptyEnv:
		msg = i18n.S("The environment variable %s named by -fdsec-password-env is not set or is empty - nothing was opened.", e.Name)
	case Cancelled:
		msg = i18n.S("Opening %s was cancelled - nothing was converted.", e.Name)
	default:
		msg = i18n.S("FileDO stopped unexpectedly while opening %s.", e.Name)
	}
	return msg
}

// outdatedSuffix names an outdated FileDO as one possible cause of "damaged" and
// "unsupported": a build older than MinFullBuild reports a suite-2 or suite-3 container that
// way.
func (e *Error) outdatedSuffix() string {
	if e.Build != "" && e.Build < MinFullBuild {
		return " " + i18n.S("The installed FileDO (build %s) is older than the first build that reads every kind of secret file - update it: winget upgrade SerZhyAle.FileDO", e.Build)
	}
	return ""
}

// Asker supplies the password. Secret is called with the container's visible base name;
// retry is true when the previous password did not open it. CanRetry says whether asking
// again can produce a different answer (a person can; an environment variable cannot).
// The returned bytes belong to the caller, which clears them.
type Asker interface {
	Secret(name string, retry bool) ([]byte, error)
	CanRetry() bool
}
