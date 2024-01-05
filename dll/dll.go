package dll

import (
	"sync"
	"unsafe"

	"github.com/lysShub/wintun-go/dll/embed"
	"golang.org/x/sys/windows"
	"golang.zx2c4.com/wireguard/windows/driver/memmod"
)

type DLL interface {
	Release() error
	FindProc(string) (*windows.Proc, error)
}

var _ DLL = (*windows.DLL)(nil)
var _ DLL = (*memDll)(nil)

type memDll memmod.Module

func (d *memDll) Release() error {
	((*memmod.Module)(d)).Free()
	return nil
}
func (d *memDll) FindProc(name string) (*windows.Proc, error) {
	ptr, err := ((*memmod.Module)(d)).ProcAddressByName(name)
	if err != nil {
		return nil, err
	}

	type dll struct {
		Dll  *DLL
		Name string
		addr uintptr
	}

	proc := &dll{Name: name, addr: ptr}
	return (*windows.Proc)(unsafe.Pointer(proc)), nil
}

type option func() (DLL, error)

func WithMemMod(data embed.MemDLL) option {
	return func() (DLL, error) {
		d, err := memmod.LoadLibrary(data)
		if err != nil {
			return nil, err
		}
		return (*memDll)(d), nil
	}
}

func WithFileMod(path string) option {
	return func() (DLL, error) {
		return windows.LoadDLL(path)
	}
}

func LoadDLL(dll option) error {
	var err error
	wintunDllLoadOnec.Do(func() {
		wintunDll, err = dll()
	})
	if err != nil {
		wintunDllLoadOnec = sync.Once{}
	} else {
		if wintunCreateAdapter, err = wintunDll.FindProc("WintunCreateAdapter"); err != nil {
			return err
		}
		if wintunOpenAdapter, err = wintunDll.FindProc("WintunOpenAdapter"); err != nil {
			return err
		}
		if wintunCloseAdapter, err = wintunDll.FindProc("WintunCloseAdapter"); err != nil {
			return err
		}
		if wintunDeleteDriver, err = wintunDll.FindProc("WintunDeleteDriver"); err != nil {
			return err
		}
		if wintunGetAdapterLuid, err = wintunDll.FindProc("WintunGetAdapterLuid"); err != nil {
			return err
		}
		if wintunGetRunningDriverVersion, err = wintunDll.FindProc("WintunGetRunningDriverVersion"); err != nil {
			return err
		}
		if wintunSetLogger, err = wintunDll.FindProc("WintunSetLogger"); err != nil {
			return err
		}
		if wintunStartSession, err = wintunDll.FindProc("WintunStartSession"); err != nil {
			return err
		}
		if wintunEndSession, err = wintunDll.FindProc("WintunEndSession"); err != nil {
			return err
		}
		if wintunGetReadWaitEvent, err = wintunDll.FindProc("WintunGetReadWaitEvent"); err != nil {
			return err
		}
		if wintunReceivePacket, err = wintunDll.FindProc("WintunReceivePacket"); err != nil {
			return err
		}
		if wintunReleaseReceivePacket, err = wintunDll.FindProc("WintunReleaseReceivePacket"); err != nil {
			return err
		}
		if wintunAllocateSendPacket, err = wintunDll.FindProc("WintunAllocateSendPacket"); err != nil {
			return err
		}
		if wintunSendPacket, err = wintunDll.FindProc("WintunSendPacket"); err != nil {
			return err
		}

	}
	return err
}
