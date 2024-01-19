package embed

import (
	_ "embed"

	"github.com/lysShub/go-dll"
)

// https://www.wintun.net/builds/wintun-0.14.1.zip

//go:embed wintun_amd64.dll
var DLL dll.MemDLL
