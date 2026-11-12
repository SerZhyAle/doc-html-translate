//go:build windows

package procrun

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestRunWindowsHelperWithoutConsole(t *testing.T) {
	if os.Getenv("DOCHT_TEST_CONSOLE_HELPER") == "1" {
		getConsoleWindow := windows.NewLazySystemDLL("kernel32.dll").NewProc("GetConsoleWindow")
		isWindowVisible := windows.NewLazySystemDLL("user32.dll").NewProc("IsWindowVisible")
		window, _, _ := getConsoleWindow.Call()
		visible, _, _ := isWindowVisible.Call(window)
		fmt.Printf("visible=%d argument=%s", visible, os.Args[len(os.Args)-1])
		os.Exit(0)
	}

	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	result, err := Run(context.Background(), Cmd{
		Tool: "test helper", Path: exe,
		Args: []string{"-test.run=^TestRunWindowsHelperWithoutConsole$", "argument with spaces"},
		Env:  []string{"DOCHT_TEST_CONSOLE_HELPER=1"}, Timeout: 3 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(result.Stdout), "visible=0 argument=argument with spaces"; got != want {
		t.Fatalf("helper output = %q, want %q", got, want)
	}
}

func TestPrepareTreeWindowsSuppressesConsole(t *testing.T) {
	cmd := exec.Command("unused")
	prepareTree(cmd)
	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.HideWindow || cmd.SysProcAttr.CreationFlags&windows.CREATE_NO_WINDOW == 0 {
		t.Fatalf("helper must start without a visible console: %+v", cmd.SysProcAttr)
	}
}

func TestRunWindowsInstalledTesseract(t *testing.T) {
	bin := os.Getenv("DOCHT_TEST_TESSERACT")
	if bin == "" {
		t.Skip("set DOCHT_TEST_TESSERACT to an installed Tesseract binary")
	}
	result, err := Run(context.Background(), Cmd{Tool: "tesseract", Path: bin, Args: []string{"--list-langs"}, Timeout: 3 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(result.Stdout), "List of available languages") {
		t.Fatalf("unexpected language list: %q", result.Stdout)
	}
}
