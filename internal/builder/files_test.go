package builder_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/davejbax/go-iso9660/internal/builder"
	"github.com/davejbax/go-iso9660/internal/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"slices"
	"testing"
	"time"
)

//var walkTestDirectory = fstest.MapFS{
//	"FOO.TXT": &fstest.MapFile{
//		Data:    []byte("foo"),
//		Mode:    0o644,
//		ModTime: time.Time{},
//		Sys:     nil,
//	},
//	"BAR.TXT": &fstest.MapFile{
//		Data:    []byte("hello world"),
//		Mode:    0o644,
//		ModTime: time.Time{},
//		Sys:     nil,
//	},
//	"BAR.DAT": &fstest.MapFile{
//		Data:    []byte("hello world"),
//		Mode:    0o644,
//		ModTime: time.Time{},
//		Sys:     nil,
//	},
//	"ABC": &fstest.MapFile{
//		Data:    nil,
//		Mode:    0o755 | fs.ModeDir,
//		ModTime: time.Time{},
//		Sys:     nil,
//	},
//	"ABC/NESTED.TXT": &fstest.MapFile{
//		Data:    []byte("hello world"),
//		Mode:    0o644,
//		ModTime: time.Time{},
//		Sys:     nil,
//	},
//	"BAR": &fstest.MapFile{
//		Data:    nil,
//		Mode:    0o755 | fs.ModeDir,
//		ModTime: time.Time{},
//		Sys:     nil,
//	},
//	"BAR/NESTED/ZZZ.TXT": &fstest.MapFile{
//		Data:    []byte("hello world"),
//		Mode:    0o644,
//		ModTime: time.Time{},
//		Sys:     nil,
//	},
//	"BAR/NESTED/AAA/ZZZZ.TXT": &fstest.MapFile{
//		Data:    []byte("hello world"),
//		Mode:    0o644,
//		ModTime: time.Time{},
//		Sys:     nil,
//	},
//}

//func TestDirectory_WriteTo(t *testing.T) {
//	d, err := builder.NewDirectoryFromFS(walkTestDirectory, ".", nil, time.Now())
//	require.NoError(t, err, "NewDirectoryFromFS should not return an error for a valid filesystem")
//
//	var expected bytes.Buffer
//
//	if _, err := d.selfRecord().WriteTo(&expected); err != nil {
//		t.Fatalf("failed to write self record to expected buffer: %v", err)
//	}
//
//	if _, err := d.parentRecord().WriteTo(&expected); err != nil {
//		t.Fatalf("failed to write parent record to expected buffer: %v", err)
//	}
//
//	// Walk() with argument false will return the directory, and then all of its descendants in breadth-first order, in
//	// the spec-enforced order of directory records.
//	// The first element is the root directory, so ignore that.
//	tree := slices.Collect(d.Walk(false))
//	children := tree[1:6]
//
//	actualChildrenNames := make([]string, len(children))
//	for i, child := range children {
//		actualChildrenNames[i] = string(child.Record().FileIdentifier)
//	}
//
//	assert.Equal(t, []string{
//		"ABC",
//		"BAR",
//		"BAR.DAT;1",
//		"BAR.TXT;1",
//		"FOO.TXT;1",
//	}, actualChildrenNames, "Walk(false) should return entries in spec-enforced order of directory records for a given set of direct descendants")
//
//	for _, entry := range children {
//		if _, err := entry.Record().WriteTo(&expected); err != nil {
//			t.Fatalf("failed to write entry to expected buffer: %v", err)
//		}
//	}
//
//	var actual bytes.Buffer
//
//	bytesWritten, err := d.WriteTo(&actual)
//	require.NoError(t, err, "WriteTo should not return an error for a valid Directory")
//
//	assert.GreaterOrEqual(t, bytesWritten, int64(expected.Len()), "WriteTo should write at least as many bytes as the concatenation of self record, parent record, and entry records")
//	assert.Equal(t, int64(actual.Len()), bytesWritten, "WriteTo should report a count of bytes written equal to the actual number of bytes written")
//	assert.Equal(t, expected.Bytes(), actual.Bytes()[:expected.Len()], "WriteTo should concatenate the self record, parent record, and child children")
//	assert.Equal(t, d.selfRecord().DataLength.RealValue(), uint32(actual.Len()), "WriteTo should write a number of bytes equal to the reported dataSize in the self directory record")
//}

// TODO test adding to a subdir doesn't depend on whether subdir added to parent before or after

func TestDirectory_Add(t *testing.T) {
	// We want to test that:
	// - Adding a directory causes it to appear in the entries
	// - Adding a file causes it to appear in the entries
	// - Entries appear in the order they were added (!!TODO!! Remove this requirement and make it so Add adds in spec order)
	// - The size of the directory is updated appropriately (even when we exceed one block!)
	// - It's possible to modify items in the tree -- i.e. Add is referencing, rather than copying

	root := builder.NewEmptyDirectory(spec.FileIdentifierSelf, time.Now(), nil)
	assert.Len(t, root.Entries(), 0, "Directory should be empty before Add()")
	assert.Equal(t, uint32(2048), root.SelfRecord().DataLength.RealValue(), "Directory DataLength should be correct before adding children")

	child := builder.NewEmptyDirectory(spec.FileIdentifier("FOO"), time.Now(), root)
	assert.Len(t, root.Entries(), 0, "Directory should remain empty even if a child is created with it as a parent")

	// Test adding a directory
	root.Add(child)
	require.Len(t, root.Entries(), 1, "Directory should have correct number of entries after adding a directory")
	assert.Equal(t, child, root.Entries()[0], "Add() should add by reference to Entries()")
	assert.Equal(t, uint32(2048), root.SelfRecord().DataLength.RealValue(), "Directory DataLength should not change if it accommodates the size of an added directory")

	// Test that adding a file works, and that Entries() respects the order of Add()
	file := builder.NewFile(spec.FileIdentifier("ZZZ"), time.Now(), 1, func() (io.Reader, error) { return bytes.NewReader([]byte("a")), nil })
	root.Add(file)
	require.Len(t, root.Entries(), 2, "Directory should have correct number of entries after adding a file")
	assert.Equal(t, []spec.FileSection{
		child,
		file,
	}, root.Entries(), "Add() should add files by reference to Entries(), and Entries() should be in the order files were added")
	assert.Equal(t, uint32(2048), root.SelfRecord().DataLength.RealValue(), "Directory DataLength should not change if it accommodates the size of an added file")

	// Test that the Directory grows as files are added
	for i := 0; i < 100; i++ {
		root.Add(builder.NewFile(spec.FileIdentifier(fmt.Sprintf("ZZZ%03d", i)), time.Now(), 1, func() (io.Reader, error) { return bytes.NewReader([]byte("a")), nil }))
	}
	assert.Equal(t, uint32(6144), root.SelfRecord().DataLength.RealValue(), "Directory DataLength should grow to accommodate a large number of files")

	t.Run("Child directories modified after being added should result in a change in the parent directory's payload", func(t *testing.T) {
		bytesBeforeChangingDirectory := bytes.NewBuffer(nil)
		_, err := root.WriteTo(bytesBeforeChangingDirectory)
		require.NoError(t, err, "WriteTo() should not produce an error")
		assert.Equal(t, 6144, bytesBeforeChangingDirectory.Len(), "WriteTo() should write the correct number of bytes")

		// Add a bunch of files to child, so that its DataLength changes. This should cause a change in the root directory's
		// payload, as the 'child' PointerRecord will have a different DataLength
		dataLengthBefore := child.PointerRecord().DataLength
		for i := 0; i < 100; i++ {
			child.Add(builder.NewFile(spec.FileIdentifier(fmt.Sprintf("ZZZ%03d", i)), time.Now(), 1, func() (io.Reader, error) { return bytes.NewReader([]byte("a")), nil }))
		}
		dataLengthAfter := child.PointerRecord().DataLength

		dataLengthBeforeBytes := make([]byte, 8)
		dataLengthAfterBytes := make([]byte, 8)
		if _, err := binary.Encode(dataLengthBeforeBytes, binary.BigEndian, dataLengthBefore); err != nil {
			t.Fatalf("failed to encode data length as binary: %v", err)
		}
		if _, err := binary.Encode(dataLengthAfterBytes, binary.BigEndian, dataLengthAfter); err != nil {
			t.Fatalf("failed to encode data length as binary: %v", err)
		}

		bytesAfterChangingDirectory := bytes.NewBuffer(nil)
		_, err = root.WriteTo(bytesAfterChangingDirectory)
		require.NoError(t, err, "WriteTo() should not produce an error")
		require.Equal(t, 6144, bytesAfterChangingDirectory.Len(), "WriteTo() should write the correct number of bytes")

		// child's PointerRecord now has a different DataLength. Since the payload of root consists of the PointerRecords of
		// its entries, the written data should now be different.
		assert.NotEqual(t, bytesAfterChangingDirectory.Bytes(), bytesBeforeChangingDirectory.Bytes(), "Add() should add by reference, such that WriteTo() reflects any changes to PointerRecords of entries in a directory after they've been added")

		// We expect as many bytes to differ as there are differing bytes in the DataLength fields
		expectedDifferingBytes := countDifferingBytes(dataLengthBeforeBytes, dataLengthAfterBytes)
		if expectedDifferingBytes == 0 {
			t.Fatal("unexpected zero number of differing DataLength bytes!")
		}
		assert.Equal(t, expectedDifferingBytes, countDifferingBytes(bytesBeforeChangingDirectory.Bytes(), bytesAfterChangingDirectory.Bytes()), "Exactly 2 bytes should differ in root directory payload after modifying the DataLength field of one of its entries")
	})
}

func countDifferingBytes(a []byte, b []byte) int {
	total := 0
	for i, v := range a {
		if i >= len(b) {
			return -1
		}

		if v != b[i] {
			total++
		}
	}

	return total
}

func TestDirectory_Walk(t *testing.T) {
	fooData := func() (io.Reader, error) {
		return bytes.NewReader([]byte("foo")), nil
	}
	fooDataLength := uint32(3)

	// <root>
	//   DIR1/
	//     DIR2/
	//       FILE3
	//     FILE2
	//   FILE1
	//
	// BFS should be DIR1, FILE1, DIR2, FILE2, FILE3
	// DFS should be DIR1, DIR2, FILE3, FILE2, FILE1
	dir0 := builder.NewEmptyDirectory(spec.FileIdentifierSelf, time.Now(), nil)
	dir1 := builder.NewEmptyDirectory(spec.FileIdentifier("DIR1"), time.Now(), dir0)
	dir2 := builder.NewEmptyDirectory(spec.FileIdentifier("DIR2"), time.Now(), dir1)

	dir2.Add(builder.NewFile(spec.FileIdentifier("FILE3"), time.Now(), fooDataLength, fooData))

	dir1.Add(dir2)
	dir1.Add(builder.NewFile(spec.FileIdentifier("FILE2"), time.Now(), fooDataLength, fooData))

	dir0.Add(dir1)
	dir0.Add(builder.NewFile(spec.FileIdentifier("FILE1"), time.Now(), fooDataLength, fooData))

	t.Run("breadth-first search", func(t *testing.T) {
		entries := slices.Collect(dir0.Walk(false))
		assert.Len(t, entries, 6, "Walk should yield as many items as there are entries in the directory tree")

		names := make([]string, len(entries))
		for i, entry := range entries {
			names[i] = string(entry.PointerRecord().FileIdentifier)
		}

		assert.Equal(t, []string{
			string(spec.FileIdentifierSelf),
			"DIR1",
			"FILE1",
			"DIR2",
			"FILE2",
			"FILE3",
		}, names, "Walk(false) should yield items in breadth-first order")
	})

	t.Run("depth-first search", func(t *testing.T) {
		entries := slices.Collect(dir0.Walk(true))
		assert.Len(t, entries, 6, "Walk should yield as many items as there are entries in the directory tree")

		names := make([]string, len(entries))
		for i, entry := range entries {
			names[i] = string(entry.PointerRecord().FileIdentifier)
		}

		assert.Equal(t, []string{
			string(spec.FileIdentifierSelf),
			"DIR1",
			"DIR2",
			"FILE3",
			"FILE2",
			"FILE1",
		}, names, "Walk(true) should yield items in depth-first order")
	})
}

func TestDirectory_Relocate(t *testing.T) {
	root := builder.NewEmptyDirectory(spec.FileIdentifierSelf, time.Now(), nil)
	child := builder.NewEmptyDirectory(spec.FileIdentifier("CHILD"), time.Now(), root)
	root.Add(child)

	assert.Equal(t, uint32(0), root.Location(), "Initial directory location should be zero for a root directory")
	assert.Equal(t, root.Location(), root.PointerRecord().ExtentLocation.RealValue(), "PointerRecord of root directory should match Location()")
	assert.Equal(t, root.Location(), root.SelfRecord().ExtentLocation.RealValue(), "SelfRecord of root directory should match Location()")
	assert.Equal(t, uint32(0), child.Location(), "Initial directory location should be zero for a child directory")
	assert.Equal(t, child.Location(), child.PointerRecord().ExtentLocation.RealValue(), "PointerRecord of child directory should match Location()")
	assert.Equal(t, child.Location(), child.SelfRecord().ExtentLocation.RealValue(), "SelfRecord of child directory should match Location()")

	root.Relocate(uint32(0x12300))
	assert.Equal(t, uint32(0x12300), root.Location(), "Root directory Location() should be correct after invoking Relocate()")
	assert.Equal(t, uint32(0x12300), root.PointerRecord().ExtentLocation.RealValue(), "Root directory PointerRecord should have updated ExtentLocation after invoking Relocate()")
	assert.Equal(t, uint32(0x12300), root.SelfRecord().ExtentLocation.RealValue(), "Root directory SelfRecord should have updated ExtentLocation after invoking Relocate()")
	assert.Equal(t, uint32(0x12300), root.ParentRecord().ExtentLocation.RealValue(), "Root directory ParentRecord should have updated ExtentLocation after invoking Relocate()")
	assert.Equal(t, uint32(0x12300), child.ParentRecord().ExtentLocation.RealValue(), "Child directory ParentRecord should have ExtentLocation matching its parent's location after relocating the parent")

	assert.Equal(t, uint32(0), child.Location(), "Child directory Location() should not change when parent is relocated")
	assert.Equal(t, uint32(0), child.PointerRecord().ExtentLocation.RealValue(), "Child directory PointerRecord() ExtentLocation should not change when parent is relocated")
	assert.Equal(t, uint32(0), child.SelfRecord().ExtentLocation.RealValue(), "Child directory SelfRecord() ExtentLocation should not change when parent is relocated")

	child.Relocate(uint32(0x45600))
	assert.Equal(t, uint32(0x45600), child.Location(), "Child directory Location() should be correct after invoking Relocate()")
	assert.Equal(t, uint32(0x45600), child.PointerRecord().ExtentLocation.RealValue(), "Child directory PointerRecord should have updated ExtentLocation after invoking Relocate()")
	assert.Equal(t, uint32(0x45600), child.SelfRecord().ExtentLocation.RealValue(), "Child directory SelfRecord should have updated ExtentLocation after invoking Relocate()")
	assert.Equal(t, uint32(0x12300), child.ParentRecord().ExtentLocation.RealValue(), "Child directory ParentRecord should remain unchanged after invoking Relocate() on the child")

	assert.Equal(t, uint32(0x12300), root.Location(), "Root directory Location() should not change when child is relocated")
	assert.Equal(t, uint32(0x12300), root.PointerRecord().ExtentLocation.RealValue(), "Root directory PointerRecord() ExtentLocation should not change when child is relocated")
	assert.Equal(t, uint32(0x12300), root.SelfRecord().ExtentLocation.RealValue(), "Root directory SelfRecord() ExtentLocation should not change when child is relocated")
	assert.Equal(t, uint32(0x12300), root.ParentRecord().ExtentLocation.RealValue(), "Root directory ParentRecord() ExtentLocation should not change when child is relocated")
}

func assertDirectoryRecord(
	t *testing.T,
	r spec.DirectoryRecord,
	prefix string,
	expectedIdentifier spec.FileIdentifier,
	expectedIdentifierLength uint8,
	expectedRecordLength uint8,
	expectedDataLength uint32,
	expectedRecordingDateTime time.Time,
	isDirectory bool,
) {
	// General checks that the file identifiers and reported lengths are what we expect
	assert.Equal(t, expectedIdentifier, r.FileIdentifier, "%s directory record should have correct file identifier", prefix)
	assert.Equal(t, expectedIdentifierLength, r.LengthOfFileIdentifier, "%s directory record should report correct file identifier length", prefix)
	assert.Equal(t, expectedRecordLength, r.Length, "%s directory record should report correct record length", prefix)
	assert.Equal(t, expectedDataLength, r.DataLength.RealValue(), "%s directory record should report correct data length", prefix)
	assert.Equal(t, expectedRecordingDateTime.Unix(), r.RecordingDateAndTime.Time().Unix(), "%s directory record should give correct recording datetime", prefix)

	// The spec imposes an even length requirement. ISO readers impose a requirement that the data length should be a
	// multiple of the logical block size, which we assume here to be 2048.
	assert.True(t, r.Length%2 == 0, "%s directory record length must be an even number of bytes", prefix)
	if isDirectory {
		assert.True(t, r.DataLength.RealValue()%2048 == 0, "%s directory record data length should be a multiple of the logical block size", prefix)
	}

	if isDirectory {
		assert.True(t, r.FileFlags&spec.FileFlagDirectory > 0, "%s directory record should have flags indicating it is a directory", prefix)
	} else {
		assert.True(t, r.FileFlags&spec.FileFlagDirectory == 0, "%s directory record should have flags indicating it is a file", prefix)
	}

	assert.Equal(t, uint32(0), r.ExtentLocation.RealValue(), "%s directory record should initially report location of 0 before locating the directory", prefix)

	// These checks ensure that we have sensible values for features that are explicitly unsupported at present
	assert.Equal(t, uint8(0), r.ExtendedAttributeRecordLength, "%s directory record should not have any extended attribute record", prefix)
	assert.Equal(t, uint16(1), r.VolumeSequenceNumber.RealValue(), "%s directory record should give record with volume sequence number of 1", prefix)
	assert.Equal(t, uint8(0), r.FileUnitSize, "%s directory record should not give interleaved record (File unit size)", prefix)
	assert.Equal(t, uint8(0), r.InterleaveGapSize, "%s directory record should not give interleaved record (interleave gap)", prefix)
}

func TestNewEmptyDirectory_WhenDirectoryHasParent(t *testing.T) {
	parentRecordedAt := time.Now()
	parent := builder.NewEmptyDirectory(spec.FileIdentifier("foobarparent"), parentRecordedAt, nil)
	require.NotNil(t, parent, "NewEmptyDirectory should not return a nil pointer")

	identifier := spec.FileIdentifier("bar")
	recordedAt := time.Date(2012, 12, 2, 6, 24, 59, 0, time.FixedZone("UTC-8", -8*60*60))
	d := builder.NewEmptyDirectory(identifier, recordedAt, parent)
	require.NotNil(t, d, "NewEmptyDirectory should not return a nil pointer when given a non-nil parent")

	require.NotNil(t, d.PointerRecord, "PointerRecord of directory should be non-nil")
	require.NotNil(t, d.SelfRecord, "SelfRecord of directory should be non-nil")
	require.NotNil(t, d.ParentRecord, "ParentRecord of directory should be non-nil")

	assertDirectoryRecord(t, d.PointerRecord(), "Pointer", identifier, 3, 36, 2048, recordedAt, true)
	assertDirectoryRecord(t, d.SelfRecord(), "Self", spec.FileIdentifierSelf, 1, 34, 2048, recordedAt, true)
	assertDirectoryRecord(t, d.ParentRecord(), "Parent", spec.FileIdentifierParent, 1, 34, 2048, parentRecordedAt, true)

	assert.Equal(t, parent, d.Parent(), "Parent() should return correct value when Directory is constructed using NewEmptyDirectory with non-nil parent argument")

	parent.Add(d)

	parent.Relocate(0x12200)
	d.Relocate(0x45600)

	assert.Equal(t, uint32(0x45600), d.PointerRecord().ExtentLocation.RealValue(), "Relocating a parented directory should update its pointer record")
	assert.Equal(t, uint32(0x45600), d.SelfRecord().ExtentLocation.RealValue(), "Relocating a parented directory should update its self record")
	assert.Equal(t, uint32(0x12200), d.ParentRecord().ExtentLocation.RealValue(), "Relocating a parented directory's parent should update the directory's parent record extent location")
	assert.Equal(t, uint32(0x12200), parent.PointerRecord().ExtentLocation.RealValue(), "Relocating a parented directory's parent should update the parent's pointer record")
	assert.Equal(t, uint32(0x12200), parent.SelfRecord().ExtentLocation.RealValue(), "Relocating a parented directory's parent should update the parent's self record")
	assert.Equal(t, parent.PointerRecord().ExtentLocation.RealValue(), d.ParentRecord().ExtentLocation.RealValue(), "A parented directory's ParentRecord should have the same ExtentLocation as the parent's PointerRecord after relocation")
	assert.Equal(t, parent.SelfRecord().ExtentLocation.RealValue(), d.ParentRecord().ExtentLocation.RealValue(), "A parented directory's ParentRecord should have the same ExtentLocation as the parent's SelfRecord after relocation")
	assert.Equal(t, parent.PointerRecord().DataLength.RealValue(), d.ParentRecord().DataLength.RealValue(), "A parented directory's ParentRecord should have the same DataLength as the parent's PointerRecord after relocation")
	assert.Equal(t, parent.SelfRecord().DataLength.RealValue(), d.ParentRecord().DataLength.RealValue(), "A parented directory's ParentRecord should have the same DataLength as the parent's SelfRecord after relocation")

	parentEntries := parent.Entries()
	if assert.Equal(t, 1, len(parentEntries)) {
		assert.Equal(t, d.PointerRecord(), parentEntries[0].PointerRecord(), "After adding a parented directory to its parent, the parent directory should have an entry with a record equal to the child's PointerRecord")
		assert.Equal(t, d, parentEntries[0], "After adding a parented directory to its parent, the child should appear in the parent's Entries as-is")
	}
}

func TestNewEmptyDirectory_WhenDirectoryHasNoParent(t *testing.T) {
	recordedAt := time.Date(2000, 1, 2, 3, 4, 5, 0, time.FixedZone("UTC-8", -8*60*60))

	identifier := spec.FileIdentifier("test")

	d := builder.NewEmptyDirectory(identifier, recordedAt, nil)
	require.NotNil(t, d, "NewEmptyDirectory should not return a nil pointer")

	require.NotNil(t, d.PointerRecord(), "PointerRecord of directory should be non-nil")
	require.NotNil(t, d.SelfRecord(), "SelfRecord of directory should be non-nil")
	require.NotNil(t, d.ParentRecord(), "ParentRecord of directory should be non-nil")

	assertDirectoryRecord(t, d.PointerRecord(), "Pointer", identifier, 4, 38, 2048, recordedAt, true)
	assertDirectoryRecord(t, d.SelfRecord(), "Self", spec.FileIdentifierSelf, 1, 34, 2048, recordedAt, true)
	assertDirectoryRecord(t, d.ParentRecord(), "Parent", spec.FileIdentifierParent, 1, 34, 2048, recordedAt, true)

	selfWithoutDifferingFields := d.SelfRecord()
	selfWithoutDifferingFields.LengthOfFileIdentifier = d.PointerRecord().LengthOfFileIdentifier
	selfWithoutDifferingFields.FileIdentifier = d.PointerRecord().FileIdentifier
	selfWithoutDifferingFields.Length = d.PointerRecord().Length
	assert.EqualValues(t, d.PointerRecord(), selfWithoutDifferingFields, "SelfRecord should be the same as PointerRecord, except for LengthOfFileIdentifier, FileIdentifier, and DataLength")

	parentWithoutDifferingFields := d.ParentRecord()
	parentWithoutDifferingFields.LengthOfFileIdentifier = d.PointerRecord().LengthOfFileIdentifier
	parentWithoutDifferingFields.FileIdentifier = d.PointerRecord().FileIdentifier
	parentWithoutDifferingFields.Length = d.PointerRecord().Length
	assert.EqualValues(t, d.PointerRecord(), parentWithoutDifferingFields, "ParentRecord should be the same as PointerRecord when creating the root directory, except for LengthOfFileIdentifier, FileIdentifier, and DataLength")
}

func TestNewFile(t *testing.T) {
	identifier := spec.FileIdentifier("FOO.DAT;1")
	now := time.Now()
	f := builder.NewFile(identifier, now, 123, func() (io.Reader, error) {
		return bytes.NewReader(make([]byte, 123)), nil
	})

	assertDirectoryRecord(t, f.PointerRecord(), "File", identifier, 9, 42, 123, now, false)
}

func TestFile_WriteTo(t *testing.T) {
	data := make([]byte, 123)
	for i := range data {
		data[i] = byte(i % 256)
	}

	f := builder.NewFile(spec.FileIdentifier("FOO.DAT;1"), time.Now(), 123, func() (io.Reader, error) {
		return bytes.NewReader(data), nil
	})

	var actualData bytes.Buffer
	bytesWritten, err := f.WriteTo(&actualData)
	require.NoError(t, err, "WriteTo() should not produce an error")
	assert.EqualValues(t, len(data), actualData.Len(), "WriteTo() should write the same number of bytes as given in NewFile()")
	assert.EqualValues(t, actualData.Len(), bytesWritten, "WriteTo() should report an accurate number of bytes written")
	assert.Equal(t, data, actualData.Bytes(), "WriteTo() should produce the same data as the file was given in NewFile()")
}

func TestFile_Relocate(t *testing.T) {
	f := builder.NewFile(spec.FileIdentifier("FOO;1"), time.Now(), 123, func() (io.Reader, error) {
		return bytes.NewReader(make([]byte, 123)), nil
	})

	assert.Equal(t, uint32(0), f.Location(), "Initial file location should be zero")
	assert.Equal(t, uint32(0), f.PointerRecord().ExtentLocation.RealValue(), "Initial file location should be zero in PointerRecord()")

	f.Relocate(uint32(0x9900))
	assert.Equal(t, uint32(0x9900), f.Location(), "File should report correct Location() after relocating")
	assert.Equal(t, uint32(0x9900), f.PointerRecord().ExtentLocation.RealValue(), "File ExtentLocation in PointerRecord() should be correct after relocating")

	f.Relocate(uint32(0x1000))
	assert.Equal(t, uint32(0x1000), f.Location(), "Successive relocations should still update Location()")
	assert.Equal(t, uint32(0x1000), f.PointerRecord().ExtentLocation.RealValue(), "Successive relocations should still update PointerRecord()'s ExtentLocation")
}
