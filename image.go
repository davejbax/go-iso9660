package iso9660

import (
	"io"
	"io/fs"
)

type Image struct{}

func NewImage(contents fs.ReadDirFS) (*Image, error) {
	return nil, nil
}

func (i *Image) WriteTo(w io.Writer) (n int64, err error) {
	return 0, nil
}
