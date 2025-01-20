package iso9660

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPrimaryVolumeDescriptor_WriteTo(t *testing.T) {
	pvd := &primaryVolumeDescriptor{}
	buff := bytes.NewBuffer(nil)

	written, err := pvd.WriteTo(buff)
	require.NoError(t, err, "Primary volume descriptor should be written without error")
	assert.Equal(t, written, buff.Len(), "Number of written bytes returned should reflect reality")
	assert.Equal(t, logicalSectorSize, buff.Len(), "Primary volume descriptor should write one logical sector of bytes")
}
