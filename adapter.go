//go:build windows
// +build windows

package wintun

import (
	"context"
	"sync"

	"github.com/pkg/errors"

	"unsafe"

	"golang.org/x/sys/windows"
	"golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"
)

type Adapter struct {
	sync.RWMutex

	handle  uintptr
	session uintptr
}

type ErrAdapterClosed struct{}

func (ErrAdapterClosed) Error() string { return "adapter closed" }

type ErrAdapterStoped struct{}

func (ErrAdapterStoped) Error() string { return "adapter stoped" }

func (a *Adapter) sessionLocked(trap uintptr, args ...uintptr) (r1, r2 uintptr, err error) {
	if a.handle == 0 {
		return 0, 0, errors.WithStack(ErrAdapterClosed{})
	} else if a.session == 0 {
		return 0, 0, errors.WithStack(ErrAdapterStoped{})
	}
	return global.calln(trap, append([]uintptr{a.session}, args...)...)
}

func (a *Adapter) Start(capacity uint32) (err error) {
	if capacity < MinRingCapacity || MaxRingCapacity < capacity {
		return errors.New("invalid ring buff capacity")
	}
	a.Lock()
	defer a.Unlock()

	if a.handle == 0 {
		return errors.WithStack(ErrAdapterClosed{})
	}
	fd, _, err := global.calln(
		global.procStartSession,
		a.handle,
		uintptr(capacity),
	)
	if err != nil {
		return err
	}
	a.session = fd
	return nil
}

func (a *Adapter) Stop() error {
	a.Lock()
	defer a.Unlock()
	return a.stopLocked()
}

func (a *Adapter) stopLocked() error {
	if a.session > 0 {
		_, _, err := global.calln(global.procEndSession, uintptr(a.session))
		if err != nil {
			return err
		}
		a.session = 0
	}
	return nil
}

func (a *Adapter) Close() error {
	a.Lock()
	defer a.Unlock()

	if a.handle > 0 {
		err := a.stopLocked()
		if err != nil {
			return err
		}

		_, _, err = global.calln(global.procCloseAdapter, a.handle)
		if err != nil {
			return err
		}
		a.handle = 0
	}
	return nil
}

func (a *Adapter) GetAdapterLuid() (winipcfg.LUID, error) {
	a.RLock()
	defer a.RUnlock()

	if a.handle == 0 {
		return 0, errors.WithStack(ErrAdapterClosed{})
	}
	var luid uint64
	_, _, err := global.calln(
		global.procGetAdapterLuid,
		a.handle,
		uintptr(unsafe.Pointer(&luid)),
	)
	if err != nil {
		return 0, err
	}
	return winipcfg.LUID(luid), nil
}

func (a *Adapter) Index() (int, error) {
	luid, err := a.GetAdapterLuid()
	if err != nil {
		return 0, err
	}

	row, err := luid.Interface()
	if err != nil {
		return 0, err
	}
	return int(row.InterfaceIndex), nil
}

func (a *Adapter) getReadWaitEvent() (windows.Handle, error) {
	r0, _, err := a.sessionLocked(global.procGetReadWaitEvent)
	if r0 == 0 {
		return 0, err
	}
	return windows.Handle(r0), nil
}

type rpack []byte

// Recv receive outbound(income adapter) ip packet, after must call ap.Release(p)
func (a *Adapter) Recv(ctx context.Context) (ip rpack, err error) {
	a.RLock()
	defer a.RUnlock()

	var size uint32
	for {
		r0, _, err := a.sessionLocked(
			global.procReceivePacket,
			(uintptr)(unsafe.Pointer(&size)),
		)
		if r0 == 0 {
			if errors.Is(err, windows.ERROR_NO_MORE_ITEMS) {
				hdl, err := a.getReadWaitEvent()
				if err != nil {
					return nil, errors.WithStack(err)
				}
				e, err := windows.WaitForSingleObject(hdl, 100)
				if err != nil {
					return nil, errors.WithStack(err)
				}
				switch e {
				case windows.WAIT_OBJECT_0:
				case uint32(windows.WAIT_TIMEOUT):
					select {
					case <-ctx.Done():
						return nil, errors.WithStack(ctx.Err())
					default:
					}
				default:
					return nil, errors.Errorf("invalid WaitForSingleObject result %d", e)
				}
				continue
			} else {
				return nil, err
			}
		}

		ptr := unsafe.Add(nil, r0)
		return unsafe.Slice((*byte)(ptr), size), nil
	}
}

func (a *Adapter) Release(p rpack) error {
	if len(p) == 0 {
		return nil
	}
	a.RLock()
	defer a.RUnlock()

	_, _, err := a.sessionLocked(
		global.procReleaseReceivePacket,
		uintptr(unsafe.Pointer(&p[0])),
	)
	return err
}

type spack []byte

func (a *Adapter) Alloc(size int) (spack, error) {
	if size == 0 {
		return spack{}, nil
	}
	a.RLock()
	defer a.RUnlock()

	r0, _, err := a.sessionLocked(
		global.procAllocateSendPacket,
		uintptr(size),
	)
	if r0 == 0 {
		return nil, err
	}

	p := (*byte)(unsafe.Add(*new(unsafe.Pointer), r0))
	return unsafe.Slice(p, size), nil
}

// Send send inbound(outgoing adapter) ip packet, ip must alloc by AllocPacket
func (a *Adapter) Send(ip spack) error {
	if len(ip) == 0 {
		return nil
	}
	a.RLock()
	defer a.RUnlock()

	_, _, err := a.sessionLocked(
		global.procSendPacket,
		uintptr(unsafe.Pointer(&ip[0])),
	)
	return err
}
