// runenv.go is the scene runner's environment: where the artifacts and fixtures live, how the
// app binaries are resolved, and how an app process is started under the Level-B environment
// redirection with its output captured and its resources sampled.

package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// runEnv is one invocation's context: the repo-rooted locations every subcommand shares.
type runEnv struct {
	artifactsDir string
	fixturesDir  string
	binDir       string

	appPath string // -app override; otherwise a HEAD build under binDir
	uiPath  string // -ui override

	isolation     string // "B-env" (redirected environment) or "none"
	probeDuration time.Duration
	sceneRunID    string
	recovery      *profileRecovery
	profiles      []string
	watch         *validityWatch
	resolvedApp   *binaryID
}

// newRunEnv resolves the default locations. The stand runs from the repository root like every
// other script here.
func newRunEnv() (*runEnv, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return &runEnv{
		artifactsDir:  filepath.Join(cwd, "temp", "auditstand", "artifacts"),
		fixturesDir:   filepath.Join(cwd, "temp", "auditstand", "fixtures"),
		binDir:        filepath.Join(cwd, "temp", "auditstand", "bin"),
		isolation:     "B-env",
		probeDuration: 10 * time.Second,
	}, nil
}

func (e *runEnv) runDir(runID string) string { return filepath.Join(e.artifactsDir, runID) }

// resolveApp returns the CLI binary path and its hash, building it from HEAD when neither -app
// nor a previous build exists. The record always carries the SHA-256 of what actually ran
// (strategic section 5.4; owner decision 4 pins the choice later).
func (e *runEnv) resolveApp() (binaryID, error) {
	if e.resolvedApp != nil {
		return *e.resolvedApp, nil
	}
	name := "doc-html-translate.exe"
	p := e.appPath
	if p == "" {
		p = filepath.Join(e.binDir, name)
	}
	p, err := filepath.Abs(p)
	if err != nil {
		return binaryID{}, err
	}
	_, statErr := os.Stat(p)
	if e.appPath == "" || statErr != nil {
		if e.appPath != "" {
			return binaryID{}, fmt.Errorf("app binary %s absent", e.appPath)
		}
		if err := os.MkdirAll(e.binDir, 0o755); err != nil {
			return binaryID{}, err
		}
		// nodynamic as every build of the app (ticket 115): without it the binary under test would
		// load libjxl.dll from the search path, which the shipped one does not.
		cmd := exec.Command("go", "build", "-tags", "nodynamic", "-o", p, "./cmd/doc-html-translate")
		cmd.Dir = repoRoot()
		out, err := cmd.CombinedOutput()
		if err != nil {
			return binaryID{}, fmt.Errorf("building the app under test: %v: %s", err, strings.TrimSpace(string(out)))
		}
	}
	sum, err := sha256File(p)
	if err != nil {
		return binaryID{}, err
	}
	e.resolvedApp = &binaryID{Path: p, SHA256: sum}
	return *e.resolvedApp, nil
}

// repoRoot is the working directory the stand was started in; run from the repository root.
func repoRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return cwd
}

// execResult is one app run's capture: the outcome capture of P7 plus the timings of P1.
type execResult struct {
	ExitCode         int
	FirstOutputMs    float64 // start to the first output line (H5's CLI bar)
	WallMs           float64 // start to exit
	StartedAt        time.Time
	Output           []outputLine
	P4Samples        []sample
	PIDs             []int // the process tree observed (the app; helpers are its children)
	combinedTail     string
	CaptureTruncated bool
}

type outputLine struct {
	AtMs  float64 `json:"at_ms"`
	Line  string  `json:"line"`
	IsErr bool    `json:"stderr,omitempty"`
}

// execOptions tunes one app run.
type execOptions struct {
	ProfileDir  string // the Level-B redirected root (LOCALAPPDATA, TEMP, .., live under it)
	MinimalPath bool   // strip PATH to the system directories (the missing-helper injection)
	ExtraEnv    []string
	Dir         string // working directory for the run
}

// execApp starts the app binary under the redirected environment, streams its output with
// timestamps, samples its process at 10 Hz and waits. The canary travels in the environment -
// logs must never echo it (P7).
func (e *runEnv) execApp(ctx context.Context, bin string, args []string, opts execOptions, canary string) (*execResult, error) {
	if e.sceneRunID != "" {
		if err := e.prepareProfile(opts.ProfileDir); err != nil {
			return nil, cannotVerify(err)
		}
		if opts.Dir != "" {
			if opts.Dir != e.runDir(e.sceneRunID) {
				if err := containedPath(e.runDir(e.sceneRunID), opts.Dir); err != nil {
					return nil, cannotVerify(err)
				}
			}
			if err := os.MkdirAll(opts.Dir, 0o700); err != nil {
				return nil, err
			}
		}
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = opts.Dir
	cmd.Env = e.childEnv(opts.ProfileDir, opts.MinimalPath, canary, opts.ExtraEnv)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	start := time.Now()
	res := &execResult{StartedAt: start}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", filepath.Base(bin), err)
	}
	res.PIDs = append(res.PIDs, cmd.Process.Pid)
	if e.watch != nil {
		e.watch.exclude(cmd.Process.Pid)
	}

	// P4 sampler at 10 Hz for the app process while it lives.
	stopSampler := make(chan struct{})
	var samplerWG sync.WaitGroup
	samplerWG.Add(1)
	go func() {
		defer samplerWG.Done()
		tick := time.NewTicker(100 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-stopSampler:
				return
			case <-tick.C:
				if s, ok := sampleProcess(cmd.Process.Pid); ok {
					res.P4Samples = append(res.P4Samples, sample{
						TMs:    time.Since(start).Milliseconds(),
						Probe:  "P4.process",
						PID:    s.PID,
						Fields: sampleFields(s),
					})
				}
			}
		}
	}()

	var wg sync.WaitGroup
	var outputMu sync.Mutex
	const maxCaptureBytes = 8 << 20
	capturedBytes := 0
	scan := func(r io.Reader, isErr bool) {
		defer wg.Done()
		data := make([]byte, 0, 4096)
		chunk := make([]byte, 4096)
		for {
			n, err := r.Read(chunk)
			if n > 0 {
				outputMu.Lock()
				if capturedBytes+n > maxCaptureBytes {
					res.CaptureTruncated = true
					outputMu.Unlock()
					data = nil
					_, _ = io.Copy(io.Discard, r)
					return
				}
				capturedBytes += n
				outputMu.Unlock()
				data = append(data, chunk[:n]...)
				for {
					idx := indexByte(data, '\n')
					if idx < 0 {
						break
					}
					line := string(data[:idx])
					data = data[idx+1:]
					outputMu.Lock()
					if res.FirstOutputMs == 0 {
						res.FirstOutputMs = float64(time.Since(start).Microseconds()) / 1000
					}
					res.Output = append(res.Output, outputLine{AtMs: float64(time.Since(start).Microseconds()) / 1000, Line: line, IsErr: isErr})
					outputMu.Unlock()
				}
			}
			if err != nil {
				if n == 0 && len(data) > 0 {
					outputMu.Lock()
					if res.FirstOutputMs == 0 {
						res.FirstOutputMs = float64(time.Since(start).Microseconds()) / 1000
					}
					res.Output = append(res.Output, outputLine{AtMs: float64(time.Since(start).Microseconds()) / 1000, Line: string(data), IsErr: isErr})
					outputMu.Unlock()
				}
				return
			}
		}
	}
	wg.Add(2)
	go scan(stdout, false)
	go scan(stderr, true)
	wg.Wait()

	waitErr := cmd.Wait()
	close(stopSampler)
	samplerWG.Wait()

	res.WallMs = float64(time.Since(start).Microseconds()) / 1000
	res.ExitCode = cmd.ProcessState.ExitCode()
	var tail bytes.Buffer
	for _, l := range res.Output {
		tail.WriteString(l.Line)
		tail.WriteByte('\n')
	}
	res.combinedTail = tail.String()
	if res.CaptureTruncated {
		return res, cannotVerify(fmt.Errorf("app output exceeded the stand's 8 MiB capture cap; outcome evidence is incomplete"))
	}
	if waitErr != nil {
		// A non-zero exit is an outcome to record, not a stand error; context cancellation is.
		if ctx.Err() != nil {
			return res, ctx.Err()
		}
	}
	return res, nil
}

// childEnv builds the Level-B environment: every writable root the app resolves through
// LOCALAPPDATA / APPDATA / TEMP / TMP / USERPROFILE is redirected under the run's profile dir,
// the canary travels in a variable the app has no reason to echo, and MinimalPath strips PATH
// to the Windows system directories (the missing-helper injection of S16).
func (e *runEnv) childEnv(profileDir string, minimalPath bool, canary string, extra []string) []string {
	local := filepath.Join(profileDir, "LocalAppData")
	roam := filepath.Join(profileDir, "AppData", "Roaming")
	tmp := filepath.Join(profileDir, "Temp")
	home := filepath.Join(profileDir, "UserProfile")
	for _, d := range []string{local, roam, tmp, home} {
		_ = os.MkdirAll(d, 0o755)
	}
	env := []string{
		"LOCALAPPDATA=" + local,
		"APPDATA=" + roam,
		"TEMP=" + tmp,
		"TMP=" + tmp,
		"USERPROFILE=" + home,
		"DOCHT_AUDIT_CANARY=" + canary,
	}
	// Keep the variables a child needs to find its own system, drop the user-specific ones the
	// app resolves through the five roots above.
	for _, kv := range os.Environ() {
		up := strings.ToUpper(kv)
		if strings.HasPrefix(up, "LOCALAPPDATA=") || strings.HasPrefix(up, "APPDATA=") ||
			strings.HasPrefix(up, "TEMP=") || strings.HasPrefix(up, "TMP=") ||
			strings.HasPrefix(up, "USERPROFILE=") || strings.HasPrefix(up, "DOCHT_AUDIT_CANARY=") {
			continue
		}
		if minimalPath && strings.HasPrefix(up, "PATH=") {
			continue
		}
		env = append(env, kv)
	}
	if minimalPath {
		env = append(env, `PATH=C:\Windows\System32;C:\Windows`)
	}
	return append(env, extra...)
}

func sampleFields(s procSample) map[string]interface{} {
	return map[string]interface{}{
		"handles":           s.Handles,
		"gdi":               s.GDI,
		"user":              s.USER,
		"threads":           s.Threads,
		"private_bytes":     s.PrivateBytes,
		"working_set_bytes": s.WorkingSetByte,
	}
}

func indexByte(b []byte, c byte) int {
	for i, v := range b {
		if v == c {
			return i
		}
	}
	return -1
}

// findRunLog returns the newest run log the app wrote into the redirected profile (the P7
// boundary-line scan reads it; the CLI writes one log per run under LOCALAPPDATA).
func findRunLog(profileDir string) (string, error) {
	logs := filepath.Join(profileDir, "LocalAppData", "doc-html-translate", "logs")
	entries, err := os.ReadDir(logs)
	if err != nil {
		return "", err
	}
	var best string
	var bestAt time.Time
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(bestAt) {
			bestAt = info.ModTime()
			best = filepath.Join(logs, e.Name())
		}
	}
	if best == "" {
		return "", fmt.Errorf("no run log under %s", logs)
	}
	return best, nil
}

// countCanary scans files for the token and returns the hit count with the paths - the P7
// canary assertion (zero hits where the design forbids the token).
func countCanary(files []string, token string) (int, []string) {
	hits := 0
	var where []string
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		if n := strings.Count(string(b), token); n > 0 {
			hits += n
			where = append(where, f)
		}
	}
	return hits, where
}
