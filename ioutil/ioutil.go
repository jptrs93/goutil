package ioutil

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net/http"
)

// Streamer reads items from a stream typically an underlying io.Reader
//
// Read returns the next object from the stream and io.EOF when there are no more items
// Note io.EOF should be returned on the Read call after the last item not with the last item!!
type Streamer[T any] interface {
	Read() (T, error)
}

type lengthPrefixedByteChunkStreamer struct {
	reader io.Reader
	err    error
}

// NewLengthPrefixedByteChunkStreamer the reader will read chunks of bytes
// from an io.Reader assuming that each chunk is prefixed by its size
// as little endian 4 byte integer
func NewLengthPrefixedByteChunkStreamer(in io.Reader) Streamer[[]byte] {
	return &lengthPrefixedByteChunkStreamer{reader: in}
}

func (c *lengthPrefixedByteChunkStreamer) Read() ([]byte, error) {
	if c.err != nil {
		return nil, c.err
	}
	buf := make([]byte, 4)
	_, err := io.ReadAtLeast(c.reader, buf, 4)
	// io.ReadAtLeast only returns EOF when 0 bytes are read
	if err != nil {
		c.err = err
		return nil, err
	}
	// convert buff to little endian integer
	toRead := int(binary.LittleEndian.Uint32(buf))
	buf = make([]byte, toRead)
	_, err = io.ReadAtLeast(c.reader, buf, toRead)
	if err != nil {
		c.err = err
		return nil, err
	}
	return buf, nil
}

// WithTransform applies a transformation to the Streamer to return a Streamer of a different type
func WithTransform[T, E any](reader Streamer[T], transform func(T) (E, error)) Streamer[E] {
	return &transformStreamer[T, E]{
		w:         reader,
		transform: transform,
	}
}

type transformStreamer[T, E any] struct {
	w         Streamer[T]
	err       error
	transform func(T) (E, error)
}

func (c *transformStreamer[T, E]) Read() (E, error) {
	var transformed E
	if c.err != nil {
		return transformed, c.err
	}
	var original T
	original, c.err = c.w.Read()
	if c.err != nil {
		return transformed, c.err
	}
	return c.transform(original)
}

type ByteSliceChannelReader struct {
	Ch     chan []byte
	buffer []byte
}

func NewByteSliceChannelReader(ch chan []byte) *ByteSliceChannelReader {
	return &ByteSliceChannelReader{Ch: ch}
}

func (r *ByteSliceChannelReader) Read(p []byte) (int, error) {
	for len(r.buffer) == 0 {
		bs, ok := <-r.Ch
		if !ok {
			if len(r.buffer) == 0 {
				return 0, io.EOF
			}
			break
		}
		r.buffer = append(r.buffer, bs...)
	}
	n := copy(p, r.buffer)
	r.buffer = r.buffer[n:]
	return n, nil
}

func (r *ByteSliceChannelReader) WriteTo(w io.Writer) (n int64, err error) {
	written := 0
	if flusher, ok := w.(http.Flusher); ok {
		for val := range r.Ch {
			n, err := w.Write(val)
			flusher.Flush()
			written += n
			if err != nil {
				break
			}
		}
	} else {
		for val := range r.Ch {
			n, err := w.Write(val)
			written += n
			if err != nil {
				break
			}
		}
	}
	return int64(written), err
}

type sseStreamer struct {
	bufReader *bufio.Reader
	err       error
}

func NewSSEStreamer(in io.Reader) Streamer[[]byte] {
	return &sseStreamer{
		bufReader: bufio.NewReader(in),
	}
}

func (c *sseStreamer) Read() ([]byte, error) {
	if c.err != nil {
		return nil, c.err
	}
	var line []byte
	var b = make([]byte, 0, 1024)
	for {
		line, c.err = c.bufReader.ReadBytes('\n')
		b = append(b, line...)
		if c.err != nil {
			if errors.Is(c.err, io.EOF) && len(b) > 0 {
				break
			}
			return c.Read()
		}
		if bytes.HasSuffix(b, []byte("\n\n")) {
			break
		}
	}
	b = bytes.TrimSuffix(b, []byte("\n\n"))
	b = bytes.TrimPrefix(b, []byte("data: "))
	return b, nil
}

type LineStreamer struct {
	bufReader *bufio.Reader
	line      []byte
	err       error
}

func NewLineStreamer(in io.Reader) Streamer[[]byte] {
	return &LineStreamer{
		bufReader: bufio.NewReader(in),
	}
}

func (s *LineStreamer) Read() ([]byte, error) {
	if s.err != nil {
		return nil, s.err
	}
	s.line, s.err = s.bufReader.ReadBytes('\n')
	if s.err != nil && errors.Is(s.err, io.EOF) {
		// we want io.EOF to be returned after the last actual data item
		return s.line, nil
	}
	return s.line, s.err
}

type byteStreamReader struct {
	streamer Streamer[[]byte]
	buffer   []byte
	err      error
	sep      []byte
	isFirst  bool
}

func ByteStreamerToIOReader(s Streamer[[]byte], sep []byte) io.Reader {
	return &byteStreamReader{streamer: s, sep: sep, isFirst: true}
}

func (r *byteStreamReader) Read(p []byte) (n int, err error) {
	if r.err == io.EOF {
		return 0, io.EOF
	}

	if len(r.buffer) == 0 {
		// note streamer should only return EOF with empty bytes
		r.buffer, r.err = r.streamer.Read()
		if !r.isFirst && len(r.sep) > 0 {
			r.buffer = append(r.sep, r.buffer...)
		} else {
			r.isFirst = false
		}
		if r.err != nil {
			return 0, r.err
		}
	}

	n = copy(p, r.buffer)
	r.buffer = r.buffer[n:]
	return n, nil
}
