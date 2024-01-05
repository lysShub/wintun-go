package raw

import (
	"errors"
	"fmt"
	"path"
	"time"

	"golang.org/x/sys/windows"
)

type DevInfoDataList struct {
	Data windows.DevInfoData
	Next *DevInfoDataList
}

var GUID_DEVCLASS_NET = windows.GUID{Data1: 0x4d36e972, Data2: 0xe325, Data3: 0x11ce, Data4: [8]byte{0xbf, 0xc1, 0x08, 0x00, 0x2b, 0xe1, 0x03, 0x18}}

func DisableAllOurAdapters(devInfo windows.DevInfo, disabledAdapters DevInfoDataList) bool {
	return false
}

func MaybeGetRunningDriverVersion(ReturnOneIfRunningInsteadOfVersion bool) DWORD {
	return 0
}

func EnsureWintunUnloaded() bool {
	loaded := false
	for tries := 0; tries < 1500; tries++ {
		if tries > 0 {
			time.Sleep(time.Millisecond * 50)
		}
		loaded = MaybeGetRunningDriverVersion(true) != 0
		if !loaded {
			break
		}
	}
	return !loaded
}

func ResourceCreateTemporaryDirectory() string {
	return ""
}

func DriverInstall(
	DevInfoExistingAdaptersForCleanup uintptr,
	ExistingAdaptersForCleanup **DevInfoDataList,
) error {

	var OurDriverData windows.Filetime // todo: set inf
	var OurDriverVersion DWORDLONG

	// todo: namespace

	devInfo, err := windows.SetupDiCreateDeviceInfoListEx(&GUID_DEVCLASS_NET, 0, "")
	if err != nil {
		return err
	}
	defer windows.SetupDiDestroyDeviceInfoList(devInfo)

	devInfoData, err := windows.SetupDiCreateDeviceInfo(
		devInfo,
		WINTUN_HWID,
		&GUID_DEVCLASS_NET,
		"",
		0,
		windows.DICD_GENERATE_ID,
	)
	if err != nil {
		return err
	}

	if err := windows.SetupDiSetDeviceRegistryProperty(
		devInfo,
		devInfoData,
		windows.SPDRP_HARDWAREID,
		[]byte(WINTUN_HWID),
	); err != nil {
		return err
	}

	if err := windows.SetupDiBuildDriverInfoList(devInfo, devInfoData, windows.SPDIT_COMPATDRIVER); err != nil {
		return err
	}

	var (
		driverData              windows.Filetime
		driverVersion           DWORDLONG
		devInfoExistingAdapters windows.DevInfo = windows.DevInfo(windows.InvalidHandle)
		existingAdapters        *DevInfoDataList
	)
	for enumIndx := 0; ; enumIndx++ {

		// todo: 参数对不上
		drvInfoData, err := windows.SetupDiEnumDriverInfo(devInfo, devInfoData, windows.SPDIT_COMPATDRIVER, enumIndx)
		if err != nil {
			if errors.Is(windows.GetLastError(), windows.ERROR_NO_MORE_ITEMS) {
				break
			} else {
				continue
			}
		}
		if IsNewer(OurDriverData, OurDriverVersion, drvInfoData.DriverDate, drvInfoData.DriverVersion) {

			if devInfoExistingAdapters == windows.DevInfo(windows.InvalidHandle) {
				devInfoExistingAdapters, err = windows.SetupDiGetClassDevsEx(
					&GUID_DEVCLASS_NET,
					WINTUN_ENUMERATOR,
					0, windows.DIGCF_PRESENT, 0, "",
				)
				if err != nil {
					return err
				}

				DisableAllOurAdapters(devInfoExistingAdapters, *existingAdapters)

				if !EnsureWintunUnloaded() {
					fmt.Println("Failed to unload existing driver, which means a reboot will likely be required")
				}

				DrvInfoDetailData, err := windows.SetupDiGetDriverInfoDetail(devInfo, devInfoData, drvInfoData)
				if err != nil {
					fmt.Println("Failed getting adapter driver info detail")
					continue
				}

				InfFileName := path.Base(DrvInfoDetailData.InfFileName())
				if err := windows.SetupUninstallOEMInf(InfFileName, windows.SUOI_FORCEDELETE); err != nil {
					fmt.Sprintf("Unable to remove existing driver %s", InfFileName)
				} else {
					continue
				}
			}

			if !IsNewer(drvInfoData.DriverDate, drvInfoData.DriverVersion, driverData, driverVersion) {
				continue
			}
		}

	}
	windows.SetupDiDestroyDriverInfoList(devInfo, devInfoData, windows.SPDIT_COMPATDRIVER)

	if driverVersion > 0 {
		fmt.Println("Using existing driver")
		return nil
	}

	fmt.Println("Installing driver")
	RandomTempSubDirectory := ResourceCreateTemporaryDirectory()
	if RandomTempSubDirectory == "" {
		fmt.Println("Failed to create temporary folder")
	}

	// todo: 一堆啥玩意

	return nil
}

func IsNewer(
	driverDate1 windows.Filetime,
	driverVersion1 DWORDLONG,
	driverDate2 windows.Filetime,
	driverVersion2 DWORDLONG,
) bool {
	if driverDate1.HighDateTime > driverDate2.HighDateTime {
		return true
	}
	if driverDate1.HighDateTime < driverDate2.HighDateTime {
		return false
	}

	if driverDate1.LowDateTime > driverDate2.LowDateTime {
		return true
	}
	if driverDate1.LowDateTime < driverDate2.LowDateTime {
		return false
	}

	if driverVersion1 > driverVersion2 {
		return true
	}
	if driverVersion1 < driverVersion2 {
		return false
	}

	return false
}
