package iso9660

import (
	"bytes"
	_ "embed"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/fs"
	"os"
	"slices"
	"strings"
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

	// Our test data selects blocks sequentially based on a depth-first search of the file tree, excluding files.
	// In order for our path table to match the test data exactly, we need to do the same thing.
	block := uint32(23)
	for entry := range d.Walk(true) {
		if _, ok := entry.(*file); ok {
			continue
		}

		entry.Relocate(block)
		block += 1
	}

	table := newPathTable(d)
	require.NotNil(t, table, "Path table returned by newPathTable should not be nil")

	var output bytes.Buffer
	count, err := table.WriteTo(&output, false) // Test data is little endian

	require.NoError(t, err, "WriteTo should not throw an error for valid input")
	assert.EqualValues(t, output.Len(), count, "WriteTo count should match actual number of written bytes")

	assert.Equal(t, testdataPathTable, output.Bytes(), "Path table encoding should match expected test data")
}

func TestPathTable_Records(t *testing.T) {
	d, err := newDirectory(os.DirFS("testdata/pathtable").(fs.ReadDirFS), ".", nil, time.Now())
	require.NoError(t, err, "newDirectory should not throw an error for valid input")

	table := newPathTable(d)
	require.NotNil(t, table, "Path table returned by newPathTable should not be nil")

	records := slices.Collect(table.Records())

	assert.Equal(t, 9, len(records), "Records() should return number of records equal to number of directories in test data")

	assert.EqualValues(t, []byte{0x00}, records[0].DirectoryIdentifier, "Root directory should be first entry in path table (ordering requirement: ordered by level in hierarchy)")
	assert.EqualValues(t, 1, records[0].ParentDirectoryNumber, "Root directory should be parented to itself in the path table")

	prevParentNumber := records[0].ParentDirectoryNumber
	prevDirectoryIdentifier := records[0].DirectoryIdentifier

	checkedDirectoryIdentifierCount := 0
	orderingErrors := []error{}

	for _, record := range records[1:] {
		checkedDirectoryIdentifier, err := checkRecordOrdering(&prevParentNumber, &prevDirectoryIdentifier, record)
		if err != nil {
			orderingErrors = append(orderingErrors, err)
		}

		if checkedDirectoryIdentifier {
			checkedDirectoryIdentifierCount += 1
		}
	}

	assert.Empty(t, orderingErrors, "Records should meet ordering criteria")
	assert.Equal(t, 5, checkedDirectoryIdentifierCount, "Record ordering check should fall through to directory identifier ordering criteria correct number of times")
}

var errRecordOrderingParentNumberViolated = errors.New("records are not in ascending order by parent record number")
var errRecordOrderingDirectoryIdentifierViolated = errors.New("records are not in ascending order by directory identifier")

// Checks whether a path table record is in the order expected by the spec.
// Returns (directory identifiers compared, ordering error)
func checkRecordOrdering(prevParentNumber *uint16, prevDirectoryIdentifier *fileIdentifier, current *pathTableRecord) (bool, error) {
	defer func() {
		*prevParentNumber = current.ParentDirectoryNumber
		*prevDirectoryIdentifier = current.DirectoryIdentifier
	}()

	// Ordering requirement #2: parent directory numbers must be ascending
	if current.ParentDirectoryNumber < *prevParentNumber {
		return false, errRecordOrderingParentNumberViolated
	} else if current.ParentDirectoryNumber > *prevParentNumber {
		// This record has a different parent directory number, and therefore lower-precedence ordering rules
		// don't apply.
		return false, nil
	}

	currentDirectoryIdentifierPadded := current.DirectoryIdentifier
	prevDirectoryIdentifierPadded := *prevDirectoryIdentifier

	// The spec requires that we pad the shortest directory identifier with 0x20.
	// Bit of a lazy hack, but do this for both identifiers to avoid having to work out which is shorter!
	for len(currentDirectoryIdentifierPadded) < len(prevDirectoryIdentifierPadded) {
		currentDirectoryIdentifierPadded = append(currentDirectoryIdentifierPadded, 0x20) // Padding byte
	}

	for len(prevDirectoryIdentifierPadded) < len(currentDirectoryIdentifierPadded) {
		prevDirectoryIdentifierPadded = append(prevDirectoryIdentifierPadded, 0x20) // Padding byte
	}

	cmp := strings.Compare(string(prevDirectoryIdentifierPadded), string(currentDirectoryIdentifierPadded))

	// Ordering requirement #3: directory identifiers should be in ascending order
	// TODO: At some point, we'll have to defer to fileIdentifier to do this comparison, and not assume that the file
	// identifiers are Go-encoded strings. The spec tells us to compare characters, not bytes.
	if cmp > 0 {
		return true, errRecordOrderingDirectoryIdentifierViolated
	}

	return true, nil
}
