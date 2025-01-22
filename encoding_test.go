package iso9660

import (
	"bytes"
	"fmt"
	"github.com/lunixbochs/struc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestUint16BothByte(t *testing.T) {
	cases := []struct {
		input    uint16
		expected [4]byte
	}{
		{0x00, [4]byte{0x00, 0x00, 0x00, 0x00}},
		{0xABCD, [4]byte{0xCD, 0xAB, 0xAB, 0xCD}},
		{0x00F7, [4]byte{0xF7, 0x00, 0x00, 0xF7}},
		{0x7F00, [4]byte{0x00, 0x7F, 0x7F, 0x00}},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%02x", c.input), func(t *testing.T) {
			t.Parallel()

			field := uint16BothByte(c.input)
			testUintField(t, field, c.expected[:], 4)
		})
	}
}

func TestUint16BothByteField_Pack_InvalidArgs(t *testing.T) {
	bothByteField := uint16BothByte(0x1234)
	written, err := bothByteField.Pack(nil, &struc.Options{})
	assert.Error(t, err, "Pack should return an error when the buffer is too small")
	assert.Equal(t, 0, written, "Pack should not write any bytes when buffer is too small")
}

func TestUint32BothByte(t *testing.T) {
	cases := []struct {
		input    uint32
		expected [8]byte
	}{
		{0x00, [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},
		{0xABCD, [8]byte{0xCD, 0xAB, 0x00, 0x00, 0x00, 0x00, 0xAB, 0xCD}},
		{0x00F7, [8]byte{0xF7, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xF7}},
		{0x7F00, [8]byte{0x00, 0x7F, 0x00, 0x00, 0x00, 0x00, 0x7F, 0x00}},
		{0x123456, [8]byte{0x56, 0x34, 0x12, 0x00, 0x00, 0x12, 0x34, 0x56}},
		{0x12345678, [8]byte{0x78, 0x56, 0x34, 0x12, 0x12, 0x34, 0x56, 0x78}},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%02x", c.input), func(t *testing.T) {
			t.Parallel()

			field := uint32BothByte(c.input)
			testUintField(t, field, c.expected[:], 8)
		})
	}
}

func TestUint32BothByteField_Pack_InvalidArgs(t *testing.T) {
	bothByteField := uint32BothByte(0x1234)
	written, err := bothByteField.Pack(nil, &struc.Options{})
	assert.Error(t, err, "Pack should return an error when the buffer is too small")
	assert.Equal(t, 0, written, "Pack should not write any bytes when buffer is too small")
}

func testUintField(t *testing.T, field struc.Custom, expected []byte, expectedSize int) {
	size := field.Size(&struc.Options{})
	assert.Equal(t, expectedSize, size, "Size method of field should be correct")

	buff := make([]byte, size)
	written, err := field.Pack(buff, &struc.Options{})

	require.NoError(t, err, "Pack should not return an error")
	assert.Equal(t, size, written, "Pack should write <size> bytes")
	assert.Equal(t, expected, buff, "Both byte field should encode to the correct value")
}

func TestNewDateTime(t *testing.T) {
	cases := []struct {
		input    time.Time
		expected [7]byte
	}{
		{time.Date(2015, 7, 31, 19, 0, 15, 0, time.UTC), [7]byte{0x73, 0x07, 0x1F, 0x13, 0x00, 0x0F, 0x00}},
		{time.Date(2000, 1, 7, 12, 26, 14, 0, time.UTC), [7]byte{0x64, 0x01, 0x07, 0x0C, 0x1A, 0x0E, 0x00}},
		{time.Date(2000, 1, 7, 12, 26, 14, 0, time.FixedZone("UTC-8", -8*60*60)), [7]byte{0x64, 0x01, 0x07, 0x14, 0x1A, 0x0E, 0x00}},
	}

	for _, c := range cases {
		dt := newDateTime(c.input)
		buff := bytes.NewBuffer(make([]byte, 0, 7))

		require.NoError(t, struc.Pack(buff, &dt), "Pack should not return an error with a dateTime struct returned by newDateTime")
		assert.Equal(t, c.expected[:], buff.Bytes(), "dateTime created from time.Time should encode to correct value")
	}
}
