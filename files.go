package iso9660

type fileFlag uint8

const (
	fileFlagExistence      fileFlag = 0x01
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
	FileIdentifier []dCharacter
}
