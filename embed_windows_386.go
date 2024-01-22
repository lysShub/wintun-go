package wintun

import (
	_ "embed"

	"github.com/lysShub/dll-go"
)

//go:embed embed/wintun_x86.dll
var DLL dll.MemDLL
