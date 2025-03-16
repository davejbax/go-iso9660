package iso9660_test

import (
	"archive/tar"
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func dumpContainerLogs(t *testing.T, container testcontainers.Container) {
	logReader, err := container.Logs(context.Background())
	if err != nil {
		t.Logf("failed to get container logs: %v", err)
	} else {
		logs, err := io.ReadAll(logReader)
		if err != nil {
			t.Logf("tried to get container logs, but could not read: %v", err)
		} else {
			_ = logReader.Close()
			t.Logf("container logs:\n%s", string(logs))
		}
	}
}

// extractISO runs a Docker container with isoPath mounted at /input/image.iso. The command in 'cmd' is executed, which
// is assumed to produce an output file /output/image.tar. This tar file is then copied to the host, and extracted into
// outputPath.
//
// The dockerfile is a path relative to the testdata directory which will be built with the testdata directory as its
// context.
func extractISO(t *testing.T, dockerfile string, cmd []string, isoPath string, outputPath string) error {
	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    filepath.Join(".", "testdata"),
			Dockerfile: dockerfile,
		},
		Cmd: cmd,
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      isoPath,
				ContainerFilePath: "/input/image.iso",
				FileMode:          0o644,
			},
		},
		WaitingFor: wait.ForExit(),
	}

	container, err := testcontainers.GenericContainer(context.Background(), testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	defer testcontainers.CleanupContainer(t, container)
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Dump the logs for ease of debugging
	dumpContainerLogs(t, container)

	state, err := container.State(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get state of container: %w", err)
	}

	assert.Equal(t, "exited", state.Status, "container should be exited")
	require.Equal(t, 0, state.ExitCode, "container should be able to extract ISO file without any errors")

	image, err := container.CopyFileFromContainer(context.Background(), "/output/image.tar")
	if err != nil {
		return fmt.Errorf("failed to copy file from container: %w", err)
	}
	defer image.Close()

	if err := extractTar(image, outputPath); err != nil {
		return fmt.Errorf("failed to extract tar obtained from container: %w", err)
	}

	return nil
}

func extractTar(data io.Reader, outputPath string) error {
	dirHeaders := make(map[string]*tar.Header)

	image := tar.NewReader(data)
	for {
		hdr, err := image.Next()
		if err == io.EOF {
			break // End of archive
		} else if err != nil {
			return fmt.Errorf("failed to extract image: %w", err)
		}

		info := hdr.FileInfo()

		outputFileName := filepath.Join(outputPath, hdr.Name)

		switch hdr.Typeflag {
		case tar.TypeDir:
			// Note: we need to add the writable bit here, as otherwise we won't be able to create files inside this
			// directory!
			if err := os.Mkdir(outputFileName, info.Mode()|0o200); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}

			dirHeaders[outputFileName] = hdr

		case tar.TypeReg:
			f, err := os.OpenFile(outputFileName, os.O_CREATE|os.O_WRONLY, info.Mode())
			if err != nil {
				return fmt.Errorf("failed to create output file: %w", err)
			}

			if _, err := io.Copy(f, image); err != nil {
				f.Close()
				return fmt.Errorf("failed to extract file from container tar archive to output: %w", err)
			}

			f.Close()

			if err := os.Chtimes(outputFileName, hdr.AccessTime, hdr.ModTime); err != nil {
				return fmt.Errorf("failed to change mtime: %w", err)
			}

		default:
			// yes, naughty to Errorf here, but we don't really need a sentinel error
			return fmt.Errorf("unexpected file type %v", hdr.Typeflag)
		}
	}

	// We need to set the dir mtimes **after** we've written all the files, because adding a file to a directory
	// will modify the directory's mtime!
	for dir, hdr := range dirHeaders {
		if err := os.Chtimes(dir, hdr.AccessTime, hdr.ModTime); err != nil {
			return fmt.Errorf("failed to change mtime: %w", err)
		}
	}

	return nil
}
