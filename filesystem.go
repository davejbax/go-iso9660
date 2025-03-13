package iso9660

import (
	"fmt"
	"github.com/davejbax/go-iso9660/internal/builder"
	"github.com/davejbax/go-iso9660/internal/encode"
	"github.com/davejbax/go-iso9660/internal/spec"
	"io"
	"io/fs"
	"path"
	"slices"
	"strings"
	"time"
)

func newDirectoryFromFS(filesystem fs.ReadDirFS, filesystemPath string, parent *builder.Directory, recordedAt time.Time) (*builder.Directory, error) {
	var identifier spec.FileIdentifier

	if parent == nil {
		// Use the root identifier if we're looking at the root Directory
		identifier = spec.FileIdentifierSelf
	} else {
		var err error
		identifier, err = encode.AsFileIdentifier(path.Base(filesystemPath), "", 1, encode.FileIdentifierEncodingDCharacter)
		if err != nil {
			return nil, fmt.Errorf("Directory has invalid name: %w", err)
		}
	}

	dir := builder.NewEmptyDirectory(identifier, recordedAt, parent)

	entries, err := filesystem.ReadDir(filesystemPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read filesystem Directory: %w", err)
	}

	// Records in a Directory must be sorted in a particular order (ECMA-119 5th edition, §10.3)
	slices.SortFunc(entries, func(a, b fs.DirEntry) int {
		return spec.CompareDirectoryEntries(&directoryEntryAdapter{a}, &directoryEntryAdapter{b})
	})

	for _, entry := range entries {
		var entryFileLike builder.RelocatableFileSection
		entryPath := path.Join(filesystemPath, entry.Name())

		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("directory '%s' has invalid info: %w", entry.Name(), err)
		}

		if entry.IsDir() {
			entryDir, err := newDirectoryFromFS(filesystem, entryPath, dir, info.ModTime())
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

		fmt.Printf("%s: adding %s, which has record identifier %s\n", filesystemPath, entryPath, string(entryFileLike.PointerRecord().FileIdentifier))
		dir.Add(entryFileLike)
	}

	return dir, nil
}

func newFile(filesystem fs.FS, filesystemPath string, recordedAt time.Time, size uint32) (*builder.File, error) {
	filenameAndExtension := path.Base(filesystemPath)
	filename := filenameAndExtension
	extension := ""

	if index := strings.LastIndex(filenameAndExtension, "."); index != -1 {
		filename = filenameAndExtension[:index]
		extension = filenameAndExtension[index+1:]
	}

	identifier, err := encode.AsFileIdentifier(filename, extension, 1, encode.FileIdentifierEncodingDCharacter)
	if err != nil {
		return nil, fmt.Errorf("could not create file identifier: %w", err)
	}

	return builder.NewFile(identifier, recordedAt, size, func() (io.Reader, error) {
		f, err := filesystem.Open(filesystemPath)
		if err != nil {
			return nil, fmt.Errorf("could not read input file '%s': %w", filesystemPath, err)
		}

		return f, nil
	}), nil
}

type directoryEntryAdapter struct {
	fs.DirEntry
}

var _ spec.DirectoryEntry = &directoryEntryAdapter{}

func (d directoryEntryAdapter) Name() string {
	index := strings.LastIndex(d.DirEntry.Name(), ".")
	if index == -1 {
		return d.DirEntry.Name()
	}

	return d.DirEntry.Name()[:index]
}

func (d directoryEntryAdapter) Extension() string {
	index := strings.LastIndex(d.DirEntry.Name(), ".")
	if index == -1 {
		return ""
	}

	return d.DirEntry.Name()[index+1:]
}

func (d directoryEntryAdapter) Version() string {
	// We don't support file versioning, so return an arbitrary value here
	return "0"
}

func (d directoryEntryAdapter) FileSectionIndex() int {
	// We don't support file sections, so return an arbitrary section index here
	return 0
}
