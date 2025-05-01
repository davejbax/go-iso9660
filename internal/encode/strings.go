package encode

import (
	"errors"
	"fmt"
	"github.com/davejbax/go-iso9660/internal/spec"
	"golang.org/x/text/encoding/unicode"
	"regexp"
	"strings"
)

var (
	// ErrBufferTooSmall indicates that the given output byte buffer is not large enough to hold the encoded result
	ErrBufferTooSmall = errors.New("provided slice buffer is not big enough to hold encoded result")

	// ErrInvalidCharacters indicates that the input string contains character incompatible with the given encoding.
	// Note that in non-strict mode, this error should not be thrown.
	ErrInvalidCharacters = errors.New("input string contains characters that violate encoding")
)

var aCharacterRegex = regexp.MustCompile(`^[A-Z0-9_!"%&'()*+,\-./:;<=>?]+$`)

var dCharacterRegex = regexp.MustCompile(`^[A-Z0-9_]+$`)

// AsACharacters converts an input string to a slice of [spec.ACharacter]. If strict is true, [ErrInvalidCharacters]
// will be returned if the input string contains characters that cannot be represented directly with a-characters.
// If tryConvert is true, characters that can be represented with minor conversions (e.g. uppercasing) will be
// converted.
func AsACharacters(input string, output []spec.ACharacter, strict bool, tryConvert bool) error {
	// TODO: this means that empty strings won't be filled with the filler byte!
	if len(input) == 0 {
		return nil
	}

	if tryConvert {
		input = strings.ToUpper(input)
	}

	if strict && !aCharacterRegex.MatchString(input) {
		return ErrInvalidCharacters
	}

	if len(output) < len(input) {
		return ErrBufferTooSmall
	}

	inputBytes := []spec.ACharacter(input)
	copy(output, inputBytes)

	for i := len(inputBytes); i < len(output); i++ {
		output[i] = spec.ACharacter(spec.FillerByte)
	}

	return nil
}

// AsDCharacters converts an input string to a slice of [spec.DCharacter]. If strict is true, [ErrInvalidCharacters]
// will be returned if the input string contains characters that cannot be represented directly with d-characters.
// If tryConvert is true, characters that can be represented with minor conversions (e.g. uppercasing) will be
// converted.
func AsDCharacters(input string, output []spec.DCharacter, strict bool, tryConvert bool) error {
	if len(input) == 0 {
		return nil
	}

	if tryConvert {
		input = strings.ToUpper(input)
		// TODO: try converting by stripping out invalid chars
	}

	if strict && !dCharacterRegex.MatchString(input) {
		return ErrInvalidCharacters
	}

	if len(output) < len(input) {
		return ErrBufferTooSmall
	}

	inputBytes := []spec.DCharacter(input)
	copy(output, inputBytes)

	for i := len(inputBytes); i < len(output); i++ {
		output[i] = spec.DCharacter(spec.FillerByte)
	}

	return nil
}

type EscapeSequence int

const (
	EscapeSequenceNone EscapeSequence = iota
	EscapeSequenceUCS2Level1
)

func (e EscapeSequence) Identifier() []byte {
	switch e {
	case EscapeSequenceNone:
		return nil
	case EscapeSequenceUCS2Level1:
		return []byte{0x25, 0x2F, 0x40} // "%\@"
	}

	panic("invalid escape sequence")
}

// ZeroCharacterArray zeros an array of characters ([spec.ACharacter], [spec.DCharacter], or [spec.CCharacterByte]),
// where 'zeroing' means 'fill with the filler byte' ([spec.FillerByte]). This zeroing process varies depending on the
// escape sequence in supplementary volume descriptors. If the target character array is of a-characters or d-characters
// or in a primary volume descriptor, escapeSeq should be [EscapeSequenceNone].
func ZeroCharacterArray(array []uint8, escapeSeq EscapeSequence) {
	switch escapeSeq {
	case EscapeSequenceNone:
		for i := 0; i < len(array); i++ {
			array[i] = spec.FillerByte
		}

	case EscapeSequenceUCS2Level1:
		// Set every even byte to 0x20, so that the array looks like { 00 20 00 20 00 20 ... }
		for i := 1; i < len(array); i += 2 {
			array[i] = spec.FillerByte
		}
	}
}

// AsCCharacters encodes an input string according to the encoding specified by the given escape sequence. This is
// used for strings that appear in supplementary volume descriptors.
//
// Output buffers will be filled with filler bytes according to the escape sequence (e.g. for UCS-2, filler bytes are
// {0x00, 0x20}) if the input string is not long enough to fill the buffer.
//
// If the input string is too long, an error will be returned.
func AsCCharacters(input string, output []spec.CCharacterByte, escapeSeq EscapeSequence) error {
	switch escapeSeq {
	case EscapeSequenceNone:
		if len(input) > len(output) {
			return ErrBufferTooSmall
		}

		written := copy(output, []byte(input))

		for i := written; i < len(output); i++ {
			output[i] = spec.FillerByte
		}

	case EscapeSequenceUCS2Level1:
		// XXX: UTF-16 isn't *really* UCS-2, and in fact is variable width (there can be 4-byte characters). However,
		// it's proooobably fine.
		encoder := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewEncoder()

		written, _, err := encoder.Transform(output, []byte(input), true)
		if err != nil {
			return fmt.Errorf("failed to transform input into UTF-16: %w", err)
		}

		if written%2 != 0 {
			panic(fmt.Sprintf("unexpected number of characters written to buffer: %d", written))
		}

		// Fill the remainder of the array with {0x00, 0x20}. If the array is odd-sized (this is possible! the spec
		// includes some odd-sized string fields), then simply leave the last byte as a zero (null terminator).
		//
		// Note: ECMA-119 briefly mentions that this filler changes to 0x00 in Joliet, but I haven't been able to find
		// a supporting reference for that -- and none of the ISOs I've generated or looked at seem to follow that.
		for i := written + 1; i < len(output); i += 2 {
			output[i] = spec.FillerByte
		}

		return nil
	}

	panic("unsupported escape sequence")
}
