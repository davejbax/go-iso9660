package spec

import (
	"encoding/binary"
	"fmt"
	"github.com/itchio/headway/counter"
	"github.com/lunixbochs/struc"
	"io"
	"iter"
	"strings"
)

// FileIdentifier is a 'file identifier', used in both files and directories. Note that the spec uses the term
// 'file name' to refer to the part of a file identifier that is distinct from the extension; a file identifier, in
// contrast, represents the combination of file name, extension, and version. For directories, directory name is
// equivalent to directory file identifier.
//
// A file identifier consists either of [DCharacter] or 'd1-characters'. The latter is a variable encoding, dependent
// on the escape sequence given in the supplementary volume descriptor. If no supplementary volume descriptor is given,
// then [DCharacter]-s are assumed.
//
// ECMA-119 (5th ed.) §8.5
type FileIdentifier []uint8

var (
	// FileIdentifierSelf is the root file identifier, and also represents the '.' entry in a directory
	FileIdentifierSelf = FileIdentifier{0x00}

	// FileIdentifierParent represents the '..' (parent) entry in a directory
	FileIdentifierParent = FileIdentifier{0x01}
)

type FileFlag uint8

const (
	FileFlagHidden         FileFlag = 0x01
	FileFlagDirectory               = 0x02
	FileFlagAssociatedFile          = 0x04
	FileFlagRecord                  = 0x08
	FileFlagProtection              = 0x10
	FileFlagMultiExtent             = 0x80
)

// DirectoryRecord specifies the file identifier, size, and location of a file or directory
//
// DirectoryRecord implements [io.WriterTo] for serialization
//
// ECMA-119 (5th ed.) §10.1
type DirectoryRecord struct {
	Length                        uint8
	ExtendedAttributeRecordLength uint8
	ExtentLocation                UInt32BothByte
	DataLength                    UInt32BothByte
	RecordingDateAndTime          DateTime
	FileFlags                     FileFlag
	FileUnitSize                  uint8
	InterleaveGapSize             uint8
	VolumeSequenceNumber          UInt16BothByte
	LengthOfFileIdentifier        uint8 `struc:"sizeof=FileIdentifier"`

	// This can actually use d1 characters, but for now just use d characters to simplify things
	// TODO: Joliet support
	FileIdentifier FileIdentifier

	// TODO system use?
}

// Ensure DirectoryRecord implements [io.WriterTo]
var _ io.WriterTo = &DirectoryRecord{}

func (d DirectoryRecord) WriteTo(w io.Writer) (int64, error) {
	cw := counter.NewWriter(w)
	if err := struc.Pack(cw, &d); err != nil {
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

// Directory records are 33 bytes
const baseDirectoryRecordSize = 33

// DirectoryRecordLength calculates the length of a [DirectoryRecord] given the length of its file identifier.
// Note that it is assumed in this implementation (and by the [DirectoryRecord] struct) that there are no system use
// bytes, i.e. LEN_SU in the spec is assumed to be zero in all cases.
//
// ECMA-119 (5th ed.) §10.1
func DirectoryRecordLength(fileIdentifierLength int) uint8 {
	// TODO system use?
	padding := 0

	// When the name is an even number of bytes, the record would end up being an odd number of bytes. Hence, we pad
	// with an extra zero byte to avoid this.
	if fileIdentifierLength%2 == 0 {
		padding = 1
	}

	return uint8(baseDirectoryRecordSize + fileIdentifierLength + padding)
}

type PointedTo interface {
	// PointerRecord is a [DirectoryRecord] describing this file or directory, which should appear in a directory or in
	// a volume descriptor to locate this file or directory.
	PointerRecord() DirectoryRecord
}

// A FileSection is a part of a file, which may be the whole file, recorded in a single extent (set of contiguous
// logical blocks). The spec allows for file sections to be interleaved, however this interface - for simplicity - does
// not.
//
// Each file section has a corresponding [DirectoryRecord] in a [Directory]. The records of file sections for a single
// file that comprises multiple file sections should appear in sequence within a [Directory].
//
// ECMA-119 (5th ed.) §7.4.4, §7.5.1. Relevant terms: 4.7 & 4.8
type FileSection interface {
	io.WriterTo
	PointedTo
}

// Directory is a single file section consisting entirely of [DirectoryRecord] structures. Each directory other than the
// root directory must be referenced by another directory. In the case of the root directory, the [Directory.SelfRecord]
// and [Directory.ParentRecord] fields should both be set to the record describing the root directory.
//
// Implementations can become a [FileSection] implementation in addition by utilising the [DirectoryWriter] utility:
// writing directories is very clearly specified in the spec, and thus is not implementation-specific, which is why this
// Directory interface is not itself a [FileSection] (as directories in themselves do not have implementation-specific
// write behaviour).
//
// ECMA-119 (5th ed.) §7.8.1
type Directory interface {
	PointedTo

	// SelfRecord is a [DirectoryRecord] describing this directory with the [FileIdentifierSelf] file identifier
	SelfRecord() DirectoryRecord

	// ParentRecord is a [DirectoryRecord] describing this directory's parent with the [FileIdentifierParent] file identifier
	ParentRecord() DirectoryRecord

	// Entries is an ordered sequence of [FileSection]-s representing the directory's direct descendants
	Entries() []FileSection
}

type DirectoryWriter struct {
	Directory Directory
}

var _ FileSection = &DirectoryWriter{}

func (d *DirectoryWriter) PointerRecord() DirectoryRecord {
	return d.Directory.PointerRecord()
}

func (d *DirectoryWriter) WriteTo(w io.Writer) (int64, error) {
	cw := counter.NewWriter(w)

	selfRecord := d.Directory.SelfRecord()
	parentRecord := d.Directory.ParentRecord()

	if _, err := selfRecord.WriteTo(cw); err != nil {
		return cw.Count(), fmt.Errorf("failed to write self record: %w", err)
	}

	if _, err := parentRecord.WriteTo(cw); err != nil {
		return cw.Count(), fmt.Errorf("failed to write parent record: %w", err)
	}

	for _, entry := range d.Directory.Entries() {
		if _, err := entry.PointerRecord().WriteTo(cw); err != nil {
			return cw.Count(), fmt.Errorf("failed to write Directory entry: %w", err)
		}
	}

	if diff := selfRecord.DataLength.RealValue() - uint32(cw.Count()); diff > 0 {
		if _, err := cw.Write(make([]byte, diff)); err != nil {
			return cw.Count(), fmt.Errorf("failed to write padding: %w", err)
		}
	}

	return cw.Count(), nil
}

// DirectoryEntry is an abstract representation of a [DirectoryRecord] for use in [CompareDirectoryEntries].
type DirectoryEntry interface {
	// Name is the file name part (not extension!) of the file or directory record, decoded as a Go string. Comparison
	// of this string with Go string comparison should be equivalent to comparison in whatever character set is used for
	// the file identifier.
	Name() string

	// Extension is the file extension
	Extension() string

	// Version is the file version, as numerical digits. These do not have to be zero-padded, and should reflect the
	// version contained in the file identifier.
	Version() string

	// IsDir returns true if the entry represents a directory, and false if it represents a file
	IsDir() bool

	// FileSectionIndex is the number of the file section represented by the directory record. In a directory,
	// successive directory records for the same file represent contiguous, ascending file section numbers.
	FileSectionIndex() int
}

// CompareDirectoryEntries compares two abstract 'directory entries', as would be contained within the data for a
// 'directory'. This results in an ordering consistent with the spec's required 'order of directory records', which is
// (in descending order of significance):
//   - Ascending by file name
//   - Ascending by file extension
//   - Descending by file version number (string-wise), where the shorter version number is padded with the '0' character.
//   - Descending according to the value of the associated file bit of the file flags field, i.e. directories come before
//     files.
//   - The order of the file sections of the file
//
// Note that a [DirectoryRecord] is not directly comparable, because [FileIdentifier] comparison is
// implementation-specific: the spec itself does not specify how to compare file identifiers, except that they must
// be compared character-wise, and that the characters are d1-characters which can be from any coded graphic character
// set (e.g. UTF-16).
//
// Returns a value < 0 if a should come before b, > 0 if b should come before a, and 0 otherwise.
//
// ECMA-119 (5th ed.) §10.3
func CompareDirectoryEntries(a, b DirectoryEntry) int {
	//// Files comprise 'file names' and 'file name extensions', separated by a period
	//aFilename, aExtension, _ := strings.Cut(a.Name(), ".")
	//bFilename, bExtension, _ := strings.Cut(b.Name(), ".")

	// First, sort by file name
	if cmp := strings.Compare(a.Name(), b.Name()); cmp != 0 {
		return cmp
	}

	// If file name is the same, compare extensions
	if cmp := strings.Compare(a.Extension(), b.Extension()); cmp != 0 {
		return cmp
	}

	// TODO: version and file section comparisons!

	aIsDir := a.IsDir()
	bIsDir := b.IsDir()

	// Sort directories before files, if everything else is equal
	if aIsDir && !bIsDir {
		return -1
	} else if !aIsDir && bIsDir {
		return 1
	}

	return 0
}

// PathTableRecord is a record in a path table, which indicates where to find a given directory on the disc by its
// extent number. This can be used by software to quickly find a given directory, rather than having to scan through
// the whole directory tree.
//
// A path table is simply a contiguous array of path table records.
//
// Note that this record is serializable with the [struc] library. Path tables can be either L-type (little endian) or
// M-type (big endian); this record type can be serialized to either, provided the correct arguments to [struc] are
// provided.
//
// ECMA-119 (5th ed.) §7.10
type PathTableRecord struct {
	LengthOfDirectoryIdentifier   uint8 `struc:"sizeof=DirectoryIdentifier"`
	ExtendedAttributeRecordLength uint8
	LocationOfExtent              uint32
	ParentDirectoryNumber         uint16
	DirectoryIdentifier           FileIdentifier
}

// An MPathTable is a path table (contiguous array of [PathTableRecord]-s) that gets serialized with big endian numbers
// ECMA-119 (5th ed.) §9.4.17
type MPathTable iter.Seq[*PathTableRecord]

func (p MPathTable) WriteTo(w io.Writer) (int64, error) {
	return writePathTable(w, iter.Seq[*PathTableRecord](p), true)
}

// An LPathTable is a path table (contiguous array of [PathTableRecord]-s) that gets serialized with little endian numbers
// ECMA-119 (5th ed.) §9.4.15
type LPathTable iter.Seq[*PathTableRecord]

func (p LPathTable) WriteTo(w io.Writer) (int64, error) {
	return writePathTable(w, iter.Seq[*PathTableRecord](p), false)
}

func writePathTable(w io.Writer, records iter.Seq[*PathTableRecord], bigEndian bool) (int64, error) {
	cw := counter.NewWriter(w)

	var byteOrder binary.ByteOrder = binary.LittleEndian
	if bigEndian {
		byteOrder = binary.BigEndian
	}

	for record := range records {
		if err := struc.PackWithOptions(cw, record, &struc.Options{Order: byteOrder}); err != nil {
			return cw.Count(), fmt.Errorf("failed to encode path table record: %w", err)
		}

		// If the directory name has an odd length, the spec requires us to add a padding byte so that this path
		// table record consists of an even number of bytes.
		if record.LengthOfDirectoryIdentifier%2 == 1 {
			if _, err := cw.Write([]byte{0}); err != nil {
				return cw.Count(), fmt.Errorf("failed to write padding byte: %w", err)
			}
		}
	}

	return cw.Count(), nil
}
