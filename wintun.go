//go:build windows
// +build windows

package wintun

import (
	"syscall"
	"unsafe"

	ddll "github.com/lysShub/divert-go/dll"
	"github.com/pkg/errors"
	"golang.org/x/sys/windows"
)

func MustLoad[T string | Mem](p T) struct{} {
	err := Load(p)
	if err != nil {
		panic(err)
	}
	return struct{}{}
}

func Load[T string | Mem](p T) error {
	if wintun.Loaded() {
		return ErrLoaded{}
	}

	switch p := any(p).(type) {
	case string:
		ddll.ResetLazyDll(wintun, p)
	case Mem:
		ddll.ResetLazyDll(wintun, p)
	default:
		panic("")
	}
	return nil
}

var (
	wintun = ddll.NewLazyDLL("wintun.dll")

	procCreateAdapter           = wintun.NewProc("CreateAdapter")
	procOpenAdapter             = wintun.NewProc("OpenAdapter")
	procCloseAdapter            = wintun.NewProc("CloseAdapter")
	procDeleteDriver            = wintun.NewProc("DeleteDriver")
	procGetAdapterLuid          = wintun.NewProc("GetAdapterLuid")
	procGetRunningDriverVersion = wintun.NewProc("GetRunningDriverVersion")
	procSetLogger               = wintun.NewProc("SetLogger")
	procStartSession            = wintun.NewProc("StartSession")
	procEndSession              = wintun.NewProc("EndSession")
	procGetReadWaitEvent        = wintun.NewProc("GetReadWaitEvent")
	procReceivePacket           = wintun.NewProc("ReceivePacket")
	procReleaseReceivePacket    = wintun.NewProc("ReleaseReceivePacket")
	procAllocateSendPacket      = wintun.NewProc("AllocateSendPacket")
	procSendPacket              = wintun.NewProc("SendPacket")
)

func CreateAdapter(name string, opts ...Option) (*Adapter, error) {
	if len(name) == 0 {
		return nil, errors.New("require adapter name")
	}

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

	r1, _, err := syscall.SyscallN(
		procCreateAdapter.Addr(),
		uintptr(unsafe.Pointer(name16)),
		uintptr(unsafe.Pointer(tunnelType16)),
		uintptr(unsafe.Pointer(o.guid)),
	)
	if err != windows.ERROR_SUCCESS {
		return nil, errors.WithStack(err)
	}
	ap := &Adapter{handle: r1}
	return ap, ap.Start(o.ringBuff)
}

func OpenAdapter(name string) (*Adapter, error) {
	if len(name) == 0 {
		return nil, errors.New("require adapter name")
	}

	var name16 *uint16
	name16, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	r1, _, err := syscall.SyscallN(procOpenAdapter.Addr(), uintptr(unsafe.Pointer(name16)))
	if err != windows.ERROR_SUCCESS {
		return nil, errors.WithStack(err)
	}
	ap := &Adapter{handle: r1}
	return ap, ap.Start(MinRingCapacity)
}

// todo: https://git.zx2c4.com/wintun-go/tree/wintun.go
func DriverVersion() (version uint32, err error) {
	r0, _, err := syscall.SyscallN(procGetRunningDriverVersion.Addr())
	if err != windows.ERROR_SUCCESS {
		return 0, errors.WithStack(err)
	}
	return uint32(r0), nil
}

func DeleteDriver() error {
	_, _, err := syscall.SyscallN(procDeleteDriver.Addr())
	if err != windows.ERROR_SUCCESS {
		return errors.WithStack(err)
	}
	return nil
}
