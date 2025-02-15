package iso9660

import "io"

type blockWriter struct {
	wrapped io.Writer

	currentBlock uint32
}

func newBlockWriter(wrapped io.Writer) *blockWriter {
	return &blockWriter{wrapped: wrapped}
}

func (*blockWriter) WriteBlock(number uint32, contents io.WriterTo) error {
	return nil
}

func (*blockWriter) BytesWritten() int64 {
	return 0
}
