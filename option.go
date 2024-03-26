package wintun

import "golang.org/x/sys/windows"

type options struct {
	tunType  string
	guid     *windows.GUID
	ringBuff uint32
}

func defaultOptions() *options {
	return &options{
		ringBuff: MinRingCapacity,
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

func RingBuff(size uint32) Option {
	return func(o *options) {
		o.ringBuff = size
	}
}
