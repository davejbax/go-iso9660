package encode

import (
	"errors"
	"github.com/davejbax/go-iso9660/internal/spec"
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

// ZeroCharacterArray zeros an array of a- or d-characters, where 'zeroing' means 'fill with the filler byte'
// ([spec.FillerByte]).
func ZeroCharacterArray[T spec.DCharacter | spec.ACharacter](array []T) {
	for i := 0; i < len(array); i++ {
		array[i] = T(spec.FillerByte)
	}
}
