package iso9660

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"testing"
	"time"
)

func createTestFile() *file {
	// EFI/BOOT/BOOTx64.EFI file from Arch Linux ISO
	return &file{
		// Note: the Arch Linux ISO uses Joliet, so the name here is in UTF16-BE encoding (thanks Microsoft...)
		name:       []uint8{0, 'B', 0, 'O', 0, 'O', 0, 'T', 0, 'x', 0, '6', 0, '4', 0, '.', 0, 'E', 0, 'F', 0, 'I'},
		location:   507811,
		recordedAt: time.Date(2025, 1, 1, 8, 45, 59, 0, time.UTC),
		flags:      0,
		dataLength: 124416,
		data: func() (io.Reader, error) {
			return bytes.NewBufferString("helloworld"), nil
		},
	}
}

func TestFile_WriteTo(t *testing.T) {
	f := createTestFile()

	reader, err := f.data()
	if err != nil {
		t.Fatalf("error creating test file reader: %v", err)
	}

	expected, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("error reading test file: %v", err)
	}

	var actual bytes.Buffer
	count, err := f.WriteTo(&actual)

	assert.NoError(t, err, "WriteTo should not return an error given a valid data source")
	assert.EqualValues(t, len(expected), actual.Len(), "WriteTo should write as many bytes as file's data field has")
	assert.EqualValues(t, actual.Len(), count, "WriteTo should return a count that matches the number of bytes written")
	assert.Equal(t, expected, actual.Bytes(), "WriteTo should write file contents verbatim")
}

func TestFile_Record(t *testing.T) {
	f := createTestFile()

	// Taken from Arch Linux 2025.01.01 x86_64 ISO
	expectedRecord := []byte{
		0x38, 0x00, 0xA3, 0xBF, 0x07, 0x00, 0x00, 0x07, 0xBF, 0xA3, 0x00, 0xE6, 0x01, 0x00, 0x00, 0x01,
		0xE6, 0x00, 0x7D, 0x01, 0x01, 0x08, 0x2D, 0x3B, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x01,
		0x16, 0x00, 0x42, 0x00, 0x4F, 0x00, 0x4F, 0x00, 0x54, 0x00, 0x78, 0x00, 0x36, 0x00, 0x34, 0x00,
		0x2E, 0x00, 0x45, 0x00, 0x46, 0x00, 0x49, 0x00,
	}

	assert.EqualValues(t, len(expectedRecord), f.RecordLength(), "RecordLength should return the correct value")

	actualRecord := f.Record()
	require.NotNil(t, actualRecord, "Record() should not return nil")

	var actualMarshalledRecord bytes.Buffer
	count, err := actualRecord.WriteTo(&actualMarshalledRecord)

	require.NoError(t, err, "Returned record's WriteTo should not return an error for valid input")
	assert.EqualValues(t, len(expectedRecord), count, "Returned record's WriteTo should return the correct number of bytes written")
	assert.EqualValues(t, actualMarshalledRecord.Len(), count, "Returned record's WriteTo should return a count that matches the number of bytes it writes")
	assert.Equal(t, expectedRecord, actualMarshalledRecord.Bytes(), "File should be marshalled correctly to expected bytes")
}

func TestDirectory_WriteTo(t *testing.T) {

}

func TestDirectory_Record(t *testing.T) {
	//d := &directory{
	//	name:       fileIdentifierSelf, // Root directory entry
	//	parent:     nil,
	//	location:   ,
	//	recordedAt: time.Time{},
	//	flags:      0,
	//	entries:    nil,
	//}
	//d.
}

func TestDirectory_RecordLength(t *testing.T) {

}
