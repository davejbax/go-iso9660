package iso9660

import "io"

type fileFlag uint8

const (
	fileFlagHidden         fileFlag = 0x01
	fileFlagDirectory               = 0x02
	fileFlagAssociatedFile          = 0x04
	fileFlagRecord                  = 0x08
	fileFlagProtection              = 0x10
	fileFlagMultiExtent             = 0x80
)

type directoryRecord struct {
	Length                        uint8
	ExtendedAttributeRecordLength uint8
	ExtentLocation                uint32BothByteField
	DataLength                    uint32BothByteField
	RecordingDateAndTime          dateTime
	FileFlags                     fileFlag
	FileUnitSize                  uint8
	InterleaveGapSize             uint8
	VolumeSequenceNumber          uint16BothByteField
	LengthOfFileIdentifier        uint8 `struc:"sizeof=FileIdentifier"`

	// This can actually use d1 characters, but for now just use d characters to simplify things
	// TODO: Joliet support
	FileIdentifier []dCharacter
}

var rootDirectoryFileIdentifier = []dCharacter{0x00}

// TODO: write failing unit tests for directory
// TODO: implement directory
type directory struct {
	parent *directory

	entries []*directoryRecord
}

func (d *directory) WriteTo(w io.Writer) (n int64, err error) {
	return 0, nil
}

func (d *directory) Record() *directoryRecord {
	return nil
}

func (d *directory) ParentRecord() *directoryRecord {
	return nil
}
