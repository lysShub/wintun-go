//go:build windows
// +build windows

package wintun

import (
	"github.com/lysShub/go-dll"
)

func LoadWintun[T string | dll.MemDLL](d T) (*Wintun, error) {
	var err error
	var t = &Wintun{}

	t.wintunDll, err = dll.LoadDLL(d)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			t.wintunDll.Release()
		}
	}()

	if t.wintunCreateAdapter, err = t.wintunDll.FindProc("WintunCreateAdapter"); err != nil {
		return nil, err
	}
	if t.wintunOpenAdapter, err = t.wintunDll.FindProc("WintunOpenAdapter"); err != nil {
		return nil, err
	}
	if t.wintunCloseAdapter, err = t.wintunDll.FindProc("WintunCloseAdapter"); err != nil {
		return nil, err
	}
	if t.wintunDeleteDriver, err = t.wintunDll.FindProc("WintunDeleteDriver"); err != nil {
		return nil, err
	}
	if t.wintunGetAdapterLuid, err = t.wintunDll.FindProc("WintunGetAdapterLUID"); err != nil {
		return nil, err
	}
	if t.wintunGetRunningDriverVersion, err = t.wintunDll.FindProc("WintunGetRunningDriverVersion"); err != nil {
		return nil, err
	}
	if t.wintunSetLogger, err = t.wintunDll.FindProc("WintunSetLogger"); err != nil {
		return nil, err
	}
	if t.wintunStartSession, err = t.wintunDll.FindProc("WintunStartSession"); err != nil {
		return nil, err
	}
	if t.wintunEndSession, err = t.wintunDll.FindProc("WintunEndSession"); err != nil {
		return nil, err
	}
	if t.wintunGetReadWaitEvent, err = t.wintunDll.FindProc("WintunGetReadWaitEvent"); err != nil {
		return nil, err
	}
	if t.wintunReceivePacket, err = t.wintunDll.FindProc("WintunReceivePacket"); err != nil {
		return nil, err
	}
	if t.wintunReleaseReceivePacket, err = t.wintunDll.FindProc("WintunReleaseReceivePacket"); err != nil {
		return nil, err
	}
	if t.wintunAllocateSendPacket, err = t.wintunDll.FindProc("WintunAllocateSendPacket"); err != nil {
		return nil, err
	}
	if t.wintunSendPacket, err = t.wintunDll.FindProc("WintunSendPacket"); err != nil {
		return nil, err
	}

	return t, nil
}
