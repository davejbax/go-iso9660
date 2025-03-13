package builder_test

import (
	"bytes"
	"github.com/davejbax/go-iso9660/internal/builder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBlockWriter_WriteBlock(t *testing.T) {
	var buff bytes.Buffer

	bw := builder.NewBlockWriter(&buff)

	require.NotNil(t, bw, "newBlockWriter should not return nil for valid input")

	testBlock16 := bytes.NewBufferString("test")

	err := bw.WriteBlock(0x16, testBlock16)
	require.NoError(t, err, "WriteBlock should not return an error for valid input")

	assert.Equal(t, 0x17*2048, buff.Len(), "WriteBlock should write all prior zeros and given block, plus padding to make whole number of blocks")
	assert.EqualValues(t, buff.Len(), bw.BytesWritten(), "BytesWritten should return correct count after writing one block")
	assert.Equal(t, make([]byte, 0x16*2048), buff.Bytes()[:0x16*2048], "WriteBlock should write zeros prior to first block")
	assert.Error(t, bw.WriteBlock(0x16, testBlock16), "WriteBlock should not allow rewriting the same block")
	assert.Error(t, bw.WriteBlock(0x00, testBlock16), "WriteBlock should not allow writing prior blocks")

	testBlock17To18Data := make([]byte, 2048+100)
	for i, _ := range testBlock17To18Data {
		testBlock17To18Data[i] = byte(i % 256)
	}

	testBlock17To18 := bytes.NewBuffer(testBlock17To18Data)
	err = bw.WriteBlock(0x17, testBlock17To18)
	require.NoError(t, err, "WriteBlock should not return an error for valid input on second block")

	assert.Equal(t, 0x19*2048, buff.Len(), "WriteBlock should write correct number of bytes when given more than one block of data to write, and pad to number of whole blocks")
	assert.EqualValues(t, buff.Len(), bw.BytesWritten(), "BytesWritten should return correct count after writing three blocks")
	assert.Equal(t, testBlock17To18Data, buff.Bytes()[0x17*2048:0x18*2048+100], "WriteBlock should write correct data when given more than one block of data to write")
	assert.Error(t, bw.WriteBlock(0x18, testBlock16), "WriteBlock should not allow rewriting a written block when given more than one block of data to write")
	assert.Error(t, bw.WriteBlock(0x16, testBlock16), "WriteBlock should not allow writing prior blocks")
}
