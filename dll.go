package wintun

import (
	"golang.org/x/sys/windows"
	"golang.zx2c4.com/wireguard/windows/driver/memmod"
)

type dll interface {
	Release() error
	FindProc(string) (uintptr, error)
	MustFindProc(string) uintptr
}

type file windows.DLL

func loadFileDLL(path string) (dll, error) {
	dll, err := windows.LoadDLL(path)
	if err != nil {
		return nil, err
	}
	return (*file)(dll), nil
}

func (d *file) FindProc(name string) (uintptr, error) {
	p, err := ((*windows.DLL)(d)).FindProc(name)
	if err != nil {
		return 0, err
	}
	return p.Addr(), nil
}
func (d *file) MustFindProc(name string) uintptr {
	hdl, err := d.FindProc(name)
	if err != nil {
		panic(err)
	}
	return hdl
}
func (d *file) Release() error {
	return ((*windows.DLL)(d)).Release()
}

type mem memmod.Module

func loadMemDLL(data []byte) (dll, error) {
	d, err := memmod.LoadLibrary(data)
	if err != nil {
		return nil, err
	}
	return (*mem)(d), nil
}

func (d *mem) FindProc(name string) (uintptr, error) {
	return ((*memmod.Module)(d)).ProcAddressByName(name)
}
func (d *mem) MustFindProc(name string) uintptr {
	hdl, err := d.FindProc(name)
	if err != nil {
		panic(err)
	}
	return hdl
}
func (d *mem) Release() error {
	((*memmod.Module)(d)).Free()
	return nil
}
