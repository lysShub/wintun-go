package wintun

import (
	_ "embed"
)

//go:embed embed/wintun_x86.dll
var DLL MemMode
