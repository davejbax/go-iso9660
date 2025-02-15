package iso9660

import (
	"fmt"
	"io"
	"io/fs"
	"time"
)

type Image struct {
	source fs.ReadDirFS
}

func NewImage(contents fs.ReadDirFS) (*Image, error) {
	return &Image{
		source: contents,
	}, nil
}

func (i *Image) WriteTo(w io.Writer) (int64, error) {
	dir, err := newDirectory(i.source, ".", nil, time.Now())
	if err != nil {
		return 0, fmt.Errorf("could not create directory: %w", err)
	}

	block := uint32(18) // 18 because 16 is PVD and 17 is TVD

	pathTable := newPathTable(dir)
	pathTableSize := pathTable.Size()

	// Generally the path table should come before the disk contents, if this was an actual CD, to make it easier to
	// skip to the relevant content
	pathTableLBlock := allocateAndIncrementBlock(&block, pathTableSize)
	pathTableMBlock := allocateAndIncrementBlock(&block, pathTableSize)

	// Set locations for the files and directories
	relocateTree(dir, &block)

	pvd, err := newPrimaryVolumeDescriptor(
		"",
		"test",
		"test",
		"publisher",
		"datapreparer",
		"application",
		block,
		pathTableSize,
		pathTableLBlock,
		0,
		pathTableMBlock,
		0,
		dir,
	)
	if err != nil {
		return 0, fmt.Errorf("could not create primary volume descriptor: %w", err)
	}

	bw := newBlockWriter(w)

	if err := bw.WriteBlock(16, pvd); err != nil {
		return bw.BytesWritten(), fmt.Errorf("failed to write PVD: %w", err)
	}

	if err := bw.WriteBlockFunc(17, writeTerminatorVolumeDescriptor); err != nil {
		return bw.BytesWritten(), fmt.Errorf("failed to write terminator volume descriptor: %w", err)
	}

	if err := bw.WriteBlock(pathTableLBlock, pathTable.LittleEndianWriterTo()); err != nil {
		return bw.BytesWritten(), fmt.Errorf("failed to write L-type path table: %w", err)
	}

	if err := bw.WriteBlock(pathTableMBlock, pathTable.BigEndianWriterTo()); err != nil {
		return bw.BytesWritten(), fmt.Errorf("failed to write M-type path table: %w", err)
	}

	for entry := range dir.Walk(false) {
		fmt.Printf("locating %s at %d\n", string(entry.Record().FileIdentifier), entry.Location())
		if err := bw.WriteBlock(entry.Location(), entry); err != nil {
			return bw.BytesWritten(), fmt.Errorf("failed to write entry: %w", err)
		}
	}

	return bw.BytesWritten(), nil
}
