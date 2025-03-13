package spec

// FillerByte is the character to be used as 'filler' in a-characters, d-characters, etc. This is only defined by the
// spec for PVDs and SVDs; an enhanced volume descriptor leaves the definition of 'filler' up to whoever.
//
// ECMA-119 (5th ed.) §8.4.3.2
const FillerByte = 0x20

// ACharacter is an 'a-character': a character from the following alphabet:
//
//	A B C D E F G H I J K L M N O P Q R S T U V W X Y Z 0 1 2 3 4 5 6 7 8 9 _
//	! " % & ' ( ) * + , - . / : ; < = > ?
//
// ECMA-119 (5th ed.) §8.4.1
type ACharacter uint8

// DCharacter is a 'd-character': a character from the following alphabet:
//
// A B C D E F G H I J K L M N O P Q R S T U V W X Y Z 0 1 2 3 4 5 6 7 8 9 _
//
// ECMA-119 (5th ed.) §8.4.1
type DCharacter uint8
