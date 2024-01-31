//go:build windows
// +build windows

package wintun

import (
	"unsafe"

	"github.com/lysShub/dll-go"
	"golang.org/x/sys/windows"
)

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
func CreateAdapter(name string, opts ...Option) (*Adapter, error) {
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

	wintun.RLock()
	r1, _, err := syscallN(
		wintun.wintunCreateAdapter,
		uintptr(unsafe.Pointer(name16)),
		uintptr(unsafe.Pointer(tunnelType16)),
		uintptr(unsafe.Pointer(o.guid)),
	)
	wintun.RUnlock()
	if r1 == 0 {
		return nil, err
	}

	wintun.refs.Add(1)
	var a = &Adapter{
		handle: r1,
	}
	return a, a.init(uint32(o.ringCap))
}

// OpenAdapter opens an existing wintun adapter.
func OpenAdapter(name string) (*Adapter, error) {
	var name16 *uint16
	name16, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}

	wintun.RLock()
	r1, _, err := syscallN(wintun.wintunOpenAdapter, uintptr(unsafe.Pointer(name16)))
	if r1 == 0 {
		return nil, err
	}
	wintun.RUnlock()

	wintun.refs.Add(1)
	var a = &Adapter{
		handle: r1,
	}
	return a, a.init(MinRingCapacity)
}

// init starts Wintun session.
func (a *Adapter) init(capacity uint32) error {
	var err error

	wintun.RLock()
	a.session, _, err = syscallN(wintun.wintunStartSession, a.handle, uintptr(capacity))
	wintun.RUnlock()
	if a.session == 0 {
		return err
	}
	return nil
}

// DriverVersion determines the version of the Wintun driver currently loaded.
func DriverVersion() (version uint32, err error) {
	wintun.RLock()
	r0, _, err := syscallN(wintun.wintunGetRunningDriverVersion)
	wintun.RUnlock()
	if r0 == 0 {
		return 0, err
	}
	return uint32(r0), nil
}

// DeleteDriver deletes the Wintun driver if there are no more adapters in use.
func DeleteDriver() error {
	if wintun.refs.Load() > 0 {
		return dll.ERR_RELEASE_DLL_IN_USE
	}

	wintun.RLock()
	r1, _, err := syscallN(wintun.wintunDeleteDriver)
	wintun.RUnlock()
	if r1 == 0 {
		return err
	}
	return Release()
}
