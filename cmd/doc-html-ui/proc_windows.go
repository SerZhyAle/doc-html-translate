//go:build windows

package main

import (
	"os/exec"
	"unsafe"

	"golang.org/x/sys/windows"
)

// prepareTree sets up a child before Start so its whole process tree can be stopped later.
func prepareTree(cmd *exec.Cmd) { hideWindow(cmd) }

// jobTree holds the converter in a job object. Kill-on-close is what stops the child when
// the GUI dies without a word (Task Manager, a crash): the kernel closes the job handle and
// takes every process in the job down with it. Grandchildren (tesseract, pdftotext) join the
// job by inheritance, so one TerminateJobObject stops them all. Processes the converter
// started in its first instant, before AssignProcessToJobObject, escape; the converter
// spawns nothing that early.
type jobTree struct{ job windows.Handle }

func attachTree(cmd *exec.Cmd) (procTree, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, err
	}
	if err := setJobLimits(job, windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE); err != nil {
		_ = windows.CloseHandle(job)
		return nil, err
	}
	h, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if err != nil {
		_ = windows.CloseHandle(job)
		return nil, err
	}
	defer func() { _ = windows.CloseHandle(h) }()
	if err := windows.AssignProcessToJobObject(job, h); err != nil {
		_ = windows.CloseHandle(job)
		return nil, err
	}
	return &jobTree{job: job}, nil
}

func (t *jobTree) Kill() error { return windows.TerminateJobObject(t.job, 1) }

// Release lets go of the job after a run ended on its own. The converter may have opened
// the result in a browser that is now a member of the job; kill-on-close is lifted first so
// closing the handle leaves that browser running.
func (t *jobTree) Release() {
	_ = setJobLimits(t.job, 0)
	_ = windows.CloseHandle(t.job)
}

func setJobLimits(job windows.Handle, flags uint32) error {
	var info windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	info.BasicLimitInformation.LimitFlags = flags
	_, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)))
	return err
}
