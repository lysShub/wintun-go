package embed

import (
	_ "embed"

	"github.com/lysShub/go-dll"
)

//go:embed wintun_arm.dll
var DLL dll.MemDLL
