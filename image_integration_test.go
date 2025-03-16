package iso9660_test

import (
	"github.com/davejbax/go-iso9660"
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
	"time"
)

// assertISOWritten uses iso9660 to create an image from a fs.ReadDirFS, and write it to a file. It asserts several
// things along the way, such as ensuring that the file is written successfully and without errors.
func assertISOWritten(t *testing.T, contents fs.ReadDirFS, outputPath string) {
	image, err := iso9660.NewImage(contents)

	require.NoError(t, err, "NewImage should not return an error for valid arguments")
	require.NotNil(t, image, "NewImage should return a non-nil image when no error")

	outputFile, err := os.OpenFile(outputPath, os.O_CREATE|os.O_RDWR, 0o600)
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
}

func TestISOIsExtractableWithXorriso(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	sourceFS := os.DirFS("testdata/imageroot").(fs.ReadDirFS)
	isoFilePath := filepath.Join(t.TempDir(), "output.iso")
	assertISOWritten(t, sourceFS, isoFilePath)

	isoExtractionPath := filepath.Join(t.TempDir(), "extracted")
	if err := extractISO(
		t,
		"extractor.Dockerfile",
		[]string{
			"/usr/bin/bash",
			"-c",
			"mkdir /output && osirrox -indev /input/image.iso -extract / /output && find /output -print && tar -C /output -cvf /output/image.tar .",
		},
		isoFilePath,
		isoExtractionPath,
	); err != nil {
		t.Fatalf("failed to extract ISO: %v", err)
	}

	assertFilesystemsEqual(t, sourceFS, os.DirFS(isoExtractionPath).(fs.ReadDirFS), ".", false)
}

func TestISOIsExtractableWith7z(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	sourceFS := os.DirFS("testdata/imageroot").(fs.ReadDirFS)
	isoFilePath := filepath.Join(t.TempDir(), "output.iso")
	assertISOWritten(t, sourceFS, isoFilePath)

	isoExtractionPath := filepath.Join(t.TempDir(), "extracted")
	if err := extractISO(
		t,
		"extractor.Dockerfile",
		[]string{
			"/usr/bin/bash",
			"-c",
			"mkdir /output && 7z x /input/image.iso -o/output && find /output -print && tar -C /output -cvf /output/image.tar .",
		},
		isoFilePath,
		isoExtractionPath,
	); err != nil {
		t.Fatalf("failed to extract ISO: %v", err)
	}

	assertFilesystemsEqual(t, sourceFS, os.DirFS(isoExtractionPath).(fs.ReadDirFS), ".", false)
}

func assertFilesystemsEqual(t *testing.T, expected fs.ReadDirFS, actual fs.ReadDirFS, dir string, checkMode bool) {
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
	slices.SortFunc(actualEntries, func(a, b fs.DirEntry) int {
		return strings.Compare(a.Name(), b.Name())
	})

	require.Equal(t, len(expectedEntries), len(actualEntries), "Number of entries in root directory of ISO file should match source data")

	for i, expectedEntry := range expectedEntries {
		entryPath := path.Join(dir, expectedEntry.Name())
		t.Run(entryPath, func(tt *testing.T) {
			//tt.Parallel()

			expectedInfo, err := expectedEntry.Info()
			if err != nil {
				tt.Fatalf("failed to get info for testdata file: %v", err)
			}

			actualEntry := actualEntries[i]

			actualInfo, err := actualEntry.Info()
			if err != nil {
				tt.Fatalf("failed to get info for extracted ISO file: %v", err)
			}

			assert.Equal(tt, expectedInfo.Name(), actualInfo.Name(), "Base name of testdata file should match source data") // TODO: do we actually want this?
			assert.Equal(tt, expectedInfo.Size(), actualInfo.Size(), "Size of testdata file should match source data")
			assert.Equal(tt, expectedInfo.IsDir(), actualInfo.IsDir(), "IsDir of testdata file should match source data")
			assert.Equal(tt, expectedInfo.ModTime().Truncate(time.Second), actualInfo.ModTime().Truncate(time.Second), "Modification time of testdata file should match source data (to within a second)")

			if checkMode {
				assert.Equal(tt, expectedInfo.Mode(), actualInfo.Mode(), "Mode of testdata file should match source data")
			}

			if actualEntry.IsDir() {
				tt.Run(expectedEntry.Name()+":Contents", func(ttt *testing.T) {
					//ttt.Parallel()
					assertFilesystemsEqual(t, expected, actual, entryPath, checkMode)
				})
			} else {
				assertFilesEqual(tt, expected, actual, entryPath)
			}
		})
	}
}

func assertFilesEqual(t *testing.T, expectedFS fs.ReadDirFS, actualFS fs.FS, path string) {
	expectedFile, err := expectedFS.Open(path)
	if err != nil {
		t.Fatalf("Failed to open source file '%s': %v", path, err)
	}

	actualFile, err := actualFS.Open(path)
	require.NoError(t, err, "Should be able to open testdata file in ISO image")

	actual, err := io.ReadAll(expectedFile)
	if err != nil {
		t.Fatalf("Failed to read testdata file '%s': %v", path, err)
	}

	expected, err := io.ReadAll(actualFile)
	require.NoError(t, err, "Should be able to read file from ISO")
	assert.Equal(t, expected, actual, "Contents of testdata file should match source data")
}
