package builder

import (
	"errors"
	"fmt"
	"github.com/itchio/headway/counter"
	"io"
)

// Logical sector size is set pretty much unanimously to 2048.
// Logical block size has to be a power of two that's smaller than (or equal to) the logical sector size.
// We don't have any reason to support logical blocks smaller than the logical sector size, and this simplifies
// implementation quite a lot.
const (
	logicalSectorSize = 2048
	logicalBlockSize  = logicalSectorSize
)

type BlockWriter struct {
	wrapped *counter.Writer

	currentBlock uint32
}

var errNonSequentialBlockWrite = errors.New("cannot write blocks in non-sequential order or rewrite existing blocks")

func NewBlockWriter(wrapped io.Writer) *BlockWriter {
	return &BlockWriter{wrapped: counter.NewWriter(wrapped)}
}

func (w *BlockWriter) WriteBlockFunc(number uint32, writeTo func(io.Writer) (int64, error)) error {
	if number < w.currentBlock {
		return errNonSequentialBlockWrite
	}

	zeroBlock := make([]byte, logicalBlockSize)
	for number > w.currentBlock {
		if _, err := w.wrapped.Write(zeroBlock); err != nil {
			return fmt.Errorf("failed to write padding prior to block: %w", err)
		}

		w.currentBlock += 1
	}

	contentsSize, err := writeTo(w.wrapped)
	if err != nil {
		return fmt.Errorf("failed to write contents to block: %w", err)
	}

	contentsBlocks := (contentsSize + logicalBlockSize - 1) / logicalBlockSize

	if contentsBlocks*logicalBlockSize > contentsSize {
		// Note: this should never be more than a block in size being allocated here
		padding := make([]byte, contentsBlocks*logicalBlockSize-contentsSize)
		if _, err := w.wrapped.Write(padding); err != nil {
			return fmt.Errorf("failed to write padding after contents: %w", err)
		}
	}

	w.currentBlock += uint32(contentsBlocks)

	return nil
}

func (w *BlockWriter) WriteBlock(number uint32, contents io.WriterTo) error {
	return w.WriteBlockFunc(number, contents.WriteTo)
}

func (w *BlockWriter) BytesWritten() int64 {
	return w.wrapped.Count()
}
