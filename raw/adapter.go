package raw

import (
	"fmt"

	"golang.org/x/sys/windows"
)

const WINTUN_HWID = "Wintun"

// #define WINTUN_ENUMERATOR (IsWindows7 ? L"ROOT\\" WINTUN_HWID : L"SWD\\" WINTUN_HWID)
const WINTUN_ENUMERATOR = "SWD\\"

func init() { // todo
	windows.GetVersion()
}

type Adapter struct {
	SwDevice          uintptr // todo: HSWDEVICE
	DevInfo           windows.DevInfo
	DevInfoData       windows.DevInfoData
	InterfaceFilename *WCHAR
	CfgInstanceID     windows.GUID
	DevInstanceID     [windows.MAX_DEVICE_ID_LEN]WCHAR
	LuidIndex         DWORD
	IfType            DWORD
	IfIndex           DWORD
}

func CreateAdapter(name string, tunnelType string, requestedGUID *windows.GUID) (*Adapter, error) {

	var adapter *Adapter

	// todo: namespace
	if err := DriverInstall(0, nil); err != nil {

	}

	fmt.Println(adapter)

	return nil, nil
}
