package iteru

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"os"
	"slices"
)

func ToSeq[T any](s []T) iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, x := range s {
			if !yield(x) {
				return
			}
		}
	}
}

func Merge[T any](s1, s2 iter.Seq[T], cmp func(a, b T) int) iter.Seq[T] {
	next1, stop1 := iter.Pull(s1)
	next2, stop2 := iter.Pull(s2)
	v1, valid1 := next1()
	v2, valid2 := next2()
	return func(yield func(T) bool) {
		defer stop1()
		defer stop2()
		for valid1 || valid2 {
			if valid1 && (!valid2 || cmp(v1, v2) <= 0) {
				if !yield(v1) {
					return
				}
				v1, valid1 = next1()
			} else {
				if !yield(v2) {
					return
				}
				v2, valid2 = next2()
			}
		}
	}
}

func Map[V, E any](s iter.Seq[V], m func(v V) E) iter.Seq[E] {
	return func(yield func(E) bool) {
		for v := range s {
			if !yield(m(v)) {
				return
			}
		}
	}
}

func Map2[K1, V1, K2, V2 any](s iter.Seq2[K1, V1], m func(k K1, v V1) (K2, V2)) iter.Seq2[K2, V2] {
	return func(yield func(K2, V2) bool) {
		for k1, v1 := range s {
			k2, v2 := m(k1, v1)
			if !yield(k2, v2) {
				return
			}
		}
	}
}

func Filter[T any](i iter.Seq[T], keep func(T) bool) iter.Seq[T] {
	return func(yield func(T) bool) {
		for v := range i {
			if keep(v) {
				if !yield(v) {
					return
				}
			}
		}
	}
}

func Filter2[T, E any](i iter.Seq2[T, E], keep func(T, E) bool) iter.Seq2[T, E] {
	return func(yield func(T, E) bool) {
		for k, v := range i {
			if keep(k, v) {
				if !yield(k, v) {
					return
				}
			}
		}
	}
}

func Dedupe[T any](i iter.Seq[T], cmp func(a, b T) int) iter.Seq[T] {
	return func(yield func(T) bool) {
		var prev T
		ind := 0
		for v := range i {
			if ind == 0 || cmp(prev, v) != 0 {
				if !yield(v) {
					return
				}
				prev = v
			}
			ind += 1
		}
	}
}

func SeqToSeq2[V1, V2, V3 any](seq iter.Seq[V1], m func(V1) (V2, V3)) iter.Seq2[V2, V3] {
	return func(yield func(V2, V3) bool) {
		for v1 := range seq {
			v2, v3 := m(v1)
			if !yield(v2, v3) {
				return
			}
		}
	}
}

// --------------------------------------------------------------------------------------
// error iterator supporting functions - typical if iterating over items originating from io
// note: when an error occurs it should be returned once then the sequence ended, io.EOF
// errors should not be returned at all instead the seq ended. Typical consumer code
// looks like:
// 	for v, err := range seqVErr {
//		if err != nil {
//			// handle err
//			break
//		}
//		// standard handling
//	}

// Dedupe2Err removes duplicates from a sorted sequence
func Dedupe2Err[T any](i iter.Seq2[T, error], cmp func(a, b T) int) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		var prev T
		ind := 0
		for v, err := range i {
			if err != nil {
				yield(v, err)
				return
			}
			if ind == 0 || cmp(prev, v) != 0 {
				if !yield(v, nil) {
					return
				}
				prev = v
			}
			ind += 1
		}
	}
}

func Empty2Err[T any]() iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		return
	}
}

func Filter2Err[T any](i iter.Seq2[T, error], keep func(T) bool) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		for v, err := range i {
			if err != nil {
				yield(v, err)
				return
			}
			if keep(v) {
				if !yield(v, nil) {
					return
				}
			}
		}
	}
}

func Map2Err[V1, V2 any](s iter.Seq2[V1, error], m func(v V1) (V2, error)) iter.Seq2[V2, error] {
	return func(yield func(V2, error) bool) {
		for v1, err := range s {
			v2, err2 := m(v1)
			if err != nil {
				yield(v2, err)
				return
			}
			if err2 != nil {
				yield(v2, err2)
				return
			}
			if !yield(v2, err) {
				return
			}
		}
	}
}

func Map2ErrNoErr[V1, V2 any](s iter.Seq2[V1, error], m func(v V1) V2) iter.Seq2[V2, error] {
	return func(yield func(V2, error) bool) {
		for v1, err := range s {
			v2 := m(v1)
			if !yield(v2, err) || err != nil {
				return
			}
		}
	}
}

func Verify2Err[T any](s iter.Seq2[T, error], ok func(a, b T) bool) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		var prev T
		i := 0
		for v, err := range s {
			if err != nil {
				yield(v, err)
				return
			}
			if i > 0 && !ok(prev, v) {
				yield(v, fmt.Errorf("verify failed for items at index %v and %v", i, i+1))
				return
			}
			if !yield(v, nil) {
				return
			}
			prev = v
			i += 1
		}
	}
}

// Merge2Err merge two paired iterators with an error second value, short circuits when either
func Merge2Err[T any](s1, s2 iter.Seq2[T, error], cmp func(a, b T) int) iter.Seq2[T, error] {
	next1, stop1 := iter.Pull2(s1)
	next2, stop2 := iter.Pull2(s2)
	v1, err1, valid1 := next1()
	v2, err2, valid2 := next2()
	return func(yield func(T, error) bool) {
		defer stop1()
		defer stop2()
		if err1 != nil {
			yield(v1, err1)
			return
		}
		if err2 != nil {
			yield(v2, err2)
			return
		}
		for valid1 || valid2 {
			if valid1 && (!valid2 || cmp(v1, v2) <= 0) {
				if !yield(v1, nil) {
					return
				}
				v1, err1, valid1 = next1()
				if err1 != nil {
					yield(v1, err1)
					return
				}
			} else {
				if !yield(v2, nil) {
					return
				}
				v2, err2, valid2 = next2()
				if err2 != nil {
					yield(v2, err2)
					return
				}
			}
		}
	}
}

// LineSeq iterates over lines from an underlying io.Reader the behaviour should be as if you
// called strings.split(text, '\n') so if the text ends in a new line the last item will be an
// empty string. If an error occurs any partial line read will be returned with the error.
func LineSeq(in io.Reader, stripNewLines bool) iter.Seq2[[]byte, error] {
	var bufReader = bufio.NewReader(in)
	return func(yield func([]byte, error) bool) {
		var err error
		var line []byte
		for {
			line, err = bufReader.ReadBytes('\n')
			if err != nil {
				if errors.Is(err, io.EOF) {
					// we want to end the loop without an error when EOF
					yield(line, nil)
				} else {
					yield(line, err)
				}
				return
			}
			if stripNewLines {
				if !yield(line[:len(line)-1], nil) {
					return
				}
			} else {
				if !yield(line, nil) {
					return
				}
			}
		}
	}
}

// SSESeq iterates over server sent events from an underlying io.Reader
func SSESeq(in io.Reader) iter.Seq2[[]byte, error] {
	bufReader := bufio.NewReader(in)
	return func(yield func([]byte, error) bool) {
		prefix := []byte("data: ")
		suffix := []byte("\n\n")
		for {
			var b = make([]byte, 0, 1024)
			for {
				line, err := bufReader.ReadBytes('\n')
				b = append(b, line...)
				if err != nil {
					if errors.Is(err, io.EOF) {
						b = bytes.TrimPrefix(bytes.TrimSuffix(b, suffix), prefix)
						yield(b, nil)
						return
					} else {
						yield(nil, err)
						return
					}
				}
				if bytes.HasSuffix(b, suffix) {
					break
				}
			}
			b = bytes.TrimPrefix(bytes.TrimSuffix(b, suffix), prefix)
			if !yield(b, nil) {
				return
			}
		}
	}
}

type IoReaderFunc func(b []byte) (int, error)

func (f IoReaderFunc) Read(b []byte) (int, error) {
	return f(b)
}

// Seq2ErrToIOReader converts a bytes iterator to an io.Reader useful if you have an
// iterator of ([]byte, error) but need to pass it to something that takes an io.Reader
func Seq2ErrToIOReader(in iter.Seq2[[]byte, error]) io.Reader {
	next, stop := iter.Pull2(in)
	var err error
	var valid = true
	var buf []byte
	var nextBytes []byte
	return IoReaderFunc(func(b []byte) (int, error) {
	top:
		if err != nil {
			return 0, err
		} else if !valid && len(buf) == 0 {
			return 0, io.EOF
		} else if !valid || len(buf) >= len(b) {
			n := copy(b, buf)
			buf = buf[n:]
			return n, nil
		}

		for len(buf) < len(b) {
			nextBytes, err, valid = next()
			if err != nil || !valid {
				stop()
				break
			}
			buf = append(buf, nextBytes...)
		}
		goto top
	})
}

func Extend2Err[T any](seqs ...iter.Seq2[T, error]) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		for _, seq := range seqs {
			for item, err := range seq {
				if err != nil {
					yield(item, err)
					return
				}
				if !yield(item, nil) {
					return
				}
			}
		}
	}
}

// LengthPrefixedBytesSeq the reader will read chunks of bytes
// from an io.Reader assuming that each chunk is prefixed by its size
// as little endian 4 byte integer
func LengthPrefixedBytesSeq(in io.Reader) iter.Seq2[[]byte, error] {
	return func(yield func(b []byte, err error) bool) {
		bufLength := make([]byte, 4)
		for {
			_, err := io.ReadAtLeast(in, bufLength, 4)
			// io.ReadAtLeast only returns EOF when 0 bytes are read
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				yield(nil, err)
				return
			}
			// convert buff to little endian integer and read that number of bytes
			toRead := int(binary.LittleEndian.Uint32(bufLength))
			bufVal := make([]byte, toRead)
			if toRead == 0 {
				// weird case where a chunk of 0 bytes is intended
				if !yield(bufVal, nil) {
					return
				}
				continue
			}

			_, err = io.ReadAtLeast(in, bufVal, toRead)
			if err != nil {
				if errors.Is(err, io.EOF) {
					// ReadAtLeast returns io.EOF only of no bytes where read
					err = io.ErrUnexpectedEOF
				}
				yield(nil, err)
				return
			}
			if !yield(bufVal, nil) {
				return
			}
		}
	}
}

func BigEndianLengthPrefixedBytesSeq(in io.Reader) iter.Seq2[[]byte, error] {
	return func(yield func(b []byte, err error) bool) {
		bufLength := make([]byte, 4)
		for {
			_, err := io.ReadAtLeast(in, bufLength, 4)
			// io.ReadAtLeast only returns EOF when 0 bytes are read
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				yield(nil, err)
				return
			}
			toRead := int(binary.BigEndian.Uint32(bufLength))
			bufVal := make([]byte, toRead)
			if toRead == 0 {
				// weird case where a chunk of 0 bytes is intended
				if !yield(bufVal, nil) {
					return
				}
				continue
			}

			_, err = io.ReadAtLeast(in, bufVal, toRead)
			if err != nil {
				if errors.Is(err, io.EOF) {
					// ReadAtLeast returns io.EOF only of no bytes where read
					err = io.ErrUnexpectedEOF
				}
				yield(nil, err)
				return
			}
			if !yield(bufVal, nil) {
				return
			}
		}
	}
}

// SortSeq2ErrDisk sorts a potentially large sequence using files on disk
func SortSeq2ErrDisk[T any](
	ctx context.Context,
	seq iter.Seq2[T, error],
	seqToReader func(iter.Seq[T]) io.Reader,
	readerToSeq func(reader io.Reader) iter.Seq2[T, error],
	cmp func(a, b T) int) iter.Seq2[T, error] {

	return func(yield func(T, error) bool) {
		files := make([]*os.File, 0)
		batch := make([]T, 0)
		batchSize := 500_000
		var err error
		var item T

		defer func() {
			for _, file := range files {
				_ = file.Close()
				_ = os.Remove(file.Name())
			}
		}()
		defer func() {
			if err != nil {
				yield(item, err)
			}
		}()

		batchCount := 0
		for item, err = range seq {
			if err != nil {
				return
			}
			batchCount++
			batch = append(batch, item)
			if batchCount > batchSize {
				slices.SortFunc(batch, cmp)
				reader := seqToReader(ToSeq(batch))
				var tmpFile *os.File
				tmpFile, err = os.CreateTemp("", "")
				if err != nil {
					return
				}
				files = append(files, tmpFile)
				if _, err = io.Copy(tmpFile, reader); err != nil {
					return
				}
				if _, err = tmpFile.Seek(0, 0); err != nil {
					return
				}
				slog.DebugContext(ctx, fmt.Sprintf("SortSeq2ErrDisk sorted batch %v of %v items", len(files), batchCount))
				batch = batch[:0]
				batchCount = 0
			}
		}
		slices.SortFunc(batch, cmp)
		i1 := func(yield func(T, error) bool) {
			for _, x := range batch {
				if !yield(x, nil) {
					return
				}
			}
		}
		slog.DebugContext(ctx, fmt.Sprintf("SortSeq2ErrDisk sorted batch %v of %v items", len(files)+1, batchCount))

		for _, file := range files {
			i1 = Merge2Err(i1, readerToSeq(file), cmp)
		}
		for b, err := range i1 {
			if !yield(b, err) {
				return
			}
		}
	}
}
