package encode

import (
	"errors"
	"fmt"
	"github.com/davejbax/go-iso9660/internal/spec"
	"golang.org/x/text/encoding/unicode"
	"strconv"
	"strings"
)

var (
	ErrUnsupportedEncoding = errors.New("unsupported file identifier encoding")
	ErrInvalidVersion      = errors.New("invalid file version number; must be in the range 1-32767 (inclusive), or 0 to specify no version")
	ErrMissingExtension    = errors.New("filename does not have an extension; file extensions are mandatory when using no escape sequence")
)

// AsFileIdentifier attempts to convert a file or directory name into a file identifier.
// The encoding of file identifiers varies depending on whether the file appears in a directory
// tree pointed to by a Primary Volume Descriptor, or a Supplementary Volume Descriptor. In the
// latter case, there should be an associated [EscapeSequence], which indicates the character
// encoding to use.
//
// For files in PVDs, escapeSeq should be [EscapeSequenceNone]. For files in SVDs, escapeSeq
// should match the escape sequence specified in the supplementary volume descriptor.
//
// Version is the file version. If the file is a directory, this should be a zero value.
//
// Filename is the full file name and extension. Note that in PVDs, an extension is REQUIRED
// for files. In SVDs with the Joliet extension, this requirement is relaxed. Additionally,
// in PVDs, an extension on a directory is invalid (this again is relaxed in SVDs with Joliet).
// If escapeSeq is [EscapeSequenceNone] and the filename violates these requirements, encoding
// will fail and an error shall be returned.
func AsFileIdentifier(filename string, version int, escapeSeq EscapeSequence) (spec.FileIdentifier, error) {
	if version < 0 || version > 32767 {
		return nil, ErrInvalidVersion
	}

	switch escapeSeq {
	case EscapeSequenceNone:
		if version > 0 {
			// We're dealing with a _file_, not a directory. Files must have a version and extension.
			filenameWithoutExtension, extension, found := strings.Cut(filename, ".")

			if !found {
				return nil, ErrMissingExtension
			}

			encodedFilename := make([]spec.DCharacter, len(filenameWithoutExtension))
			if err := AsDCharacters(filenameWithoutExtension, encodedFilename, true, true); err != nil {
				return nil, fmt.Errorf("could not encode filename as d-characters: %w", err)
			}

			encodedExtension := make([]spec.DCharacter, len(extension))
			if err := AsDCharacters(extension, encodedExtension, true, true); err != nil {
				return nil, fmt.Errorf("could not encode extension as d-characters: %w", err)
			}

			encodedVersion := strconv.Itoa(version)

			fi := make(spec.FileIdentifier, 0, len(encodedFilename)+1+len(encodedExtension)+1+len(encodedVersion))
			fi = append(fi, encodedFilename...)
			fi = append(fi, '.')
			fi = append(fi, encodedExtension...)
			fi = append(fi, ';')
			fi = append(fi, encodedVersion...)

			return fi, nil
		} else {
			// Directories have no extension or version, and hence we try to encode it directly.

			encodedFilename := make([]spec.DCharacter, len(filename))
			if err := AsDCharacters(filename, encodedFilename, true, true); err != nil {
				return nil, fmt.Errorf("could not encode directory name as d-characters: %w", err)
			}

			return encodedFilename, nil
		}

	case EscapeSequenceUCS2Level1:
		encoder := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewEncoder()

		// TODO: check for invalid bytes in filename for Joliet

		fi, err := encoder.Bytes([]byte(filename))
		if err != nil {
			return nil, fmt.Errorf("could not encode filename: %w", err)
		}

		if version > 0 {
			fi = append(fi, uint8(0x00), uint8(';'))

			versionString := strconv.Itoa(version)
			encodedVersion, err := encoder.Bytes([]byte(versionString))
			if err != nil {
				return nil, fmt.Errorf("could not encode version: %w", err)
			}

			fi = append(fi, encodedVersion...)
		}

		return fi, nil

	default:
		return nil, ErrUnsupportedEncoding
	}
}
