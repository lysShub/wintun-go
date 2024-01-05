package raw

import (
	"github.com/Microsoft/go-winio/pkg/guid"
)

// base type

type (
	WCHAR     = uint16
	DWORD     = uint32
	DWORDLONG = uint64
)

const (
// INVALID_HANDLE_VALUE = -1
)

type ()

var _ guid.GUID
