package spec

import (
	"fmt"
	"github.com/itchio/headway/counter"
	"github.com/lunixbochs/struc"
	"io"
)

var (
	// StandardIdentifier represents the version of the ISO9660 standard used. Always 'CD001'
	//
	// ECMA-119 (5th ed.) §9.1.3
	StandardIdentifier = [5]uint8{0x43, 0x44, 0x30, 0x30, 0x31}
)

// VolumeDescriptorType identifies the type of volume descriptor in a contiguous array of volume descriptors ("volume
// descriptor set") -- each of which describe the actual volume (in simplified terms, the CD image as a whole).
// Note that types 4-254 are reserved.
//
// ECMA-119 (5th ed.) §9.1.2
type VolumeDescriptorType uint8

const (
	VolumeDescriptorTypeBootRecord    VolumeDescriptorType = 0
	VolumeDescriptorTypePrimary                            = 1
	VolumeDescriptorTypeSupplementary                      = 2
	VolumeDescriptorTypePartition                          = 3
	VolumeDescriptorTypeTerminator                         = 255
)

// FileStructureVersion is a field in the [PrimaryVolumeDescriptor] which indicates the version of the specification for
// records in a directory and in a path table.
//
// ECMA-119 (5th ed.) §9.4.31
type FileStructureVersion uint8

const (
	FileStructureVersionPrimary       FileStructureVersion = 1
	FileStructureVersionSupplementary                      = 1
	FileStructureVersionEnhanced                           = 2
)

// VolumeDescriptor is the base structure for volume descriptors. All volume descriptors start with this structure
// and follow with descriptor-specific data.
//
// Volume descriptors are structures that occur at successive blocks starting at block 16. This list of volume
// descriptors must be terminated by a [TerminatorVolumeDescriptor].
//
// ECMA-119 (5th ed.) §9
type VolumeDescriptor struct {
	Kind                    VolumeDescriptorType
	StandardIdentifier      [5]uint8
	VolumeDescriptorVersion uint8
}

// PrimaryVolumeDescriptor is a type of volume descriptor that contains metadata about the CD image, in addition to
// a pointer to the root directory and path tables -- either of which can be used to traverse the volume.
//
// ECMA-119 (5th ed.) §9.4
type PrimaryVolumeDescriptor struct {
	Header                         *VolumeDescriptor
	Unused8                        uint8
	SystemIdentifier               [32]ACharacter
	VolumeIdentifier               [32]DCharacter
	Unused73                       [8]uint8
	VolumeSpaceSize                UInt32BothByte
	Unused89                       [32]uint8
	VolumeSetSize                  UInt16BothByte
	VolumeSequenceNumber           UInt16BothByte
	LogicalBlockSize               UInt16BothByte
	PathTableSize                  UInt32BothByte
	LocationTypeLPathTable         uint32 `struc:"little"`
	LocationTypeLOptionalPathTable uint32 `struc:"little"`
	LocationTypeMPathTable         uint32 `struc:"big"`
	LocationTypeMOptionalPathTable uint32 `struc:"big"`
	RootDirectoryRecord            *DirectoryRecord
	VolumeSetIdentifier            [128]DCharacter
	PublisherIdentifier            [128]ACharacter
	DataPreparerIdentifier         [128]DCharacter
	ApplicationIdentifier          [128]ACharacter

	// TODO: these are actually d1-characters, I think? Except maybe not in a PVD?
	// They're file identifiers.
	// TODO: figure out how to implement file identifiers in a Joliet-or-not-agnostic sort of way.
	CopyrightFileIdentifier     [37]DCharacter
	AbstractFileIdentifier      [37]DCharacter
	BibliographicFileIdentifier [37]DCharacter
	VolumeCreationDateTime      LongDateTime
	VolumeModificationDateTime  LongDateTime
	VolumeExpirationDateTime    LongDateTime
	VolumeEffectiveDateTime     LongDateTime
	FileStructureVersion        FileStructureVersion
	Reserved883                 uint8
	ApplicationUse              [512]uint8
	Reserved1396                [653]uint8
}

func (p *PrimaryVolumeDescriptor) WriteTo(w io.Writer) (int64, error) {
	cw := counter.NewWriter(w)

	if err := struc.Pack(cw, p); err != nil {
		return cw.Count(), fmt.Errorf("could not pack structure: %w", err)
	}

	return cw.Count(), nil
}

// TerminatorVolumeDescriptor is a volume descriptor with no payload that signals the end of the volume descriptor set.
//
// ECMA-119 (5th ed.) §9.3
var TerminatorVolumeDescriptor = &VolumeDescriptor{
	Kind:                    VolumeDescriptorTypeTerminator,
	StandardIdentifier:      StandardIdentifier,
	VolumeDescriptorVersion: 1, // Always 1
}
