package spec

// UInt16BothByte is an unsigned 16-bit integer represented in both big endian and little endian, in a 32-bit
// integer container.
//
// The encoding is [ <little endian unsigned 16-bit integer>, <big endian unsigned 16-bit integer> ]
//
// UInt16BothByte can be encoded by the [struc] library.
type UInt16BothByte struct {
	Value uint32 `struc:"uint32,big"`
}

func (u UInt16BothByte) RealValue() uint16 {
	return uint16(u.Value & 0xFFFF)
}

// UInt32BothByte is an unsigned 32-bit integer represented in both big endian and little endian, in a 64-bit
// integer container.
//
// The encoding is [ <little endian unsigned 32-bit integer>, <big endian unsigned 32-bit integer> ]
//
// UInt32BothByte can be encoded by the [struc] library.
type UInt32BothByte struct {
	Value uint64 `struc:"uint64,big"`
}

func (u UInt32BothByte) RealValue() uint32 {
	return uint32(u.Value & 0xFFFFFFFF)
}
