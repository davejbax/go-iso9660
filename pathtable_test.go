package iso9660

import (
	"bytes"
	_ "embed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/fs"
	"os"
	"testing"
	"time"
)

// pathtable.dat is the path table obtained by using a third-party tool to create an ISO
// from the contents of testdata/pathtable/
//
//go:embed testdata/pathtable.dat
var testdataPathTable []byte

func TestPathTable_WriteTo(t *testing.T) {
	d, err := newDirectory(os.DirFS("testdata/pathtable").(fs.ReadDirFS), ".", nil, time.Now())
	require.NoError(t, err, "newDirectory should not throw an error for valid input")

	table := newPathTable(d)
	var output bytes.Buffer

	count, err := table.WriteTo(&output, false) // Test data is little endian
	require.NoError(t, err, "WriteTo should not throw an error for valid input")
	assert.EqualValues(t, output.Len(), count, "WriteTo count should match actual number of written bytes")

	assert.Equal(t, testdataPathTable, output.Bytes(), "Path table encoding should match expected test data")
}
