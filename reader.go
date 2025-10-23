// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package openzl

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Reader implements io.ReadCloser for streaming decompression.
//
// Reader reads compressed frames from the underlying reader and decompresses
// them on demand. This allows decompression of large data streams without
// loading the entire compressed input into memory.
//
// Example:
//
//	file, _ := os.Open("input.zl")
//	reader, _ := openzl.NewReader(file)
//	defer reader.Close()
//
//	// Decompress data as it's read
//	io.Copy(destWriter, reader)
//
// The Reader reads frames written by Writer, which have a 4-byte little-endian
// frame length header followed by compressed data.
type Reader struct {
	r            io.Reader     // Underlying reader for compressed data
	decompressor *Decompressor // Reusable decompressor context
	buf          []byte        // Buffer for decompressed data from current frame
	bufPos       int           // Current read position in buffer
	bufSize      int           // Amount of valid data in buffer
	closed       bool          // Whether Close() has been called
	eof          bool          // Whether we've reached end-of-stream marker
	err          error         // Sticky error from previous operations
}

// NewReader creates a new Reader that reads compressed data from r and
// decompresses it.
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
	if r == nil {
		return nil, fmt.Errorf("nil reader")
	}

	// Create reusable decompressor
	decompressor, err := NewDecompressor()
	if err != nil {
		return nil, fmt.Errorf("create decompressor: %w", err)
	}

	return &Reader{
		r:            r,
		decompressor: decompressor,
	}, nil
}

// Read decompresses data from the underlying reader into p.
//
// Read implements the io.Reader interface. It reads and decompresses frames
// as needed to fill p. When the end-of-stream marker is reached, Read returns
// io.EOF.
//
// If an error occurs, the Reader enters an error state and all subsequent
// Read calls will return the same error.
func (r *Reader) Read(p []byte) (n int, err error) {
	if r.closed {
		return 0, fmt.Errorf("read from closed Reader")
	}
	if r.err != nil {
		return 0, r.err
	}
	if r.eof {
		return 0, io.EOF
	}

	totalRead := 0

	for totalRead < len(p) {
		// If buffer is empty, read and decompress next frame
		if r.bufPos >= r.bufSize {
			if err := r.readFrame(); err != nil {
				if err == io.EOF {
					r.eof = true
					if totalRead > 0 {
						return totalRead, nil
					}
					return 0, io.EOF
				}
				r.err = err
				if totalRead > 0 {
					return totalRead, nil
				}
				return 0, err
			}
		}

		// Copy from buffer to output
		available := r.bufSize - r.bufPos
		toCopy := len(p) - totalRead
		if toCopy > available {
			toCopy = available
		}

		copy(p[totalRead:], r.buf[r.bufPos:r.bufPos+toCopy])
		r.bufPos += toCopy
		totalRead += toCopy
	}

	return totalRead, nil
}

// readFrame reads and decompresses the next frame from the underlying reader.
func (r *Reader) readFrame() error {
	// Read 4-byte frame header (little-endian compressed size)
	var header [4]byte
	if _, err := io.ReadFull(r.r, header[:]); err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return io.EOF
		}
		return fmt.Errorf("read header: %w", err)
	}

	// Parse frame size
	frameSize := binary.LittleEndian.Uint32(header[:])

	// Zero-length frame is end-of-stream marker
	if frameSize == 0 {
		return io.EOF
	}

	// Read compressed frame data
	compressed := make([]byte, frameSize)
	if _, err := io.ReadFull(r.r, compressed); err != nil {
		if err == io.EOF {
			return io.ErrUnexpectedEOF
		}
		return fmt.Errorf("read frame: %w", err)
	}

	// Decompress frame
	decompressed, err := r.decompressor.Decompress(compressed)
	if err != nil {
		return fmt.Errorf("decompress: %w", err)
	}

	// Store decompressed data in buffer
	r.buf = decompressed
	r.bufPos = 0
	r.bufSize = len(decompressed)

	return nil
}

// Close releases resources associated with the Reader.
//
// Calling Close() multiple times is safe and has no effect after the first call.
func (r *Reader) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true

	// Close decompressor
	r.decompressor.Close()

	return nil
}

// Reset resets the Reader to read from a new underlying reader.
//
// This allows reuse of the Reader and its internal decompressor context for
// better performance when decompressing multiple streams.
//
// If the Reader was previously closed, Reset will create a new decompressor.
//
// Example:
//
//	reader, _ := openzl.NewReader(file1)
//	io.Copy(dest1, reader)
//	reader.Close()
//
//	reader.Reset(file2)  // Reuse the reader
//	io.Copy(dest2, reader)
//	reader.Close()
func (r *Reader) Reset(reader io.Reader) error {
	if reader == nil {
		return fmt.Errorf("nil reader")
	}

	// If closed, need to recreate decompressor
	if r.closed || r.decompressor == nil {
		decompressor, err := NewDecompressor()
		if err != nil {
			return fmt.Errorf("create decompressor: %w", err)
		}
		r.decompressor = decompressor
	}

	// Reset state
	r.r = reader
	r.buf = nil
	r.bufPos = 0
	r.bufSize = 0
	r.closed = false
	r.eof = false
	r.err = nil

	return nil
}

// Ensure Reader implements io.ReadCloser
var _ io.ReadCloser = (*Reader)(nil)
