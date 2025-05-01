package encode_test

import (
	"errors"
	"github.com/davejbax/go-iso9660/internal/encode"
	"github.com/davejbax/go-iso9660/internal/spec"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAsFileIdentifier_PVD(t *testing.T) {
	cases := []struct {
		name          string
		filename      string
		version       uint
		expected      spec.FileIdentifier
		expectedError error
	}{
		{
			name:     "Spec-compliant input filename",
			filename: "VALIDFILE.TXT",
			version:  1,
			expected: spec.FileIdentifier("VALIDFILE.TXT;1"),
		},
		{
			name:     "Spec-compliant input filename with version >1",
			filename: "VALIDFILE.TXT",
			version:  5,
			expected: spec.FileIdentifier("VALIDFILE.TXT;5"),
		},
		{
			name:     "Spec-compliant input directory",
			filename: "DIRECTORY",
			version:  0,
			expected: spec.FileIdentifier("DIRECTORY"),
		},
		{
			name:     "Should be able to convert nearly-correct filename (convert to uppercase)",
			filename: "soMEfiLe.tXt",
			version:  1,
			expected: spec.FileIdentifier("SOMEFILE.TXT;1"),
		},
		{
			name:     "Should convert uncorrectable characters to underscores",
			filename: "f;-! oo bar.tx?t",
			version:  1,
			expected: spec.FileIdentifier("F____OO_BAR.TXT_T;1"),
		},
		{
			name:     "Should convert uncorrectable characters to underscores in directory names",
			filename: "f;-! oo bar.tx?t;1",
			version:  0,
			expected: spec.FileIdentifier("F____OO_BAR_TXT_T_1"),
		},
		{
			name:     "Missing file extension should still add separator 1 to file identifier",
			filename: "NOFILEEXTENSION",
			version:  1,
			expected: spec.FileIdentifier("NOFILEEXTENSION.;1"),
		},
		{
			name:     "Directory with extension should be transformed into compliant identifier",
			filename: "DIRECTORY.THING",
			version:  0,
			expected: spec.FileIdentifier("DIRECTORY_THING"),
		},
		{
			name:     "Filename+extensions less than or equal to 30 characters should remain unmodified",
			filename: "SOMEVERYLONGFILENAMEITHINK.DOCX",
			version:  1,
			expected: spec.FileIdentifier("SOMEVERYLONGFILENAMEITHINK.DOCX;1"),
		},
		{
			name:     "Filename+extensions longer than 30 characters should truncate the filename",
			filename: "THISISTHEVERYLONGFILENAMEITHINK.DOCX",
			version:  1,
			expected: spec.FileIdentifier("THISISTHEVERYLONGFILENAMEITHIN.DOCX;1"),
		},
		// Error cases
		{
			name:          "Version should not be able to exceed 32767",
			filename:      "SOMEFILE.TXT",
			version:       32768,
			expectedError: encode.ErrInvalidVersion,
		},
		{
			name:          "Filename+extensions longer than 30 characters that cannot truncate filename to get below limit should throw an error",
			filename:      "FOO.THISISASUPERLONGFILEEXTENSIONTHATISMORETHAN30CHARACTERS",
			version:       1,
			expectedError: errors.New("TODO"),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			fi, err := encode.AsFileIdentifier(c.filename, c.version, encode.EscapeSequenceNone)

			if c.expectedError != nil {
				assert.ErrorIs(t, err, c.expectedError)
				assert.Nil(t, fi, "File identifier should be nil when an error is returned")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.expected, fi)
			}
		})
	}
}

func TestAsFileIdentifier_UCS2(t *testing.T) {
	cases := []struct {
		name          string
		filename      string
		version       uint
		expected      spec.FileIdentifier
		expectedError error
	}{
		{
			name:     "Spec-compliant input filename",
			filename: "VALIDFILE.TXT",
			version:  1,
			expected: spec.FileIdentifier("\x00V\x00A\x00L\x00I\x00D\x00F\x00I\x00L\x00E\x00.\x00T\x00X\x00T\x00;\x001"),
		},
		{
			name:     "Spec-compliant input filename with version >1",
			filename: "VALIDFILE.TXT",
			version:  5,
			expected: spec.FileIdentifier("\x00V\x00A\x00L\x00I\x00D\x00F\x00I\x00L\x00E\x00.\x00T\x00X\x00T\x00;\x005"),
		},
		{
			name:     "Spec-compliant input directory",
			filename: "DIRECTORY",
			version:  0,
			expected: spec.FileIdentifier("\x00D\x00I\x00R\x00E\x00C\x00T\x00O\x00R\x00Y"),
		},
		{
			name:     "Mixed case filename (still ASCII)",
			filename: "soMEfiLe.tXt",
			version:  1,
			expected: spec.FileIdentifier("\x00s\x00o\x00M\x00E\x00f\x00i\x00L\x00e\x00.\x00t\x00X\x00t\x00;\x001"),
		},
		{
			name:     "File without extension is permitted",
			filename: "soMEfiLe",
			version:  1,
			expected: spec.FileIdentifier("\x00s\x00o\x00M\x00E\x00f\x00i\x00L\x00e\x00;\x001"),
		},
		{
			name:     "Directory with extension is permitted",
			filename: "directory.txt",
			version:  0,
			expected: spec.FileIdentifier("\x00d\x00i\x00r\x00e\x00c\x00t\x00o\x00r\x00y\x00.\x00t\x00x\x00t"),
		},
		// Long filenames allowed
		{
			name:     "Special characters should be permitted",
			filename: "f-! oo bar.tx?t",
			version:  1,
			expected: spec.FileIdentifier("\x00f\x00-\x00!\x00 \x00o\x00o\x00 \x00b\x00a\x00r\x00.\x00t\x00x\x00?\x00t\x00;\x001"),
		},
		{
			name:     "Should convert uncorrectable characters to underscores in filenames",
			filename: "f*/:;?\\\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0A\x0B\x0C\x0D\x0E\x0F\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1A\x1B\x1C\x1D\x1E\x1F",
			version:  1,
			expected: spec.FileIdentifier("\x00f\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_\x00_"),
		},
		{
			name:     "Filename+extensions up to 64 characters should remain unmodified",
			filename: "this is a very long filename - Joliet permits up to 64 chars.txt",
			version:  1,
			expected: spec.FileIdentifier("\x00t\x00h\x00i\x00s\x00 \x00i\x00s\x00 \x00a\x00 \x00v\x00e\x00r\x00y\x00 \x00l\x00o\x00n\x00g\x00 \x00f\x00i\x00l\x00e\x00n\x00a\x00m\x00e\x00 \x00-\x00 \x00J\x00o\x00l\x00i\x00e\x00t\x00 \x00p\x00e\x00r\x00m\x00i\x00t\x00s\x00 \x00u\x00p\x00 \x00t\x00o\x00 \x006\x004\x00 \x00c\x00h\x00a\x00r\x00s\x00.\x00t\x00x\x00t\x00;\x001"),
		},
		{
			name:     "Filename+extensions over 64 characters should be truncated",
			filename: "this is a very long filename - but it is over 64 characters long.txt",
			version:  1,
			expected: spec.FileIdentifier("\x00t\x00h\x00i\x00s\x00 \x00i\x00s\x00 \x00a\x00 \x00v\x00e\x00r\x00y\x00 \x00l\x00o\x00n\x00g\x00 \x00f\x00i\x00l\x00e\x00n\x00a\x00m\x00e\x00 \x00-\x00 \x00b\x00u\x00t\x00 \x00i\x00t\x00 \x00i\x00s\x00 \x00o\x00v\x00e\x00r\x00 \x006\x004\x00 \x00c\x00h\x00a\x00r\x00a\x00c\x00t\x00e\x00r\x00s\x00 \x00.\x00t\x00x\x00t\x00;\x001"),
		},
		{
			name:     "Unicode characters should encode correctly as UCS2 big endian",
			filename: "かわいい.exe",
			version:  1,
			expected: spec.FileIdentifier("\x30\x4b\x30\x8f\x30\x44\x30\x44\x00.\x00e\x00x\x00e\x00;\x001"),
		},
		// Error cases
		{
			name:          "Version should not be able to exceed 32767",
			filename:      "SOMEFILE.TXT",
			version:       32768,
			expectedError: encode.ErrInvalidVersion,
		},
		{
			name:          "Filename+extensions longer than 64 characters that cannot truncate filename to get below limit should throw an error",
			filename:      "thi.s is a very long filename - but it is over 64 characters long txt",
			version:       1,
			expectedError: errors.New("TODO"),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			fi, err := encode.AsFileIdentifier(c.filename, c.version, encode.EscapeSequenceUCS2Level1)

			if c.expectedError != nil {
				assert.ErrorIs(t, err, c.expectedError)
				assert.Nil(t, fi, "File identifier should be nil when an error is returned")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.expected, fi)
			}
		})
	}
}
