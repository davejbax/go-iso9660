package encode_test

import (
	"fmt"
	"github.com/davejbax/go-iso9660/internal/encode"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAsUInt16BothByte(t *testing.T) {
	cases := []struct {
		input    uint16
		expected uint32
	}{
		{0x00, 0x00000000},
		{0xABCD, 0xCDABABCD},
		{0x00F7, 0xF70000F7},
		{0x7F00, 0x007F7F00},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%02x", c.input), func(t *testing.T) {
			t.Parallel()

			encoded := encode.AsUInt16BothByte(c.input)
			assert.Equal(t, c.expected, encoded.Value, "Encoded value should match expected encoding")
		})
	}
}

func TestAsUInt32BothByte(t *testing.T) {
	cases := []struct {
		input    uint32
		expected uint64
	}{
		{0x00, 0x0000000000000000},
		{0xABCD, 0xCDAB00000000ABCD},
		{0x00F7, 0xF7000000000000F7},
		{0x7F00, 0x007F000000007F00},
		{0x123456, 0x5634120000123456},
		{0x12345678, 0x7856341212345678},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%02x", c.input), func(t *testing.T) {
			t.Parallel()

			encoded := encode.AsUInt32BothByte(c.input)
			assert.Equal(t, c.expected, encoded.Value, "Encoded value should match expected encoding")
		})
	}
}
