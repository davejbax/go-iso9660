package iso9660

import (
	"fmt"
	"github.com/davejbax/go-iso9660/internal/builder"
	"github.com/davejbax/go-iso9660/internal/encode"
	"github.com/davejbax/go-iso9660/internal/spec"
	"io"
	"slices"
)

const (
	pvdBlock = 16
	svdBlock = 17
)

// block represents an allocated block in the ISO image, along with its associated writer
type block struct {
	location uint32
	writer   io.WriterTo
}

// volumeDescriptorInputs are common inputs to a volume descriptor factory
type volumeDescriptorInputs struct {
	totalBlocks        uint32
	pathTableSize      uint32
	pathTableLLocation uint32
	pathTableMLocation uint32
	rootDir            *builder.Directory
}

// buildVolume allocates blocks for:
// - a pathtable
// - a directory tree (without files)
//
// and returns a finalizer, which:
// - allocates a block for the volume descriptor
// - allocates blocks for the files at the end of the volume
func (b *Builder) buildVolume(
	image *Image,
	currentBlock *uint32,
	writeFileBlocks bool,
	escapeSeq encode.EscapeSequence,
	descriptorFactory func(inputs *volumeDescriptorInputs) (*block, error),
) (func(image *Image, totalBlocks uint32) error, error) {
	rootDir, err := newDirectoryFromFS(b.source, ".", nil, b.recordedAt, escapeSeq)
	if err != nil {
		return nil, fmt.Errorf("failed to build PVD directory: %w", err)
	}

	pathTable := builder.NewPathTable(rootDir)

	// Place the path tables first in the volume
	pathTableMLocation := builder.AllocateAndIncrementBlock(currentBlock, pathTable.Size())
	pathTableLLocation := builder.AllocateAndIncrementBlock(currentBlock, pathTable.Size())
	image.blocks = append(image.blocks, &block{location: pathTableMLocation, writer: pathTable.MPathTable()})
	image.blocks = append(image.blocks, &block{location: pathTableLLocation, writer: pathTable.LPathTable()})

	// Then, the PVD directories (not files) in breadth-first order
	for dir := range rootDir.Walk(false) {
		// Only locate directories at this stage
		if _, ok := dir.(*builder.Directory); !ok {
			continue
		}

		dir.Relocate(builder.AllocateAndIncrementBlock(currentBlock, dir.PointerRecord().DataLength.RealValue()))
		image.blocks = append(image.blocks, &block{location: dir.Location(), writer: dir})
	}

	return func(image *Image, totalBlocks uint32) error {
		for file := range rootDir.Walk(false) {
			if _, ok := file.(*builder.File); !ok {
				continue
			}

			file.Relocate(builder.AllocateAndIncrementBlock(&totalBlocks, file.PointerRecord().DataLength.RealValue()))

			// Only actually write a block if size is non-zero, and if we're the volume responsible for writing file
			// blocks (we only need to write them once, since file contents don't change per primary/supplementary volume)
			if file.PointerRecord().DataLength.RealValue() > 0 && writeFileBlocks {
				image.blocks = append(image.blocks, &block{location: file.Location(), writer: file})
			}
		}

		descriptor, err := descriptorFactory(&volumeDescriptorInputs{
			totalBlocks:        totalBlocks,
			pathTableSize:      pathTable.Size(),
			pathTableLLocation: pathTableLLocation,
			pathTableMLocation: pathTableMLocation,
			rootDir:            rootDir,
		})
		if err != nil {
			return fmt.Errorf("failed to build volume descriptor: %w", err)
		}

		image.blocks = append(image.blocks, descriptor)

		return nil
	}, nil
}

func (b *Builder) Build() (*Image, error) {
	image := &Image{}

	hasSVD := b.svdOptions != nil

	// The first block we write our non-descriptor data to should be *after* the volume descriptors.
	// PVD, SVD, and TVD each occupy one block. Hence, if we're PVD-only, our starting block is two
	// after the PVD: [0-15 reserved, 16=PVD, 17=TVD, 18=start]
	// If we have an SVD, it's two after the SVD instead.
	currentBlock := uint32(pvdBlock + 2)
	if hasSVD {
		currentBlock = uint32(svdBlock + 2)
	}

	image.blocks = append(image.blocks, &block{
		location: currentBlock - 1, // TVD is by definition the block before our starting block
		writer:   &spec.TerminatorVolumeDescriptor{},
	})

	finalizePVD, err := b.buildVolume(image, &currentBlock, true, encode.EscapeSequenceNone, func(inputs *volumeDescriptorInputs) (*block, error) {
		pvd, err := builder.NewPrimaryVolumeDescriptor(
			"",
			b.metadata.VolumeIdentifier,
			b.metadata.VolumeSetIdentifier,
			b.metadata.PublisherIdentifier,
			dataPreparer,
			b.metadata.ApplicationIdentifier,
			inputs.totalBlocks,
			inputs.pathTableSize,
			inputs.pathTableLLocation,
			0,
			inputs.pathTableMLocation,
			0,
			inputs.rootDir,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to build primary volume descriptor: %w", err)
		}

		return &block{location: pvdBlock, writer: pvd}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build PVD: %w", err)
	}

	var finalizeSVD func(image *Image, totalBlocks uint32) error

	if hasSVD {
		finalizeSVD, err = b.buildVolume(image, &currentBlock, false, b.svdOptions.escapeSequence, func(inputs *volumeDescriptorInputs) (*block, error) {
			svd, err := builder.NewSupplementaryVolumeDescriptor(
				"",
				b.metadata.VolumeIdentifier,
				b.metadata.VolumeSetIdentifier,
				b.metadata.PublisherIdentifier,
				dataPreparer,
				b.metadata.ApplicationIdentifier,
				inputs.totalBlocks,
				inputs.pathTableSize,
				inputs.pathTableLLocation,
				0,
				inputs.pathTableMLocation,
				0,
				inputs.rootDir,
				b.svdOptions.escapeSequence,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to build supplementary volume descriptor: %w", err)
			}

			return &block{location: svdBlock, writer: svd}, nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to build SVD: %w", err)
		}
	}

	if err := finalizePVD(image, currentBlock); err != nil {
		return nil, fmt.Errorf("failed to finalize PVD: %w", err)
	}

	if hasSVD {
		if err := finalizeSVD(image, currentBlock); err != nil {
			return nil, fmt.Errorf("failed to finalize SVD: %w", err)
		}
	}

	// [builder.BlockWriter] doesn't like to write non-sequentially, so sort blocks by their block index/location
	slices.SortFunc(image.blocks, func(a, b *block) int {
		return int(a.location) - int(b.location)
	})

	return image, nil
}
