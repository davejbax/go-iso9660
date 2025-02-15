package iso9660

import (
	"errors"
	"fmt"
	"github.com/itchio/headway/counter"
	"io"
)

type blockWriter struct {
	wrapped *counter.Writer

	currentBlock uint32
}

var errNonSequentialBlockWrite = errors.New("cannot write blocks in non-sequential order or rewrite existing blocks")

func newBlockWriter(wrapped io.Writer) *blockWriter {
	return &blockWriter{wrapped: counter.NewWriter(wrapped)}
}

func (w *blockWriter) WriteBlock(number uint32, contents io.WriterTo) error {
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

	contentsSize, err := contents.WriteTo(w.wrapped)
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

func (w *blockWriter) BytesWritten() int64 {
	return w.wrapped.Count()
}
