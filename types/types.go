package types

import (
	"encoding/binary"
)

type NET_LUID struct {
	Value uint64
	Info  info
}

type info uint64

func (i *info) NetLuidIndex() uint32 {
	bs := binary.BigEndian.AppendUint64(nil, uint64(*i))

	return binary.BigEndian.Uint32(bs[3:6])
}
func (i *info) IfType() uint16 {
	bs := binary.BigEndian.AppendUint64(nil, uint64(*i))
	return binary.BigEndian.Uint16(bs[6:])
}
