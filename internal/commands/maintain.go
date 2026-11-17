package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"doc-html-translate/internal/i18n"
	"doc-html-translate/internal/ocr"
	"doc-html-translate/internal/readiness"
	"doc-html-translate/internal/report"
	"doc-html-translate/internal/windowsreg"
)

// The maintenance group (ticket E0147): `ocr download`, `report build`, `integration add`, `default-handler set` and
// `default-handler release`, the verbs of the five flags -ocr-download, -report, -register-openwith, -register and
// -unregister. The flags stay as they are (APP-COMMANDS rule 2) and call the same functions of `ocr`, `report` and
// `windowsreg` the verbs call here; what differs is only what rule 4 asks of a verb: it never prompts, offers Settings or
// waits for Enter, so none of the five reads standard input.

// ReasonPackaged is the product's own reason token (rule 12 lets a product add one): the registry verbs are refused under a
// Store package, which declares the associations itself and virtualizes the registry. It goes with exit 11.
const ReasonPackaged = "packaged"

// ReasonExists is APP-COMMANDS 0.3 item V.3's rule-12 reason token for an existing --out target without --overwrite (exit 11).
const ReasonExists = "exists"

// ExitNoSpace is rule 11's exit 13: the archive could not be written for lack of space.
const ExitNoSpace = 13

// maintainGroup is the set of verbs served by doc-html-translate.exe alone (Env.Maintain): the registry functions write the
// running executable as the handler, and the five flags exist only in that program. The launcher refuses them with the
// sentence that says which program runs them (assumption W2).
var maintainGroup = map[string]bool{
	"ocr download": true, "report build": true, "integration add": true, "default-handler set": true, "default-handler release": true,
}

// effects is what the verbs do to the machine. The tests replace it, so no test of this package writes the registry,
// reaches the network or fills the user's report folder.
type effects struct {
	ensureInstalled     func(ctx context.Context, code string) (fetched bool, err error)
	buildReport         func(opts report.BuildOptions) (path string, dropped int, err error)
	registerOpenWith    func() ([]string, error)
	registerContextMenu func() ([]string, error)
	registerHandler     func() (windowsreg.Registration, error)
	unregister          func() ([]string, error)
	handlerStatus       func() windowsreg.Status
	packaged            func() bool
}

var fx = effects{
	ensureInstalled: func(ctx context.Context, code string) (bool, error) {
		return ocr.EnsureInstalled(ctx, code, nil)
	},
	buildReport:         report.Build,
	registerOpenWith:    windowsreg.RegisterOpenWith,
	registerContextMenu: windowsreg.RegisterContextMenu,
	registerHandler:     windowsreg.RegisterHandler,
	unregister:          windowsreg.Unregister,
	handlerStatus:       windowsreg.HandlerStatus,
	packaged:            readiness.Packaged,
}

// The sentences the group reuses from the console's own prose (internal/i18n has all of them in every language).
const (
	msgOpenWithFailed = `  Could not add the "Open with" entry: %v`
	msgMenuFailed     = `  Could not add the right-click "Convert to HTML" entry: %v`
	msgSetFailed      = "  Could not set as default handler: %v"
)

// say prints one sentence on standard error and returns code. The sentence is an internal/i18n key; the leading spaces some
// of the console's keys carry are dropped.
func (f *front) say(code int, key string, args ...any) int {
	_, stderr := f.streams()
	fmt.Fprintf(stderr, "Error: %s\n", strings.TrimSpace(i18n.T(f.lang(), key, args...)))
	return code
}

// usageText is a refusal whose sentence is already in the interface language: exit 10, nothing was done.
func (f *front) usageText(msg string) int {
	_, stderr := f.streams()
	fmt.Fprintf(stderr, "Error: %s\n", msg)
	return ExitUsage
}

// notNil is an empty list as `[]` in the data, never `null`.
func notNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// list is a list as one readable value: the words separated by a space, or `none`.
func list(s []string) string {
	if len(s) == 0 {
		return "none"
	}
	return strings.Join(s, " ")
}

// refusePackaged is rule 12's `packaged`: under a Store package the three registry verbs change nothing and say so.
func (f *front) refusePackaged(verb string) (int, bool) {
	if !fx.packaged() {
		return 0, false
	}
	return f.say(ExitNotFound, msgPackaged, verb), true
}

// OCRDownloadData is the data of `ocr download`.
type OCRDownloadData struct {
	Lang       string `json:"lang"`
	Installed  bool   `json:"installed"`
	Downloaded bool   `json:"downloaded"`
}

// ocrDownload serves `ocr download`: the pack is in the per-user folder when the verb returns 0. A pack that is already
// there and verified is not fetched again (assumption W7).
func (f *front) ocrDownload(inv *Invocation) int {
	asJSON, uerr := inv.Bool("json")
	if uerr != nil {
		return f.refuse(uerr)
	}
	code, uerr := inv.String("lang")
	if uerr != nil {
		return f.refuse(uerr)
	}
	if err := ocr.CheckLang(code); err != nil {
		// A code outside the catalogue is a wrong value: the sentence names the catalogue, and nothing was fetched.
		return f.usageText(ocr.ErrorText(err, f.lang()))
	}
	fetched, err := fx.ensureInstalled(context.Background(), code)
	if err != nil {
		return f.failText(ocr.ErrorText(err, f.lang()))
	}
	d := OCRDownloadData{Lang: code, Installed: true, Downloaded: fetched}
	return f.respond(inv, asJSON, encode(d), fmt.Sprintf("lang: %s\ninstalled: %v\ndownloaded: %v\n", d.Lang, d.Installed, d.Downloaded))
}

// ReportBuildData is the data of `report build`.
type ReportBuildData struct {
	Path        string `json:"path"`
	DroppedLogs int    `json:"droppedLogs"`
}

// reportBuild serves `report build`: the archive of the recent run logs and the environment summary. Everything in it
// passes the report's own redaction, so no document name or key is in it, as for -report.
func (f *front) reportBuild(inv *Invocation) int {
	asJSON, uerr := inv.Bool("json")
	if uerr != nil {
		return f.refuse(uerr)
	}
	out, uerr := inv.String("out")
	if uerr != nil {
		return f.refuse(uerr)
	}
	overwrite, uerr := inv.Bool("overwrite")
	if uerr != nil {
		return f.refuse(uerr)
	}
	path, dropped, err := fx.buildReport(report.BuildOptions{
		AppVersion: f.env.Version,
		Packaged:   fx.packaged(),
		At:         time.Now(),
		Out:        out,
		Overwrite:  overwrite,
	})
	switch {
	case err == nil:
	case errors.Is(err, report.ErrOutExists):
		return f.say(ExitNotFound, msgOutExists, "--out", "--overwrite")
	case errors.Is(err, report.ErrOutFolderMissing):
		return f.say(ExitNotFound, msgOutFolderMissing, "--out")
	case report.IsNoSpace(err):
		return f.say(ExitNoSpace, msgNoSpace)
	default:
		return f.failText(err.Error())
	}
	d := ReportBuildData{Path: path, DroppedLogs: dropped}
	return f.respond(inv, asJSON, encode(d), fmt.Sprintf("path: %s\ndroppedLogs: %d\n", d.Path, d.DroppedLogs))
}

// IntegrationAddData is the data of `integration add`.
type IntegrationAddData struct {
	Extensions []string `json:"extensions"`
}

// integrationAdd serves `integration add`: "Open with" and the right-click entry, per user, no default handler change. The
// order and the stop on a failed "Open with" are the flag's (-register-openwith); a failed right-click entry after it is
// exit 1 here, where the flag only warns (assumption W3).
func (f *front) integrationAdd(inv *Invocation) int {
	asJSON, uerr := inv.Bool("json")
	if uerr != nil {
		return f.refuse(uerr)
	}
	if code, refused := f.refusePackaged("integration add"); refused {
		return code
	}
	advertised, err := fx.registerOpenWith()
	if err != nil {
		return f.say(ExitFailed, msgOpenWithFailed, err)
	}
	if _, err := fx.registerContextMenu(); err != nil {
		return f.say(ExitFailed, msgMenuFailed, err)
	}
	d := IntegrationAddData{Extensions: notNil(advertised)}
	return f.respond(inv, asJSON, encode(d), "extensions: "+list(d.Extensions)+"\n")
}

// DefaultHandlerSetData is the data of `default-handler set`.
type DefaultHandlerSetData struct {
	Complete bool     `json:"complete"`
	Default  []string `json:"default"`
	Blocked  []string `json:"blocked"`
	Unknown  []string `json:"unknown"`
	Failed   []string `json:"failed"`
}

// defaultHandlerSet serves `default-handler set`: the product becomes the default handler where Windows lets it, and the
// right-click entry is added. Exit 0 whenever the keys were written; `complete` and `blocked` say what Windows did with
// them (assumption W12). No Settings offer and no pause: those belong to the flag.
func (f *front) defaultHandlerSet(inv *Invocation) int {
	asJSON, uerr := inv.Bool("json")
	if uerr != nil {
		return f.refuse(uerr)
	}
	if code, refused := f.refusePackaged("default-handler set"); refused {
		return code
	}
	reg, err := fx.registerHandler()
	if err != nil {
		return f.say(ExitFailed, msgSetFailed, err)
	}
	if _, err := fx.registerContextMenu(); err != nil {
		return f.say(ExitFailed, msgMenuFailed, err)
	}
	d := DefaultHandlerSetData{
		Complete: reg.Complete(), Default: notNil(reg.Default), Blocked: notNil(reg.Blocked),
		Unknown: notNil(reg.Unknown), Failed: notNil(reg.Failed),
	}
	readable := fmt.Sprintf("complete: %v\ndefault: %s\nblocked: %s\nunknown: %s\nfailed: %s\n",
		d.Complete, list(d.Default), list(d.Blocked), list(d.Unknown), list(d.Failed))
	return f.respond(inv, asJSON, encode(d), readable)
}

// DefaultHandlerReleaseData is the data of `default-handler release`.
type DefaultHandlerReleaseData struct {
	Released     []string `json:"released"`
	StillDefault []string `json:"stillDefault"`
}

// defaultHandlerRelease serves `default-handler release`: the association is handed back and the right-click entry and
// "Open with" stay. Nothing to release is exit 0.
func (f *front) defaultHandlerRelease(inv *Invocation) int {
	asJSON, uerr := inv.Bool("json")
	if uerr != nil {
		return f.refuse(uerr)
	}
	if code, refused := f.refusePackaged("default-handler release"); refused {
		return code
	}
	released, err := fx.unregister()
	if err != nil {
		return f.failText(err.Error())
	}
	d := DefaultHandlerReleaseData{Released: notNil(released), StillDefault: notNil(fx.handlerStatus().Default)}
	return f.respond(inv, asJSON, encode(d), fmt.Sprintf("released: %s\nstillDefault: %s\n", list(d.Released), list(d.StillDefault)))
}
