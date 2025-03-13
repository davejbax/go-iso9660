package encode_test

import (
	"bytes"
	"github.com/davejbax/go-iso9660/internal/encode"
	"github.com/lunixbochs/struc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestAsDateTime(t *testing.T) {
	cases := []struct {
		input    time.Time
		expected [7]byte
	}{
		{time.Date(2015, 7, 31, 19, 0, 15, 0, time.UTC), [7]byte{0x73, 0x07, 0x1F, 0x13, 0x00, 0x0F, 0x00}},
		{time.Date(2000, 1, 7, 12, 26, 14, 0, time.UTC), [7]byte{0x64, 0x01, 0x07, 0x0C, 0x1A, 0x0E, 0x00}},
		{time.Date(2000, 1, 7, 12, 26, 14, 0, time.FixedZone("UTC-8", -8*60*60)), [7]byte{0x64, 0x01, 0x07, 0x14, 0x1A, 0x0E, 0x00}},
	}

	for _, c := range cases {
		dt := encode.AsDateTime(c.input)
		buff := bytes.NewBuffer(make([]byte, 0, 7))

		require.NoError(t, struc.Pack(buff, &dt), "Pack should not return an error with a dateTime struct returned by newDateTime")
		assert.Equal(t, c.expected[:], buff.Bytes(), "dateTime created from time.Time should encode to correct value")
	}
}
