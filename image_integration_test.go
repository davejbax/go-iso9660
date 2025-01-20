package iso9660_test

import (
	"github.com/davejbax/go-iso9660"
	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/backend/file"
	"github.com/diskfs/go-diskfs/filesystem"
	diskfsiso "github.com/diskfs/go-diskfs/filesystem/iso9660"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestImage(t *testing.T) {
	contents := os.DirFS("testdata").(fs.ReadDirFS)
	image, err := iso9660.NewImage(contents)

	require.NoError(t, err, "NewImage should not return an error for valid arguments")
	require.NotNil(t, image, "NewImage should return a non-nil image when no error")

	outputFile, err := os.OpenFile(filepath.Join(t.TempDir(), "output.iso"), os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("Failed to open output file: %v", err)
	}
	defer outputFile.Close()

	written, err := image.WriteTo(outputFile)
	require.NoError(t, err, "WriteTo should not return an error for valid arguments")

	stat, err := os.Stat(outputFile.Name())
	if err != nil {
		t.Fatalf("Failed to stat output file: %v", err)
	}

	assert.Equal(t, stat.Size(), written, "Bytes written returned by WriteTo should match actual file size")

	backend, err := file.OpenFromPath(outputFile.Name(), true)
	if err != nil {
		// The file hasn't been read yet, so this error indicates something that isn't our fault
		t.Fatalf("Failed to open output file for reading: %v", err)
	}

	disk, err := diskfs.OpenBackend(backend)
	require.NoError(t, err, "Should be able to open ISO file")

	fs, err := disk.GetFilesystem(0) // Partition 0 is whole disk
	require.NoError(t, err, "Should be able to read filesystem from ISO file")

	iso, ok := fs.(*diskfsiso.FileSystem)
	require.Equal(t, true, ok, "Filesystem in written image should be an ISO9660 filesystem")

	assertFilesystemsEqual(t, contents, iso, "/")
}

func assertFilesystemsEqual(t *testing.T, expected fs.ReadDirFS, actual filesystem.FileSystem, dir string) {
	expectedEntries, err := expected.ReadDir(dir)
	if err != nil {
		t.Fatalf("Failed to read testdata directory '%s': %v", dir, err)
	}

	// Sort the expected entries, because we need to compare file-by-file, and we can't guarantee the order that these
	// files are presented in
	slices.SortFunc(expectedEntries, func(a, b fs.DirEntry) int {
		return strings.Compare(a.Name(), b.Name())
	})

	actualEntries, err := actual.ReadDir(dir)
	require.NoError(t, err, "Should be able to read root directory in ISO filesystem")

	// Sort the actual entries, because the order of files returned by the dirfs library is not something under test,
	// and the order needs to match the expected entries so we can compare file-by-file
	slices.SortFunc(actualEntries, func(a, b os.FileInfo) int {
		return strings.Compare(a.Name(), b.Name())
	})

	require.Equal(t, len(expectedEntries), len(actualEntries), "Number of entries in root directory of ISO file should match source data")

	for i, expectedEntry := range expectedEntries {
		entryPath := path.Join(dir, expectedEntry.Name())
		t.Run(entryPath, func(tt *testing.T) {
			tt.Parallel()

			expectedInfo, err := expectedEntry.Info()
			if err != nil {
				tt.Fatalf("failed to get info for testdata file: %v", err)
			}

			actualEntry := actualEntries[i]

			assert.Equal(tt, expectedInfo.Name(), actualEntry.Name(), "Base name of testdata file should match source data") // TODO: do we actually want this?
			assert.Equal(tt, expectedInfo.Size(), actualEntry.Size(), "Size of testdata file should match source data")
			assert.Equal(tt, expectedInfo.Mode(), actualEntry.Mode(), "Mode of testdata file should match source data")
			assert.Equal(tt, expectedInfo.IsDir(), actualEntry.IsDir(), "IsDir of testdata file should match source data")
			assert.Equal(tt, expectedInfo.ModTime(), actualEntry.ModTime(), "Modification time of testdata file should match source data")

			if actualEntry.IsDir() {
				t.Run(expectedEntry.Name()+": contents", func(tt *testing.T) {
					tt.Parallel()
					assertFilesystemsEqual(t, expected, actual, entryPath)
				})
			} else {
				assertFilesEqual(tt, expected, actual, entryPath)
			}
		})
	}
}

func assertFilesEqual(t *testing.T, source fs.ReadDirFS, actualFS filesystem.FileSystem, path string) {
	expectedFile, err := source.Open(path)
	if err != nil {
		t.Fatalf("Failed to open source file '%s': %v", path, err)
	}

	actualFile, err := actualFS.OpenFile(path, os.O_RDONLY)
	require.NoError(t, err, "Should be able to open testdata file in ISO image")

	actual, err := io.ReadAll(expectedFile)
	if err != nil {
		t.Fatalf("Failed to read testdata file '%s': %v", path, err)
	}

	expected, err := io.ReadAll(actualFile)
	require.NoError(t, err, "Should be able to read file from ISO")
	assert.Equal(t, expected, actual, "Contents of testdata file should match source data")
}
