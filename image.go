package iso9660

import (
	"fmt"
	"github.com/davejbax/go-iso9660/internal/builder"
	"github.com/davejbax/go-iso9660/internal/spec"
	"github.com/itchio/headway/counter"
	"github.com/lunixbochs/struc"
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
	// TODO: probably move this to the constructor?
	dir, err := newDirectoryFromFS(i.source, ".", nil, time.Now())
	if err != nil {
		return 0, fmt.Errorf("could not create directory: %w", err)
	}

	block := uint32(18) // 18 because 16 is PVD and 17 is TVD

	pathTable := builder.NewPathTable(dir)
	pathTableSize := pathTable.Size()

	// Generally the path table should come before the disk contents, if this was an actual CD, to make it easier to
	// skip to the relevant content
	pathTableLBlock := builder.AllocateAndIncrementBlock(&block, pathTableSize)
	pathTableMBlock := builder.AllocateAndIncrementBlock(&block, pathTableSize)

	// Set locations for the files and directories
	builder.RelocateTree(dir, &block)

	pvd, err := builder.NewPrimaryVolumeDescriptor(
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

	bw := builder.NewBlockWriter(w)

	if err := bw.WriteBlock(16, pvd); err != nil {
		return bw.BytesWritten(), fmt.Errorf("failed to write PVD: %w", err)
	}

	if err := bw.WriteBlockFunc(17, func(w io.Writer) (int64, error) {
		cw := counter.NewWriter(w)
		if err := struc.Pack(cw, spec.TerminatorVolumeDescriptor); err != nil {
			return cw.Count(), fmt.Errorf("could not pack structure: %w", err)
		}

		return cw.Count(), nil
	}); err != nil {
		return bw.BytesWritten(), fmt.Errorf("failed to write terminator volume descriptor: %w", err)
	}

	if err := bw.WriteBlock(pathTableLBlock, pathTable.LPathTable()); err != nil {
		return bw.BytesWritten(), fmt.Errorf("failed to write L-type path table: %w", err)
	}

	if err := bw.WriteBlock(pathTableMBlock, pathTable.MPathTable()); err != nil {
		return bw.BytesWritten(), fmt.Errorf("failed to write M-type path table: %w", err)
	}

	for entry := range dir.Walk(false) {
		if err := bw.WriteBlock(entry.Location(), entry); err != nil {
			return bw.BytesWritten(), fmt.Errorf("failed to write entry: %w", err)
		}
	}

	return bw.BytesWritten(), nil
}
