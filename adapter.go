//go:build windows
// +build windows

package wintun

import (
	"context"

	"github.com/pkg/errors"

	"unsafe"

	"golang.org/x/sys/windows"
	"golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"
)

type Adapter struct {
	handle  uintptr
	session uintptr
}

func (a *Adapter) Start(capacity uint32) (err error) {
	a.session, _, err = global.calln(
		global.procStartSession,
		a.handle,
		uintptr(capacity),
	)
	if a.session == 0 {
		return errors.WithStack(err)
	}
	return nil
}

func (a *Adapter) Stop() error {
	_, _, err := global.calln(global.procEndSession, uintptr(a.session))
	if err != windows.ERROR_SUCCESS {
		return errors.WithStack(err)
	}
	return nil
}

func (a *Adapter) Close() (err error) {
	err = a.Stop()
	if err != nil {
		return errors.WithStack(err)
	}

	_, _, err = global.calln(global.procCloseAdapter, a.handle)
	if err != windows.ERROR_SUCCESS {
		return err
	}
	return nil
}

func (a *Adapter) GetAdapterLuid() (winipcfg.LUID, error) {
	var luid uint64
	_, _, err := global.calln(
		global.procGetAdapterLuid,
		a.handle,
		uintptr(unsafe.Pointer(&luid)),
	)
	if err != windows.ERROR_SUCCESS {
		return 0, errors.WithStack(err)
	}
	return winipcfg.LUID(luid), nil
}

func (a *Adapter) Index() (int, error) {
	luid, err := a.GetAdapterLuid()
	if err != nil {
		return 0, errors.WithStack(err)
	}

	row, err := luid.Interface()
	if err != nil {
		return 0, errors.WithStack(err)
	}
	return int(row.InterfaceIndex), nil
}

func (s *Adapter) getReadWaitEvent() (windows.Handle, error) {
	r0, _, err := global.calln(global.procGetReadWaitEvent, uintptr(s.session))
	if r0 == 0 {
		return 0, errors.WithStack(err)
	}
	return windows.Handle(r0), nil
}

type Packet []byte

func (a *Adapter) Receive(ctx context.Context) (rp Packet, err error) {
	var size uint32
	for {
		r0, _, err := global.calln(
			global.procReceivePacket,
			a.session,
			(uintptr)(unsafe.Pointer(&size)),
		)
		if r0 == 0 {
			if errors.Is(err, windows.ERROR_NO_MORE_ITEMS) {
				hdl, err := a.getReadWaitEvent() // todo: store this handle?
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

func (a *Adapter) AllocPacket(packetSize uint32) (Packet, error) {
	r0, _, err := global.calln(
		global.procAllocateSendPacket,
		uintptr(a.session),
		uintptr(packetSize),
	)
	if r0 == 0 {
		return nil, err
	}

	p := (*byte)(unsafe.Add(*new(unsafe.Pointer), r0))
	return unsafe.Slice(p, packetSize), nil
}

func (a *Adapter) Send(p Packet) error {
	_, _, err := global.calln(
		global.procSendPacket,
		uintptr(a.session),
		uintptr(unsafe.Pointer(&p[0])),
	)
	if err != windows.ERROR_SUCCESS {
		return errors.WithStack(err)
	}
	return nil
}

func (a Adapter) ReleasePacket(p Packet) error {
	_, _, err := global.calln(
		global.procReleaseReceivePacket,
		uintptr(a.session),
		uintptr(unsafe.Pointer(&p[0])),
	)
	if err != windows.ERROR_SUCCESS {
		return errors.WithStack(err)
	}
	return nil
}
