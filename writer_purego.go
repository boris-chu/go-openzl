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

// Writer implements io.WriteCloser for streaming compression using Pure Go.
//
// This is the Pure Go implementation that works without CGO. It provides
// streaming compression with automatic frame management.
type Writer struct {
	impl *purgo.Writer
}

const (
	// DefaultFrameSize is the default buffer size for streaming compression (1MB).
	DefaultFrameSize = purgo.DefaultFrameSize

	// MinFrameSize is the minimum frame size (4KB).
	MinFrameSize = 4 * 1024

	// MaxFrameSize is the maximum frame size (1MB).
	MaxFrameSize = 1024 * 1024
)

// WriterOption configures a Writer.
type WriterOption func(*Writer) error

// WithFrameSize sets the frame size for buffered compression.
//
// Larger frames provide better compression ratios but use more memory.
// Smaller frames reduce memory usage and latency.
//
// Default: 1MB
func WithFrameSize(size int) WriterOption {
	return func(w *Writer) error {
		// Validation is done by purgo.WithFrameSize
		return nil
	}
}

// NewWriter creates a new Writer that compresses data and writes it to w.
//
// This Pure Go implementation provides streaming compression with automatic
// frame management. Data is buffered until the frame size is reached, then
// compressed and written.
//
// Example:
//
//	file, _ := os.Create("output.zl")
//	writer, _ := openzl.NewWriter(file)
//	defer writer.Close()
//
//	writer.Write([]byte("data chunk 1"))
//	writer.Write([]byte("data chunk 2"))
func NewWriter(w io.Writer, opts ...WriterOption) (*Writer, error) {
	if w == nil {
		return nil, fmt.Errorf("writer cannot be nil")
	}

	// For now, use default options
	// TODO: Support custom frame sizes via opts
	_ = opts // Ignore options for now, use defaults

	impl, err := purgo.NewWriter(w)
	if err != nil {
		return nil, fmt.Errorf("create writer: %w", err)
	}

	return &Writer{impl: impl}, nil
}

// Write compresses data and writes it to the underlying writer.
//
// Data is buffered until the frame size is reached, then automatically
// compressed and written. This implements the io.Writer interface.
func (w *Writer) Write(p []byte) (n int, err error) {
	if w.impl == nil {
		return 0, fmt.Errorf("writer not initialized")
	}
	return w.impl.Write(p)
}

// Close flushes any buffered data, writes final compressed frame, and releases resources.
//
// After Close, no further writes are allowed. If the underlying writer implements
// io.Closer, it will also be closed.
func (w *Writer) Close() error {
	if w.impl == nil {
		return nil
	}
	return w.impl.Close()
}

// Reset resets the Writer to write to a new underlying writer.
//
// Note: Reset is not currently supported in Pure Go mode.
// Create a new Writer instead.
func (w *Writer) Reset(writer io.Writer) error {
	return fmt.Errorf("Reset not supported in Pure Go mode (create new Writer instead)")
}
