//go:build !windows

package windowsreg

import "testing"

// Off Windows every registration entry point must fail loudly rather than report success,
// so a caller never tells the user an association was made.
func TestRegistrationUnsupportedOffWindows(t *testing.T) {
	calls := map[string]func() ([]string, error){
		"RegisterHandler":        RegisterHandler,
		"RegisterOpenWith":       RegisterOpenWith,
		"RegisterOpenWithFor":    func() ([]string, error) { return RegisterOpenWithFor("/usr/bin/app") },
		"RegisterContextMenu":    RegisterContextMenu,
		"RegisterContextMenuFor": func() ([]string, error) { return RegisterContextMenuFor("/usr/bin/app") },
		"Unregister":             Unregister,
	}
	for name, call := range calls {
		if got, err := call(); err == nil || got != nil {
			t.Errorf("%s = %v, %v; want nil and an error", name, got, err)
		}
	}
	if IsDefaultHandler() {
		t.Error("IsDefaultHandler = true off Windows")
	}
}
