package iso9660

import (
	"fmt"
	"github.com/itchio/headway/counter"
	"github.com/lunixbochs/struc"
	"io"
)

const logicalSectorSize = 2048

type volumeDescriptorType uint8

// Volume descriptor types; note that 4-254 are reserved
const (
	volumeDescriptorTypeBootRecord    volumeDescriptorType = 0
	volumeDescriptorTypePrimary                            = 1
	volumeDescriptorTypeSupplementary                      = 2
	volumeDescriptorTypePartition                          = 3
	volumeDescriptorTypeTerminator                         = 255
)

// volumeDescriptor is the base structure for volume descriptors. All volume descriptors start with this structure
// and follow with descriptor-specific data.
type volumeDescriptor struct {
	Kind                    volumeDescriptorType
	StandardIdentifier      [5]uint8
	VolumeDescriptorVersion uint8
}

type primaryVolumeDescriptor struct {
	Header                         volumeDescriptor
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
	RootDirectoryRecord            [34]uint8
	VolumeSetIdentifier            [128]dCharacter
	PublisherIdentifier            [128]aCharacter
	DataPreparerIdentifier         [128]dCharacter
	ApplicationIdentifier          [128]aCharacter
	CopyrightFileIdentifier        [37]uint8
	AbstractFileIdentifier         [37]uint8
	BibliographicFileIdentifier    [37]uint8
	VolumeCreationDateTime         [17]uint8
	VolumeModificationDateTime     [17]uint8
	VolumeExpirationDateTime       [17]uint8
	VolumeEffectiveDateTime        [17]uint8
	FileStructureVersion           uint8
	Reserved883                    uint8
	ApplicationUse                 [512]uint8
	Reserved1396                   [653]uint8
}

func (p *primaryVolumeDescriptor) WriteTo(w io.Writer) (n int64, err error) {
	cw := counter.NewWriter(w)

	if err := struc.Pack(cw, p); err != nil {
		return cw.Count(), fmt.Errorf("could not pack structure: %w", err)
	}

	return cw.Count(), nil
}
