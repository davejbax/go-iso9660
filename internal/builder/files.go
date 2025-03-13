package builder

import (
	"fmt"
	"github.com/davejbax/go-iso9660/internal/encode"
	"github.com/davejbax/go-iso9660/internal/spec"
	"io"
	"iter"
	"time"
)

// RelocatableFileSection is an extension of [spec.FileSection] that allows for relocating the file section (which is
// generally the same thing as a file or directory).
//
// This delegates locating files/directories on a disk to the caller,
// which allows for some separation of concerns and is generally neater than trying to tell a directory 'start on this
// block and allocate as you please'. In addition, it solves the problem of path table generation, since a path table
// size depends on the size of the directory tree, and should come before a directory on disk (this produces a chicken-
// and-egg problem).
type RelocatableFileSection interface {
	spec.FileSection

	Location() uint32
	Relocate(newLocation uint32)

	recordLength() uint8
	children() []RelocatableFileSection
}

type Directory struct {
	entries        []RelocatableFileSection
	record         *spec.DirectoryRecord
	parent         *Directory
	realDataLength uint32
}

var _ RelocatableFileSection = &Directory{}
var _ spec.Directory = &Directory{}

func NewEmptyDirectory(identifier spec.FileIdentifier, recordedAt time.Time, parent *Directory) *Directory {
	pointerRecord := &spec.DirectoryRecord{
		Length:                        spec.DirectoryRecordLength(len(identifier)),
		ExtendedAttributeRecordLength: 0,
		ExtentLocation:                encode.AsUInt32BothByte(0),
		DataLength:                    encode.AsUInt32BothByte(logicalBlockSize), // This is the real initial size, rounded up to a block
		RecordingDateAndTime:          encode.AsDateTime(recordedAt),
		FileFlags:                     spec.FileFlagDirectory,

		// These fields are used for interleaving and hence we leave them unset
		FileUnitSize:      0,
		InterleaveGapSize: 0,

		// We aren't supporting multiple volumes currently
		// TODO[multivolume]
		VolumeSequenceNumber: encode.AsUInt16BothByte(1),

		LengthOfFileIdentifier: uint8(len(identifier)),
		FileIdentifier:         identifier,
	}

	d := &Directory{
		record: pointerRecord,
		parent: parent,

		// Every Directory always contains a . (self) and .. (parent) pointerRecord
		// Hence, the Directory payload starts off being that long, assuming there are no entries.
		realDataLength: uint32(spec.DirectoryRecordLength(len(spec.FileIdentifierSelf)) + spec.DirectoryRecordLength(len(spec.FileIdentifierParent))),
	}

	if parent == nil {
		d.parent = d
	}

	return d
}

func (d *Directory) PointerRecord() spec.DirectoryRecord {
	return *d.record
}

func (d *Directory) SelfRecord() spec.DirectoryRecord {
	selfRecord := *d.record
	selfRecord.LengthOfFileIdentifier = uint8(len(spec.FileIdentifierSelf))
	selfRecord.FileIdentifier = spec.FileIdentifierSelf
	selfRecord.Length = spec.DirectoryRecordLength(len(spec.FileIdentifierSelf))

	return selfRecord

	//parentRecord := pointerRecord
	//if parent != nil {
	//	parentRecord = parent.Record()
	//}
	//
	//parentRecord.LengthOfFileIdentifier = uint8(len(spec.FileIdentifierParent))
	//parentRecord.FileIdentifier = spec.FileIdentifierParent
	//parentRecord.Length = spec.DirectoryRecordLength(len(spec.FileIdentifierParent))
}

func (d *Directory) ParentRecord() spec.DirectoryRecord {
	parentRecord := *d.record
	if d.parent != nil {
		parentRecord = d.parent.PointerRecord()
	}

	parentRecord.LengthOfFileIdentifier = uint8(len(spec.FileIdentifierParent))
	parentRecord.FileIdentifier = spec.FileIdentifierParent
	parentRecord.Length = spec.DirectoryRecordLength(len(spec.FileIdentifierParent))

	return parentRecord
}

func (d *Directory) Location() uint32 {
	return d.PointerRecord().ExtentLocation.RealValue()
}

func (d *Directory) Relocate(newLocation uint32) {
	d.record.ExtentLocation = encode.AsUInt32BothByte(newLocation)
}

// Add adds a file section as a direct descendant of a directory.
//
// IMPORTANT: since file sections and identifiers themselves are not directly comparable, it is up to the caller to
// ensure entries are inserted in a spec-compliant sorting order. To ensure this, use [spec.CompareDirectoryEntries]
// before invoking Add.
func (d *Directory) Add(f RelocatableFileSection) {
	d.entries = append(d.entries, f)
	
	// Round the DataLength to the nearest block. For some reason, ISO readers expect this.
	// I guess it sorta makes sense, since it'll be stored across a full extent of blocks.
	d.realDataLength += uint32(f.recordLength())
	d.record.DataLength = encode.AsUInt32BothByte((d.realDataLength + logicalBlockSize - 1) / logicalBlockSize * logicalBlockSize)
}

func (d *Directory) Parent() *Directory {
	return d.parent
}

// Walk returns all directory descendants, including itself, in either breadth-first or depth-first order. In breadth-
// first mode, Walk should obey the ordering constraints implied by [spec.CompareDirectoryEntries].
func (d *Directory) Walk(depthFirst bool) iter.Seq[RelocatableFileSection] {
	return func(yield func(RelocatableFileSection) bool) {
		queue := []RelocatableFileSection{d}
		for len(queue) > 0 {
			node := queue[0]
			queue = queue[1:]

			if !yield(node) {
				break
			}

			if depthFirst {
				queue = append(node.children(), queue...)
			} else {
				queue = append(queue, node.children()...)
			}
		}
	}
}

func (d *Directory) Entries() []spec.FileSection {
	entries := make([]spec.FileSection, len(d.entries))

	for i, entry := range d.entries {
		entries[i] = entry
	}

	return entries
}

func (d *Directory) WriteTo(w io.Writer) (int64, error) {
	dw := &spec.DirectoryWriter{Directory: d}
	return dw.WriteTo(w)
}

func (d *Directory) children() []RelocatableFileSection {
	return d.entries
}

func (d *Directory) recordLength() uint8 {
	return spec.DirectoryRecordLength(len(d.PointerRecord().FileIdentifier))
}

type File struct {
	name       spec.FileIdentifier
	location   uint32
	recordedAt time.Time
	flags      spec.FileFlag

	dataSize uint32
	data     func() (io.Reader, error)
}

func NewFile(identifier spec.FileIdentifier, recordedAt time.Time, dataSize uint32, data func() (io.Reader, error)) *File {
	return &File{
		name:       identifier,
		location:   0,
		recordedAt: recordedAt,
		flags:      0,
		dataSize:   dataSize,
		data:       data,
	}
}

func (f *File) WriteTo(w io.Writer) (int64, error) {
	r, err := f.data()
	if err != nil {
		return 0, fmt.Errorf("failed to get File data: %w", err)
	}

	return io.Copy(w, r)
}

func (f *File) PointerRecord() spec.DirectoryRecord {
	return spec.DirectoryRecord{
		Length:                        f.recordLength(),
		ExtendedAttributeRecordLength: 0,
		ExtentLocation:                encode.AsUInt32BothByte(f.location),
		DataLength:                    encode.AsUInt32BothByte(f.dataSize),
		RecordingDateAndTime:          encode.AsDateTime(f.recordedAt),
		FileFlags:                     f.flags,
		// These fields are used for interleaving and hence we leave them unset
		FileUnitSize:      0,
		InterleaveGapSize: 0,
		// We aren't supporting multiple volumes currently
		// TODO[multivolume]
		VolumeSequenceNumber: encode.AsUInt16BothByte(1),

		LengthOfFileIdentifier: uint8(len(f.name)),
		FileIdentifier:         f.name,
	}
}

func (f *File) Relocate(newLocation uint32) {
	f.location = newLocation
}

func (f *File) Location() uint32 {
	return f.location
}

func (f *File) children() []RelocatableFileSection {
	// Files have no children
	return nil
}

func (f *File) recordLength() uint8 {
	return spec.DirectoryRecordLength(len(f.name))
}

func (f *File) dataLength() uint32 {
	return f.dataSize
}

var _ RelocatableFileSection = &File{}
