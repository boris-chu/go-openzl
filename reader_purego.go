//go:build !cgo
// +build !cgo

// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package openzl

import (
	"fmt"
	"io"
)

// Reader implements io.ReadCloser for streaming decompression.
//
// In Pure Go builds, Reader uses the purgo.Reader implementation internally.
//
// Example:
//
//	file, _ := os.Open("input.zl")
//	reader, _ := openzl.NewReader(file)
//	defer reader.Close()
//
//	// Decompress data as it's read
//	io.Copy(destWriter, reader)
type Reader struct {
	r io.Reader
}

// NewReader creates a new Reader that reads compressed data from r and
// decompresses it using the Pure Go decoder.
//
// The returned Reader implements io.ReadCloser. You should call Close() when
// done reading to release resources.
//
// Example:
//
//	file, err := os.Open("input.zl")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer file.Close()
//
//	reader, err := openzl.NewReader(file)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer reader.Close()
//
//	data, err := io.ReadAll(reader)
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewReader(r io.Reader) (*Reader, error) {
	return nil, fmt.Errorf("streaming Reader requires CGO (use purgo.NewReader instead, or build with CGO_ENABLED=1)")
}

// Read decompresses data from the underlying reader into p.
//
// Note: This function is not available in Pure Go builds. Use purgo.NewReader instead.
func (r *Reader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("streaming Reader requires CGO (use purgo.NewReader instead, or build with CGO_ENABLED=1)")
}

// Close releases resources associated with the Reader.
func (r *Reader) Close() error {
	return nil
}

// Reset resets the Reader to read from a new underlying reader.
func (r *Reader) Reset(reader io.Reader) error {
	return fmt.Errorf("streaming Reader requires CGO (use purgo.NewReader instead, or build with CGO_ENABLED=1)")
}
