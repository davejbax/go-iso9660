package builder

import (
	"github.com/davejbax/go-iso9660/internal/spec"
	"iter"
)

type PathTable struct {
	root *Directory
}

func NewPathTable(root *Directory) *PathTable {
	return &PathTable{root: root}
}

func (p *PathTable) Records() iter.Seq[*spec.PathTableRecord] {
	return func(yield func(*spec.PathTableRecord) bool) {
		directoryNumbers := make(map[RelocatableFileSection]int)
		currentDirectoryNumber := 1

		// Path table needs to be in breadth-first search order, because the spec requires that records are ordered by
		// level in the Directory hierarchy first and foremost, and secondly by directory identifiers
		for entry := range p.root.Walk(false) {
			dir, ok := entry.(*Directory)
			if !ok {
				continue
			}

			parentNumber := 1
			if parent := dir.Parent(); parent != nil && parent != entry {
				var ok bool
				parentNumber, ok = directoryNumbers[parent]
				if !ok {
					// This shouldn't happen: Directory.Walk() is a breadth-first search, which should give us all parent
					// directories before we get to the children.
					panic("unexpected Directory iteration order failure: expected to have written parent Directory already, but parent Directory has not been written")
				}
			}

			record := &spec.PathTableRecord{
				LengthOfDirectoryIdentifier:   dir.PointerRecord().LengthOfFileIdentifier,
				ExtendedAttributeRecordLength: 0,
				LocationOfExtent:              dir.PointerRecord().ExtentLocation.RealValue(),
				ParentDirectoryNumber:         uint16(parentNumber),
				DirectoryIdentifier:           dir.PointerRecord().FileIdentifier,
			}

			if !yield(record) {
				break
			}

			directoryNumbers[entry] = currentDirectoryNumber
			currentDirectoryNumber += 1
		}
	}
}

func (p *PathTable) Size() uint32 {
	total := uint32(0)

	for record := range p.Records() {
		total += 8 + uint32(record.LengthOfDirectoryIdentifier)
		if record.LengthOfDirectoryIdentifier%2 == 1 {
			total += 1 // Padding
		}
	}

	return total
}

func (p *PathTable) MPathTable() spec.MPathTable {
	return spec.MPathTable(p.Records())
}

func (p *PathTable) LPathTable() spec.LPathTable {
	return spec.LPathTable(p.Records())
}
