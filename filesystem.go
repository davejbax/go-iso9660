package iso9660

import (
	"fmt"
	"io"
	"io/fs"
	"path"
	"slices"
	"strings"
	"time"
)

func newDirectory(filesystem fs.ReadDirFS, filesystemPath string, parent *directory, recordedAt time.Time) (*directory, error) {
	var identifier fileIdentifier

	if parent == nil {
		// Use the root identifier if we're looking at the root directory
		identifier = fileIdentifierSelf
	} else {
		var err error
		identifier, err = newFileIdentifier(path.Base(filesystemPath), "", 0, fileIdentifierEncodingDCharacter)
		if err != nil {
			return nil, fmt.Errorf("directory has invalid name: %w", err)
		}
	}

	dir := &directory{
		name:       identifier,
		parent:     parent,
		location:   0,
		recordedAt: recordedAt,
		flags:      0,
		entries:    nil,
	}

	// No parent; therefore, we're the root directory
	if parent == nil {
		dir.parent = dir
	}

	entries, err := filesystem.ReadDir(filesystemPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read filesystem directory: %w", err)
	}

	// Records in a directory must be sorted in a particular order (ECMA-119 5th edition, §10.3)
	slices.SortFunc(entries, compareDirEntries)

	for _, entry := range entries {
		var entryFileLike fileLike
		entryPath := path.Join(filesystemPath, entry.Name())

		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("directory '%s' has invalid info: %w", entry.Name(), err)
		}

		if entry.IsDir() {
			entryDir, err := newDirectory(filesystem, entryPath, dir, info.ModTime())
			if err != nil {
				return nil, fmt.Errorf("failed to create subdirectory '%s': %w", entryPath, err)
			}

			entryFileLike = entryDir
		} else {
			// TODO: handle case where file is > 4GB here
			entryFile, err := newFile(filesystem, entryPath, info.ModTime(), uint32(info.Size()))
			if err != nil {
				return nil, fmt.Errorf("failed to create file '%s': %w", entryPath, err)
			}

			entryFileLike = entryFile
		}

		dir.entries = append(dir.entries, entryFileLike)
	}

	return dir, nil
}

func newFile(filesystem fs.FS, filesystemPath string, recordedAt time.Time, size uint32) (*file, error) {
	filenameAndExtension := path.Base(filesystemPath)
	filename := filenameAndExtension
	extension := ""

	if index := strings.LastIndex(filenameAndExtension, "."); index != -1 {
		filename = filenameAndExtension[:index]
		extension = filenameAndExtension[index+1:]
	}

	identifier, err := newFileIdentifier(filename, extension, 1, fileIdentifierEncodingDCharacter)
	if err != nil {
		return nil, fmt.Errorf("could not create file identifier: %w", err)
	}

	return &file{
		name:       identifier,
		location:   0,
		recordedAt: recordedAt,
		flags:      0,
		dataLength: size,
		data: func() (io.Reader, error) {
			f, err := filesystem.Open(filesystemPath)
			if err != nil {
				return nil, fmt.Errorf("could not read input file '%s': %w", filesystemPath, err)
			}

			return f, nil
		},
	}, nil
}

func compareDirEntries(a, b fs.DirEntry) int {
	// Files comprise 'file names' and 'file name extensions', separated by a period
	aFilename, aExtension, _ := strings.Cut(a.Name(), ".")
	bFilename, bExtension, _ := strings.Cut(a.Name(), ".")

	// First, sort by file name
	if cmp := strings.Compare(aFilename, bFilename); cmp != 0 {
		return cmp
	}

	// If file name is the same, compare extensions
	if cmp := strings.Compare(aExtension, bExtension); cmp != 0 {
		return cmp
	}

	aIsDir := a.IsDir()
	bIsDir := b.IsDir()

	// Sort directories before files, if everything else is equal
	if aIsDir && !bIsDir {
		return 1
	} else if !aIsDir && bIsDir {
		return -1
	}

	return 0
}
