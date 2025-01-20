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
