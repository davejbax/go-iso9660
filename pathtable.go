package iso9660

import (
	"encoding/binary"
	"fmt"
	"github.com/itchio/headway/counter"
	"github.com/lunixbochs/struc"
	"io"
	"iter"
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
	return &pathTable{root: root}
}

func (p *pathTable) Records() iter.Seq[*pathTableRecord] {
	return func(yield func(*pathTableRecord) bool) {
		directoryNumbers := make(map[fileLike]int)
		currentDirectoryNumber := 1

		// Path table needs to be in breadth-first search order, because the spec requires that records are ordered by
		// level in the directory hierarchy first and foremost, and secondly by
		for entry := range p.root.Walk(false) {
			dir, ok := entry.(*directory)
			if !ok {
				continue
			}

			parentNumber := 1
			if parent := dir.Parent(); parent != nil && parent != entry {
				var ok bool
				parentNumber, ok = directoryNumbers[parent]
				if !ok {
					// This shouldn't happen: directory.Walk() is a breadth-first search, which should give us all parent
					// directories before we get to the children.
					panic("unexpected directory iteration order failure: expected to have written parent directory already, but parent directory has not been written")
				}
			}

			record := &pathTableRecord{
				LengthOfDirectoryIdentifier:   uint8(len(dir.name)),
				ExtendedAttributeRecordLength: 0,
				LocationOfExtent:              dir.location,
				ParentDirectoryNumber:         uint16(parentNumber),
				DirectoryIdentifier:           dir.name,
			}

			if !yield(record) {
				break
			}

			directoryNumbers[entry] = currentDirectoryNumber
			currentDirectoryNumber += 1
		}
	}
}

func (p *pathTable) Size() uint32 {
	total := uint32(0)

	for record := range p.Records() {
		total += 8 + uint32(record.LengthOfDirectoryIdentifier)
		if record.LengthOfDirectoryIdentifier%2 == 1 {
			total += 1 // Padding
		}
	}

	return total
}

func (p *pathTable) WriteTo(w io.Writer, bigEndian bool) (int64, error) {
	cw := counter.NewWriter(w)

	var byteOrder binary.ByteOrder = binary.LittleEndian
	if bigEndian {
		byteOrder = binary.BigEndian
	}

	for record := range p.Records() {
		if err := struc.PackWithOptions(cw, record, &struc.Options{Order: byteOrder}); err != nil {
			return cw.Count(), fmt.Errorf("failed to encode path table record: %w", err)
		}

		// If the directory name has an odd length, the spec requires us to add a padding byte so that this path
		// table record consists of an even number of bytes.
		if record.LengthOfDirectoryIdentifier%2 == 1 {
			if _, err := cw.Write([]byte{0}); err != nil {
				return cw.Count(), fmt.Errorf("failed to write padding byte: %w", err)
			}
		}
	}

	return cw.Count(), nil
}
