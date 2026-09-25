//go:build windows

package procrun

import (
	"os/exec"
	"strconv"
	"unsafe"

	"golang.org/x/sys/windows"
)

// processTree is a job object holding the helper. KILL_ON_JOB_CLOSE makes closing the handle
// end every process in the job, including children the helper started after it was assigned.
type processTree struct {
	job windows.Handle
}

func prepareTree(*exec.Cmd) {}

// attachTree puts the started helper into a new job. It returns nil when that fails (an old
// Windows without nested jobs, or a parent job that forbids it); killTree then falls back to
// taskkill /T.
func attachTree(cmd *exec.Cmd) *processTree {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		_ = windows.CloseHandle(job)
		return nil
	}
	proc, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if err != nil {
		_ = windows.CloseHandle(job)
		return nil
	}
	defer func() { _ = windows.CloseHandle(proc) }()
	if err := windows.AssignProcessToJobObject(job, proc); err != nil {
		_ = windows.CloseHandle(job)
		return nil
	}
	return &processTree{job: job}
}

func killTree(cmd *exec.Cmd, t *processTree) error {
	if t != nil {
		if err := windows.TerminateJobObject(t.job, 1); err == nil {
			return nil
		}
	}
	// taskkill walks the parent-pid links, which is what the job would have given us.
	_ = exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
	return cmd.Process.Kill()
}

func releaseTree(t *processTree) {
	if t != nil {
		_ = windows.CloseHandle(t.job)
	}
}
