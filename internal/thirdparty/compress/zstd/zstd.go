// Package zstd is a stub implementation of zstd compression.
// This stub satisfies the interface required by go.mongodb.org/mongo-driver without
// performing real compression. It is used only when the real klauspost/compress
// package cannot be downloaded. Do NOT use in production.
package zstd

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

// EncoderLevel controls the compression level.
type EncoderLevel int

const (
	SpeedFastest           EncoderLevel = 1
	SpeedDefault           EncoderLevel = 3
	SpeedBetterCompression EncoderLevel = 6
	SpeedBestCompression   EncoderLevel = 11
)

// EncoderLevelFromZstd converts a zstd compression level integer to EncoderLevel.
func EncoderLevelFromZstd(level int) EncoderLevel {
	switch {
	case level <= int(SpeedFastest):
		return SpeedFastest
	case level >= int(SpeedBestCompression):
		return SpeedBestCompression
	default:
		return EncoderLevel(level)
	}
}

// EncoderOption is a functional option for the Encoder.
type EncoderOption func(*Encoder)

// WithWindowSize sets the window size option (no-op in stub).
func WithWindowSize(_ int) EncoderOption { return func(*Encoder) {} }

// WithEncoderLevel sets the encoder level (no-op in stub).
func WithEncoderLevel(_ EncoderLevel) EncoderOption { return func(*Encoder) {} }

// Encoder is a stub zstd encoder.
type Encoder struct {
	w io.Writer
}

// NewWriter creates a new stub encoder.
func NewWriter(w io.Writer, opts ...EncoderOption) (*Encoder, error) {
	enc := &Encoder{w: w}
	for _, o := range opts {
		o(enc)
	}
	return enc, nil
}

// EncodeAll encodes src appended to dst (stub: length-prefixed copy).
func (e *Encoder) EncodeAll(src, dst []byte) []byte {
	var buf [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(buf[:], uint64(len(src)))
	dst = append(dst, buf[:n]...)
	return append(dst, src...)
}

// Decoder is a stub zstd decoder.
type Decoder struct{}

// NewReader creates a stub decoder.
func NewReader(r io.Reader) (*Decoder, error) {
	return &Decoder{}, nil
}

// DecodeAll decodes src into dst (stub: reverses EncodeAll).
func (d *Decoder) DecodeAll(src, dst []byte) ([]byte, error) {
	r := bytes.NewReader(src)
	l, err := binary.ReadUvarint(r)
	if err != nil {
		return nil, errors.New("zstd: corrupt input")
	}
	payload := make([]byte, l)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, errors.New("zstd: corrupt input")
	}
	return append(dst, payload...), nil
}

// Reset resets the decoder (no-op in stub).
func (d *Decoder) Reset(r io.Reader) error { return nil }

// Close closes the decoder (no-op in stub).
func (d *Decoder) Close() {}
