package encode

import (
	"errors"
	"fmt"
	"github.com/davejbax/go-iso9660/internal/spec"
	"strconv"
)

type FileIdentifierEncoding int

const (
	FileIdentifierEncodingDCharacter FileIdentifierEncoding = iota
)

var (
	ErrUnsupportedEncoding = errors.New("unsupported file identifier encoding")
	ErrInvalidVersion      = errors.New("invalid file version number; must be in the range 1-32767 (inclusive)")
)

func AsFileIdentifier(filename string, extension string, version int, encoding FileIdentifierEncoding) (spec.FileIdentifier, error) {
	if version < 1 || version > 32767 {
		return nil, ErrInvalidVersion
	}

	switch encoding {
	case FileIdentifierEncodingDCharacter:
		encodedFilename := make([]spec.DCharacter, len(filename))
		if err := AsDCharacters(filename, encodedFilename, true, true); err != nil {
			return nil, fmt.Errorf("could not encode filename as d-characters: %w", err)
		}

		encodedExtension := make([]spec.DCharacter, len(extension))
		if err := AsDCharacters(extension, encodedExtension, true, true); err != nil {
			return nil, fmt.Errorf("could not encode extension as d-characters: %w", err)
		}

		versionString := strconv.Itoa(version)

		fi := make(spec.FileIdentifier, 0, len(encodedFilename)+1+len(encodedExtension)+1+len(versionString))

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
		return nil, ErrUnsupportedEncoding
	}
}
