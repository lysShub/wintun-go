package wintun

import (
	_ "embed"
)

//go:embed embed/wintun_arm.dll
var DLL Mem
