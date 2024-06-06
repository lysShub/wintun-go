package wintun

import (
	_ "embed"
)

// from https://www.wintun.net/builds/wintun-0.14.1.zip
//
//go:embed embed/wintun_amd64.dll
var DLL Mem
