//go:build windows

package fdsec

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// restrictToUser creates dir with a protected DACL that grants full control to the current user
// and SYSTEM only, set at creation so there is no window in which it inherits a wider one.
func restrictToUser(dir string) error {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return err
	}
	sd, err := windows.SecurityDescriptorFromString("D:PAI(A;OICI;FA;;;" + user.User.Sid.String() + ")(A;OICI;FA;;;SY)")
	if err != nil {
		return err
	}
	sa := &windows.SecurityAttributes{SecurityDescriptor: sd}
	sa.Length = uint32(unsafe.Sizeof(*sa))
	p, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return err
	}
	return windows.CreateDirectory(p, sa)
}
