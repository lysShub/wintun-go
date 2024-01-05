package embed

import (
	_ "embed"
)

type MemDLL []byte

//go:embed wintun_amd64.dll
var Amd64 MemDLL

//go:embed wintun_arm.dll
var Arm MemDLL

//go:embed wintun_arm64.dll
var Arm64 MemDLL

//go:embed wintun_x86.dll
var X86 MemDLL
