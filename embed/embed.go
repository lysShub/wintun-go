package embed

import (
	_ "embed"

	"github.com/lysShub/go-dll"
)

// https://www.wintun.net/builds/wintun-0.14.1.zip

//go:embed wintun_amd64.dll
var Amd64 dll.MemDLL

//go:embed wintun_arm.dll
var Arm dll.MemDLL

//go:embed wintun_arm64.dll
var Arm64 dll.MemDLL

//go:embed wintun_x86.dll
var X86 dll.MemDLL
