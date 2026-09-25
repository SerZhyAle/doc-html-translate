# The Go and JS editions have drifted on eighteen behaviours

**Status:** Draft
**Priority:** 60
**Date:** 2026-09-26

> Filed by the pre-release audit, [ticket 34](done/34_2026-09-25_full-code-audit-pre-release.md). Finding ids
> refer to its register, [`FINDINGS.md`](done/34_2026-09-25_full-code-audit-pre-release/FINDINGS.md).
> docs/PARITY.md is the authority on which side is right.

## 1. Problem

The same input gives different results in the desktop app and the extension, or docs/PARITY.md says
something the code does not:

- EPUB: **B46** (the extension halves of E6, E12, E14: XHTML parsed as HTML, UTF-8-only decoding, SVG
  cover text lost), **E36** (TOC href resolution), **E37** (root-relative links), **B48** (container
  fallback), **E39** (symlink entries - PARITY is wrong).
- FB2: **B34** (stanza title and subtitle dropped), **B35** (cover and missing-image placeholder).
- TXT: **X34** (form feed and Unicode line separators).
- Markdown: **E26** (nested heading split). HTML input: **E28** (where the language comes from).
- Comics: **B30** (symlink entries), **B31** (TAR name precedence), **B32** (no-signature fallback).
- MOBI: **B33** (a document-supplied link marker survives).
- OCR: **B64** (column-order confidence floor on rescue passes), **B38** (CJK class), **B56**
  (`ocr-screen.js` missing from `configs/parity-map.json`).
- Docs only: **B42**, **E29** (stale constants and names in PARITY.md).

## 2. Goals

For each item: make both editions behave the same, or record the divergence in docs/PARITY.md with its
consequence - never leave it unwritten. Pin every value that can be pinned in `tests/parity_test.go`.

## 3. Constraints

- One ticket, both editions, per the parity process. OCR items follow the `OCR-PIPELINE` contract:
  amend the catalog first if a stage changes.

## 4. Acceptance

- Each item is closed by a paired Go and JS test on one fixture, or by a PARITY.md divergence entry.
