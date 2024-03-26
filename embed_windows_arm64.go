package wintun

import (
	_ "embed"
)

//go:embed embed/wintun_arm64.dll
var DLL Mem
