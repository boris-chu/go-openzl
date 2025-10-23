// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package openzl

import (
	"fmt"
	"io"
)

// Writer implements io.WriteCloser for streaming compression.
//
// Writer buffers input data and compresses it in chunks, writing compressed
// frames to the underlying writer. This allows compression of large data streams
// without loading the entire input into memory.
//
// Example:
//
//	file, _ := os.Create("output.zl")
//	writer, _ := openzl.NewWriter(file)
//	defer writer.Close()
//
//	// Compress data as it's written
//	io.Copy(writer, sourceReader)
//
// Important: You must call Close() to flush any buffered data and ensure
// all compressed data is written to the underlying writer.
type Writer struct {
	w          io.Writer     // Underlying writer for compressed data
	compressor *Compressor   // Reusable compressor context
	buf        []byte        // Buffer for incoming uncompressed data
	bufSize    int           // Current amount of data in buffer
	frameSize  int           // Size of each compression frame (default 64KB)
	closed     bool          // Whether Close() has been called
	err        error         // Sticky error from previous operations
}

const (
	// DefaultFrameSize is the default buffer size for streaming compression.
	// 64KB provides a good balance between compression ratio and memory usage.
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
// Larger frame sizes generally provide better compression ratios but use more
// memory. Smaller frame sizes reduce memory usage but may reduce compression ratio.
//
// The frame size must be between MinFrameSize (4KB) and MaxFrameSize (1MB).
// If not specified, DefaultFrameSize (64KB) is used.
func WithFrameSize(size int) WriterOption {
	return func(w *Writer) error {
		if size < MinFrameSize || size > MaxFrameSize {
			return fmt.Errorf("frame size must be between %d and %d bytes", MinFrameSize, MaxFrameSize)
		}
		w.frameSize = size
		w.buf = make([]byte, size)
		return nil
	}
}

// NewWriter creates a new Writer that compresses data and writes it to w.
//
// The returned Writer implements io.WriteCloser. You must call Close() when
// done writing to flush any buffered data.
//
// Example:
//
//	file, err := os.Create("output.zl")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer file.Close()
//
//	writer, err := openzl.NewWriter(file)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer writer.Close()
//
//	fmt.Fprintf(writer, "Hello, OpenZL!")
func NewWriter(w io.Writer, opts ...WriterOption) (*Writer, error) {
	if w == nil {
		return nil, fmt.Errorf("nil writer")
	}

	// Create reusable compressor
	compressor, err := NewCompressor()
	if err != nil {
		return nil, fmt.Errorf("create compressor: %w", err)
	}

	writer := &Writer{
		w:          w,
		compressor: compressor,
		frameSize:  DefaultFrameSize,
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(writer); err != nil {
			compressor.Close()
			return nil, err
		}
	}

	// Allocate buffer if not already done by options
	if writer.buf == nil {
		writer.buf = make([]byte, writer.frameSize)
	}

	return writer, nil
}

// Write compresses data and writes it to the underlying writer.
//
// Write buffers input data until a full frame is available, then compresses
// and writes the frame. This implements the io.Writer interface.
//
// If an error occurs, the Writer enters an error state and all subsequent
// Write calls will return the same error.
func (w *Writer) Write(p []byte) (n int, err error) {
	if w.closed {
		return 0, fmt.Errorf("write to closed Writer")
	}
	if w.err != nil {
		return 0, w.err
	}

	written := 0
	for len(p) > 0 {
		// Copy as much as possible to buffer
		available := w.frameSize - w.bufSize
		toCopy := len(p)
		if toCopy > available {
			toCopy = available
		}

		copy(w.buf[w.bufSize:], p[:toCopy])
		w.bufSize += toCopy
		p = p[toCopy:]
		written += toCopy

		// If buffer is full, compress and write it
		if w.bufSize == w.frameSize {
			if err := w.flush(); err != nil {
				w.err = err
				return written, err
			}
		}
	}

	return written, nil
}

// flush compresses and writes the current buffer to the underlying writer.
func (w *Writer) flush() error {
	if w.bufSize == 0 {
		return nil
	}

	// Compress the buffered data
	compressed, err := w.compressor.Compress(w.buf[:w.bufSize])
	if err != nil {
		return fmt.Errorf("compress: %w", err)
	}

	// Write frame header: 4-byte little-endian compressed size
	header := []byte{
		byte(len(compressed)),
		byte(len(compressed) >> 8),
		byte(len(compressed) >> 16),
		byte(len(compressed) >> 24),
	}

	if _, err := w.w.Write(header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	// Write compressed data
	if _, err := w.w.Write(compressed); err != nil {
		return fmt.Errorf("write compressed: %w", err)
	}

	// Reset buffer
	w.bufSize = 0

	return nil
}

// Close flushes any buffered data, writes final compressed frame, and releases resources.
//
// You must call Close() to ensure all data is written. Calling Close() multiple
// times is safe and has no effect after the first call.
func (w *Writer) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true

	// Flush any remaining buffered data
	if w.bufSize > 0 {
		if err := w.flush(); err != nil {
			w.compressor.Close()
			return err
		}
	}

	// Write end-of-stream marker (zero-length frame)
	header := []byte{0, 0, 0, 0}
	if _, err := w.w.Write(header); err != nil {
		w.compressor.Close()
		return fmt.Errorf("write end marker: %w", err)
	}

	// Close compressor
	w.compressor.Close()

	return nil
}

// Reset resets the Writer to write to a new underlying writer.
//
// This allows reuse of the Writer and its internal compressor context for
// better performance when compressing multiple streams.
//
// If the Writer was previously closed, Reset will create a new compressor.
//
// Example:
//
//	writer, _ := openzl.NewWriter(file1)
//	io.Copy(writer, data1)
//	writer.Close()
//
//	writer.Reset(file2)  // Reuse the writer
//	io.Copy(writer, data2)
//	writer.Close()
func (w *Writer) Reset(writer io.Writer) error {
	if writer == nil {
		return fmt.Errorf("nil writer")
	}

	// Flush any pending data first
	if !w.closed && w.bufSize > 0 {
		if err := w.flush(); err != nil {
			return err
		}
	}

	// If closed, need to recreate compressor
	if w.closed || w.compressor == nil {
		compressor, err := NewCompressor()
		if err != nil {
			return fmt.Errorf("create compressor: %w", err)
		}
		w.compressor = compressor
	}

	// Reset state
	w.w = writer
	w.bufSize = 0
	w.closed = false
	w.err = nil

	return nil
}

// Ensure Writer implements io.WriteCloser
var _ io.WriteCloser = (*Writer)(nil)
