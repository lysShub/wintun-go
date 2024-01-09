//go:build windows
// +build windows

package wintun

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type Adapter struct {
	handle uintptr
}

// Close releases Wintun adapter resources and, if adapter was created with CreateAdapter, removes adapter.
func (a *Adapter) Close() error {
	_, _, err := syscall.SyscallN(wintunCloseAdapter, a.handle)
	if err != syscall.Errno(0) {
		return err
	}
	return nil
}

// GetAdapterLuid returns the LUID of the adapter.
func (a *Adapter) GetAdapterLuid() (uint64, error) {
	var luid uint64
	_, _, err := syscall.SyscallN(wintunGetAdapterLuid, a.handle, uintptr(unsafe.Pointer(&luid)))
	if err != syscall.Errno(0) {
		return 0, err
	}
	return luid, nil
}

var (
	modiphlpapi                     = windows.NewLazySystemDLL("iphlpapi.dll")
	procConvertInterfaceLuidToIndex = modiphlpapi.NewProc("ConvertInterfaceLuidToIndex")
)

func (a *Adapter) InterfaceIndex() (int, error) {
	luid, err := a.GetAdapterLuid()
	if err != nil {
		return -1, err
	}

	var idx uint32
	_, _, err = syscall.SyscallN(
		procConvertInterfaceLuidToIndex.Addr(),
		uintptr(unsafe.Pointer(&luid)),
		uintptr(unsafe.Pointer(&idx)),
	)
	if err != syscall.Errno(0) {
		return -1, err
	}
	return int(idx), nil
}

type Session uintptr

// StartSession starts Wintun session.
//
//	capacity: rings capacity, must be between MIN_RING_CAPACITY and MAX_RING_CAPACITY, must be a power of two.
func (a *Adapter) StartSession(capacity uint32) (Session, error) {
	r1, _, err := syscall.SyscallN(wintunStartSession, a.handle, uintptr(capacity))
	if r1 == 0 {
		return 0, err
	}
	return Session(r1), nil
}

// Close ends Wintun session.
func (s Session) Close() error {
	_, _, err := syscall.SyscallN(wintunEndSession, uintptr(s))
	if err != syscall.Errno(0) {
		return err
	}
	return nil
}

func (s Session) getReadWaitEvent() (windows.Handle, error) {
	r0, _, err := syscall.SyscallN(wintunGetReadWaitEvent, uintptr(s))
	if err != syscall.Errno(0) {
		return windows.InvalidHandle, err
	}

	return windows.Handle(r0), nil
}

type Packet []byte

// ReceivePacket Retrieves one or packet. After the packet content is consumed,
// call Release with Packet returned from this function to release internal buffer.
// This function is thread-safe.
//
//	If the function fails, possible errors include the following:
//	 *         ERROR_HANDLE_EOF     Wintun adapter is terminating;
//	 *         ERROR_NO_MORE_ITEMS  Wintun buffer is exhausted;
//	 *         ERROR_INVALID_DATA   Wintun buffer is corrupt
func (s Session) ReceivePacket() (rp Packet, err error) {
	var size uint32
	for {
		r0, _, err := syscall.SyscallN(wintunReceivePacket, uintptr(s), (uintptr)(unsafe.Pointer(&size)))
		if r0 == 0 {
			if err == windows.ERROR_NO_MORE_ITEMS {
				hdl, err := s.getReadWaitEvent()
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
func (s Session) AllocateSendPacket(packetSize uint32) (Packet, error) {
	r0, _, err := syscall.SyscallN(wintunAllocateSendPacket, uintptr(s), uintptr(packetSize))
	if r0 == 0 {
		return nil, err
	}

	p := (*byte)(unsafe.Add(*new(unsafe.Pointer), r0))
	return unsafe.Slice(p, packetSize), nil
}

// SendPacket sends the packet and releases internal buffer.
// is thread-safe, but the AllocateSendPacket order of calls define
// the packet sending order. this means the packet is not guaranteed to be sent in the SendPacket yet.
func (s Session) SendPacket(p Packet) error {
	_, _, err := syscall.SyscallN(
		wintunSendPacket,
		uintptr(s),
		uintptr(unsafe.Pointer(&p[0])),
	)
	if err != syscall.Errno(0) {
		return err
	}
	return nil
}

// Release releases internal buffer after the received packet has been processed by the client.
// this function is thread-safe.
func (s Session) Release(p Packet) error {
	_, _, err := syscall.SyscallN(
		wintunReleaseReceivePacket,
		uintptr(s),
		uintptr(unsafe.Pointer(&p[0])),
	)
	if err != syscall.Errno(0) {
		return err
	}
	return nil
}
