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

	block := uint32(17)

	pathTable := newPathTable(dir)
	pathTableSize := pathTable.Size()

	// Generally the path table should come before the disk contents, if this was an actual CD, to make it easier to
	// skip to the relevant content
	pathTableLBlock := allocateAndIncrementBlock(&block, pathTableSize)
	pathTableMBlock := allocateAndIncrementBlock(&block, pathTableSize)

	// Set locations for the files and directories
	relocateTree(dir, &block)

	_, err = newPrimaryVolumeDescriptor(
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

	return 0, nil
}
