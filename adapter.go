//go:build windows
// +build windows

package wintun

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"
)

type Adapter struct {
	wintun *Wintun

	handle  uintptr
	session uintptr
}

// init starts Wintun session.
func (a *Adapter) init(capacity uint32) error {
	var err error
	a.session, _, err = syscall.SyscallN(a.wintun.wintunStartSession, a.handle, uintptr(capacity))
	if a.session == 0 {
		return err
	}
	return nil
}

// Close releases Wintun adapter resources and, if adapter was created with CreateAdapter, removes adapter.
func (a *Adapter) Close() error {
	_, _, err := syscall.SyscallN(a.wintun.wintunEndSession, uintptr(a.session))
	if err != syscall.Errno(0) {
		return err
	}

	_, _, err = syscall.SyscallN(a.wintun.wintunCloseAdapter, a.handle)
	if err != syscall.Errno(0) {
		return err
	}

	a.wintun.refs.Add(-1)
	return nil
}

// GetAdapterLuid returns the LUID of the adapter.
func (a *Adapter) GetAdapterLuid() (winipcfg.LUID, error) {
	var luid uint64
	_, _, err := syscall.SyscallN(a.wintun.wintunGetAdapterLuid, a.handle, uintptr(unsafe.Pointer(&luid)))
	if err != syscall.Errno(0) {
		return 0, err
	}
	return winipcfg.LUID(luid), nil
}

func (a *Adapter) InterfaceIndex() (int, error) {
	luid, err := a.GetAdapterLuid()
	if err != nil {
		return -1, err
	}

	row, err := luid.Interface()
	if err != nil {
		return -1, err
	}
	return int(row.InterfaceIndex), nil
}

func (s *Adapter) getReadWaitEvent() (windows.Handle, error) {
	r0, _, err := syscall.SyscallN(s.wintun.wintunGetReadWaitEvent, uintptr(s.session))
	if err != syscall.Errno(0) {
		return windows.InvalidHandle, err
	}

	return windows.Handle(r0), nil
}

type Packet []byte

// ReceivePacket receives one outbound packet. After the packet content is consumed,
// call Release with Packet returned from this function to release internal buffer.
// This function is thread-safe.
//
//	If the function fails, possible errors include the following:
//	 *         ERROR_HANDLE_EOF     Wintun adapter is terminating;
//	 *         ERROR_NO_MORE_ITEMS  Wintun buffer is exhausted;
//	 *         ERROR_INVALID_DATA   Wintun buffer is corrupt
func (a *Adapter) ReceivePacket() (rp Packet, err error) {
	var size uint32
	for {
		r0, _, err := syscall.SyscallN(a.wintun.wintunReceivePacket, uintptr(a.session), (uintptr)(unsafe.Pointer(&size)))
		if r0 == 0 {
			if err == windows.ERROR_NO_MORE_ITEMS {
				hdl, err := a.getReadWaitEvent() // todo: store this handle?
				if err != nil {
					return nil, err
				}
				event, err := windows.WaitForSingleObject(hdl, windows.INFINITE)
				if err != nil {
					return nil, err
				} else if event != windows.WAIT_OBJECT_0 {
					return nil, fmt.Errorf("call WaitForSingleObject enent %d, error %s", event, windows.GetLastError())
				}
				continue
			} else {
				return nil, err
			}
		}

		return unsafe.Slice((*byte)(unsafe.Add(*new(unsafe.Pointer), r0)), size), nil
	}
}

// AllocateSendPacket allocates memory for a packet to send, is thread-safe and
// the AllocateSendPacket order of calls define the packet sending order.
//
//	If the function fails, possible errors include the following:
//	 *         ERROR_HANDLE_EOF       Wintun adapter is terminating;
//	 *         ERROR_BUFFER_OVERFLOW  Wintun buffer is full;
func (a *Adapter) AllocateSendPacket(packetSize uint32) (Packet, error) {
	r0, _, err := syscall.SyscallN(a.wintun.wintunAllocateSendPacket, uintptr(a.session), uintptr(packetSize))
	if r0 == 0 {
		return nil, err
	}

	p := (*byte)(unsafe.Add(*new(unsafe.Pointer), r0))
	return unsafe.Slice(p, packetSize), nil
}

// SendPacket sends one inbound packet and releases internal buffer.
// is thread-safe, but the AllocateSendPacket order of calls define
// the packet sending order. this means the packet is not guaranteed to be sent in the SendPacket yet.
func (a *Adapter) SendPacket(p Packet) error {
	_, _, err := syscall.SyscallN(
		a.wintun.wintunSendPacket,
		uintptr(a.session),
		uintptr(unsafe.Pointer(&p[0])),
	)
	if err != syscall.Errno(0) {
		return err
	}
	return nil
}

// ReleasePacket releases internal buffer after the received packet has been processed by the client.
// this function is thread-safe.
func (a Adapter) ReleasePacket(p Packet) error {
	_, _, err := syscall.SyscallN(
		a.wintun.wintunReleaseReceivePacket,
		uintptr(a.session),
		uintptr(unsafe.Pointer(&p[0])),
	)
	if err != syscall.Errno(0) {
		return err
	}
	return nil
}
