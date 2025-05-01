package spec

// FillerByte is the character to be used as 'filler' in a-characters, d-characters, etc. This is only defined by the
// spec for PVDs and SVDs; an enhanced volume descriptor leaves the definition of 'filler' up to whoever.
//
// ECMA-119 (5th ed.) §8.4.3.2
const FillerByte = uint8(0x20)

// ACharacter is an 'a-character': a character from the following alphabet:
//
//	A B C D E F G H I J K L M N O P Q R S T U V W X Y Z 0 1 2 3 4 5 6 7 8 9 _
//	! " % & ' ( ) * + , - . / : ; < = > ?
//
// ECMA-119 (5th ed.) §8.4.1
type ACharacter = uint8

// DCharacter is a 'd-character': a character from the following alphabet:
//
// A B C D E F G H I J K L M N O P Q R S T U V W X Y Z 0 1 2 3 4 5 6 7 8 9 _
//
// ECMA-119 (5th ed.) §8.4.1
type DCharacter = uint8

// CCharacterByte is a single byte in a character encoding scheme conveyed by the
// charset specified by the supplementary volume descriptor's escape sequence.
// This character set can have characters comprising multiple bytes, and hence the
// distinction of calling this type a _byte_: this is the lowest common denominator
// of all possible escape sequences.
//
// One consequence of this is that an array of CCharacterByte-s of length n does NOT
// imply n characters: simply that the encoded characters must fit in a byte array
// of length n. For UCS-2, the number of characters would be n / 2.
//
// Technically, a1-characters and d1-characters also exist in the specification
// as subsets of c-characters, however the spec does not specify what this subset
// should be (this is left up to an agreement between the originator and recipient).
// Hence, for simplicity, we do not make a distinction here.
//
// ECMA-119 (5th ed.) §8.4.2
type CCharacterByte = uint8
