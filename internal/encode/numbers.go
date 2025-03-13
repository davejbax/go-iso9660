package encode

import (
	"github.com/davejbax/go-iso9660/internal/spec"
)

// AsUInt16BothByte creates a [UInt16BothByte] from a unsigned 16-bit integer
func AsUInt16BothByte(value uint16) spec.UInt16BothByte {
	// Both byte representation of MS LS is LS MS MS LS
	value32 := uint32(value)
	return spec.UInt16BothByte{
		Value: ((value32 & 0xFF) << 24) |
			((value32 & 0xFF00) << 8) |
			value32,
	}
}

// AsUInt32BothByte creates a [UInt32BothByte] from a unsigned 32-bit integer
func AsUInt32BothByte(value uint32) spec.UInt32BothByte {
	// Both representation of ST UV WX YZ is YZ WX UV ST ST UV WX YZ
	value64 := uint64(value)

	return spec.UInt32BothByte{
		Value: ((value64 & 0xFF) << 56) |
			((value64 & 0xFF00) << 40) |
			((value64 & 0xFF0000) << 24) |
			((value64 & 0xFF000000) << 8) |
			value64,
	}
}
