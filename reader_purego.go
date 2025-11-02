//go:build !cgo
// +build !cgo

// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package openzl

import (
	"fmt"
	"io"

	"github.com/boris-chu/go-openzl/purgo"
)

// Reader implements io.ReadCloser for streaming decompression using Pure Go.
//
// This is the Pure Go implementation that works without CGO. It provides
// streaming decompression with the io.Reader interface.
type Reader struct {
	impl *purgo.Reader
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
	impl, err := purgo.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("create reader: %w", err)
	}
	return &Reader{impl: impl}, nil
}

// Read decompresses data from the underlying reader into p.
//
// This implements the io.Reader interface.
func (r *Reader) Read(p []byte) (n int, err error) {
	if r.impl == nil {
		return 0, fmt.Errorf("reader not initialized")
	}
	return r.impl.Read(p)
}

// Close releases resources associated with the Reader.
//
// If the underlying reader implements io.Closer, it will also be closed.
func (r *Reader) Close() error {
	if r.impl == nil {
		return nil
	}
	return r.impl.Close()
}

// Reset resets the Reader to read from a new underlying reader.
//
// Note: Reset is not currently supported in Pure Go mode.
// Create a new Reader instead.
func (r *Reader) Reset(reader io.Reader) error {
	return fmt.Errorf("Reset not supported in Pure Go mode (create new Reader instead)")
}
