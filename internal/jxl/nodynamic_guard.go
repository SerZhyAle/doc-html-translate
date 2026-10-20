//go:build !nodynamic

package jxl

// Without the nodynamic build tag, github.com/gen2brain/jpegxl asks Windows to load libjxl.dll by bare
// name in its package init, before main: a libjxl.dll in the current directory or anywhere on PATH is
// mapped into every start of the app, and replaces the pinned WASM decoder (ticket 115). The tag keeps
// the WASM decoder only. This file refuses a build that forgot it, so a missed path fails here instead
// of shipping; the scripts set it for every go command through scripts/lib/gobuildtags.ps1.
const _ int = "build with -tags nodynamic (ticket 115)"
