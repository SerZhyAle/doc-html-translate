# Pointer: FDSEC-BEHAVIOUR, FDSEC-FORMAT

- **Id:** `FDSEC-BEHAVIOUR`, `FDSEC-FORMAT`
- **Version:** 1.5 / 1.3
- **Home:** the shared contracts catalog, `secure-container/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, through the installed FileDO command line - never an implementation of the format (adopted 2026-10-06, ticket 94)
- **Wire carrier:** FileDO's exit-code classes (`FDSEC-BEHAVIOUR` section 7.1); the container bytes are FileDO's alone

This product opens a FileDO secret file (`.fd-sec`) by calling the FileDO installed on the machine
(`unsecure`, the password in the child's environment only) and converting what comes out. It owes the
behaviour contract its reader-side sections and the format contract only the credential-free length screen.

**What this repo owes them**

- `FDSEC-BEHAVIOUR` section 7: the three outcome classes - wrong password or not a secret file or altered,
  damaged, unsupported - are never merged in wording and a wrong password is never called damage; exit
  classes 2 / 3 / 4 / 5 / 6 of section 7.1 are mapped in [`../../internal/fdsec/restore.go`](../../internal/fdsec/restore.go).
- Section 8.1: no-echo prompt in a console, a masked field in the GUI window, one scripted source (a named
  environment variable, `-fdsec-password-env`); asked once per batch and only after every target was screened;
  off a terminal with no source the run fails at once naming the sources and never waits; an empty password is
  accepted and labelled "obfuscation only - no secrecy" where it is typed.
- Section 8.2 and invariant 8: neither the password nor the sealed true name reaches a log, the console, a
  window, a report, a command line or a file name another program can see; FileDO's own output is never
  forwarded (a successful restore prints the true name).
- The hand-off is exactly `--no-history <container> unsecure to <private folder> pe:<VAR> -y` (FileDO's own syntax):
  no inline password, no `del` or `start`, and no `history.json` dropped into the folder the app runs in.
- Section 10: the honest statement - the converted pages are not encrypted - is logged after every success.
- Section 14.1 and the "nothing bulk" rule: nothing recovered from a container is launched, only supported
  document and picture types are converted, and at most one plain copy exists at a time.
- `FDSEC-FORMAT` section 4.5 rule 5 and section 18.4 only: the credential-free length screen
  (`fdsec.ScreenLength`). The container is never read here.
- The plain copy lives in a per-run folder of `doc-html-translate-fdsec` inside the user's temp folder, restricted to
  the user and swept at the next start. This departs from the non-normative reveal precedent of section 14.1
  (a folder of the per-user application-data root) on purpose: a packaged process's new folder there is
  redirected into the package's private store and the installed FileDO would not see what it wrote
  (measured, ticket 94 research note section 4.1).

**Conformance.** The stand-in FileDO tests of [`../../internal/fdsec/`](../../internal/fdsec/) (every exit class,
the retry rule, the absence of the password and the sealed name from every error, one plain copy at a time, the
sweep), [`../../internal/pipeline/container_test.go`](../../internal/pipeline/container_test.go) (the run end to
end, identity, reuse without a question, no output folder after a refusal), the window's side in
[`../../cmd/doc-html-ui/fdsecanswer_test.go`](../../cmd/doc-html-ui/fdsecanswer_test.go), and one run against the real
FileDO in [`../../tests/fdsec_e2e_test.go`](../../tests/fdsec_e2e_test.go) (skips - COULD NOT VERIFY - when
FileDO is not installed). The browser extension edition declines the feature: see
[`../PARITY.md`](../PARITY.md) "Intentional divergences".
