//go:build windows
// +build windows

package wintun

import (
	"sync"
	"sync/atomic"

	"github.com/lysShub/dll-go"
)

var wintun = struct {
	refs atomic.Int32

	sync.RWMutex
	dll dll.DLL

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
}{}

func MustLoad[T string | dll.MemDLL](d T) {
	if err := Load(d); err != nil {
		panic(err)
	}
}

func Load[T string | dll.MemDLL](d T) (err error) {
	wintun.Lock()
	defer wintun.Unlock()
	defer func() {
		if err != nil {
			resetWintunUnLock()
		}
	}()

	if wintun.dll != nil {
		return nil
	} else {
		wintun.dll, err = dll.LoadDLL(d)
		if err != nil {
			return err
		}

		if wintun.wintunCreateAdapter, err = wintun.dll.FindProc("WintunCreateAdapter"); err != nil {
			return err
		}
		if wintun.wintunOpenAdapter, err = wintun.dll.FindProc("WintunOpenAdapter"); err != nil {
			return err
		}
		if wintun.wintunCloseAdapter, err = wintun.dll.FindProc("WintunCloseAdapter"); err != nil {
			return err
		}
		if wintun.wintunDeleteDriver, err = wintun.dll.FindProc("WintunDeleteDriver"); err != nil {
			return err
		}
		if wintun.wintunGetAdapterLuid, err = wintun.dll.FindProc("WintunGetAdapterLUID"); err != nil {
			return err
		}
		if wintun.wintunGetRunningDriverVersion, err = wintun.dll.FindProc("WintunGetRunningDriverVersion"); err != nil {
			return err
		}
		if wintun.wintunSetLogger, err = wintun.dll.FindProc("WintunSetLogger"); err != nil {
			return err
		}
		if wintun.wintunStartSession, err = wintun.dll.FindProc("WintunStartSession"); err != nil {
			return err
		}
		if wintun.wintunEndSession, err = wintun.dll.FindProc("WintunEndSession"); err != nil {
			return err
		}
		if wintun.wintunGetReadWaitEvent, err = wintun.dll.FindProc("WintunGetReadWaitEvent"); err != nil {
			return err
		}
		if wintun.wintunReceivePacket, err = wintun.dll.FindProc("WintunReceivePacket"); err != nil {
			return err
		}
		if wintun.wintunReleaseReceivePacket, err = wintun.dll.FindProc("WintunReleaseReceivePacket"); err != nil {
			return err
		}
		if wintun.wintunAllocateSendPacket, err = wintun.dll.FindProc("WintunAllocateSendPacket"); err != nil {
			return err
		}
		if wintun.wintunSendPacket, err = wintun.dll.FindProc("WintunSendPacket"); err != nil {
			return err
		}
	}

	return nil
}

func Release() error {
	wintun.Lock()
	defer wintun.Unlock()

	if wintun.dll == nil {
		return dll.ERR_RELEASE_DLL_NOT_LOAD
	}
	if wintun.refs.Load() > 0 {
		return dll.ERR_RELEASE_DLL_IN_USE
	}

	if err := wintun.dll.Release(); err != nil {
		return err
	}

	resetWintunUnLock()
	return nil
}

func resetWintunUnLock() {
	wintun.refs.Store(0)
	// wintun.RWMutex
	wintun.dll = nil
	wintun.wintunCreateAdapter = 0
	wintun.wintunOpenAdapter = 0
	wintun.wintunCloseAdapter = 0
	wintun.wintunDeleteDriver = 0
	wintun.wintunGetAdapterLuid = 0
	wintun.wintunGetRunningDriverVersion = 0
	wintun.wintunSetLogger = 0
	wintun.wintunStartSession = 0
	wintun.wintunEndSession = 0
	wintun.wintunGetReadWaitEvent = 0
	wintun.wintunReceivePacket = 0
	wintun.wintunReleaseReceivePacket = 0
	wintun.wintunAllocateSendPacket = 0
	wintun.wintunSendPacket = 0
}
