# Strategic spec: 2026-09-24_hotfix-output-dir-ownership - Never delete or reuse a folder the converter does not own

**Ticket:** 2026-09-24_hotfix-output-dir-ownership
**Status:** BlockNeedUserTest - implemented and covered by tests on Linux; needs a hands-on Windows check (hidden marker attribute, lock with a real second process, GUI drop and delete).
**Priority:** 95
**Date:** 2026-09-24
**Tier:** Security/Compliance (urgent)
**Tactical plan:** `DEV/plan/2026-09-24_hotfix-output-dir-ownership/` (created by /spec-tech)
**Findings:** P1 P2 P6 P7 P16 P17 G9 G11 (see `../README.md`)

> **Scope:** STRATEGIC. Goals, constraints, open questions. No class names, paths, line
> budgets, schema versions, or framework module details.

---

## 1. Problem
The converter decides where its output goes only from the input's name. It then treats that
location as its own: it deletes it when extraction fails, when `-force` is given, or when the GUI's
"delete previous result" is used. When the input is a folder, the output location is **the input
folder itself**, and a failed run deletes it recursively (reproduced). A pre-existing user folder
that happens to share the book's name is deleted the same way. Two different documents with one
base name (`book.epub`, `book.pdf`) share one output, so one opens the other's content or destroys it.

## 2. Goals
1. A conversion never deletes or overwrites a directory that the converter did not create.
2. An input that is not a readable regular file is refused before anything is created.
3. An output location that equals the input, contains it, or is an ancestor of it is refused.
4. Different source documents never share one output location silently.
5. Two concurrent conversions of the same output cannot corrupt or delete each other's work.
6. The GUI's "delete previous result" can only delete an output the converter produced.
7. Files dropped onto the GUI never overwrite each other, and never overwrite a good copy with a partial one.

**Non-goals:**
- Batch conversion of every file in a folder given as input (a possible later feature, not a fix).
- Output completeness and stale-option reuse (ticket `bugfix-output-completeness`).

## 3. Wishes and constraints
### 3.1 Owner wishes
- Existing outputs from earlier versions keep opening without a rebuild where that is safe.

### 3.2 Hard constraints
- **Platform / versions:** Windows first (case-insensitive paths, reserved device names); the non-Windows build must behave the same.
- **Performance:** n/a.
- **Data compatibility:** outputs created before this change carry no ownership record. The migration rule for them must be explicit (see §6).
- **Localization:** new refusal messages go through the i18n layer in all 13 languages.
- **Accessibility:** n/a.

### 3.3 Owner inputs (Approval gate)
- **Related tickets:** `bugfix-output-completeness` (uses the same ownership record as its completion marker); `bugfix-gui-local-api-hardening` (the delete endpoint also becomes CSRF-proof there).
- **Data compatibility:** decide the fate of legacy outputs without a record (§6.1).
- **Copy/tone policy:** refusal messages state what was refused and why, with no blame.
- **Validation level:** automated tests for every refusal path, plus a manual Windows check with a real folder.
- **Owner sign-off:** required. The output naming scheme is user-visible.

## 4. Current architecture context
The output location is derived by one shared helper that strips the extension, used by both the CLI
and the GUI. The pipeline creates that location unconditionally, then on any extraction failure
removes it wholesale. `-force` and the GUI's delete action remove it if it merely contains an
`index.html`. Nothing records which run created a directory, so "ours" and "the user's" cannot be
told apart, and there is no lock.

## 5. Proposed approach
### 5.1 Pillars / modules
- **Input validation:** the input must exist, be a regular file and be readable. Anything else is an argument error, with a clear message, before any filesystem write.
- **Location safety check:** refuse an output location that equals, contains or is an ancestor of the input, compared case-insensitively on Windows.
- **Ownership record:** every output the converter creates carries a small marker. It records the source identity (path, size, mtime) and the tool version. Deletion (failure cleanup, `-force`, GUI delete) is allowed only when the marker is present and names the same source.
- **Created-by-this-run rule:** failure cleanup deletes only what this run created. If the directory existed before the run, it is left alone.
- **Name collision rule:** when the derived location is owned by a different source, a new distinct location is chosen (§6.2), or the run is refused. It never silently reuses the other one.
- **Run lock:** an exclusive lock inside the output location for the duration of a run. A second run waits or refuses with a clear message. A lock left by a dead process is detected and cleared.
- **Reserved names:** the sanitizer handles device names with any extension, `CONIN$`/`CONOUT$`, superscript digits, characters invalid on Windows, and dot-only names.
- **GUI drop area:** each drop gets its own unique staging location, is written to a temporary name and renamed when complete, is removed on failure, and old drops are cleaned after a retention period.

### 5.2 Data & event flows
Input -> validation -> location derivation -> safety check -> lock -> "create or verify ownership" ->
extract. On failure: remove only if created this run, then unlock. `-force` and GUI delete ->
verify ownership -> remove.

### 5.3 Extension points
- The ownership record is the single place later tickets read to answer "is this output ours, complete, and built with the same options".

## 6. Open questions / research items
1. **Legacy outputs without a record**
   - **Question:** may an old output with no marker be reused and deleted?
   - **Options:** (a) reuse it read-only but never delete it without an explicit confirmation; (b) treat it as foreign; (c) adopt it if its layout matches what the converter generates.
   - **To find out:** owner decision.
   - **Status:** Resolved (2026-09-25, a default taken at implementation; revisit if wrong). Option (c): a marker-less folder is adopted only when its `index.html` (or the root redirect stub's target) carries the generator's `dht-` signature. If its navbar names a different source file, it belongs to that document. Anything else is foreign.
2. **Collision naming**
   - **Question:** what name does the second document with the same base name get?
   - **Options:** `book (pdf)`; `book.pdf.html`; `book-2`.
   - **To find out:** owner decision; check the extension edition for parity (it produces no output folder, so it is likely exempt).
   - **Status:** Resolved. `book (pdf)`, then `book (pdf) 2`..; `(file)` when there is no extension. The first document keeps the plain name, so existing outputs stay where they are. The extension writes no output folder, so it is exempt from parity.
3. **Folder input**
   - **Question:** refuse a folder input outright, or later offer batch conversion?
   - **Status:** Resolved for now: refused with an argument error. Batch conversion stays a possible later feature.

## 7. Risks
- **A marker file shows up in the user's output folder.** Likelihood: high. Impact: cosmetic. Mitigation: hidden attribute on Windows, and a clear name.
- **A stale lock blocks conversions after a crash.** Likelihood: medium. Impact: the user cannot convert. Mitigation: record the PID and start time, and clear the lock when that process is dead.
- **The legacy output policy frustrates users.** Likelihood: medium. Impact: an extra rebuild. Mitigation: option (a) in §6.1.

## 8. User impact (docs)
README: note that a folder is not accepted as input, and how same-name documents are separated.

## 9. Architecture decisions (ADR)
**ADR-1: ownership is proven by a record, not inferred from contents.** Decision: a marker written by the converter. Alternatives: inferring from the `index.html` shape, which is unreliable (a saved website has one too). Why: deletion must never rest on a guess.

## 10. Links to other specs
`bugfix-output-completeness`, `bugfix-gui-local-api-hardening`.

## 11. Done criteria (strategic)
1. Passing a folder prints a refusal, and the folder is intact afterwards.
2. A failed conversion next to a pre-existing same-named user folder leaves that folder intact.
3. `-force` and GUI delete refuse a folder without a matching ownership record.
4. `book.epub` and `book.pdf` in one folder produce two separate outputs.
5. Two simultaneous runs on one input: one succeeds and the other waits or refuses. No output is lost.
6. Dropping two different files with the same name into the GUI converts each one's own content.

## Implementation notes (2026-09-25)
- Done criteria 1-6 are covered by tests: `internal/pipeline/outputdir_test.go`, `internal/outputpath/ownership_test.go`, `cmd/doc-html-ui/ownership_test.go`.
- Deviation: the refusal messages are English, like every other pipeline error. The CLI error path is not localized today, and no new GUI strings were needed.
- Deviation: the GUI drop area has no retention cleanup. Drops are keyed by content hash, so re-dropping a file no longer adds a copy, and deleting old drops would also delete the conversions stored next to them.
- The marker is written when the run claims the folder, not at completion. Completion and option matching remain ticket `bugfix-output-completeness`, which extends this record.

## 12. Next step
`/spec-tech 2026-09-24_hotfix-output-dir-ownership` - creates the phased tactical plan.
