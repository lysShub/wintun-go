//go:build windows
// +build windows

package wintun

import (
	"os"
	"sync"
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
	"golang.org/x/sys/windows"
)

var global = wintun{}

func MustLoad[T string | MemMode](p T) struct{} {
	err := Load(p)
	if err != nil {
		panic(err)
	}
	return struct{}{}
}

func Load[T string | MemMode](p T) error {
	global.Lock()
	defer global.Unlock()
	if global.dll != nil {
		return errors.WithStack(ErrLoaded{})
	}

	var err error
	switch p := any(p).(type) {
	case string:
		global.dll, err = loadFileDLL(p)
		if err != nil {
			return errors.WithStack(err)
		}
	case MemMode:
		global.dll, err = loadMemDLL(DLL)
		if err != nil {
			return errors.WithStack(err)
		}
	default:
		return windows.ERROR_INVALID_PARAMETER
	}

	err = global.init()
	return errors.WithStack(err)
}

type ErrLoaded struct{}

func (e ErrLoaded) Error() string {
	return "wintun loaded"
}

func Release() error {
	global.Lock()
	defer global.Unlock()
	if global.dll == nil {
		return nil
	}

	err := global.dll.Release()
	global.dll = nil
	return errors.WithStack(err)
}

type wintun struct {
	sync.RWMutex
	dll dll

	procCreateAdapter           uintptr
	procOpenAdapter             uintptr
	procCloseAdapter            uintptr
	procDeleteDriver            uintptr
	procGetAdapterLuid          uintptr
	procGetRunningDriverVersion uintptr
	procSetLogger               uintptr
	procStartSession            uintptr
	procEndSession              uintptr
	procGetReadWaitEvent        uintptr
	procReceivePacket           uintptr
	procReleaseReceivePacket    uintptr
	procAllocateSendPacket      uintptr
	procSendPacket              uintptr
}

func (w *wintun) init() (err error) {
	if global.procCreateAdapter, err = global.dll.FindProc("WintunCreateAdapter"); err != nil {
		goto ret
	}
	if global.procOpenAdapter, err = global.dll.FindProc("WintunOpenAdapter"); err != nil {
		goto ret
	}
	if global.procCloseAdapter, err = global.dll.FindProc("WintunCloseAdapter"); err != nil {
		goto ret
	}
	if global.procDeleteDriver, err = global.dll.FindProc("WintunDeleteDriver"); err != nil {
		goto ret
	}
	if global.procGetAdapterLuid, err = global.dll.FindProc("WintunGetAdapterLUID"); err != nil {
		goto ret
	}
	if global.procGetRunningDriverVersion, err = global.dll.FindProc("WintunGetRunningDriverVersion"); err != nil {
		goto ret
	}
	if global.procSetLogger, err = global.dll.FindProc("WintunSetLogger"); err != nil {
		goto ret
	}
	if global.procStartSession, err = global.dll.FindProc("WintunStartSession"); err != nil {
		goto ret
	}
	if global.procEndSession, err = global.dll.FindProc("WintunEndSession"); err != nil {
		goto ret
	}
	if global.procGetReadWaitEvent, err = global.dll.FindProc("WintunGetReadWaitEvent"); err != nil {
		goto ret
	}
	if global.procReceivePacket, err = global.dll.FindProc("WintunReceivePacket"); err != nil {
		goto ret
	}
	if global.procReleaseReceivePacket, err = global.dll.FindProc("WintunReleaseReceivePacket"); err != nil {
		goto ret
	}
	if global.procAllocateSendPacket, err = global.dll.FindProc("WintunAllocateSendPacket"); err != nil {
		goto ret
	}
	if global.procSendPacket, err = global.dll.FindProc("WintunSendPacket"); err != nil {
		goto ret
	}

ret:
	if err != nil {
		w.dll.Release()
		w.dll = nil
	}
	return err
}

func (w *wintun) calln(trap uintptr, args ...uintptr) (r1, r2 uintptr, err error) {
	w.RLock()
	defer w.RUnlock()

	if w.dll == nil {
		return 0, 0, os.ErrClosed
	}

	var e syscall.Errno
	r1, r2, e = syscall.SyscallN(trap, args...)

	return r1, r2, e
}

func CreateAdapter(name string, opts ...Option) (*Adapter, error) {
	var o = defaultOptions()
	for _, fn := range opts {
		fn(o)
	}

	name16, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	tunnelType16, err := windows.UTF16PtrFromString(o.tunType)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	r1, _, err := global.calln(
		global.procCreateAdapter,
		uintptr(unsafe.Pointer(name16)),
		uintptr(unsafe.Pointer(tunnelType16)),
		uintptr(unsafe.Pointer(o.guid)),
	)
	if r1 == 0 {
		return nil, errors.WithStack(err)
	}
	ap := &Adapter{handle: r1}
	return ap, ap.Start(uint32(o.ringCap))
}

func OpenAdapter(name string) (*Adapter, error) {
	var name16 *uint16
	name16, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	r1, _, err := global.calln(global.procOpenAdapter, uintptr(unsafe.Pointer(name16)))
	if r1 == 0 {
		return nil, errors.WithStack(err)
	}
	ap := &Adapter{handle: r1}
	return ap, ap.Start(uint32(MinRingCapacity))
}

func DriverVersion() (version uint32, err error) {
	r0, _, err := global.calln(global.procGetRunningDriverVersion)
	if r0 == 0 {
		return 0, errors.WithStack(err)
	}
	return uint32(r0), nil
}

func DeleteDriver() error {
	r1, _, err := global.calln(global.procDeleteDriver)
	if r1 == 0 {
		return errors.WithStack(err)
	}
	return nil
}
