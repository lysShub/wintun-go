//go:build windows
// +build windows

package wintun

import (
	"fmt"
	"sync/atomic"
	"syscall"
	"unsafe"

	"github.com/lysShub/dll-go"
	"golang.org/x/sys/windows"
)

// todo: add ctx
type Wintun struct {
	wintunDll dll.DLL
	refs      atomic.Int32

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
	if t.refs.Load() > 0 {
		return fmt.Errorf("can't close wintun used by %d adapters", t.refs.Load())
	}

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

	t.refs.Add(1)
	return &Adapter{
		wintun: t,
		handle: r1,
	}, nil
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

	t.refs.Add(1)
	return &Adapter{
		wintun: t,
		handle: r1,
	}, nil
}

// DeleteDriver deletes the Wintun driver if there are no more adapters in use.
func (t *Wintun) DeleteDriver() error {
	if err := t.Close(); err != nil {
		return err
	}

	r1, _, err := syscall.SyscallN(t.wintunDeleteDriver)
	if r1 == 0 {
		return err
	}
	return nil
}
