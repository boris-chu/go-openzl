//go:build !cgo
// +build !cgo

// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package openzl

import (
	"fmt"
	"io"
)

// Writer implements io.WriteCloser for streaming compression.
//
// Note: Writer requires CGO for compression. This is not available in Pure Go builds.
type Writer struct {
	w io.Writer
}

const (
	// DefaultFrameSize is the default buffer size for streaming compression.
	DefaultFrameSize = 64 * 1024

	// MinFrameSize is the minimum frame size (4KB).
	MinFrameSize = 4 * 1024

	// MaxFrameSize is the maximum frame size (1MB).
	MaxFrameSize = 1024 * 1024
)

// WriterOption configures a Writer.
type WriterOption func(*Writer) error

// WithFrameSize sets the frame size for buffered compression.
//
// Note: Writer requires CGO. Build with CGO_ENABLED=1 to use this function.
func WithFrameSize(size int) WriterOption {
	return func(w *Writer) error {
		return fmt.Errorf("Writer requires CGO (build with CGO_ENABLED=1)")
	}
}

// NewWriter creates a new Writer that compresses data and writes it to w.
//
// Note: Compression requires CGO. Build with CGO_ENABLED=1 to use this function.
func NewWriter(w io.Writer, opts ...WriterOption) (*Writer, error) {
	return nil, fmt.Errorf("Writer requires CGO (build with CGO_ENABLED=1)")
}

// Write compresses data and writes it to the underlying writer.
//
// Note: Compression requires CGO. Build with CGO_ENABLED=1 to use this function.
func (w *Writer) Write(p []byte) (n int, err error) {
	return 0, fmt.Errorf("Writer requires CGO (build with CGO_ENABLED=1)")
}

// Close flushes any buffered data, writes final compressed frame, and releases resources.
func (w *Writer) Close() error {
	return nil
}

// Reset resets the Writer to write to a new underlying writer.
func (w *Writer) Reset(writer io.Writer) error {
	return fmt.Errorf("Writer requires CGO (build with CGO_ENABLED=1)")
}
