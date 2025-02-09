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
	relocateTree(dir, &block)

	_, err = newPrimaryVolumeDescriptor(
		"",
		"test",
		"test",
		"publisher",
		"datapreparer",
		"application",
		block,
		0,
		0,
		0,
		0,
		0,
		dir,
	)
	if err != nil {
		return 0, fmt.Errorf("could not create primary volume descriptor: %w", err)
	}

	return 0, nil
}
