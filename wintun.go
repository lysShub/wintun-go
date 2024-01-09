//go:build windows
// +build windows

package wintun

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"github.com/lysShub/go-dll"
	"golang.org/x/sys/windows"
)

var (
	wintunDllLoadOnec sync.Once
	wintunDll         dll.DLL

	wintunCreateAdapter           uintptr //*windows.Proc
	wintunOpenAdapter             uintptr //*windows.Proc
	wintunCloseAdapter            uintptr //*windows.Proc
	wintunDeleteDriver            uintptr //*windows.Proc
	wintunGetAdapterLuid          uintptr //*windows.Proc
	wintunGetRunningDriverVersion uintptr //*windows.Proc
	wintunSetLogger               uintptr //*windows.Proc
	wintunStartSession            uintptr //*windows.Proc
	wintunEndSession              uintptr //*windows.Proc
	wintunGetReadWaitEvent        uintptr //*windows.Proc
	wintunReceivePacket           uintptr //*windows.Proc
	wintunReleaseReceivePacket    uintptr //*windows.Proc
	wintunAllocateSendPacket      uintptr //*windows.Proc
	wintunSendPacket              uintptr //*windows.Proc
)

// DriverVersion determines the version of the Wintun driver currently loaded.
func DriverVersion() (version uint32, err error) {
	r0, _, err := syscall.SyscallN(wintunGetRunningDriverVersion)
	if r0 == 0 {
		return 0, err
	}
	return uint32(r0), nil
}

// CreateAdapter creates a new wintun adapter.
func CreateAdapter(name, tunType string, guid *windows.GUID) (adapter *Adapter, err error) {
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
	r1, _, err := syscall.SyscallN(wintunCreateAdapter, uintptr(unsafe.Pointer(name16)), uintptr(unsafe.Pointer(tunnelType16)), uintptr(unsafe.Pointer(guid)))
	if r1 == 0 {
		return nil, err
	}
	return &Adapter{handle: r1}, nil
}

// OpenAdapter opens an existing wintun adapter.
func OpenAdapter(name string) (adapter *Adapter, err error) {
	var name16 *uint16
	name16, err = windows.UTF16PtrFromString(name)
	if err != nil {
		return
	}
	r1, _, err := syscall.SyscallN(wintunOpenAdapter, uintptr(unsafe.Pointer(name16)))
	if r1 == 0 {
		return nil, err
	}
	return &Adapter{handle: r1}, nil
}

// DeleteDriver deletes the Wintun driver if there are no more adapters in use.
func DeleteDriver() error {
	r1, _, err := syscall.SyscallN(wintunDeleteDriver)
	if r1 == 0 {
		return err
	}
	return nil
}

type loggerLevel int

const (
	logInfo loggerLevel = iota
	logWarn
	logErr
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
func SetLogger(logger LoggerCallback) error {
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

	_, _, err := syscall.SyscallN(wintunSetLogger, callback)
	if err != syscall.Errno(0) {
		return err
	}
	return nil
}
