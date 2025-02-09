package iso9660

import (
	"fmt"
	"github.com/itchio/headway/counter"
	"github.com/lunixbochs/struc"
	"io"
)

// Logical sector size is set pretty much unanimously to 2048.
// Logical block size has to be a power of two that's smaller than (or equal to) the logical sector size.
// We don't have any reason to support logical blocks smaller than the logical sector size, and this simplifies
// implementation quite a lot.
const (
	logicalSectorSize = 2048
	logicalBlockSize  = logicalSectorSize
)

type volumeDescriptorType uint8

// Volume descriptor types; note that 4-254 are reserved
const (
	volumeDescriptorTypeBootRecord    volumeDescriptorType = 0
	volumeDescriptorTypePrimary                            = 1
	volumeDescriptorTypeSupplementary                      = 2
	volumeDescriptorTypePartition                          = 3
	volumeDescriptorTypeTerminator                         = 255
)

const (
	fileStructureVersionPrimary       = 1
	fileStructureVersionSupplementary = 1
	fileStructureVersionEnhanced      = 2
)

// volumeDescriptor is the base structure for volume descriptors. All volume descriptors start with this structure
// and follow with descriptor-specific data.
type volumeDescriptor struct {
	Kind                    volumeDescriptorType
	StandardIdentifier      [5]uint8
	VolumeDescriptorVersion uint8
}

type primaryVolumeDescriptor struct {
	Header                         *volumeDescriptor
	Unused8                        uint8
	SystemIdentifier               [32]aCharacter
	VolumeIdentifier               [32]dCharacter
	Unused73                       [8]uint8
	VolumeSpaceSize                uint32BothByteField
	Unused89                       [32]uint8
	VolumeSetSize                  uint16BothByteField
	VolumeSequenceNumber           uint16BothByteField
	LogicalBlockSize               uint16BothByteField
	PathTableSize                  uint32BothByteField
	LocationTypeLPathTable         uint32 `struc:"little"`
	LocationTypeLOptionalPathTable uint32 `struc:"little"`
	LocationTypeMPathTable         uint32 `struc:"big"`
	LocationTypeMOptionalPathTable uint32 `struc:"big"`
	RootDirectoryRecord            *directoryRecord
	VolumeSetIdentifier            [128]dCharacter
	PublisherIdentifier            [128]aCharacter
	DataPreparerIdentifier         [128]dCharacter
	ApplicationIdentifier          [128]aCharacter

	// TODO: these are actually d1-characters, I think? Except maybe not in a PVD?
	// They're file identifiers.
	// TODO: figure out how to implement file identifiers in a Joliet-or-not-agnostic sort of way.
	CopyrightFileIdentifier     [37]dCharacter
	AbstractFileIdentifier      [37]dCharacter
	BibliographicFileIdentifier [37]dCharacter
	VolumeCreationDateTime      longDateTime
	VolumeModificationDateTime  longDateTime
	VolumeExpirationDateTime    longDateTime
	VolumeEffectiveDateTime     longDateTime
	FileStructureVersion        uint8
	Reserved883                 uint8
	ApplicationUse              [512]uint8
	Reserved1396                [653]uint8
}

// TODO: unit test this!
func newPrimaryVolumeDescriptor(
	systemIdentifier, volumeIdentifier, volumeSetIdentifier, publisherIdentifier, dataPreparerIdentifier, applicationIdentifier string,
	volumeSpaceSize uint32,
	pathTableSize uint32,
	pathTableLLocationBlockNumber uint32,
	pathTableLOptionalLocationBlockNumber uint32,
	pathTableMLocationBlockNumber uint32,
	pathTableMOptionalLocationBlockNumber uint32,
	rootDirectory directory,
) (*primaryVolumeDescriptor, error) {
	pvd := &primaryVolumeDescriptor{
		Header: &volumeDescriptor{
			Kind:                    volumeDescriptorTypePrimary,
			StandardIdentifier:      standardIdentifier,
			VolumeDescriptorVersion: 1, // Always 1
		},

		VolumeSpaceSize: uint32BothByte(volumeSpaceSize),

		// We don't currently support multiple volumes in a volume set
		// TODO[multivolume]: add support, if it'd be useful
		VolumeSetSize:        uint16BothByte(1),
		VolumeSequenceNumber: uint16BothByte(1),

		// This is a fixed value to make our implementation simpler;
		// Most ISOs that I've seen do a similar thing.
		LogicalBlockSize: uint16BothByte(logicalBlockSize),

		PathTableSize:                  uint32BothByte(pathTableSize),
		LocationTypeLPathTable:         pathTableLLocationBlockNumber,
		LocationTypeLOptionalPathTable: pathTableLOptionalLocationBlockNumber,
		LocationTypeMPathTable:         pathTableMLocationBlockNumber,
		LocationTypeMOptionalPathTable: pathTableMOptionalLocationBlockNumber,

		RootDirectoryRecord: rootDirectory.Record(),

		// We fill in all of these below
		SystemIdentifier:       [32]aCharacter{},
		VolumeIdentifier:       [32]dCharacter{},
		VolumeSetIdentifier:    [128]dCharacter{},
		PublisherIdentifier:    [128]aCharacter{},
		DataPreparerIdentifier: [128]dCharacter{},
		ApplicationIdentifier:  [128]aCharacter{},

		// These are optional, and hence for now, we don't support setting these fields
		// TODO[future]: add support for copyright, abstract, bibliography, volume create/mod/expire/effective times
		CopyrightFileIdentifier:     [37]dCharacter{},
		AbstractFileIdentifier:      [37]dCharacter{},
		BibliographicFileIdentifier: [37]dCharacter{},
		VolumeCreationDateTime:      zeroLongDateTime,
		VolumeModificationDateTime:  zeroLongDateTime,
		VolumeExpirationDateTime:    zeroLongDateTime,
		VolumeEffectiveDateTime:     zeroLongDateTime,

		FileStructureVersion: fileStructureVersionPrimary,

		// This is an arbitrary field of which the use isn't specified by the standard, hence we leave it blank
		ApplicationUse: [512]uint8{},
	}

	// TODO: consider whether to make 'strict' and 'tryConvert' params to this function
	if err := strToACharacters(systemIdentifier, pvd.SystemIdentifier[:], true, true); err != nil {
		return nil, fmt.Errorf("could not encode system identifier: %w", err)
	}

	if err := strToDCharacters(volumeIdentifier, pvd.VolumeIdentifier[:], true, true); err != nil {
		return nil, fmt.Errorf("could not encode volume identifier: %w", err)
	}

	if err := strToDCharacters(volumeSetIdentifier, pvd.VolumeSetIdentifier[:], true, true); err != nil {
		return nil, fmt.Errorf("could not encode volume set identifier: %w", err)
	}

	if err := strToACharacters(publisherIdentifier, pvd.PublisherIdentifier[:], true, true); err != nil {
		return nil, fmt.Errorf("could not encode volume set identifier: %w", err)
	}

	if err := strToDCharacters(dataPreparerIdentifier, pvd.DataPreparerIdentifier[:], true, true); err != nil {
		return nil, fmt.Errorf("could not encode data preparer identifier: %w", err)
	}

	if err := strToACharacters(applicationIdentifier, pvd.ApplicationIdentifier[:], true, true); err != nil {
		return nil, fmt.Errorf("could not encode application identifier: %w", err)
	}

	// These fields MUST be zeroed in order to indicate that they don't exist.
	// I don't know what would happen if they were 0x00s rather than the filler byte; I assume the CD drive would explode.
	zeroCharacterArray(pvd.CopyrightFileIdentifier[:])
	zeroCharacterArray(pvd.AbstractFileIdentifier[:])
	zeroCharacterArray(pvd.BibliographicFileIdentifier[:])

	return pvd, nil
}

func (p *primaryVolumeDescriptor) WriteTo(w io.Writer) (int64, error) {
	cw := counter.NewWriter(w)

	if err := struc.Pack(cw, p); err != nil {
		return cw.Count(), fmt.Errorf("could not pack structure: %w", err)
	}

	return cw.Count(), nil
}
