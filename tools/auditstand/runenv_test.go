package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

func TestAuditstandOutputHelper(t *testing.T) {
	if os.Getenv("AUDITSTAND_OUTPUT_HELPER") != "1" {
		return
	}
	var wg sync.WaitGroup
	for _, f := range []*os.File{os.Stdout, os.Stderr} {
		wg.Add(1)
		go func(f *os.File) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				fmt.Fprintf(f, "line %d\n", i)
			}
			fmt.Fprint(f, "unterminated")
		}(f)
	}
	wg.Wait()
	os.Exit(0)
}

func TestExecAppCapturesBothStreamsAndUnterminatedOutput(t *testing.T) {
	bin, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	e := &runEnv{}
	r, err := e.execApp(ctx, bin, []string{"-test.run=^TestAuditstandOutputHelper$"}, execOptions{ProfileDir: t.TempDir(), ExtraEnv: []string{"AUDITSTAND_OUTPUT_HELPER=1"}}, "canary")
	if err != nil {
		t.Fatal(err)
	}
	stdout, stderr := 0, 0
	for _, line := range r.Output {
		if line.IsErr {
			stderr++
		} else {
			stdout++
		}
	}
	if stdout != 101 || stderr != 101 || r.ExitCode != 0 || r.FirstOutputMs <= 0 {
		t.Fatalf("stdout=%d stderr=%d exit=%d first=%f", stdout, stderr, r.ExitCode, r.FirstOutputMs)
	}
}

func TestAuditstandCLIHelper(t *testing.T) {
	if os.Getenv("AUDITSTAND_CLI_HELPER") != "1" {
		return
	}
	os.Args = []string{"auditstand", "-SelfTest", "-unknown"}
	main()
}

func TestCLIArgumentFailureEndsWithVerdict(t *testing.T) {
	bin, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	r, err := (&runEnv{}).execApp(ctx, bin, []string{"-test.run=^TestAuditstandCLIHelper$"}, execOptions{ProfileDir: t.TempDir(), ExtraEnv: []string{"AUDITSTAND_CLI_HELPER=1"}}, "canary")
	if err != nil {
		t.Fatal(err)
	}
	if r.ExitCode != 2 || len(r.Output) == 0 {
		t.Fatalf("exit=%d lines=%d", r.ExitCode, len(r.Output))
	}
	last := r.Output[len(r.Output)-1]
	if last.IsErr || last.Line != "auditstand -Arguments: COULD NOT VERIFY (flag provided but not defined: -unknown)" {
		t.Fatalf("last line is not the verdict: %q", last.Line)
	}
}
