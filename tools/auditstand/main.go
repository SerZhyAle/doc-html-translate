// main.go wires the runner contract of strategic section 5.4: one instrument, one flag per
// action, every exit in the shared CHECK-VERDICT vocabulary.
//
//	auditstand -SelfTest               matrix integrity, capability coverage, fixture presence
//	auditstand -Fixtures [-Big]        build the synthetic corpus (deterministic)
//	auditstand -Control                the whole probe set on the idle desktop, no app started
//	auditstand -Scene S03 [-Cycles N]  run one scene; pass only against a same-day control
//	auditstand -Summary                campaign state; exit 0 only with every cell closed
//	auditstand -Plan                   print the matrix state without judging anything
package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

const usage = `auditstand - the deep dynamic audit's measuring stand (ticket 113)

Usage:
  auditstand -SelfTest
  auditstand -Fixtures [-Big]
  auditstand -Control
  auditstand -BrowserCheck [-allow-key-inject]
  auditstand -Scene <id> [-Cycles N]
  auditstand -Summary
  auditstand -Plan
  auditstand -Scene S11 -dry-run
  auditstand -Recover <recovery.json> -recovery-sha256 <hash-from-record>
  auditstand -VerifyRecovery <recovery.json> -recovery-sha256 <hash-from-record>

Actions (exactly one per run):
  -SelfTest      matrix integrity + capability coverage (CLAUDE.md/AGENTS.md pins) + fixtures
  -Fixtures      generate the synthetic corpus deterministically into -fixtures
  -Control       run every available probe on the idle desktop, no app started (exit 2 when the
                 machine's own noise exceeds a budget's jitter threshold)
  -BrowserCheck  pre-launch the instrumented browser (step 00.08: touches the real browser
                 profile) and verify the marker page, the optional injected key, the
                 capture-diff fallback and the closed port
  -Scene <id>    run a scene; refuses without a same-day control and without fixtures; scenes
                 that can write the registry refuse until the Level-B key backup exists
  -Summary       campaign state from the recorded runs; exit 0 only when every cell is measured
                 or reasoned n/a and no P0/P1 finding is open
  -Plan          print the matrix state (scenes x properties x budgets)

Common flags:
  -artifacts <dir>   run-record root (default temp/auditstand/artifacts)
  -fixtures <dir>    fixture root (default temp/auditstand/fixtures)
  -app <path>        the CLI binary under test (default: a HEAD build under temp/auditstand/bin)
  -ui <path>         the GUI binary under test (recorded, used by the GUI scenes)
  -window <title>    the P1 window-marker substring (default DOC-HTML-UI)
  -probe-duration <d>  per-probe control duration (default 10s)
  -allow-key-inject  opt in to the keyboard-hook probe's injected keys (stand-owned desktop only)
  -isolation B-full  redirected folders and a hashed named-key registry backup
  -dry-run          with -Scene: print the backup plan without launching or touching registry
`

func main() {
	fs := flag.NewFlagSet("auditstand", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() { fmt.Fprint(os.Stdout, usage) }

	selfTest := fs.Bool("SelfTest", false, "matrix integrity, capability coverage, fixture presence")
	control := fs.Bool("Control", false, "run the probe set on the idle desktop")
	browserCheck := fs.Bool("BrowserCheck", false, "pre-launch the instrumented browser and verify step 00.08")
	fixtures := fs.Bool("Fixtures", false, "generate the synthetic corpus")
	big := fs.Bool("Big", false, "with -Fixtures: also build the multi-GB fixtures")
	summary := fs.Bool("Summary", false, "print the campaign state")
	plan := fs.Bool("Plan", false, "print the matrix state")
	recoverPath := fs.String("Recover", "", "restore a hashed recovery manifest after an interrupted stand run")
	verifyRecovery := fs.String("VerifyRecovery", "", "verify byte-identical registry restoration from a hashed manifest")
	recoveryHash := fs.String("recovery-sha256", "", "expected recovery manifest SHA-256 from record.json")
	scene := fs.String("Scene", "", "scene id to run (e.g. S03)")
	cycles := fs.Int("Cycles", 1, "cycles for -Scene")
	dryRun := fs.Bool("dry-run", false, "with -Scene: print the isolation backup plan without touching the registry or launching the app")
	artifacts := fs.String("artifacts", "", "run-record root")
	fixturesDir := fs.String("fixtures", "", "fixture root")
	app := fs.String("app", "", "CLI binary under test")
	ui := fs.String("ui", "", "GUI binary under test")
	window := fs.String("window", "", "P1 window-marker substring")
	probeDuration := fs.Duration("probe-duration", 10*time.Second, "per-probe control duration")
	keyInjectFlag := fs.Bool("allow-key-inject", false, "opt in to the hook probe's key injection")
	isolation := fs.String("isolation", "B-env", `isolation level recorded into every run: "B-env" (redirected environment; registry-touching scenes refuse), "B-full" or "A" once their step lands`)

	if err := fs.Parse(os.Args[1:]); err != nil {
		emitVerdict("-Arguments", verdictCouldNotVerify, err.Error())
		os.Exit(2)
	}
	args := fs.Args()
	if len(args) > 0 {
		fmt.Fprintf(os.Stdout, "unexpected arguments: %v\n\n", args)
		fmt.Fprint(os.Stdout, usage)
		emitVerdict("-Arguments", verdictCouldNotVerify, "unexpected positional arguments")
		os.Exit(2)
	}

	env, err := newRunEnv()
	if err != nil {
		fmt.Fprintln(os.Stdout, "auditstand:", err)
		emitVerdict("-Arguments", verdictCouldNotVerify, err.Error())
		os.Exit(2)
	}
	if *artifacts != "" {
		env.artifactsDir = *artifacts
	}
	if *fixturesDir != "" {
		env.fixturesDir = *fixturesDir
	}
	if *app != "" {
		env.appPath = *app
	}
	if *ui != "" {
		env.uiPath = *ui
	}
	if *window != "" {
		auditWindowMarker = *window
	}
	if *keyInjectFlag {
		setKeyInject(true)
	}
	if *isolation != "" {
		env.isolation = *isolation
	}

	actions := 0
	for _, b := range []*bool{selfTest, control, browserCheck, fixtures, summary, plan} {
		if *b {
			actions++
		}
	}
	if *scene != "" {
		actions++
	}
	if *recoverPath != "" {
		actions++
	}
	if *verifyRecovery != "" {
		actions++
	}
	if actions != 1 {
		fmt.Fprint(os.Stdout, "exactly one action flag is required\n\n")
		fmt.Fprint(os.Stdout, usage)
		emitVerdict("-Arguments", verdictCouldNotVerify, "exactly one action flag is required")
		os.Exit(2)
	}
	if (*dryRun && *scene == "") || (*big && !*fixtures) || *cycles < 1 || *probeDuration <= 0 {
		emitVerdict("-Arguments", verdictCouldNotVerify, "-dry-run needs -Scene; -Big needs -Fixtures; cycles and probe duration must be positive")
		os.Exit(2)
	}

	switch {
	case *recoverPath != "" || *verifyRecovery != "":
		path := *recoverPath
		check := "-Recover"
		if path == "" {
			path = *verifyRecovery
			check = "-VerifyRecovery"
		}
		if err := containedPath(env.artifactsDir, path); err != nil {
			emitVerdict(check, verdictCouldNotVerify, err.Error())
			os.Exit(2)
		}
		roots, err := registrationRoots(repoRoot(), env.appPath, env.uiPath)
		if err != nil {
			emitVerdict(check, verdictCouldNotVerify, err.Error())
			os.Exit(2)
		}
		store, err := newRegistryStore()
		if err != nil {
			emitVerdict(check, verdictCouldNotVerify, err.Error())
			os.Exit(2)
		}
		r, err := loadProfileRecovery(path, *recoveryHash, store, roots)
		if err != nil {
			emitVerdict(check, verdictCouldNotVerify, err.Error())
			os.Exit(2)
		}
		if *recoverPath != "" {
			if err := r.restore(); err != nil {
				emitVerdict(check, verdictFail, err.Error())
				os.Exit(1)
			}
		}
		if err := r.verifyRestored(); err != nil {
			emitVerdict(check, verdictFail, err.Error())
			os.Exit(1)
		}
		emitVerdict(check, verdictPass, "registry pre-state verified; recovery exports retained")
		os.Exit(0)
	case *selfTest:
		res := runSelfTest(env)
		for _, d := range res.defects {
			fmt.Println("defect:", d)
		}
		for _, n := range res.notes {
			fmt.Println("advisory:", n)
		}
		for _, m := range outOfScopeMatches {
			fmt.Println("info:", m)
		}
		if res.verdict == verdictPass || res.verdict == verdictAdvisory {
			recs, _ := loadAllRecords(env.artifactsDir)
			printMatrixState(recs)
		}
		emitVerdict("-SelfTest", res.verdict, res.detail)
		os.Exit(exitCode(res.verdict))

	case *fixtures:
		check := "-Fixtures"
		if *big {
			// The multi-GB fixtures are the one generation step that can fill a disk; the
			// tactical plan gates -Big on a free-space check before anything is written.
			if free, err := freeBytes(env.fixturesDir); err != nil {
				emitVerdict(check, verdictCouldNotVerify, "free space unknown: "+err.Error())
				os.Exit(2)
			} else if free < 6<<30 {
				emitVerdict(check, verdictCouldNotVerify, fmt.Sprintf("less than 6 GiB free (%.1f GiB) for the big fixtures", float64(free)/(1<<30)))
				os.Exit(2)
			}
		}
		built, err := generateFixtures(env.fixturesDir, *big)
		if err != nil {
			fmt.Fprintln(os.Stdout, "auditstand -Fixtures:", err)
			emitVerdict(check, verdictCouldNotVerify, err.Error())
			os.Exit(2)
		}
		for _, id := range built {
			fmt.Println("built:", id)
		}
		_, corrupt, err := checkFixtures(env.fixturesDir)
		if err != nil {
			emitVerdict(check, verdictCouldNotVerify, err.Error())
			os.Exit(2)
		}
		if len(corrupt) > 0 {
			emitVerdict(check, verdictFail, fmt.Sprintf("%d fixture(s) failed their own hash: %v", len(corrupt), corrupt))
			os.Exit(1)
		}
		emitVerdict(check, verdictPass, fmt.Sprintf("%d fixture(s)", len(built)))
		os.Exit(0)

	case *control:
		check := "-Control"
		res, err := runControl(env, *probeDuration)
		if err != nil {
			fmt.Fprintln(os.Stdout, "auditstand -Control:", err)
			emitVerdict(check, verdictCouldNotVerify, err.Error())
			os.Exit(2)
		}
		emitVerdict(check, res.verdict, res.detail)
		os.Exit(exitCode(res.verdict))

	case *browserCheck:
		check := "-BrowserCheck"
		res, err := runBrowserCheck(env, *keyInjectFlag)
		if err != nil {
			fmt.Fprintln(os.Stdout, "auditstand -BrowserCheck:", err)
			emitVerdict(check, verdictCouldNotVerify, err.Error())
			os.Exit(2)
		}
		emitVerdict(check, res.verdict, res.detail)
		os.Exit(exitCode(res.verdict))

	case *scene != "":
		check := "-Scene " + *scene
		if *dryRun {
			if _, ok := sceneByID(sceneID(*scene)); !ok {
				emitVerdict(check, verdictCouldNotVerify, "unknown scene")
				os.Exit(2)
			}
			roots, err := registrationRoots(repoRoot(), env.appPath, env.uiPath)
			if err != nil {
				emitVerdict(check, verdictCouldNotVerify, err.Error())
				os.Exit(2)
			}
			for _, root := range roots {
				fmt.Println("backup: HKCU\\" + root)
			}
			fmt.Println("recovery: hash every export and the manifest; restore raw value differences; verify byte-identical; delete owned profiles; halt on failure")
			emitVerdict(check, verdictPass, "dry-run; no registry writes or app launch")
			os.Exit(0)
		}
		res, err := runScene(env, sceneID(*scene), *cycles)
		if err != nil {
			fmt.Fprintln(os.Stdout, "auditstand "+check+":", err)
			emitVerdict(check, verdictCouldNotVerify, err.Error())
			os.Exit(2)
		}
		if res.runID != "" {
			fmt.Printf("run record: %s\n", env.runDir(res.runID))
		}
		emitVerdict(check, res.verdict, res.detail)
		os.Exit(exitCode(res.verdict))

	case *summary:
		res := runSummary(env)
		emitVerdict("-Summary", res.verdict, res.detail)
		os.Exit(exitCode(res.verdict))

	case *plan:
		recs, _ := loadAllRecords(env.artifactsDir)
		printMatrixState(recs)
		emitVerdict("-Plan", verdictPass, "")
		os.Exit(0)
	}
}

func exitCode(v verdict) int {
	switch v {
	case verdictPass:
		return 0
	case verdictAdvisory:
		return 3
	case verdictFail:
		return 1
	default:
		return 2
	}
}
