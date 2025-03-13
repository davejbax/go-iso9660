package builder_test

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/davejbax/go-iso9660/internal/builder"
	"github.com/davejbax/go-iso9660/internal/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"slices"
	"strings"
	"testing"
	"time"
)

func buildTestDirectory() *builder.Directory {
	dummyData := func() (io.Reader, error) {
		return bytes.NewReader([]byte("foo")), nil
	}

	root := builder.NewEmptyDirectory(spec.FileIdentifierSelf, time.Now(), nil)

	dir0 := builder.NewEmptyDirectory(spec.FileIdentifier("APPLE"), time.Now(), root)
	dir0_0 := builder.NewEmptyDirectory(spec.FileIdentifier("MELON"), time.Now(), dir0)
	dir0_0_0 := builder.NewEmptyDirectory(spec.FileIdentifier("BANANA"), time.Now(), dir0_0)
	dir0_0_1 := builder.NewEmptyDirectory(spec.FileIdentifier("PINEAPPLE"), time.Now(), dir0_0)
	dir0_0_1_0 := builder.NewFile(spec.FileIdentifier("BBBBBBBB.TXT;1"), time.Now(), 3, dummyData)
	dir0_1 := builder.NewFile(spec.FileIdentifier("ZZZZ.TXT;1"), time.Now(), 3, dummyData)
	dir1 := builder.NewEmptyDirectory(spec.FileIdentifier("BANANA"), time.Now(), root)
	dir1_0 := builder.NewEmptyDirectory(spec.FileIdentifier("1234"), time.Now(), dir1)
	dir1_1 := builder.NewEmptyDirectory(spec.FileIdentifier("APPLE"), time.Now(), dir1)
	dir1_2 := builder.NewEmptyDirectory(spec.FileIdentifier("PINEAPPLE"), time.Now(), dir1)
	dir1_3 := builder.NewFile(spec.FileIdentifier("A.DAT;1"), time.Now(), 3, dummyData)
	dir2 := builder.NewFile(spec.FileIdentifier("AARDVARK.MP3;1"), time.Now(), 3, dummyData)

	root.Add(dir0)
	root.Add(dir1)
	root.Add(dir2)

	dir0.Add(dir0_0)
	dir0.Add(dir0_1)

	dir0_0.Add(dir0_0_0)
	dir0_0.Add(dir0_0_1)

	dir0_0_1.Add(dir0_0_1_0)

	dir1.Add(dir1_0)
	dir1.Add(dir1_1)
	dir1.Add(dir1_2)
	dir1.Add(dir1_3)

	return root
}

func TestPathTable_Records(t *testing.T) {
	d := buildTestDirectory()

	table := builder.NewPathTable(d)
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
func checkRecordOrdering(prevParentNumber *uint16, prevDirectoryIdentifier *spec.FileIdentifier, current *spec.PathTableRecord) (bool, error) {
	defer func() {
		*prevParentNumber = current.ParentDirectoryNumber
		*prevDirectoryIdentifier = current.DirectoryIdentifier
	}()

	// Ordering requirement #2: parent directory numbers must be ascending
	if current.ParentDirectoryNumber < *prevParentNumber {
		return false, fmt.Errorf("%w: previously-seen directory %s has a higher number (%d) than current directory %s (%d)",
			errRecordOrderingParentNumberViolated,
			string(*prevDirectoryIdentifier),
			*prevParentNumber,
			string(current.DirectoryIdentifier),
			current.ParentDirectoryNumber,
		)
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
