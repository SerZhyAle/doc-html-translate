//go:build windows

package main

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

func processCPUTimes() (map[int]cpuCounter, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(snap)
	var pe windows.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	if err := windows.Process32First(snap, &pe); err != nil {
		return nil, err
	}
	out := map[int]cpuCounter{}
	for {
		h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pe.ProcessID)
		if err == nil {
			var created, exit, kernel, user windows.Filetime
			if windows.GetProcessTimes(h, &created, &exit, &kernel, &user) == nil {
				out[int(pe.ProcessID)] = cpuCounter{
					Ticks:  (uint64(kernel.HighDateTime)<<32 | uint64(kernel.LowDateTime)) + (uint64(user.HighDateTime)<<32 | uint64(user.LowDateTime)),
					Parent: int(pe.ParentProcessID),
					Image:  windows.UTF16ToString(pe.ExeFile[:]),
				}
			}
			windows.CloseHandle(h)
		}
		if err := windows.Process32Next(snap, &pe); err != nil {
			if err != windows.ERROR_NO_MORE_FILES {
				return nil, err
			}
			break
		}
	}
	return out, nil
}

func desktopUnlocked() error {
	h, _, err := user32.NewProc("OpenInputDesktop").Call(0, 0, 1 /*DESKTOP_READOBJECTS*/)
	if h == 0 {
		return fmt.Errorf("OpenInputDesktop: %v", err)
	}
	defer user32.NewProc("CloseDesktop").Call(h)
	buf := make([]uint16, 256)
	var needed uint32
	r, _, err := user32.NewProc("GetUserObjectInformationW").Call(h, 2 /*UOI_NAME*/, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)*2), uintptr(unsafe.Pointer(&needed)))
	if r == 0 {
		return fmt.Errorf("desktop name: %v", err)
	}
	if windows.UTF16ToString(buf) != "Default" {
		return fmt.Errorf("input desktop is not unlocked Default")
	}
	return nil
}
