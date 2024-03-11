package wintun

import "golang.org/x/sys/windows"

type options struct {
	tunType string
	guid    *windows.GUID
	ringCap int
}

func defaultOptions() *options {
	return &options{
		ringCap: MinRingCapacity,
	}
}

type Option func(*options)

func TunType(typ string) Option {
	return func(o *options) {
		o.tunType = typ
	}
}

func Guid(guid *windows.GUID) Option {
	return func(o *options) {
		o.guid = guid
	}
}

func RingBuff(size int) Option {
	return func(o *options) {
		o.ringCap = size
	}
}
