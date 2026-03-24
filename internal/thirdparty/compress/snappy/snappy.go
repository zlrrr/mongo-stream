// Package snappy is a stub implementation of Snappy compression.
// This stub satisfies the interface required by go.mongodb.org/mongo-driver without
// performing real compression. It is used only when the real klauspost/compress
// package cannot be downloaded. Do NOT use in production.
package snappy

import (
	"encoding/binary"
	"errors"
)

// Encode returns a snappy-encoded form of src.
// This stub simply prefixes the source with its varint-encoded length.
func Encode(dst, src []byte) []byte {
	n := len(src)
	buf := make([]byte, binary.MaxVarintLen64+n)
	written := binary.PutUvarint(buf, uint64(n))
	copy(buf[written:], src)
	return buf[:written+n]
}

// DecodedLen returns the length of the decoded data.
func DecodedLen(src []byte) (int, error) {
	l, n := binary.Uvarint(src)
	if n <= 0 {
		return 0, errors.New("snappy: corrupt input")
	}
	return int(l), nil
}

// Decode decodes src into dst.
func Decode(dst, src []byte) ([]byte, error) {
	l, n := binary.Uvarint(src)
	if n <= 0 {
		return nil, errors.New("snappy: corrupt input")
	}
	if dst == nil {
		dst = make([]byte, l)
	}
	copy(dst, src[n:n+int(l)])
	return dst[:l], nil
}
