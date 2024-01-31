package wintun

import (
	"syscall"

	"golang.org/x/sys/windows"
)

func syscallN(trap uintptr, args ...uintptr) (r1, r2 uintptr, err syscall.Errno) {
	// avoid syscallN(0,...) FAIL
	if trap == 0 {
		return 0, 0, windows.ERROR_INVALID_HANDLE
	}

	// special, all wintun function first parameter is not null/0
	if len(args) > 0 && args[0] == 0 {
		return 0, 0, windows.ERROR_INVALID_HANDLE
	}

	return syscall.SyscallN(trap, args...)
}
