package dll

import (
	"errors"
	"syscall"
	"unsafe"

	"github.com/lysShub/wintun-go/types"
	"golang.org/x/sys/windows"
)

type Adapter struct {
	handle uintptr
}

// Close releases Wintun adapter resources and, if adapter was created with WintunCreateAdapter, removes adapter.
func (a *Adapter) Close() error {
	syscall.SyscallN(wintunCloseAdapter.Addr(), a.handle)
	return nil
}

// GetAdapterLuid returns the LUID of the adapter.
func (a *Adapter) GetAdapterLuid() *types.NET_LUID {
	var luid = &types.NET_LUID{}
	syscall.SyscallN(wintunGetAdapterLuid.Addr(), a.handle, uintptr(unsafe.Pointer(&luid)))
	return luid
}

type Session uintptr

// StartSession starts Wintun session.
//
//	capacity: rings capacity, must be between WINTUN_MIN_RING_CAPACITY and WINTUN_MAX_RING_CAPACITY, must be a power of two.
func (a *Adapter) StartSession(capacity uint32) (Session, error) {
	r0, _, _ := syscall.SyscallN(wintunStartSession.Addr(), a.handle, uintptr(capacity))
	if r0 == 0 {
		return 0, windows.GetLastError()
	}
	return Session(r0), nil
}

// Close ends Wintun session.
func (s Session) Close() error {
	syscall.SyscallN(wintunEndSession.Addr(), uintptr(s))
	return nil
}

func (s Session) getReadWaitEvent() windows.Handle {
	r0, _, _ := syscall.SyscallN(wintunGetReadWaitEvent.Addr(), uintptr(s))
	return windows.Handle(r0)
}

/* func (s Session) ReceivePacket(b []byte) (n int, err error) {
	var size uint32
	r0, _, e := syscall.SyscallN(wintunReceivePacket.Addr(), uintptr(s), (uintptr)(unsafe.Pointer(&size)))
	if r0 == 0 {
		return 0, e
	}
	var p = (*byte)(unsafe.Add(*new(unsafe.Pointer), r0))

	copy(b, unsafe.Slice(p, min(len(b), int(size))))
	if n < len(b) {
		s.releaseReceivePacket(p)
		return n, io.ErrShortBuffer
	}

	return n, s.releaseReceivePacket(p)
} */

type RawPacket struct {
	session Session
	IP      []byte
}

// ReceivePacket Retrieves one or packet. After the packet content is consumed, call WintunReleaseReceivePacket with Packet returned from this function to release internal buffer. This function is thread-safe.
//
//	If the function fails, possible errors include the following:
//	 *         ERROR_HANDLE_EOF     Wintun adapter is terminating;
//	 *         ERROR_NO_MORE_ITEMS  Wintun buffer is exhausted;
//	 *         ERROR_INVALID_DATA   Wintun buffer is corrupt
func (s Session) ReceivePacket() (rp *RawPacket, err error) {
	var size uint32
	for {
		r0, _, _ := syscall.SyscallN(wintunReceivePacket.Addr(), uintptr(s), (uintptr)(unsafe.Pointer(&size)))
		if r0 == 0 {
			e := windows.GetLastError()
			if errors.Is(e, windows.ERROR_NO_MORE_ITEMS) {
				s.getReadWaitEvent()
				continue
			} else {
				return nil, e
			}
		}

		var p unsafe.Pointer
		return &RawPacket{
			session: s,
			IP:      (unsafe.Slice((*byte)(unsafe.Add(p, r0)), size)),
		}, nil
	}
}

// AllocateSendPacket allocates memory for a packet to send, is thread-safe and the WintunAllocateSendPacket order of calls define the packet sending order.
//
//	If the function fails, possible errors include the following:
//	 *         ERROR_HANDLE_EOF       Wintun adapter is terminating;
//	 *         ERROR_BUFFER_OVERFLOW  Wintun buffer is full;
func (s Session) AllocateSendPacket(packetSize uint32) (*RawPacket, error) {
	r0, _, _ := syscall.SyscallN(wintunAllocateSendPacket.Addr(), uintptr(s), uintptr(packetSize))
	if r0 == 0 {
		return nil, windows.GetLastError()
	}

	var p = (*byte)(unsafe.Add(*new(unsafe.Pointer), r0))
	return &RawPacket{
		session: s,
		IP:      unsafe.Slice(p, packetSize),
	}, nil
}

// SendPacket sends the packet and releases internal buffer.  is thread-safe, but the AllocateSendPacket order of calls define the packet sending order. this means the packet is not guaranteed to be sent in the SendPacket yet.
func (s *RawPacket) SendPacket() {
	syscall.SyscallN(
		wintunSendPacket.Addr(),
		uintptr(s.session),
		uintptr(unsafe.Pointer(&s.IP[0])),
	)
	s.Release()
}

// Release releases internal buffer after the received packet has been processed by the client. this function is thread-safe.
func (p *RawPacket) Release() {
	syscall.SyscallN(
		wintunReleaseReceivePacket.Addr(),
		uintptr(p.session),
		uintptr(unsafe.Pointer(&p.IP[0])),
	)
}
