package iso9660

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

func uint16BothByte(value uint16) uint32 {
	// Both byte representation of MS LS is LS MS MS LS
	return ((uint32(value) & 0xFF00) << 8) | ((uint32(value) & 0xFF) << 24) | uint32(value)
}

func uint32BothByte(value uint32) uint64 {
	// Both representation of ST UV WX YZ is YZ WX UV ST ST UV WX YZ
	value64 := uint64(value)

	return ((value64 & 0xFF) << 56) |
		((value64 & 0xFF00) << 40) |
		((value64 & 0xFF0000) << 24) |
		((value64 & 0xFF000000) << 8) |
		value64
}
