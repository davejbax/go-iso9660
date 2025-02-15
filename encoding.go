package iso9660

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/lunixbochs/struc"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	errUnimplemented  = errors.New("method is not implemented")
	errBufferTooSmall = errors.New("provided slice buffer is not big enough to pack all data into")

	// TODO: export?
	errInvalidCharacters = errors.New("input string contains characters that violate encoding")
)

const fillerByte = 0x20

// a-characters are:
//
//	A B C D E F G H I J K L M N O P Q R S T U V W X Y Z 0 1 2 3 4 5 6 7 8 9 _
//	! " % & ' ( ) * + , - . / : ; < = > ?
type aCharacter uint8

var aCharacterRegex = regexp.MustCompile(`^[A-Z0-9_!"%&'()*+,\-./:;<=>?]+$`)

func strToACharacters(input string, output []aCharacter, strict bool, tryConvert bool) error {
	if tryConvert {
		input = strings.ToUpper(input)
	}

	if strict && !aCharacterRegex.MatchString(input) {
		return errInvalidCharacters
	}

	if len(output) < len(input) {
		return errBufferTooSmall
	}

	inputBytes := []aCharacter(input)
	copy(output, inputBytes)

	for i := len(inputBytes); i < len(output); i++ {
		output[i] = aCharacter(fillerByte)
	}

	return nil
}

// d-characters are:
// A B C D E F G H I J K L M N O P Q R S T U V W X Y Z 0 1 2 3 4 5 6 7 8 9 _
type dCharacter uint8

var dCharacterRegex = regexp.MustCompile(`^[A-Z0-9_]+$`)

func strToDCharacters(input string, output []dCharacter, strict bool, tryConvert bool) error {
	if len(input) == 0 {
		return nil
	}

	if tryConvert {
		input = strings.ToUpper(input)
	}

	if strict && !dCharacterRegex.MatchString(input) {
		return errInvalidCharacters
	}

	if len(output) < len(input) {
		return errBufferTooSmall
	}

	inputBytes := []dCharacter(input)
	copy(output, inputBytes)

	for i := len(inputBytes); i < len(output); i++ {
		output[i] = dCharacter(fillerByte)
	}

	// TODO: possibly fill with 0x20?
	return nil
}

func zeroCharacterArray[T dCharacter | aCharacter](array []T) {
	for i := 0; i < len(array); i++ {
		array[i] = T(fillerByte)
	}
}

// TODO: Joliet support, and/or general support for other escape sequences in the supplementary volume
// See ISO 2022 for escape sequences; notable ones are:
// - 25 2F 40/43/45 -- all UTF-16; required to be one of these for Joliet
// - all zeros for d-characters

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

type dateTime struct {
	YearsSince1900            uint8
	Month                     uint8
	Day                       uint8
	Hour                      uint8
	Minute                    uint8
	Second                    uint8
	GMTOffsetIn15MinIntervals uint8
}

func newDateTime(t time.Time) dateTime {
	t = t.UTC()
	return dateTime{
		YearsSince1900:            uint8(t.Year() - 1900),
		Month:                     uint8(t.Month()),
		Day:                       uint8(t.Day()),
		Hour:                      uint8(t.Hour()),
		Minute:                    uint8(t.Minute()),
		Second:                    uint8(t.Second()),
		GMTOffsetIn15MinIntervals: 0,
	}
}

type longDateTime struct {
	YearDigits                [4]uint8
	MonthDigits               [2]uint8
	DayDigits                 [2]uint8
	HourDigits                [2]uint8
	MinuteDigits              [2]uint8
	SecondDigits              [2]uint8
	CentisecondsDigits        [2]uint8
	GMTOffsetIn15MinIntervals uint8
}

var zeroLongDateTime = longDateTime{
	YearDigits:                [4]uint8{'0', '0', '0', '0'},
	MonthDigits:               [2]uint8{'0', '0'},
	DayDigits:                 [2]uint8{'0', '0'},
	HourDigits:                [2]uint8{'0', '0'},
	MinuteDigits:              [2]uint8{'0', '0'},
	SecondDigits:              [2]uint8{'0', '0'},
	CentisecondsDigits:        [2]uint8{'0', '0'},
	GMTOffsetIn15MinIntervals: 0,
}

type fileIdentifier []uint8

var (
	fileIdentifierSelf   = fileIdentifier{0x00}
	fileIdentifierParent = fileIdentifier{0x01}

	errUnsupportedEncoding = errors.New("unsupported file identifier encoding")
	errInvalidVersion      = errors.New("invalid file version number; must be in the range 1-32767 (inclusive)")
)

type fileIdentifierEncoding int

const (
	fileIdentifierEncodingDCharacter fileIdentifierEncoding = iota
)

func newFileIdentifier(filename string, extension string, version int, encoding fileIdentifierEncoding) (fileIdentifier, error) {
	if version < 1 || version > 32767 {
		return nil, errInvalidVersion
	}

	switch encoding {
	case fileIdentifierEncodingDCharacter:
		encodedFilename := make([]dCharacter, len(filename))
		if err := strToDCharacters(filename, encodedFilename, true, true); err != nil {
			return nil, fmt.Errorf("could not encode filename as d-characters: %w", err)
		}

		encodedExtension := make([]dCharacter, len(extension))
		if err := strToDCharacters(extension, encodedExtension, true, true); err != nil {
			return nil, fmt.Errorf("could not encode extension as d-characters: %w", err)
		}

		versionString := strconv.Itoa(version)

		fi := make(fileIdentifier, 0, len(encodedFilename)+1+len(encodedExtension)+1+len(versionString))

		for _, v := range encodedFilename {
			fi = append(fi, uint8(v))
		}

		if extension != "" {
			// Separator 1 (a period, before file extension)
			fi = append(fi, uint8('.'))

			for _, v := range encodedExtension {
				fi = append(fi, uint8(v))
			}

			// Separator 2 (a semicolon, before version)
			fi = append(fi, uint8(';'))

			for _, v := range versionString {
				fi = append(fi, uint8(v))
			}
		}

		return fi, nil
	default:
		return nil, errUnsupportedEncoding
	}
}
