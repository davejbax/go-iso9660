package builder

import (
	"fmt"
	"github.com/davejbax/go-iso9660/internal/encode"
	"github.com/davejbax/go-iso9660/internal/spec"
)

func NewPrimaryVolumeDescriptor(
	systemIdentifier, volumeIdentifier, volumeSetIdentifier, publisherIdentifier, dataPreparerIdentifier, applicationIdentifier string,
	volumeSpaceSize uint32,
	pathTableSize uint32,
	pathTableLLocationBlockNumber uint32,
	pathTableLOptionalLocationBlockNumber uint32,
	pathTableMLocationBlockNumber uint32,
	pathTableMOptionalLocationBlockNumber uint32,
	rootDirectory *Directory,
) (*spec.PrimaryVolumeDescriptor, error) {
	rootRecord := rootDirectory.SelfRecord()
	pvd := &spec.PrimaryVolumeDescriptor{
		Header: &spec.VolumeDescriptor{
			Kind:                    spec.VolumeDescriptorTypePrimary,
			StandardIdentifier:      spec.StandardIdentifier,
			VolumeDescriptorVersion: 1, // Always 1
		},

		VolumeSpaceSize: encode.AsUInt32BothByte(volumeSpaceSize),

		// We don't currently support multiple volumes in a volume set
		// TODO[multivolume]: Add support, if it'd be useful
		VolumeSetSize:        encode.AsUInt16BothByte(1),
		VolumeSequenceNumber: encode.AsUInt16BothByte(1),

		// This is a fixed value to make our implementation simpler;
		// Most ISOs that I've seen do a similar thing.
		LogicalBlockSize: encode.AsUInt16BothByte(logicalBlockSize),

		PathTableSize:                  encode.AsUInt32BothByte(pathTableSize),
		LocationTypeLPathTable:         pathTableLLocationBlockNumber,
		LocationTypeLOptionalPathTable: pathTableLOptionalLocationBlockNumber,
		LocationTypeMPathTable:         pathTableMLocationBlockNumber,
		LocationTypeMOptionalPathTable: pathTableMOptionalLocationBlockNumber,

		RootDirectoryRecord: &rootRecord,

		// We fill in all of these below
		SystemIdentifier:       [32]spec.ACharacter{},
		VolumeIdentifier:       [32]spec.DCharacter{},
		VolumeSetIdentifier:    [128]spec.DCharacter{},
		PublisherIdentifier:    [128]spec.ACharacter{},
		DataPreparerIdentifier: [128]spec.DCharacter{},
		ApplicationIdentifier:  [128]spec.ACharacter{},

		// These are optional, and hence for now, we don't support setting these fields
		// TODO[future]: Add support for copyright, abstract, bibliography, volume create/mod/expire/effective times
		CopyrightFileIdentifier:     [37]spec.DCharacter{},
		AbstractFileIdentifier:      [37]spec.DCharacter{},
		BibliographicFileIdentifier: [37]spec.DCharacter{},
		VolumeCreationDateTime:      spec.ZeroLongDateTime,
		VolumeModificationDateTime:  spec.ZeroLongDateTime,
		VolumeExpirationDateTime:    spec.ZeroLongDateTime,
		VolumeEffectiveDateTime:     spec.ZeroLongDateTime,

		FileStructureVersion: spec.FileStructureVersionPrimary,

		// This is an arbitrary field of which the use isn't specified by the standard, hence we leave it blank
		ApplicationUse: [512]uint8{},
	}

	// TODO: consider whether to make 'strict' and 'tryConvert' params to this function
	if err := encode.AsACharacters(systemIdentifier, pvd.SystemIdentifier[:], true, true); err != nil {
		return nil, fmt.Errorf("could not encode system identifier: %w", err)
	}

	if err := encode.AsDCharacters(volumeIdentifier, pvd.VolumeIdentifier[:], true, true); err != nil {
		return nil, fmt.Errorf("could not encode volume identifier: %w", err)
	}

	if err := encode.AsDCharacters(volumeSetIdentifier, pvd.VolumeSetIdentifier[:], true, true); err != nil {
		return nil, fmt.Errorf("could not encode volume set identifier: %w", err)
	}

	if err := encode.AsACharacters(publisherIdentifier, pvd.PublisherIdentifier[:], true, true); err != nil {
		return nil, fmt.Errorf("could not encode volume set identifier: %w", err)
	}

	if err := encode.AsDCharacters(dataPreparerIdentifier, pvd.DataPreparerIdentifier[:], true, true); err != nil {
		return nil, fmt.Errorf("could not encode data preparer identifier: %w", err)
	}

	if err := encode.AsACharacters(applicationIdentifier, pvd.ApplicationIdentifier[:], true, true); err != nil {
		return nil, fmt.Errorf("could not encode application identifier: %w", err)
	}

	// These fields MUST be zeroed in order to indicate that they don't exist.
	// I don't know what would happen if they were 0x00s rather than the filler byte; I assume the CD drive would explode.
	encode.ZeroCharacterArray(pvd.CopyrightFileIdentifier[:], encode.EscapeSequenceNone)
	encode.ZeroCharacterArray(pvd.AbstractFileIdentifier[:], encode.EscapeSequenceNone)
	encode.ZeroCharacterArray(pvd.BibliographicFileIdentifier[:], encode.EscapeSequenceNone)

	return pvd, nil
}

func NewSupplementaryVolumeDescriptor(
	systemIdentifier, volumeIdentifier, volumeSetIdentifier, publisherIdentifier, dataPreparerIdentifier, applicationIdentifier string,
	volumeSpaceSize uint32,
	pathTableSize uint32,
	pathTableLLocationBlockNumber uint32,
	pathTableLOptionalLocationBlockNumber uint32,
	pathTableMLocationBlockNumber uint32,
	pathTableMOptionalLocationBlockNumber uint32,
	rootDirectory *Directory,
	escapeSequence encode.EscapeSequence,
) (*spec.SupplementaryVolumeDescriptor, error) {
	rootRecord := rootDirectory.PointerRecord()

	svd := &spec.SupplementaryVolumeDescriptor{
		Header: &spec.VolumeDescriptor{
			Kind:                    spec.VolumeDescriptorTypeSupplementary,
			StandardIdentifier:      spec.StandardIdentifier,
			VolumeDescriptorVersion: 1, // Always 1
		},

		VolumeFlags:     0, // Escape sequence should always be a standard - we don't implement non-standard ones
		VolumeSpaceSize: encode.AsUInt32BothByte(volumeSpaceSize),

		VolumeSetSize:        encode.AsUInt16BothByte(1),
		VolumeSequenceNumber: encode.AsUInt16BothByte(1),

		LogicalBlockSize: encode.AsUInt16BothByte(logicalBlockSize),

		// TODO[multivolume]: Add support, if it'd be useful
		// We don't currently support multiple volumes in a volume set
		PathTableSize:                  encode.AsUInt32BothByte(pathTableSize),
		LocationTypeLPathTable:         pathTableLLocationBlockNumber,
		LocationTypeLOptionalPathTable: pathTableLOptionalLocationBlockNumber,
		LocationTypeMPathTable:         pathTableMLocationBlockNumber,
		LocationTypeMOptionalPathTable: pathTableMOptionalLocationBlockNumber,

		RootDirectoryRecord: &rootRecord,

		// We fill in all of these below
		EscapeSequences:        [32]uint8{},
		SystemIdentifier:       [32]spec.CCharacterByte{},
		VolumeIdentifier:       [32]spec.CCharacterByte{},
		VolumeSetIdentifier:    [128]spec.CCharacterByte{},
		PublisherIdentifier:    [128]spec.CCharacterByte{},
		DataPreparerIdentifier: [128]spec.CCharacterByte{},
		ApplicationIdentifier:  [128]spec.CCharacterByte{},

		// These are optional, and hence for now, we don't support setting these fields
		// TODO[future]: Add support for copyright, abstract, bibliography, volume create/mod/expire/effective times
		CopyrightFileIdentifier:     [37]spec.CCharacterByte{},
		AbstractFileIdentifier:      [37]spec.CCharacterByte{},
		BibliographicFileIdentifier: [37]spec.CCharacterByte{},
		VolumeCreationDateTime:      spec.ZeroLongDateTime,
		VolumeModificationDateTime:  spec.ZeroLongDateTime,
		VolumeExpirationDateTime:    spec.ZeroLongDateTime,
		VolumeEffectiveDateTime:     spec.ZeroLongDateTime,

		FileStructureVersion: spec.FileStructureVersionSupplementary,
	}

	// Only add one escape sequence: we don't currently support others, as there isn't a need for them -- we only have
	// one charset!
	copy(svd.EscapeSequences[:], escapeSequence.Identifier())

	if err := encode.AsCCharacters(systemIdentifier, svd.SystemIdentifier[:], escapeSequence); err != nil {
		return nil, fmt.Errorf("could not encode system identifier: %w", err)
	}

	if err := encode.AsCCharacters(volumeIdentifier, svd.VolumeIdentifier[:], escapeSequence); err != nil {
		return nil, fmt.Errorf("could not encode volume identifier: %w", err)
	}

	if err := encode.AsCCharacters(volumeSetIdentifier, svd.VolumeSetIdentifier[:], escapeSequence); err != nil {
		return nil, fmt.Errorf("could not encode volume set identifier: %w", err)
	}

	if err := encode.AsCCharacters(publisherIdentifier, svd.PublisherIdentifier[:], escapeSequence); err != nil {
		return nil, fmt.Errorf("could not encode volume set identifier: %w", err)
	}

	if err := encode.AsCCharacters(dataPreparerIdentifier, svd.DataPreparerIdentifier[:], escapeSequence); err != nil {
		return nil, fmt.Errorf("could not encode data preparer identifier: %w", err)
	}

	if err := encode.AsCCharacters(applicationIdentifier, svd.ApplicationIdentifier[:], escapeSequence); err != nil {
		return nil, fmt.Errorf("could not encode application identifier: %w", err)
	}

	encode.ZeroCharacterArray(svd.CopyrightFileIdentifier[:], escapeSequence)
	encode.ZeroCharacterArray(svd.AbstractFileIdentifier[:], escapeSequence)
	encode.ZeroCharacterArray(svd.BibliographicFileIdentifier[:], escapeSequence)

	return svd, nil
}
