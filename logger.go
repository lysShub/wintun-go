package wintun

import (
	"fmt"
	"log"
	"runtime"
	"syscall"

	"golang.org/x/sys/windows"
)

type loggerLevel int

const (
	LogInfo loggerLevel = iota
	LogWarn
	LogErr
)

type LoggerCallback func(level loggerLevel, timestamp uint64, msg *uint16) uintptr

func Message(level loggerLevel, timestamp uint64, msg *uint16) uintptr {
	if tw, ok := log.Default().Writer().(interface {
		WriteWithTimestamp(p []byte, ts int64) (n int, err error)
	}); ok {
		tw.WriteWithTimestamp([]byte(log.Default().Prefix()+windows.UTF16PtrToString(msg)), (int64(timestamp)-116444736000000000)*100)
	} else {
		log.Println(windows.UTF16PtrToString(msg))
	}
	return 0
}

// SetLogger sets logger callback function.
//
//	logger may be called from various threads concurrently, set to nil to disable
func (t *Wintun) SetLogger(logger LoggerCallback) error {
	var callback uintptr
	if logger != nil {
		switch runtime.GOARCH {
		case "386":
			callback = windows.NewCallback(func(level loggerLevel, timestampLow, timestampHigh uint32, msg *uint16) {
				logger(level, uint64(timestampHigh)<<32|uint64(timestampLow), msg)
			})
		case "arm":
			callback = windows.NewCallback(func(level loggerLevel, _, timestampLow, timestampHigh uint32, msg *uint16) {
				logger(level, uint64(timestampHigh)<<32|uint64(timestampLow), msg)
			})
		case "amd64", "arm64":
			callback = windows.NewCallback(logger)
		default:
			return fmt.Errorf("not support windows arch %s", runtime.GOARCH)
		}
	}

	_, _, err := syscall.SyscallN(t.wintunSetLogger, callback)
	if err != syscall.Errno(0) {
		return err
	}
	return nil
}
