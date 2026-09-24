# Flags written after the input path are silently ignored

**Status:** Implemented 2026-08-11 - both shapes landed (operands permute, extra operands refused);
measured against a fresh build, see "What landed".
**Priority:** 41
**Date:** 2026-08-11

> Cross-edition feature ticket. One feature = one ticket covering every edition.
> Read [`docs/PARITY.md`](../../../docs/PARITY.md) before starting; update it when a shared invariant moves.

## What / why

Measured 2026-08-11 against the freshly built binary while preparing a manual OCR test. This command:

```
doc-html-translate image.png -folder temp/testkit/out -ocr -notranslate -force
```

wrote its output **next to the input image**, not into the named folder. `-folder` was ignored, and
so were `-notranslate` and `-force`. Nothing warned. The run printed `Done.` and exited 0.

The cause is standard Go flag parsing: it stops at the first non-flag argument, so everything after
the file name is collected as trailing operands. The app reads the first operand as the input file
and never looks at the rest - they are dropped without a word.

Damage, worst first:

1. **An explicit spend cap can disappear.** `-max-cost` after the path is dropped, so
   `doc-html-translate -google book.epub -max-cost 5` runs with no cap. The interactive cost dialog
   still fires above 1000 characters, so this is not a silent bill - but the one guard the user typed
   on purpose is gone, and a user who sets a cap is exactly the user who will click through a dialog.
2. **Output lands somewhere else.** The book appears beside the source instead of in the named
   folder. When the source sits in a synced or read-only directory that is a surprise, and the user
   goes looking where they asked for it and finds nothing.
3. **Every switch becomes a coin flip.** `-ocr`, `-force`, `-multipage`, `-ocr-lang`, `-dst`,
   `-notranslate` - all silently ineffective in that position, and each failure looks like the
   feature not working rather than the argument not arriving.

Why this matters more here than in a developer tool: the audience is not CLI-native. This is a
right-click / GUI product whose command line is reached through documentation examples, and the habit
those users bring from everyday tools - `command file --flag` - is exactly the broken order. The GUI
is unaffected because it builds its own argument vector, which is also why this went unnoticed for so
long.

Two candidate shapes, to be decided in the tactical pass and not here:

- **Accept flags anywhere.** Hoist operands to the end before parsing. This is what users expect and
  it breaks no invocation that works today, but it *is* a change of public CLI semantics for the
  pathological case of a file literally named like a flag - which CLAUDE.md says not to do casually.
- **Refuse loudly.** If a trailing operand starts with `-`, exit with a named error that prints the
  corrected command line. Smaller, cannot surprise anyone, and turns a silently wrong result into an
  instruction.

The refusal is worth having under either choice: even with hoisting, a genuine typo should be named
rather than swallowed.

## What landed (2026-08-11)

**Both shapes, because they answer different halves of the defect.**

*Hoisting*, in `config.permuteArgs`: operands move behind the flags before `flag.Parse` sees them, so
the two orders are one order. Arity comes from the FlagSet itself rather than a guess, so `-split 5000
book.epub` keeps its value and `-ocr book.epub` does not swallow the path; an undefined flag consumes
nothing and reaches Parse, whose error for it is the one worth showing; `--` still ends flag parsing,
which is how a file named like a flag is passed. Flags-first invocations permute to themselves, so no
working command line changed meaning - which is what keeps this inside "don't change public CLI flag
semantics": the only invocations whose meaning moves are the ones that were silently broken.

*Refusal*, answering this ticket's own open question: a second operand is now named and refused
instead of dropped. It is nearly always an unquoted Windows path with a space in it, and dropping it
turned a quoting mistake into a "file not found" about a path the user never typed. Known trade,
recorded rather than discovered later: dragging several files onto the exe used to convert the first
one silently and now refuses the batch with an explanation. Converting one of N without saying so was
not a feature worth keeping; real batch conversion is a separate ask.

The usage block now opens with a synopsis stating that both orders work, so `-h` says what the parser
does.

Measured against a freshly built binary, not reasoned about:

- `doc-html-translate image.png -folder temp/verify/out -ocr -notranslate -noopen -force` puts the
  output in the named folder and leaves nothing beside the source. Before: the reverse of both.
- `-ocr "C:\My" "Book.epub"` exits 1 with `unexpected extra argument "Book.epub" after "C:\My"` and
  the quoting hint. (`%q` was replaced by explicit quotes around `%s`: every path here is a Windows
  path, and `%q` doubled the backslashes into something the user never typed.)
- The GUI's own argument vector is now run through the real parser by
  `TestAssembledArgsSurviveTheCLIParser` in `cmd/doc-html-ui`, so a future parser change that drops a
  flag fails in the suite instead of in someone's conversion.

## Edition parity checklist

| Edition | Status | Notes / rationale |
|---|---|---|
| CLI (`doc-html-translate`) | `[x]` | `permuteArgs` + the extra-operand refusal, pinned by six tests in `internal/config` |
| GUI (`doc-html-ui`) | `[x]` | unchanged by design (it builds its own argv, flags first) and now pinned: its assembled vector is parsed by the real parser in `TestAssembledArgsSurviveTheCLIParser` |
| MSIX Store app | `[x]` | inherits the GUI; nothing edition-specific in the parser |
| Browser extension | Not applicable | no command line exists on that side |
| Website / docs | `[x]` | checked, not assumed: every example in `README*.md` and `docs*.html` is flags-first (`-google "book.epub"`, `-folder "D:\out" "book.pdf"`, ..), which stays correct. No prose change - the fix removes a footgun, it does not add a documented feature |

## Shared invariants touched

None. Argument parsing is desktop-only and has no twin in `docs/PARITY.md`.

## Cross-references

- [`internal/config/flags.go`](../../../internal/config/flags.go) - `ParseArgs`, and the `fs.Args()` call
  that takes only the first operand.
- [`cmd/doc-html-translate/main.go`](../../../cmd/doc-html-translate/main.go) - the caller.
- [`cmd/doc-html-ui/main.go`](../../../cmd/doc-html-ui/main.go) - the GUI's argv construction, which must
  keep working under any change here.
- [`internal/windowsreg`](../../../internal/windowsreg) - the registered right-click / "Open with"
  command templates, which encode an argument order of their own.

## Done criteria

- [x] A command with flags after the path either honours them or fails with a named error - decided,
      implemented, and pinned by a test that states the chosen behaviour.
- [x] `-h` output and every documented example agree with the implemented order.
- [x] The GUI still converts, exercised rather than reasoned about - its own argument vector is built
      and parsed in a test, which is the coupling a manual click would have proved less durably.
- [x] The registered shell commands still work: the templates are `"<exe>" "%1"`
      (`register_windows.go`), one quoted operand and no flags, so they parse identically before and
      after - and an entry written by an older build still works, because flags-first is unchanged.
- [x] `./scripts/test.ps1` green; changelog entry in `DEV/CHANGELOG.md`.

## Open questions

- ~~Do the registry handler templates, the installer's context-menu entries, or any script in this
  repo depend on the current operand position?~~ Answered: no. All three registration paths write
  `"<exe>" "%1"`, and the GUI builds flags-first.
- ~~Should an unknown trailing operand (a second file name) also be refused?~~ Answered: yes, refused
  with a named error. See "What landed" for the drag-several-files trade.
- Still open, and deliberately not taken here: real batch conversion. Refusing several documents is
  honest, but converting them is what a user dragging five files actually wanted. That is a feature,
  not this bug fix.
