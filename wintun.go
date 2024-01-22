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

type options struct {
	tunType string
	guid    *windows.GUID
	ringCap int
}

type Option func(*options)

func TunType(typ string) Option {
	return func(o *options) {
		o.tunType = typ
	}
}

func Guid(guid *windows.GUID) Option {
	return func(o *options) {
		o.guid = guid
	}
}

// RingBuff ring capacity: rings capacity, must be between MIN_RING_CAPACITY and MAX_RING_CAPACITY,
// must be a power of two.
func RingBuff(size int) Option {
	return func(o *options) {
		o.ringCap = size
	}
}

// CreateAdapter creates a new wintun adapter.
func (t *Wintun) CreateAdapter(name string, opts ...Option) (*Adapter, error) {
	name16, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}

	var o = &options{
		ringCap: MinRingCapacity,
	}
	for _, fn := range opts {
		fn(o)
	}

	tunnelType16, err := windows.UTF16PtrFromString(o.tunType)
	if err != nil {
		return nil, err
	}
	r1, _, err := syscall.SyscallN(
		t.wintunCreateAdapter,
		uintptr(unsafe.Pointer(name16)),
		uintptr(unsafe.Pointer(tunnelType16)),
		uintptr(unsafe.Pointer(o.guid)),
	)
	if r1 == 0 {
		return nil, err
	}

	t.refs.Add(1)
	var a = &Adapter{
		wintun: t,
		handle: r1,
	}
	return a, a.init(uint32(o.ringCap))
}

// OpenAdapter opens an existing wintun adapter.
func (t *Wintun) OpenAdapter(name string) (*Adapter, error) {
	var name16 *uint16
	name16, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}
	r1, _, err := syscall.SyscallN(t.wintunOpenAdapter, uintptr(unsafe.Pointer(name16)))
	if r1 == 0 {
		return nil, err
	}

	t.refs.Add(1)
	var a = &Adapter{
		wintun: t,
		handle: r1,
	}
	return a, a.init(MinRingCapacity)
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
