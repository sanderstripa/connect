//go:build windows

package main

import (
	"golang.org/x/sys/windows"
	"unsafe"
)

func containChromeProcesses() (func(), error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, err
	}
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	_, err = windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits)))
	if err == nil {
		err = windows.AssignProcessToJobObject(job, windows.CurrentProcess())
	}
	if err != nil {
		windows.CloseHandle(job)
		return nil, err
	}
	return func() {
		// Normal shutdown already stopped the cores. Do not terminate our own host
		// when releasing the final job handle during its deferred cleanup.
		limits.BasicLimitInformation.LimitFlags = 0
		windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits)))
		windows.CloseHandle(job)
	}, nil
}
