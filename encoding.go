package iso9660

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/lunixbochs/struc"
	"io"
)

var (
	errUnimplemented  = errors.New("method is not implemented")
	errBufferTooSmall = errors.New("provider byte slice is not big enough to pack into")
)

// a-characters are:
//
//	A B C D E F G H I J K L M N O P Q R S T U V W X Y Z 0 1 2 3 4 5 6 7 8 9 _
//	! " % & ' ( ) * + , - . / : ; < = > ?
type aCharacter uint8

// d-characters are:
// A B C D E F G H I J K L M N O P Q R S T U V W X Y Z 0 1 2 3 4 5 6 7 8 9 _
type dCharacter uint8

const (
	separator1 = 0x2E
	separator2 = 0x3B
)

var (
	// Identifier for the version of the ISO9660 standard used. Always 'CD001'
	standardIdentifier = [5]uint8{0x43, 0x44, 0x30, 0x30, 0x31}
)

func uint32BothByte(value uint32) uint32BothByteField {
	// Both representation of ST UV WX YZ is YZ WX UV ST ST UV WX YZ
	value64 := uint64(value)

	return uint32BothByteField(((value64 & 0xFF) << 56) |
		((value64 & 0xFF00) << 40) |
		((value64 & 0xFF0000) << 24) |
		((value64 & 0xFF000000) << 8) |
		value64)
}

func uint16BothByte(value uint16) uint16BothByteField {
	// Both byte representation of MS LS is LS MS MS LS
	value32 := uint32(value)
	return uint16BothByteField(((value32 & 0xFF) << 24) |
		((value32 & 0xFF00) << 8) |
		value32)
}

type uint16BothByteField uint32

var _ struc.Custom = uint16BothByteField(0)

func (f uint16BothByteField) Pack(p []byte, _ *struc.Options) (int, error) {
	if len(p) < 4 {
		return 0, errBufferTooSmall
	}

	buff := bytes.NewBuffer(make([]byte, 0, 4))
	if err := binary.Write(buff, binary.BigEndian, uint32(f)); err != nil {
		return 0, fmt.Errorf("failed to write uint32 to buffer: %w", err)
	}

	written := copy(p, buff.Bytes())
	if written != buff.Len() {
		// This should never happen: we ensured that p has 4 or more bytes; buff should have 4 bytes; copy()#
		// writes the minimum of these two lengths
		panic(fmt.Sprintf("unexpected number of bytes written: expected %d, got %d", buff.Len(), written))
	}

	// Should always be 4 bytes written
	return written, nil
}

func (f uint16BothByteField) Unpack(_ io.Reader, _ int, _ *struc.Options) error {
	return errUnimplemented
}

func (f uint16BothByteField) String() string {
	return "uint16BothByteField"
}

func (f uint16BothByteField) Size(_ *struc.Options) int {
	return 4
}

type uint32BothByteField uint64

var _ struc.Custom = uint32BothByteField(0)

func (f uint32BothByteField) Pack(p []byte, _ *struc.Options) (int, error) {
	if len(p) < 8 {
		return 0, errBufferTooSmall
	}

	buff := bytes.NewBuffer(make([]byte, 0, 8))
	if err := binary.Write(buff, binary.BigEndian, uint64(f)); err != nil {
		return 0, fmt.Errorf("failed to write uint64 to buffer: %w", err)
	}

	written := copy(p, buff.Bytes())
	if written != buff.Len() {
		// This should never happen: we ensured that p has 4 or more bytes; buff should have 4 bytes; copy()#
		// writes the minimum of these two lengths
		panic(fmt.Sprintf("unexpected number of bytes written: expected %d, got %d", buff.Len(), written))
	}

	// Should always be 8 bytes written
	return written, nil
}

func (f uint32BothByteField) Unpack(_ io.Reader, _ int, _ *struc.Options) error {
	return errUnimplemented
}

func (f uint32BothByteField) String() string {
	return "uint32BothByteField"
}

func (f uint32BothByteField) Size(_ *struc.Options) int {
	return 8
}
