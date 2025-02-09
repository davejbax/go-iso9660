package iso9660

import (
	"io"
)

type pathTableRecord struct {
	LengthOfDirectoryIdentifier   uint8 `struc:"sizeof=DirectoryIdentifier"`
	ExtendedAttributeRecordLength uint8
	LocationOfExtent              uint32
	ParentDirectoryNumber         uint16
	DirectoryIdentifier           fileIdentifier
}

type pathTable struct {
	root *directory
}

func newPathTable(root *directory) *pathTable {
	return nil
}

func (*pathTable) WriteTo(w io.Writer, bigEndian bool) (int64, error) {
	return 0, nil
}
