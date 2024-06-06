package wintun

import (
	"context"
	"log/slog"
	"runtime"
	"syscall"

	"github.com/pkg/errors"
	"golang.org/x/sys/windows"
)

type LoggerLevel int

const (
	Info LoggerLevel = iota
	Warn
	Error
)

type LoggerCallback func(level LoggerLevel, timestamp uint64, msg *uint16) uintptr

func DefaultCallback(log *slog.Logger) LoggerCallback {
	return func(level LoggerLevel, timestamp uint64, msg *uint16) uintptr {
		var sl slog.Level
		switch level {
		case Info:
			sl = slog.LevelInfo
		case Warn:
			sl = slog.LevelWarn
		case Error:
			sl = slog.LevelError
		default:
			sl = slog.LevelDebug
		}
		log.LogAttrs(
			context.Background(), sl,
			windows.UTF16PtrToString(msg),
		)
		return 0
	}
}

func SetLogger(logger LoggerCallback) error {
	var callback uintptr
	if logger != nil {
		switch runtime.GOARCH {
		case "386":
			callback = windows.NewCallback(func(level LoggerLevel, timestampLow, timestampHigh uint32, msg *uint16) {
				logger(level, uint64(timestampHigh)<<32|uint64(timestampLow), msg)
			})
		case "arm":
			callback = windows.NewCallback(func(level LoggerLevel, _, timestampLow, timestampHigh uint32, msg *uint16) {
				logger(level, uint64(timestampHigh)<<32|uint64(timestampLow), msg)
			})
		case "amd64", "arm64":
			callback = windows.NewCallback(logger)
		default:
			return errors.Errorf("not support windows arch %s", runtime.GOARCH)
		}
	}

	_, _, err := syscall.SyscallN(procSetLogger.Addr(), callback)
	if err != windows.ERROR_SUCCESS {
		return errors.WithStack(err)
	}
	return nil
}
