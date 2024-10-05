package iou

import (
	"compress/gzip"
	"io"
	"sync/atomic"
)

type CountingReader struct {
	reader    io.Reader
	bytesRead int64
}

func NewCountingReader(r io.Reader) *CountingReader {
	return &CountingReader{reader: r}
}

func (cr *CountingReader) Read(p []byte) (n int, err error) {
	n, err = cr.reader.Read(p)
	atomic.AddInt64(&cr.bytesRead, int64(n))
	return
}

func (cr *CountingReader) BytesRead() int64 {
	return atomic.LoadInt64(&cr.bytesRead)
}

// GzipCompressReader wraps an io.Reader so that the bytes are compressed as they are read
func GzipCompressReader(r io.Reader) io.Reader {
	pipeReader, pipeWriter := io.Pipe()
	gzipWriter := gzip.NewWriter(pipeWriter)
	go func() {
		_, err := io.Copy(gzipWriter, r)
		gzipWriter.Close()
		if err != nil {
			_ = pipeWriter.CloseWithError(err)
		} else {
			_ = pipeWriter.Close()
		}
	}()
	return pipeReader
}
