//go:build windows

package dialog

import "testing"

type boxCall struct {
	owner          uintptr
	title, message string
	style          uintptr
}

// fakeMessageBox records each dialog instead of showing it and answers with ret. The console
// owner is pinned to owner.
func fakeMessageBox(t *testing.T, owner, ret uintptr) *[]boxCall {
	t.Helper()
	var calls []boxCall
	origBox, origOwner := messageBox, consoleOwner
	t.Cleanup(func() { messageBox, consoleOwner = origBox, origOwner })
	consoleOwner = func() uintptr { return owner }
	messageBox = func(owner uintptr, title, message string, style uintptr) uintptr {
		calls = append(calls, boxCall{owner, title, message, style})
		return ret
	}
	return &calls
}

// The cost question must default to "do not spend": Cancel is the default button, so Enter
// declines, and OK/Cancel (unlike Yes/No) gives Escape and the close box the Cancel path.
func TestConfirmDefaultsToCancel(t *testing.T) {
	const idCancel = uintptr(2)
	// 0 is what MessageBoxW returns when it cannot show the box at all; an unanswered
	// question must never read as consent.
	for _, tc := range []struct {
		name string
		ret  uintptr
		want bool
	}{{"ok", idOK, true}, {"cancel or escape", idCancel, false}, {"box failed", 0, false}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(HostEnv, "")
			calls := fakeMessageBox(t, 0x1234, tc.ret)
			if got := Confirm("Cost", "Spend $3?"); got != tc.want {
				t.Errorf("Confirm = %v, want %v", got, tc.want)
			}
			want := boxCall{0x1234, "Cost", "Spend $3?", mbOKCancel | mbIconWarning | mbDefButton2 | mbSetForeground}
			if len(*calls) != 1 || (*calls)[0] != want {
				t.Errorf("dialogs shown = %+v, want one %+v", *calls, want)
			}
		})
	}
}

// Without a console to own it the box is made topmost, so it cannot open behind another window.
func TestConfirmWithoutOwnerIsTopmost(t *testing.T) {
	t.Setenv(HostEnv, "")
	calls := fakeMessageBox(t, 0, 0)
	Confirm("Cost", "Spend $3?")
	if len(*calls) != 1 || (*calls)[0].style&mbTopmost == 0 {
		t.Errorf("dialogs shown = %+v, want one topmost box", *calls)
	}
}

func TestShowWarning(t *testing.T) {
	t.Setenv(HostEnv, "")
	calls := fakeMessageBox(t, 0x1234, 0)
	ShowWarning("PDF", "pdftotext was blocked")
	want := boxCall{0x1234, "PDF", "pdftotext was blocked", mbOK | mbIconWarning | mbSetForeground}
	if len(*calls) != 1 || (*calls)[0] != want {
		t.Errorf("dialogs shown = %+v, want one %+v", *calls, want)
	}
}

// Under the GUI no native box may open at all: it would sit behind the GUI's window.
func TestHostedDialogsShowNoNativeBox(t *testing.T) {
	t.Setenv(HostEnv, HostStdio)
	calls := fakeMessageBox(t, 0x1234, idOK)
	withHostPipes(t, "yes\n", func() {
		if !Confirm("Cost", "Spend $3?") {
			t.Error("Confirm = false for a GUI that answered yes")
		}
		ShowWarning("PDF", "pdftotext was blocked")
	})
	if len(*calls) != 0 {
		t.Errorf("native boxes shown under the GUI: %+v", *calls)
	}
}
