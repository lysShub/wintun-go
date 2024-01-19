//go:build windows
// +build windows

package wintun

import (
	"fmt"
	"log"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/lysShub/go-dll"
	"golang.org/x/sys/windows"
)

type Wintun struct {
	wintunDll dll.DLL

	// todo: add ctx

	wintunCreateAdapter           uintptr
	wintunOpenAdapter             uintptr
	wintunCloseAdapter            uintptr
	wintunDeleteDriver            uintptr
	wintunGetAdapterLuid          uintptr
	wintunGetRunningDriverVersion uintptr
	wintunSetLogger               uintptr
	wintunStartSession            uintptr
	wintunEndSession              uintptr
	wintunGetReadWaitEvent        uintptr
	wintunReceivePacket           uintptr
	wintunReleaseReceivePacket    uintptr
	wintunAllocateSendPacket      uintptr
	wintunSendPacket              uintptr
}

func (t *Wintun) Close() error {
	// todo: add test
	return t.wintunDll.Release()
}

// DriverVersion determines the version of the Wintun driver currently loaded.
func (t *Wintun) DriverVersion() (version uint32, err error) {
	r0, _, err := syscall.SyscallN(t.wintunGetRunningDriverVersion)
	if r0 == 0 {
		return 0, err
	}
	return uint32(r0), nil
}

// CreateAdapter creates a new wintun adapter.
func (t *Wintun) CreateAdapter(name, tunType string, guid *windows.GUID) (adapter *Adapter, err error) {
	var name16 *uint16
	name16, err = windows.UTF16PtrFromString(name)
	if err != nil {
		return
	}
	var tunnelType16 *uint16
	tunnelType16, err = windows.UTF16PtrFromString(tunType)
	if err != nil {
		return
	}
	r1, _, err := syscall.SyscallN(t.wintunCreateAdapter, uintptr(unsafe.Pointer(name16)), uintptr(unsafe.Pointer(tunnelType16)), uintptr(unsafe.Pointer(guid)))
	if r1 == 0 {
		return nil, err
	}
	return &Adapter{handle: r1}, nil
}

// OpenAdapter opens an existing wintun adapter.
func (t *Wintun) OpenAdapter(name string) (adapter *Adapter, err error) {
	var name16 *uint16
	name16, err = windows.UTF16PtrFromString(name)
	if err != nil {
		return
	}
	r1, _, err := syscall.SyscallN(t.wintunOpenAdapter, uintptr(unsafe.Pointer(name16)))
	if r1 == 0 {
		return nil, err
	}
	return &Adapter{
		wintun: t,
		handle: r1,
	}, nil
}

// DeleteDriver deletes the Wintun driver if there are no more adapters in use.
func (t *Wintun) DeleteDriver() error {
	r1, _, err := syscall.SyscallN(t.wintunDeleteDriver)
	if r1 == 0 {
		return err
	}
	return nil
}

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
