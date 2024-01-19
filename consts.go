//go:build windows
// +build windows

package wintun

const (
	//  minimum ring capacity
	MinRingCapacity = 0x20000 /* 128kiB */

	// maximum ring capacity
	MaxRingCapacity = 0x4000000 /* 64MiB */
)
