package iso9660

import (
	"fmt"
	"github.com/itchio/headway/counter"
	"github.com/lunixbochs/struc"
	"io"
	"iter"
	"time"
)

type fileFlag uint8

const (
	fileFlagHidden         fileFlag = 0x01
	fileFlagDirectory               = 0x02
	fileFlagAssociatedFile          = 0x04
	fileFlagRecord                  = 0x08
	fileFlagProtection              = 0x10
	fileFlagMultiExtent             = 0x80
)

const (
	baseDirectoryRecordSize = 33
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
	FileIdentifier fileIdentifier
}

func (d *directoryRecord) WriteTo(w io.Writer) (int64, error) {
	cw := counter.NewWriter(w)
	if err := struc.Pack(cw, d); err != nil {
		return cw.Count(), fmt.Errorf("failed to pack directory record: %w", err)
	}

	// We don't have a magical 'padding' field, so we need to pad ourselves
	if remainder := int64(d.Length) - cw.Count(); remainder > 0 {
		if _, err := cw.Write(make([]byte, remainder)); err != nil {
			return cw.Count(), fmt.Errorf("failed to pad directory record: %w", err)
		}
	}

	return cw.Count(), nil
}

func directoryRecordLength(nameLength int) uint8 {
	padding := 0

	// When the name is an even number of bytes, the record would end up being an odd number of bytes. Hence, we pad
	// with an extra zero byte to avoid this.
	if nameLength%2 == 0 {
		padding = 1
	}

	return uint8(baseDirectoryRecordSize + nameLength + padding)
}

type fileLike interface {
	Record() *directoryRecord
	RecordLength() uint8
	DataLength() uint32
	Relocate(newLocation uint32)
	Entries() []fileLike
}

type directory struct {
	name       fileIdentifier
	parent     *directory
	location   uint32
	recordedAt time.Time
	flags      fileFlag

	entries []fileLike
}

func (d *directory) WriteTo(w io.Writer) (int64, error) {
	cw := counter.NewWriter(w)

	selfRecord := d.selfRecord()
	parentRecord := d.parentRecord()

	if err := struc.Pack(cw, selfRecord); err != nil {
		return cw.Count(), fmt.Errorf("failed to write self record: %w", err)
	}

	if err := struc.Pack(cw, parentRecord); err != nil {
		return cw.Count(), fmt.Errorf("failed to write parent record: %w", err)
	}

	for _, entry := range d.entries {
		if err := struc.Pack(cw, entry.Record()); err != nil {
			return cw.Count(), fmt.Errorf("failed to write directory entry: %w", err)
		}
	}

	return cw.Count(), nil
}

func (d *directory) Record() *directoryRecord {
	return &directoryRecord{
		Length:                        d.RecordLength(),
		ExtendedAttributeRecordLength: 0,
		ExtentLocation:                uint32BothByte(d.location),
		DataLength:                    uint32BothByte(d.DataLength()),
		RecordingDateAndTime:          newDateTime(d.recordedAt),
		FileFlags:                     fileFlagDirectory | d.flags,

		// These fields are used for interleaving and hence we leave them unset
		FileUnitSize:      0,
		InterleaveGapSize: 0,

		// We aren't supporting multiple volumes currently
		// TODO[multivolume]
		VolumeSequenceNumber: uint16BothByte(1),

		LengthOfFileIdentifier: uint8(len(d.name)),
		FileIdentifier:         d.name,
	}
}

func (d *directory) RecordLength() uint8 {
	return directoryRecordLength(len(d.name))
}

func (d *directory) Relocate(newLocation uint32) {
	d.location = newLocation
}

func (d *directory) Entries() []fileLike {
	return d.entries
}

func (d *directory) DataLength() uint32 {
	// Every directory always contains a . (self) and .. (parent) record
	// Hence, the directory payload is always going to be that long at least.
	size := uint32(directoryRecordLength(len(fileIdentifierSelf)) + directoryRecordLength(len(fileIdentifierParent)))

	for _, entry := range d.entries {
		size += uint32(entry.RecordLength())
	}

	return size
}

func (d *directory) Walk(depthFirst bool) iter.Seq[fileLike] {
	return func(yield func(fileLike) bool) {
		queue := []fileLike{d}
		for len(queue) > 0 {
			node := queue[0]
			queue = queue[1:]

			if !yield(node) {
				break
			}

			if depthFirst {
				queue = append(node.Entries(), queue...)
			} else {
				queue = append(queue, node.Entries()...)
			}
		}
	}
}

func (d *directory) Parent() fileLike {
	return d.parent
}

func (d *directory) selfRecord() *directoryRecord {
	dr := d.Record()
	dr.LengthOfFileIdentifier = uint8(len(fileIdentifierSelf))
	dr.FileIdentifier = fileIdentifierSelf
	dr.Length = directoryRecordLength(len(fileIdentifierSelf))

	return dr
}

func (d *directory) parentRecord() *directoryRecord {
	dr := d.parent.Record()
	dr.LengthOfFileIdentifier = uint8(len(fileIdentifierParent))
	dr.FileIdentifier = fileIdentifierParent
	dr.Length = directoryRecordLength(len(fileIdentifierParent))

	return dr
}

type file struct {
	name       fileIdentifier
	location   uint32
	recordedAt time.Time
	flags      fileFlag

	dataLength uint32
	data       func() (io.Reader, error)
}

func (f *file) WriteTo(w io.Writer) (int64, error) {
	r, err := f.data()
	if err != nil {
		return 0, fmt.Errorf("failed to get file data: %w", err)
	}

	return io.Copy(w, r)
}

func (f *file) Record() *directoryRecord {
	return &directoryRecord{
		Length:                        f.RecordLength(),
		ExtendedAttributeRecordLength: 0,
		ExtentLocation:                uint32BothByte(f.location),
		DataLength:                    uint32BothByte(f.dataLength),
		RecordingDateAndTime:          newDateTime(f.recordedAt),
		FileFlags:                     f.flags,
		// These fields are used for interleaving and hence we leave them unset
		FileUnitSize:      0,
		InterleaveGapSize: 0,
		// We aren't supporting multiple volumes currently
		// TODO[multivolume]
		VolumeSequenceNumber: uint16BothByte(1),

		LengthOfFileIdentifier: uint8(len(f.name)),
		FileIdentifier:         f.name,
	}
}

func (f *file) RecordLength() uint8 {
	return directoryRecordLength(len(f.name))
}

func (f *file) Relocate(newLocation uint32) {
	f.location = newLocation
}

func (f *file) Entries() []fileLike {
	// Files have no entries
	return nil
}

func (f *file) DataLength() uint32 {
	return f.dataLength
}
