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
	_, err := newDirectory(i.source, ".", nil, time.Now())
	if err != nil {
		return 0, fmt.Errorf("could not create directory: %w", err)
	}

	return 0, nil
}
