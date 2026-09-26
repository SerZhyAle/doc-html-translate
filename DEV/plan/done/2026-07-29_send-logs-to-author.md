# Strategic spec: 2026-07-29_send-logs-to-author - Send logs to the author

**Ticket:** 2026-07-29_send-logs-to-author
**Status:** BlockNeedUserTest (2026-08-11 - all 7 phases built and checked; the four hands-on criteria
in the tactical INDEX's completion gate need a human)
**Priority:** 50
**Date:** 2026-07-29
**Tier:** Moderate
**Tactical plan:** [`2026-07-29_send-logs-to-author/INDEX.md`](2026-07-29_send-logs-to-author/INDEX.md) - 7 phases

> **Scope:** STRATEGIC. Goals, constraints, open questions. No class names, paths, line
> budgets, schema versions, or framework module details.

---

## 1. Problem

When a conversion goes wrong, the user has no way to hand the author any evidence. The run
log exists only while the app is open: the desktop UI streams it into a pane that is wiped at
the start of the next run and gone when the window closes, and the command-line edition
writes it to a console that scrolls away. The only reporting channel today is a plain
"Feedback" mail link, so a bug report is either a hand-selected fragment of a log pane or a
description from memory - and the author usually cannot reproduce the failure without the
exact file, settings and versions the log would have named. The affected area is the app's
diagnostics and the settings surface of the desktop UI.

## 2. Goals

1. Run logs survive the session: after a conversion the log is on disk in a per-user
   location, so it can still be produced an hour later or after a crash.
2. The settings surface gains an "About the program" section that states what the user is
   running (product, version, edition, engine) and carries one button that starts a report.
3. One press of that button produces a single archive of the recent logs plus a short
   environment summary, without the user hunting for files.
4. The user's own mail program opens, addressed to the author, with the subject and a short
   body template already filled in, and the archive is put in front of the user in the same
   press: its folder opens with the file selected and its path goes on the clipboard, so
   attaching it is one drag or one paste.
5. Nothing ever leaves the machine unless the user presses Send in their own mail program.
   The app performs no upload of its own, so the product's "no telemetry" claim stays true.
6. The browser extension, which has no log store to archive, gets the equivalent it can
   honestly offer: a copy-diagnostics action that puts a short report text on the clipboard.
7. The capability is described on every user-facing surface - the readme, the documentation
   pages and the product site - in every locale those surfaces already support.

**Non-goals:**

- Automatic or background crash reporting, telemetry, or any upload from the app itself.
- Sending mail from the app (no SMTP, no credentials, no author-side intake service).
- Putting the source document, the converted output, or any book content in the archive.
- A support ticket system, a reply channel, or any tracking of what was sent.
- Programmatic attachment of the archive to the message (owner decision - see ADR-3).
- A log archive in the browser extension (it has no filesystem log store - the clipboard text
  is the whole extension scope; see ADR-4).

## 3. Wishes and constraints

### 3.1 Owner wishes

- The About section doubles as the place a user reads their version before reporting, so it
  should be worth opening even when nothing is wrong.
- The user can look inside the archive before sending it - trust comes from being able to see
  what is being handed over, not from a promise.
- A way to wipe the stored logs from the same section, for a user who does not want a run
  history on disk at all.
- The report should arrive pre-sorted: a subject line the author can filter on, and enough of
  an environment summary that the first reply is not "which version?".

### 3.2 Hard constraints

- **Platform / versions:** Windows only, matching the app. Must work identically in the
  portable build, the installer build and the packaged Store build - the last one runs from a
  read-only install directory, so both the log store and the archive live in the per-user
  writable location the app already uses for its other state.
- **Performance:** persisting the log must not slow a conversion measurably and must never be
  able to fail it - if the log cannot be written, the conversion still runs and succeeds.
  Building the archive is a user-initiated action and may take a moment, but the UI stays
  responsive and says what it is doing.
- **Data compatibility:** additive only. Existing settings and existing outputs are
  untouched; a first run after the update simply starts a log history that was not there
  before. No stored format changes shape.
- **Localization:** every new user-visible string in the desktop UI **and in the extension's
  options page** ships in all thirteen interface languages both already support; the
  documentation pages and the site follow their existing locale sets.
- **Accessibility:** the About section and its button are keyboard reachable, the button
  announces its result as text (not colour alone), and the fallback instruction is readable
  text rather than an icon.
- **Privacy:** the archive must not contain a secret. The translation API key is stored next
  to the app's other per-user state and must never be copied in, quoted, or logged. Anything
  in the archive that identifies the user (file paths, document names, machine name) is either
  reduced or plainly disclosed before sending - never both hidden and included.

### 3.3 Owner inputs (Approval gate)

- **Related tickets:** none blocking. Touches the same surfaces as
  `2026-07-28_thirteen-ui-languages` (new strings enter
  the thirteen-language set) and is a new row for
  `2026-07-01_cross-edition-parity`.
- **Copy/tone policy:** the button and the prefilled mail body are the author's own voice -
  short, no apology, no support-desk register. The mail body is written in the user's
  interface language, but the environment summary inside the archive stays English so the
  author reads one format.
- **Data compatibility:** additive; no migration.
- **Platform constraints:** the packaged Store build is the gating flavour - the feature is
  not done until it is proven there, not only in the portable build.
- **Localization:** thirteen interface languages for the app's UI strings and for the
  extension's; readme in the three author-proofread languages; site pages in their full
  existing locale set.
- **Validation level:** automated checks for the archive's contents (a secret-shaped string
  never appears, the size cap holds, the archive is well formed) plus one hands-on pass per
  build flavour for the mail hand-off, because how a mail program renders a prefilled message
  is not something a test can assert.
- **Owner sign-off:** **given 2026-07-29 on all three §6 decisions** - mail hand-off is the
  mail-link mechanism plus the reveal-and-clipboard path, with no programmatic attachment
  attempt (ADR-3); the archive carries logs plus the environment summary plus the last run's
  settings, keeping document file names (ADR-5); the extension is in scope with a
  clipboard-only diagnostics action (ADR-4).

## 4. Current architecture context

Logging in this product is a print channel, not a store: the conversion pipeline and every
format reader emit timestamped lines to standard output, and whoever launched the converter
decides what happens to them. On the command line that is the console. In the desktop UI it
is a stream read from the converter process and appended to a log pane in the page - a pane
that is deliberately cleared when a new run starts, and that has no life beyond the window.
Nothing writes a log file, and nothing keeps a run history, so there is no artefact for a
report to be built from. That is the reason this cannot be solved by adding a button alone.

The settings surface has the second half of the gap. The desktop UI is a single page of
grouped, collapsible option sections plus a header byline that already carries a product-site
link and a feedback mail link. There is no section that describes the installation itself, so
there is no natural home for a diagnostics action - and the byline's mail link is exactly the
channel that today produces reports with no evidence attached.

## 5. Proposed approach

Three responsibilities, deliberately separated so each can fail without taking the others
down.

First, **the log becomes an artefact.** The component that emits log lines also appends them
to a per-run file under the app's existing per-user state location, keeping a bounded history
of recent runs. This is a side channel: the console and the UI pane keep behaving exactly as
they do now, and a failure to write the file is swallowed rather than surfaced as a conversion
error. Because the desktop UI drives the converter as a separate process, the converter itself
owns this - which also means a command-line user's failure is captured, not only a UI user's.

Second, **the report becomes a package.** On request, a collector gathers the recent log
files and one generated environment summary - product version, build flavour, operating system
build, interface language, which optional engines are present and reachable, and the settings
that shaped the last run - passes them through a redaction step, and writes one compressed
archive to the per-user location. The redaction step is a hard gate, not a courtesy: the
archive is the only thing that leaves, so the rule that a secret cannot appear in it is
enforced where the archive is built, and asserted by a test rather than by review.

Third, **the hand-off stays in the user's hands, and it is one shape rather than two.** The
app opens the user's default mail program with the author's address, a subject naming product
and version, and a short body template; in the same press it opens the archive's folder with
the file selected and puts the path on the clipboard, and says in one sentence that the archive
must be dragged or pasted into the message. There is deliberately no attempt to attach the file
programmatically: the mail-link mechanism has no attachment field at all, and the platform
interface that does have one depends on which mail program is registered - so an attempt would
succeed for some users, fail silently for others, and leave the UI unable to say truthfully
what happened. One predictable path that works in every build, including the packaged Store
one, is worth more than an attachment that sometimes appears. The message sits unsent in the
user's own mail program until the user sends it, and the app never opens a network connection
of its own for this feature.

The button that starts all this lives in a new About section of the settings surface, which
also displays the version and edition information a reporter is always asked for. The browser
extension gets the same intent in the shape its sandbox allows: an About block on its options
page with a copy-diagnostics action that puts a short report text on the clipboard, next to the
feedback link that already exists.

### 5.1 Pillars / modules

- **Log persistence.** Goal: no run is unreportable after the fact. Requirements: bounded by
  count and total size so it cannot grow without limit; per-run granularity so a report can
  carry the failure and its neighbours; write failures are silent and non-fatal; behaves
  identically under the read-only packaged install.
- **Report packaging.** Goal: one file, safe to send, useful to receive. Requirements: a
  deterministic, self-describing name carrying product, version and timestamp; an English
  environment summary; the last run's settings with every secret field reduced; document file
  names kept but paths below the user's profile shortened; a redaction pass with an automated
  assertion; a size cap chosen against what mail providers accept, with the oldest logs dropped
  first and the user told when something was dropped.
- **Mail hand-off.** Goal: the shortest honest path from "it broke" to "the author has the
  evidence". Requirements: the author's canonical contact address, taken from the single place
  the product already defines it rather than re-typed; prefilled subject and body; the archive
  revealed in its folder with the path on the clipboard in the same action; wording that tells
  the user to attach it and never claims the app did.
- **About section.** Goal: the place a user looks when something is wrong. Requirements:
  version, edition and engine status; the existing links; the report button with visible
  progress and result; a control to open the archive itself for inspection and one to clear the
  stored logs; every string localized.
- **Command-line parity.** Goal: the two editions do not disagree about what a report is. The
  archive-building step is exposed as a command-line action as well, so the same report can be
  produced without the UI and the UI has no logic the engine lacks.
- **Extension diagnostics.** Goal: a report from the browser edition that is worth reading,
  without inventing a log store the sandbox cannot host. Requirements: an About block on the
  options page carrying version and the existing feedback link, plus one action that copies a
  short English report text - extension version, browser and platform, interface language,
  the document format and page count of the last opened document, the active option set, and
  the last error the viewer recorded - to the clipboard; the button confirms in the user's
  language that the text was copied; no document content, no page text, no host list beyond
  what the user configured.
- **Editions and surfaces.** All four ship: command line, desktop UI, packaged Store build
  (archive plus mail hand-off) and browser extension (clipboard text). The difference in shape
  between them is intentional and is recorded in the parity document as such (ADR-4).

### 5.2 Data & event flows

- Conversion run: UI or console -> converter process -> log channel -> console/stream **and**
  per-run log file in the per-user store.
- Report: About section -> local request to the app's own service -> collector reads the log
  store, generates the environment summary, redacts, writes the archive -> result path returned
  to the section.
- Hand-off: About section -> platform mail link with address, subject and body -> the user's
  mail program, message unsent; in the same action the archive's folder is opened with the file
  selected, the path goes on the clipboard, and the instruction is shown.
- Housekeeping: the log store is trimmed to its bounds at the start of each run, and can be
  emptied on request from the About section.
- Extension: options page About block -> viewer's recorded last-document and last-error state
  plus the extension's own version and option set -> report text -> clipboard, with a
  confirmation shown in place.

### 5.3 Extension points

- The set of things the environment summary reports must be cheap to extend as the product
  grows engines and formats, without touching the packaging or hand-off steps.
- The redaction rules are a list, not a hard-coded pass: a future secret or a future
  personal-data field is added in one place and inherited by every report.
- The hand-off is one replaceable step, so a different attachment mechanism (or a future
  non-mail channel) can be substituted without changing what a report is.
- The archive layout is documented as what the author receives, so a future intake helper on
  the author's side can rely on it.

## 6. Open questions / research items

1. **Can the mail program actually be handed an attachment?**
   - **Question:** the standard mail-link mechanism carries address, subject and body but has
     no attachment field, and mail programs ignore attempts to add one. So either a different
     platform mechanism is used, or the file reaches the message by the user's own hand.
   - **Options:** (a) the legacy platform mail interface, which does support an attachment but
     only with a mail program registered for it, and which is a poor fit for the packaged
     Store build; (b) mail link for the message plus reveal-and-clipboard for the file, with
     wording that never claims an attachment the app did not make; (c) attempt (a), fall back
     to (b).
   - **Status:** **Resolved 2026-07-29 - option (b).** No programmatic attachment attempt at
     all. One path, identical in every build, and the UI can state truthfully what it did. See
     ADR-3.
2. **What exactly goes in the archive?**
   - **Question:** logs only, or logs plus the environment summary plus the current settings?
     Do full file paths and document names stay, get shortened to file names, or get reduced
     to their shape?
   - **Options:** minimal (logs, redacted paths) / standard (logs, environment summary,
     settings, file names kept) / full (nothing reduced, contents disclosed before sending).
   - **Status:** **Resolved 2026-07-29 - standard.** Logs plus an English environment summary
     plus the last run's settings; document file names are kept because a log that cannot be
     matched to a document cannot be acted on; paths below the user's profile are shortened;
     the translation API key never appears. See ADR-5.
3. **Is the browser extension in scope?**
   - **Question:** the extension has no log store, so either it is declined or it gets a
     differently-shaped action.
   - **Status:** **Resolved 2026-07-29 - in scope now, clipboard only.** An About block on the
     options page with a copy-diagnostics action. No archive, no mail hand-off, no log store.
     See ADR-4.
4. **How much history, and for how long?**
   - **Question:** the bound on the log store - number of runs, total size, age, or a
     combination - and whether the user is told it exists on first run.
   - **To find out:** typical log size across the corpus, including the noisy long runs (large
     scanned documents with image recognition), which set the upper end.
   - **Status:** Open - a default can be chosen during the tactical plan and tuned later.
5. **Does the report subject need a machine-readable tag?**
   - **Question:** a fixed prefix plus version makes author-side filtering trivial but makes
     the subject less human.
   - **Status:** Open - cosmetic, decide while writing the copy.

## 7. Risks

| Risk | Likelihood | Impact | Mitigation |
|------|:----------:|--------|-----------|
| The user sends the mail without attaching the archive | Medium | The author receives a report with no evidence - the exact problem this feature exists to fix | The instruction is part of the same press, not a later step: folder open with the file selected, path on the clipboard, and the prefilled body itself carries the "attach the archive" line, so an unattached report still reads as incomplete to its own sender |
| A secret or credential ends up in the archive | Low | A user leaks their own translation API key by their own hand | Redaction enforced where the archive is built, plus an automated assertion that no secret-shaped string can appear; the user can inspect the archive first |
| Personal data (paths, document titles, machine name) leaves without the user realising | Medium | Privacy expectation broken, and the product's privacy statement becomes inaccurate | Disclose the archive's contents in the UI and in the documentation; let the user open it before sending; keep the privacy page, the permission list and the store data-safety form saying the same thing |
| The archive is too large to mail | Medium | The message bounces or the user gives up | Cap the archive, drop oldest logs first, say what was dropped, and state the cap in the documentation |
| The packaged Store build cannot write the log store or launch the mail program | Medium | The feature is dead exactly where the least technical users are | Reuse the per-user writable location the app already relies on under that build, and make the Store flavour a required hands-on check rather than an assumed inheritance |
| Persisting logs introduces a new failure path into every conversion | Low | A conversion fails for a reason unrelated to the document | Logging to file is a swallowed side effect - never propagates, never blocks, never retried |
| "No telemetry" reads as untrue once the app can build a report | Low | Store review friction, or a fair accusation of overclaiming | State it precisely everywhere: the app never sends anything; a report is a file the user mails themselves |
| Log files accumulate on disk indefinitely | Low | Slow disk growth on a heavy user's machine | Bound by count and size, trimmed at the start of each run, plus a clear-logs control |
| The extension's clipboard text needs a permission it does not have, or a store reviewer reads the new diagnostics action as data collection | Medium | A rejected extension submission, or a permission added that the privacy page does not explain | Write to the clipboard from the user's own click, which needs no added permission; keep the report text to what the user can already see; state it in the privacy page and the store data-safety answers before submitting |
| The extension's report text and the app's archive drift into two different notions of a report | Medium | The author gets incomparable reports from the two editions and the parity document goes stale | Define the report's field list once, mirror it in the parity document's invariant table, and cross-reference the Go and JS sides there |

## 8. User impact (docs)

New capability, and it needs to be visible on every surface: the desktop app can keep a
history of its recent run logs and, from the new About section in settings, pack them into one
archive and open a pre-addressed message to the author so a bug report carries evidence; the
browser extension gets the same intent as a copy-diagnostics action on its options page.
Documentation must state plainly what the archive contains, that the user attaches it and
presses Send themselves, and that nothing is ever sent automatically. Covered surfaces: the
readme in its author-proofread languages, the documentation pages in their existing locales,
the product site pages in their full locale set, the privacy statement, the store data-safety
answers, and the release changelog entry.

## 9. Architecture decisions (ADR)

**ADR-1: persist logs as run files rather than snapshotting the UI's log pane.**
Decision: the converter appends every line it prints to a bounded per-user run-log store.
Alternatives: capture the desktop UI's log pane on demand; keep an in-memory ring buffer for
the current session only. Why: the pane is empty before the first run of a session, cleared at
the start of each run, and gone after a crash - which is precisely the case a report is needed
for; and the converter, not the UI, is the process that produces the interesting lines, so a
UI-side capture would miss command-line failures entirely.

**ADR-2: the user's own mail program sends the report; the app never does.**
Decision: hand off a prefilled message and let the user press Send. Alternatives: send by mail
from the app with an embedded account; upload to an author-side intake endpoint. Why: no
credentials to ship or leak, no service to run, nothing that could be read as background
telemetry, and the user sees exactly what leaves their machine before it leaves.

**ADR-3: the app never attaches the archive - it hands over the message and the file
separately.** Decision (owner, 2026-07-29): open the mail program with address, subject and
body via the standard mail link, and in the same action reveal the archive with its path on the
clipboard; make no programmatic attachment attempt. Alternatives: the legacy platform mail
interface, which can attach but only with a mail program registered for it; a hybrid that tries
that first and falls back. Why: the standard mail link has no attachment field, and the
attaching interface depends on software the app cannot assume - so a hybrid would work for some
users, fail invisibly for others, and force the UI to describe an outcome it cannot reliably
detect. One path that behaves the same in the portable, installer and packaged Store builds is
worth more than an attachment that sometimes appears, and the cost to the user is one drag.

**ADR-4: the browser extension ships the feature as clipboard text, not as an archive.**
Decision (owner, 2026-07-29): the extension's options page gains an About block with a
copy-diagnostics action; it does not gain a log store, an archive, or a mail hand-off.
Alternatives: decline the extension entirely; build a log store in extension storage. Why: the
browser sandbox has no place to keep a run history worth archiving, and inventing one would add
storage and permissions for little diagnostic gain - but the useful part of a report (version,
browser, format, page count, options, last error) is already in the extension's own state and
costs nothing to hand over. The shape difference between the editions is intentional and is
recorded in the parity document rather than left to be discovered.

**ADR-5: the archive carries logs, the environment summary and the last run's settings, with
document names kept.** Decision (owner, 2026-07-29): keep document file names; shorten paths
below the user's profile; never include a secret. Alternatives: logs only with names reduced to
their shape; nothing reduced at all. Why: the report exists to be acted on, and a log whose
document cannot be identified usually cannot be matched to a reproducible case - while the
user's account name and folder layout add nothing diagnostic and are the part worth dropping.

## 10. Links to other specs

- `2026-07-01_cross-edition-parity` - gains the ADR-4 row.
- `2026-07-28_thirteen-ui-languages` - the locale set the
  new app and extension strings must land in.
- [docs/PARITY.md](../../../docs/PARITY.md) - owns the author's contact address, gains the report
  field list as a shared invariant, and gains the ADR-4 divergence entry.
- `_TEMPLATE_cross-edition.md` - this ticket is a cross-edition
  feature; its edition checklist is carried by the tactical plan.

## 11. Done criteria (strategic)

1. After a conversion finishes - or fails - a log file for that run exists in the per-user
   location and contains the same lines the user saw, and the same is true for a conversion
   started from the command line with no UI involved.
2. A conversion still succeeds when the log store cannot be written to, and the user is not
   shown an error about it.
3. The settings surface has an About section that states version, edition and engine status,
   and it is present and correctly localized in all thirteen interface languages.
4. Pressing the report button produces one archive in the per-user location whose name carries
   product, version and timestamp, within the stated size cap; when logs were dropped to meet
   the cap, the UI says so.
5. The archive contains no secret: an automated check fails if a credential-shaped string can
   reach it, and the translation API key is provably absent.
6. The archive's contents match what the documentation says they are, and the user can open it
   from the About section before sending.
7. Pressing the button opens the default mail program with the author's canonical address, a
   subject naming product and version, and the body template in the user's interface language;
   the message is unsent and its body contains the line telling the user to attach the archive.
8. The same press opens the archive's folder with the file selected and puts the path on the
   clipboard, and the on-screen wording tells the user to attach it - the UI never claims the
   app attached anything.
9. Cancelling at any point sends nothing: no network connection is opened by the app for this
   feature, verified by observation and not only by reading the code.
10. The whole flow is proven by hand in the packaged Store build, not only in the portable
    build.
11. The extension's options page shows an About block with its version and a copy-diagnostics
    action; pressing it puts the report text on the clipboard, confirms in the user's language,
    requires no permission the extension did not already hold, and the text contains no
    document content.
12. The readme, the documentation pages, the site pages and the privacy statement describe the
    feature - both shapes, app and extension - in every locale each of those surfaces supports;
    the store data-safety answers match; the changelog carries its entry.

## 12. Next step

`/spec-tech 2026-07-29_send-logs-to-author` - creates the phased tactical plan. All three
blocking owner decisions were taken on 2026-07-29 (§3.3, §6 items 1-3), so nothing gates it.
