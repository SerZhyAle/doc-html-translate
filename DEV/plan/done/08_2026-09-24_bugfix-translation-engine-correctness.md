# Strategic spec: 08_2026-09-24_bugfix-translation-engine-correctness - Correct, bounded and secret-safe translation

**Ticket:** 08_2026-09-24_bugfix-translation-engine-correctness
**Status:** BlockNeedUserTest - implemented and covered by HTTP-stub tests on Linux; needs a hands-on run against the real Google API (done criteria 1 and 2: unplugged network with a real key, and "It's" into German) and a real Ollama model (criterion 6), plus the unattended `-max-cost` run on Windows without a MessageBox.
**Priority:** 85
**Date:** 2026-09-24
**Tier:** Moderate
**Tactical plan:** `DEV/plan/done/08_2026-09-24_bugfix-translation-engine-correctness/` (created by /spec-tech)
**Findings:** T1 T2 T3 T4 T6 T7 T8 (= P11) T9 P10 P20 (see the [findings register](../../research/audit_2026-09-24/README.md))

> **Scope:** STRATEGIC.

---

## 1. Problem
Paid translation has several faults that cost money or corrupt text:
- **Key leak:** the API key travels in the request URL, so any network error prints it to the console and writes it into the persistent run log.
- **Corrupted text:** plain text is sent as HTML, and the entity-escaped reply is stored as text. Apostrophes show up as `&#39;`, and `<` in prose is mangled.
- **No segment cap:** batches have no limit on segment count. A page with many short strings exceeds the provider's per-request limit and stops the whole translation.
- **Unchecked reply length:** a short reply either crashes the cache or shifts every later translation onto the wrong text.
- **Weak rate-limit handling:** retries give up after about 7 seconds, and some rate-limit replies are never retried.
- **Cost guard gaps:** the guard is skipped for small documents, ignores the title and TOC, counts bytes instead of characters, and treats an invalid limit as "no limit".
- **Blocking dialog:** the cost confirmation blocks any unattended run.
- **Ollama:** multi-line text is truncated to its first line.

## 2. Goals
1. The API key never appears in any console line, log, report or error message.
2. Translated text reads correctly: entities are decoded, and markup characters in the source survive intact.
3. Every request respects the provider's per-request segment and size limits.
4. A reply whose shape does not match the request is an error, never a crash or a silent misplacement.
5. Transient throttling is waited out within a bounded time, honouring the provider's advice.
6. The cost estimate counts every billable character (pages, title, TOC) in the units the provider bills. It applies at any document size, and an invalid limit is rejected at startup.
7. An unattended run can pre-approve spending up to a limit, without a dialog.
8. Ollama translates multi-line segments completely, and a malformed reply cannot overwrite another segment.

**Non-goals:**
- Choosing a new engine or model.
- Partial-output handling (ticket `bugfix-output-completeness`).

## 3. Wishes and constraints
### 3.2 Hard constraints
- **Platform / versions:** Google Cloud Translation v2 REST; local Ollama.
- **Performance:** no more than a small increase in request count from the segment cap.
- **Data compatibility:** the translation cache on disk must stay readable, or be versioned.
- **Localization:** the cost dialog and error text stay in all 13 languages.

### 3.3 Owner inputs (Approval gate)
- **Related tickets:** `bugfix-output-completeness`, `chore-hygiene-and-test-gaps` (flag validation).
- **Performance budget:** a worst-case retry wait budget per request (§6.2).
- **Copy/tone policy:** the cost dialog wording for the pre-approved case.
- **Validation level:** HTTP-stub tests for each fault (a key in an error, an entity reply, 200 segments, a short reply, 429 with `Retry-After`, 403 rateLimitExceeded, a multi-line Ollama segment).
- **Owner sign-off:** required. `-max-cost` semantics are a public flag, and the change is limited to making it stricter.

## 4. Current architecture context
The Google client batches by bytes, puts the key in the query, retries three times on 429/5xx
only, and copies the reply array unchecked. A cache layer maps unique strings to replies by index.
The pipeline estimates cost from page segments only, inside a size threshold, and asks via a
modal dialog. The Ollama client builds a numbered prompt and parses numbered lines with a
single-line pattern.

## 5. Proposed approach
### 5.1 Pillars / modules
- **Credential transport:** the key goes in a request header. Any transport error is scrubbed of the URL before it is wrapped. The run log gets the same redaction the report archive already applies.
- **Text-mode contract:** text is either sent as plain text, or escaped before sending and unescaped on return. Pick one (§6.1).
- **Request shaping:** batches are bounded by segment count and by size.
- **Reply validation:** the count and shape are checked per batch; a mismatch is a retryable or fatal error with context.
- **Throttling policy:** honour `Retry-After`, capped exponential backoff with jitter, and treat rate-limit 403s as retryable.
- **Cost model:** a character-accurate estimate over every translated string. It applies whenever a limit is set, and the limit is validated at parse time.
- **Unattended approval:** within a user-set limit no dialog appears. An explicit approval flag, or a "limit set means approved" rule, is decided by the owner.
- **Ollama framing:** segments are normalized to one line, or a structured output format is used; the first answer per index wins, and out-of-range indices are ignored. A separate timeout covers the cold model load, and requests are cancellable.

### 5.2 Data & event flows
Segments -> cache lookup -> request shaper -> client (key in header) -> reply validator -> cache ->
DOM. The cost model reads the same segment set, plus title and TOC, before the first request.

## 6. Open questions / research items
1. **Plain-text vs escaped HTML**
   - **Question:** plain-text mode loses nothing for text nodes; does the provider translate identically in both modes?
   - **To find out:** a side-by-side sample on 50 real segments.
   - **Status:** Decided: keep `format=html`; each segment is HTML-escaped before sending and the reply is `html.UnescapeString`-ed. The round trip is deterministic: `It's` never shows `&#39;`, and `a<b` survives. The side-by-side quality sample was not needed for this choice, since the provider sees the same mode as before.
2. **Retry budget**
   - **Question:** what is the maximum total wait per request before giving up?
   - **Status:** Decided: 60 s of waiting per request at most. `Retry-After` is honoured (seconds or an HTTP date); otherwise capped exponential backoff (1 s doubling, 16 s cap) with jitter. Retried: 429, 5xx, a transport error, and 403 with reason `rateLimitExceeded` / `userRateLimitExceeded`. A `Retry-After` past the budget gives up at once. The sleep and the clock are injectable for tests.
3. **Unattended approval shape**
   - **Question:** new flag vs "`-max-cost` set implies approval up to it"?
   - **Status:** Decided: an explicitly set `-max-cost` (above 0) is the pre-approval - an estimate within it translates with no dialog; one above it is refused as before. Without a limit the confirmation behaves as today (asked above 1000 characters). The dialog wording is unchanged; the README documents the unattended run.
4. **Other decisions**
   - **Status:** Decided: the key goes in the `X-Goog-Api-Key` header, transport errors are reduced to their cause (no URL) and scrubbed of the key before wrapping, and the run log gets the report archive's redaction (`report.Redact`) as it is written. Requests are bounded to 128 segments (v2 limit) and 5000 characters. A reply of the wrong length is retried once, then fails with context; the cache refuses a wrong-length reply instead of panicking. The cost counts characters (runes) over every translated string including the title and TOC labels, and applies at any size whenever a limit is set; a negative, NaN or infinite `-max-cost` is an argument error at parse time (exit 1). Ollama sends a segment containing a line break on its own (no numbered framing), the first answer per number wins, out-of-range numbers are ignored, the first request (cold model load) gets a 15-minute timeout and later ones 5 minutes, and every request takes the run's context. The cache key is unchanged (`src:dst:text`, in memory only).

## 7. Risks
- **Plain-text mode changes translation quality.** Likelihood: low. Impact: noticeably different wording. Mitigation: the §6.1 sample.
- **The cache format changes and invalidates old caches.** Likelihood: medium. Impact: extra paid re-translation. Mitigation: keep the cache key stable, or version the cache and migrate it.

## 8. User impact (docs)
README: `-max-cost` counts characters including title and TOC, and how to run unattended.

## 9. Architecture decisions (ADR)
**ADR-1: secrets never enter URLs.** Why: URLs are logged by every layer (errors, proxies, crash dumps).

## 10. Links to other specs
`bugfix-output-completeness`, `chore-hygiene-and-test-gaps`.

## 11. Done criteria (strategic)
1. With the network unplugged, no output or log line contains the key.
2. "It's" translated into German shows no `&#39;`.
3. A page with 300 short strings translates fully.
4. `-max-cost 0.01` refuses a 1 000-character Cyrillic book only if its true character cost exceeds $0.01.
5. `-max-cost -1` is rejected at startup.
6. An Ollama segment with two lines comes back with both lines translated.

## 12. Next step
`/spec-tech 08_2026-09-24_bugfix-translation-engine-correctness`

## Implementation

- **Google client** (`internal/translator/google.go`, `retry.go`): key in the `X-Goog-Api-Key` header; `transportCause` drops the `*url.Error` wrapper and `scrub` removes the key from any provider text (T1). Segments are HTML-escaped out and unescaped back (T2). `batchTexts` bounds batches by 128 segments and 5000 characters (T3). Each reply's count is checked; a mismatch is retried once, then an error (T4). `retryPolicy` implements the §6.2 budget with an injectable sleep, clock and jitter (T6).
- **Cache** (`internal/translator/cache.go`): a reply of the wrong length is an error, never a panic or a shifted result (T4).
- **Run log** (`internal/logging`): `SetRunLogFilter`; `cmd/doc-html-translate` installs `report.Redact` before the log starts (T1).
- **Cost** (`internal/pipeline/cost.go`): `billableChars` counts runes over pages, title and TOC labels; a set limit is enforced at any size and pre-approves (no dialog) within it; without a limit the dialog asks above 1000 characters as before (P10, P11/T8). `internal/config`: `-max-cost` negative, NaN or Inf is refused (P20).
- **Ollama** (`internal/translator/ollama.go`): multi-line segments are sent singly, first answer per number wins, out-of-range numbers are ignored (T7); per-request timeouts with a longer one until the model has answered once, and every request carries the run's context (T9).
- **Tests:** key only in the header and never in an error (`TestAPIKeyTravelsInHeaderOnly`, `TestNetworkErrorNeverShowsKey`, `TestErrorBodyEchoingKeyIsScrubbed`, `TestRunLogFilterRewritesLogOnly`); entity reply (`TestEntityRoundTrip`); 300 segments (`TestThreeHundredSegments`, `TestBatchTextsBounds`); short reply (`TestShortReplyIsAnError`, `TestShortReplyRecoveredOnRetry`, `TestCacheRejectsWrongLength`); 429 with `Retry-After` (`TestRetryHonoursRetryAfterSeconds`, `TestRetryAfterHTTPDate`, `TestRetryAfterBeyondBudgetGivesUpAtOnce`, `TestRetryBudgetIsBounded`); 403 (`TestForbiddenRateLimitIsRetried`, `TestForbiddenRefusalIsNotRetried`); multi-line and `1984.` (`TestOllamaMultiLineSegmentKeepsEveryLine`, `TestParseNumberedFirstAnswerWins`, `TestOllama1984LineKeepsItsSlot`, `TestOllamaLoadTimeoutOnlyUntilReady`); cost (`TestMaxCostCountsCharactersNotBytes`, `TestNoLimitStillAsks`, `TestBillableCharsIncludeTitleAndTOC`); `-max-cost` values (`TestParseArgsRejectsInvalidMaxCost`).
- **Done criteria:** 3, 4 and 5 are proven by the tests above; 1, 2 and 6 are proven against stubs and need the hands-on check named in the status line.
- **Deviation:** the `-max-cost` refusal is an English argument error like every other `ParseArgs` error - the interface language is resolved only after the arguments are parsed.
- **Docs:** README (EN/RU/UK) `-max-cost` row and behaviour notes (characters incl. title and TOC, unattended runs, the key in a header); AGENTS.md cost line. docs/PARITY.md is unchanged: the extension has no paid engine or cost formula.
