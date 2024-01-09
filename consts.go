//go:build windows
// +build windows

package wintun

const (

	//  minimum ring capacity
	WINTUN_MIN_RING_CAPACITY = 0x20000 /* 128kiB */

	// maximum ring capacity
	WINTUN_MAX_RING_CAPACITY = 0x4000000 /* 64MiB */

	// maximum IP packet size
	WINTUN_MAX_IP_PACKET_SIZE = 0xFFFF
)
