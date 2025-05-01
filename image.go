package iso9660

import (
	"fmt"
	"github.com/davejbax/go-iso9660/internal/builder"
	"github.com/davejbax/go-iso9660/internal/encode"
	"github.com/itchio/headway/counter"
	"io"
	"io/fs"
	"time"
)

const dataPreparer = "GOISO9660"

type Builder struct {
	source fs.ReadDirFS

	recordedAt time.Time

	svdOptions *svdOptions
	metadata   Metadata
}

func New(contents fs.ReadDirFS) *Builder {
	return &Builder{
		source:     contents,
		recordedAt: time.Now(),
	}
}

func (b *Builder) WithJoliet() *Builder {
	if b.svdOptions == nil {
		b.svdOptions = &svdOptions{}
	}

	b.svdOptions.escapeSequence = encode.EscapeSequenceUCS2Level1

	return b
}

func (b *Builder) WithMetadata(m Metadata) *Builder {
	b.metadata = m

	return b
}

type Metadata struct {
	VolumeIdentifier      string
	VolumeSetIdentifier   string
	PublisherIdentifier   string
	ApplicationIdentifier string
}

type svdOptions struct {
	escapeSequence encode.EscapeSequence
}

type Image struct {
	blocks []*block
}

func (i *Image) WriteTo(w io.Writer) (int64, error) {
	cw := counter.NewWriter(w)
	bw := builder.NewBlockWriter(cw)

	for _, block := range i.blocks {
		if err := bw.WriteBlock(block.location, block.writer); err != nil {
			return cw.Count(), fmt.Errorf("failed to write block: %w", err)
		}
	}

	return cw.Count(), nil
}
