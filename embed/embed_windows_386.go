package embed

import (
	_ "embed"

	"github.com/lysShub/go-dll"
)

//go:embed wintun_x86.dll
var DLL dll.MemDLL
