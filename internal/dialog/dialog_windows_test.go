//go:build windows

package dialog

import "testing"

type boxCall struct {
	title, message string
	style          uintptr
}

// fakeMessageBox records each dialog instead of showing it and answers with ret.
func fakeMessageBox(t *testing.T, ret uintptr) *[]boxCall {
	t.Helper()
	var calls []boxCall
	orig := messageBox
	t.Cleanup(func() { messageBox = orig })
	messageBox = func(title, message string, style uintptr) uintptr {
		calls = append(calls, boxCall{title, message, style})
		return ret
	}
	return &calls
}

func TestConfirmYesNo(t *testing.T) {
	const idNo = uintptr(7)
	// 0 is what MessageBoxW returns when it cannot show the box at all; an unanswered
	// question must never read as consent.
	for _, tc := range []struct {
		name string
		ret  uintptr
		want bool
	}{{"yes", idYes, true}, {"no", idNo, false}, {"box failed", 0, false}} {
		t.Run(tc.name, func(t *testing.T) {
			calls := fakeMessageBox(t, tc.ret)
			if got := ConfirmYesNo("Register", "Make this app the default?"); got != tc.want {
				t.Errorf("ConfirmYesNo = %v, want %v", got, tc.want)
			}
			want := boxCall{"Register", "Make this app the default?", mbYesNo | mbIconQuestion}
			if len(*calls) != 1 || (*calls)[0] != want {
				t.Errorf("dialogs shown = %+v, want one %+v", *calls, want)
			}
		})
	}
}

func TestShowWarning(t *testing.T) {
	calls := fakeMessageBox(t, 0)
	ShowWarning("PDF", "pdftotext was blocked")
	want := boxCall{"PDF", "pdftotext was blocked", mbOK | mbIconWarning}
	if len(*calls) != 1 || (*calls)[0] != want {
		t.Errorf("dialogs shown = %+v, want one %+v", *calls, want)
	}
}
