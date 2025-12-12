package binaryu

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"time"

	"github.com/google/uuid"
)

type Encodable interface {
	Encode() []byte
}

func WriteUUID(b *bytes.Buffer, d uuid.UUID) {
	b.Write(d[:])
}

func WriteString(b *bytes.Buffer, d string) {
	WriteInt32(b, int32(len(d)))
	b.Write([]byte(d))
}

func WriteUInt8(b *bytes.Buffer, d uint8) {
	b.WriteByte(d)
}

func WriteInt32(b *bytes.Buffer, d int32) {
	bs := make([]byte, 4)
	binary.BigEndian.PutUint32(bs, uint32(d))
	b.Write(bs)
}

func WriteInt64(b *bytes.Buffer, d int64) {
	bs := make([]byte, 8)
	binary.BigEndian.PutUint64(bs, uint64(d))
	b.Write(bs)
}

func WriteFloat64(b *bytes.Buffer, d float64) {
	bs := make([]byte, 8)
	binary.BigEndian.PutUint64(bs, math.Float64bits(d))
	b.Write(bs)
}

func WriteBool(b *bytes.Buffer, d bool) {
	if d {
		b.WriteByte(byte(1))
	} else {
		b.WriteByte(byte(0))
	}
}

func WriteTimeMs(b *bytes.Buffer, time time.Time) {
	WriteInt64(b, time.UTC().UnixMilli())
}

func WriteTimeS(b *bytes.Buffer, time time.Time) {
	WriteInt64(b, time.UTC().Unix())
}

func WriteMap[K comparable, V any](b *bytes.Buffer, m map[K]V, keyWriter func(*bytes.Buffer, K), valueWriter func(*bytes.Buffer, V)) {
	WriteInt32(b, int32(len(m)))
	for k, v := range m {
		keyWriter(b, k)
		valueWriter(b, v)
	}
}

func WriteSlice[V Encodable](b *bytes.Buffer, m []V) {
	WriteInt32(b, int32(len(m)))
	for _, v := range m {
		b.Write(v.Encode())
	}
}

func ReadUUID(r io.Reader) (uuid.UUID, error) {
	var d uuid.UUID
	_, err := io.ReadFull(r, d[:])
	if err != nil {
		return uuid.UUID{}, err
	}
	return d, nil
}

func ReadString(r io.Reader) (string, error) {
	length, err := ReadInt32(r)
	if err != nil {
		return "", err
	}

	bs := make([]byte, length)
	_, err = io.ReadFull(r, bs)
	if err != nil {
		return "", err
	}
	return string(bs), nil
}

func ReadInt32(r io.Reader) (int32, error) {
	bs := make([]byte, 4)
	_, err := io.ReadFull(r, bs)
	if err != nil {
		return 0, err
	}
	return int32(binary.BigEndian.Uint32(bs)), nil
}

func ReadInt64(r io.Reader) (int64, error) {
	bs := make([]byte, 8)
	_, err := io.ReadFull(r, bs)
	if err != nil {
		return 0, err
	}
	return int64(binary.BigEndian.Uint64(bs)), nil
}

func ReadBool(r io.Reader) (bool, error) {
	bs := make([]byte, 1)
	_, err := io.ReadFull(r, bs)
	if err != nil {
		return false, err
	}
	return bs[0] != 0, nil
}

func ReadTimeMs(r io.Reader) (time.Time, error) {
	ms, err := ReadInt64(r)
	if err != nil {
		return time.Time{}, err
	}
	return time.UnixMilli(ms), nil
}

func ReadTimeS(r io.Reader) (time.Time, error) {
	s, err := ReadInt64(r)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(s, 0), nil
}

func ReadFloat64(r io.Reader) (float64, error) {
	bs := make([]byte, 8)
	_, err := io.ReadFull(r, bs)
	if err != nil {
		return 0, err
	}
	return math.Float64frombits(binary.BigEndian.Uint64(bs)), nil
}

func ReadMap[K comparable, V any](r io.Reader, keyReader func(io.Reader) (K, error), valueReader func(io.Reader) (V, error)) (map[K]V, error) {
	length, err := ReadInt32(r)
	if err != nil {
		return nil, err
	}

	m := make(map[K]V, length)
	for i := int32(0); i < length; i++ {
		k, err := keyReader(r)
		if err != nil {
			return nil, err
		}
		v, err := valueReader(r)
		if err != nil {
			return nil, err
		}
		m[k] = v
	}
	return m, nil
}

func ReadSlice[V any](r io.Reader, valueReader func(io.Reader) (V, error)) ([]V, error) {
	length, err := ReadInt32(r)
	if err != nil {
		return nil, err
	}

	slice := make([]V, length)
	for i := int32(0); i < length; i++ {
		v, err := valueReader(r)
		if err != nil {
			return nil, err
		}
		slice[i] = v
	}
	return slice, nil
}
