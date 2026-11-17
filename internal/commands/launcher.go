package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"doc-html-translate/internal/fsutil"
	"doc-html-translate/internal/i18n"
)

func (f *front) pipeName() string {
	if f.env.PipeName != "" {
		return f.env.PipeName
	}
	if f.env.Launcher != nil && f.env.Launcher.PipeName != "" {
		return f.env.Launcher.PipeName
	}
	return DefaultPipeName()
}

func (f *front) mutexName() string {
	if f.env.MutexName != "" {
		return f.env.MutexName
	}
	if f.env.Launcher != nil && f.env.Launcher.MutexName != "" {
		return f.env.Launcher.MutexName
	}
	return DefaultMutexName()
}

func (f *front) forward(cmd *Command, rawArgs []string, resultPath string, allowFg bool) Response {
	cwd, _ := os.Getwd()
	timeout := 10 * time.Second
	if cmd != nil && cmd.TimeoutMs != nil && *cmd.TimeoutMs > 0 {
		timeout = time.Duration(*cmd.TimeoutMs) * time.Millisecond
	}
	req := Request{
		SchemaVersion: SchemaVersion,
		Command:       cmd.Name,
		Args:          rawArgs,
		WorkingDir:    cwd,
		Origin:        "cli",
		RequestID:     newRequestID(),
		Wait:          "done",
		ResultPath:    resultPath,
	}
	return dialAndExchange(f.pipeName(), f.mutexName(), req, DeliveryBound, timeout, allowFg, f.lang())
}

func (f *front) emitForwardResponse(inv *Invocation, resp Response) int {
	asJSON, uerr := inv.Bool("json")
	if uerr != nil {
		return f.refuse(uerr)
	}
	resultPath, uerr := inv.String("result")
	if uerr != nil {
		return f.refuse(uerr)
	}
	if resp.Status == StatusDone && resp.ExitCode == ExitDone && resultPath != "" && len(resp.Data) > 0 {
		out := append(append([]byte(nil), resp.Data...), '\n')
		if err := fsutil.WriteFile(resultPath, out, 0o644); err != nil {
			return f.fail(msgResultFailed)
		}
	}
	stdout, stderr := f.streams()
	if asJSON {
		_, _ = stdout.Write(encodePipeResponse(f.lang(), resp))
		return resp.ExitCode
	}
	if resp.ExitCode == ExitDone {
		fmt.Fprintln(stdout, resp.Message)
	} else {
		fmt.Fprintf(stderr, "Error: %s\n", resp.Message)
	}
	return resp.ExitCode
}

func (f *front) emitLocalDone(inv *Invocation, data any, msgKey string, msgArgs ...any) int {
	asJSON, uerr := inv.Bool("json")
	if uerr != nil {
		return f.refuse(uerr)
	}
	resultPath, uerr := inv.String("result")
	if uerr != nil {
		return f.refuse(uerr)
	}
	raw := encodeCompact(data)
	if resultPath != "" {
		out := append(append([]byte(nil), raw...), '\n')
		if err := fsutil.WriteFile(resultPath, out, 0o644); err != nil {
			return f.fail(msgResultFailed)
		}
	}
	msg := i18n.T(f.lang(), msgKey, msgArgs...)
	stdout, _ := f.streams()
	if asJSON {
		resp := Response{
			SchemaVersion: SchemaVersion,
			RequestID:     newRequestID(),
			Status:        StatusDone,
			ExitCode:      ExitDone,
			Message:       msg,
			Data:          raw,
		}
		_, _ = stdout.Write(encodePipeResponse(f.lang(), resp))
		return ExitDone
	}
	fmt.Fprintln(stdout, msg)
	return ExitDone
}

func (f *front) launcherStatus(cmd *Command, inv *Invocation) int {
	if _, uerr := inv.Bool("json"); uerr != nil {
		return f.refuse(uerr)
	}
	resultPath, uerr := inv.String("result")
	if uerr != nil {
		return f.refuse(uerr)
	}
	resp := f.forward(cmd, nil, resultPath, false)
	return f.emitForwardResponse(inv, resp)
}

func (f *front) launcherQuit(cmd *Command, inv *Invocation, _ []string) int {
	cancelRuns, uerr := inv.Bool("cancel-runs")
	if uerr != nil {
		return f.refuse(uerr)
	}
	if _, uerr := inv.Bool("json"); uerr != nil {
		return f.refuse(uerr)
	}
	resultPath, uerr := inv.String("result")
	if uerr != nil {
		return f.refuse(uerr)
	}
	var fwdArgs []string
	if cancelRuns {
		fwdArgs = []string{"--cancel-runs"}
	}
	resp := f.forward(cmd, fwdArgs, resultPath, false)
	if resp.Reason == ReasonNotRunning {
		resp = Response{
			SchemaVersion: SchemaVersion,
			RequestID:     resp.RequestID,
			Status:        StatusDone,
			ExitCode:      ExitDone,
			Message:       i18n.T(f.lang(), msgNotRunning),
			Data:          encodeCompact(QuitData{CancelledRuns: 0}),
		}
	}
	return f.emitForwardResponse(inv, resp)
}

func (f *front) launcherShow(cmd *Command, inv *Invocation) int {
	if _, uerr := inv.Bool("json"); uerr != nil {
		return f.refuse(uerr)
	}
	resultPath, uerr := inv.String("result")
	if uerr != nil {
		return f.refuse(uerr)
	}
	if f.env.Launcher.BypassSingleInstance {
		return f.env.Launcher.StartFirstInstance(nil, nil, func() {
			_ = f.emitLocalDone(inv, ShowData{Focused: true, Reopened: true}, msgShowDone)
		})
	}
	lock, acquired, err := AcquireInstanceLock(f.mutexName())
	if err != nil {
		return f.fail(msgPipeFailed)
	}
	if acquired {
		return f.env.Launcher.StartFirstInstance(lock, nil, func() {
			_ = f.emitLocalDone(inv, ShowData{Focused: true, Reopened: true}, msgShowDone)
		})
	}
	resp := f.forward(cmd, nil, resultPath, true)
	return f.emitForwardResponse(inv, resp)
}

func (f *front) launcherOpen(cmd *Command, inv *Invocation) int {
	if _, uerr := inv.Bool("json"); uerr != nil {
		return f.refuse(uerr)
	}
	resultPath, uerr := inv.String("result")
	if uerr != nil {
		return f.refuse(uerr)
	}
	cwd, _ := os.Getwd()
	paths, missing := resolveExistingPaths(cwd, inv.Strings("in"))
	if missing {
		return f.targetMissing()
	}
	unique := dedupPaths(paths)
	if f.env.Launcher.BypassSingleInstance {
		return f.env.Launcher.StartFirstInstance(nil, unique, func() {
			_ = f.emitLocalDone(inv, OpenData{Queued: len(unique), Reopened: true}, msgOpenDone, len(unique))
		})
	}
	lock, acquired, err := AcquireInstanceLock(f.mutexName())
	if err != nil {
		return f.fail(msgPipeFailed)
	}
	if acquired {
		return f.env.Launcher.StartFirstInstance(lock, unique, func() {
			_ = f.emitLocalDone(inv, OpenData{Queued: len(unique), Reopened: true}, msgOpenDone, len(unique))
		})
	}
	fwdArgs := make([]string, 0, len(unique)*2)
	for _, p := range unique {
		fwdArgs = append(fwdArgs, "--in", p)
	}
	resp := f.forward(cmd, fwdArgs, resultPath, true)
	return f.emitForwardResponse(inv, resp)
}

func (f *front) launcherBareShow() int {
	lock, acquired, err := AcquireInstanceLock(f.mutexName())
	if err != nil {
		return ExitFailed
	}
	if acquired {
		return f.env.Launcher.StartFirstInstance(lock, nil, nil)
	}
	showCmd := &Command{Name: "show"}
	resp := f.forward(showCmd, nil, "", true)
	return resp.ExitCode
}

func (f *front) launcherDefaultOpen(args []string) (bool, int) {
	if len(args) > 0 && strings.HasPrefix(args[0], "-") {
		if _, err := os.Stat(args[0]); err != nil {
			return true, f.refuse(usage(ReasonUnknownVerb, msgUnknownCommand, args[0]))
		}
	}
	cwd, _ := os.Getwd()
	paths, missing := resolveExistingPaths(cwd, args)
	if missing {
		return true, f.targetMissing()
	}
	unique := dedupPaths(paths)
	if f.env.Launcher.BypassSingleInstance {
		return false, 0
	}
	lock, acquired, err := AcquireInstanceLock(f.mutexName())
	if err != nil {
		return true, ExitFailed
	}
	if acquired {
		return true, f.env.Launcher.StartFirstInstance(lock, unique, nil)
	}
	fwdArgs := make([]string, 0, len(unique)*2)
	for _, p := range unique {
		fwdArgs = append(fwdArgs, "--in", p)
	}
	openCmd := &Command{Name: "open"}
	resp := f.forward(openCmd, fwdArgs, "", true)
	if resp.ExitCode != ExitDone {
		_, stderr := f.streams()
		fmt.Fprintf(stderr, "Error: %s\n", resp.Message)
	}
	return true, resp.ExitCode
}

func dedupPaths(paths []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		clean := filepath.Clean(p)
		key := clean
		if runtime.GOOS == "windows" {
			key = strings.ToLower(clean)
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, clean)
	}
	return out
}
