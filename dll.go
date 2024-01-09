//go:build windows
// +build windows

package wintun

import (
	"sync"

	"github.com/lysShub/go-dll"
)

func LoadDLL[T string | dll.MemDLL](d T) error {
	var err error
	wintunDllLoadOnec.Do(func() {
		wintunDll, err = dll.LoadDLL(d)
		if err == nil {
			if wintunCreateAdapter, err = wintunDll.FindProc("WintunCreateAdapter"); err != nil {
				return
			}
			if wintunOpenAdapter, err = wintunDll.FindProc("WintunOpenAdapter"); err != nil {
				return
			}
			if wintunCloseAdapter, err = wintunDll.FindProc("WintunCloseAdapter"); err != nil {
				return
			}
			if wintunDeleteDriver, err = wintunDll.FindProc("WintunDeleteDriver"); err != nil {
				return
			}
			if wintunGetAdapterLuid, err = wintunDll.FindProc("WintunGetAdapterLUID"); err != nil {
				return
			}
			if wintunGetRunningDriverVersion, err = wintunDll.FindProc("WintunGetRunningDriverVersion"); err != nil {
				return
			}
			if wintunSetLogger, err = wintunDll.FindProc("WintunSetLogger"); err != nil {
				return
			}
			if wintunStartSession, err = wintunDll.FindProc("WintunStartSession"); err != nil {
				return
			}
			if wintunEndSession, err = wintunDll.FindProc("WintunEndSession"); err != nil {
				return
			}
			if wintunGetReadWaitEvent, err = wintunDll.FindProc("WintunGetReadWaitEvent"); err != nil {
				return
			}
			if wintunReceivePacket, err = wintunDll.FindProc("WintunReceivePacket"); err != nil {
				return
			}
			if wintunReleaseReceivePacket, err = wintunDll.FindProc("WintunReleaseReceivePacket"); err != nil {
				return
			}
			if wintunAllocateSendPacket, err = wintunDll.FindProc("WintunAllocateSendPacket"); err != nil {
				return
			}
			if wintunSendPacket, err = wintunDll.FindProc("WintunSendPacket"); err != nil {
				return
			}
		}
	})
	if err != nil {
		wintunDllLoadOnec = sync.Once{}
	}
	return err
}
