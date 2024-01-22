//go:build windows
// +build windows

package wintun

import (
	"github.com/lysShub/dll-go"
)

func LoadWintun[T string | dll.MemDLL](d T) (*Wintun, error) {
	var tun = &Wintun{}
	var err error

	tun.wintunDll, err = dll.LoadDLL(d)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			tun.Close()
		}
	}()
	if tun.wintunCreateAdapter, err = tun.wintunDll.FindProc("WintunCreateAdapter"); err != nil {
		return nil, err
	}
	if tun.wintunOpenAdapter, err = tun.wintunDll.FindProc("WintunOpenAdapter"); err != nil {
		return nil, err
	}
	if tun.wintunCloseAdapter, err = tun.wintunDll.FindProc("WintunCloseAdapter"); err != nil {
		return nil, err
	}
	if tun.wintunDeleteDriver, err = tun.wintunDll.FindProc("WintunDeleteDriver"); err != nil {
		return nil, err
	}
	if tun.wintunGetAdapterLuid, err = tun.wintunDll.FindProc("WintunGetAdapterLUID"); err != nil {
		return nil, err
	}
	if tun.wintunGetRunningDriverVersion, err = tun.wintunDll.FindProc("WintunGetRunningDriverVersion"); err != nil {
		return nil, err
	}
	if tun.wintunSetLogger, err = tun.wintunDll.FindProc("WintunSetLogger"); err != nil {
		return nil, err
	}
	if tun.wintunStartSession, err = tun.wintunDll.FindProc("WintunStartSession"); err != nil {
		return nil, err
	}
	if tun.wintunEndSession, err = tun.wintunDll.FindProc("WintunEndSession"); err != nil {
		return nil, err
	}
	if tun.wintunGetReadWaitEvent, err = tun.wintunDll.FindProc("WintunGetReadWaitEvent"); err != nil {
		return nil, err
	}
	if tun.wintunReceivePacket, err = tun.wintunDll.FindProc("WintunReceivePacket"); err != nil {
		return nil, err
	}
	if tun.wintunReleaseReceivePacket, err = tun.wintunDll.FindProc("WintunReleaseReceivePacket"); err != nil {
		return nil, err
	}
	if tun.wintunAllocateSendPacket, err = tun.wintunDll.FindProc("WintunAllocateSendPacket"); err != nil {
		return nil, err
	}
	if tun.wintunSendPacket, err = tun.wintunDll.FindProc("WintunSendPacket"); err != nil {
		return nil, err
	}

	return tun, nil
}
