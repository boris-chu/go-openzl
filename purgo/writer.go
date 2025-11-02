package purgo

import (
	"bytes"
	"fmt"
	"io"
)

// Writer provides streaming compression using the io.Writer interface.
//
// This allows incremental compression by buffering input data and writing
// compressed frames when the buffer is full or when Flush() is called.
//
// Example usage:
//
//	file, _ := os.Create("data.zl")
//	writer, _ := purgo.NewWriter(file)
//	defer writer.Close()
//
//	writer.Write([]byte("chunk 1"))
//	writer.Write([]byte("chunk 2"))
//	writer.Close() // Flushes remaining data
type Writer struct {
	// w is the underlying writer for compressed output
	w io.Writer

	// buf accumulates uncompressed data until frame size is reached
	buf *bytes.Buffer

	// frameSize is the target size for each frame (before compression)
	// Default: 1MB
	frameSize int

	// closed tracks whether Close() has been called
	closed bool

	// bytesWritten tracks total uncompressed bytes written
	bytesWritten int64

	// framesWritten tracks number of frames written
	framesWritten int
}

// WriterOption configures a Writer.
type WriterOption func(*Writer)

// WithFrameSize sets the target frame size for compression.
//
// Larger frames provide better compression ratios but use more memory.
// Smaller frames reduce memory usage and latency.
//
// Default: 1MB (1048576 bytes)
//
// Example:
//
//	writer, _ := purgo.NewWriter(file, purgo.WithFrameSize(512*1024)) // 512KB frames
func WithFrameSize(size int) WriterOption {
	return func(w *Writer) {
		if size > 0 {
			w.frameSize = size
		}
	}
}

const (
	// DefaultFrameSize is the default target frame size (1MB)
	DefaultFrameSize = 1024 * 1024 // 1MB
)

// NewWriter creates a new streaming compression writer.
//
// Data written to the Writer is buffered until the frame size is reached,
// then compressed and written to the underlying writer.
//
// Parameters:
//   - w: io.Writer to receive compressed output
//   - opts: Optional WriterOption functions for configuration
//
// Returns:
//   - *Writer: Streaming compression writer
//   - error: nil on success
//
// Example:
//
//	file, _ := os.Create("data.zl")
//	writer, _ := purgo.NewWriter(file)
//	defer writer.Close()
//
//	io.Copy(writer, inputFile) // Compress inputFile to data.zl
func NewWriter(w io.Writer, opts ...WriterOption) (*Writer, error) {
	if w == nil {
		return nil, fmt.Errorf("purgo: writer cannot be nil")
	}

	writer := &Writer{
		w:         w,
		buf:       new(bytes.Buffer),
		frameSize: DefaultFrameSize,
		closed:    false,
	}

	// Apply options
	for _, opt := range opts {
		opt(writer)
	}

	return writer, nil
}

// Write writes uncompressed data to the Writer.
//
// Data is buffered until the frame size is reached, then compressed
// and written to the underlying writer. This implements the io.Writer
// interface.
//
// Parameters:
//   - p: Uncompressed data to write
//
// Returns:
//   - n: Number of bytes written (always len(p) on success)
//   - err: Error if write fails
//
// Example:
//
//	writer, _ := purgo.NewWriter(file)
//	n, _ := writer.Write([]byte("hello world"))
func (w *Writer) Write(p []byte) (int, error) {
	if w.closed {
		return 0, fmt.Errorf("purgo: write to closed writer")
	}

	if len(p) == 0 {
		return 0, nil
	}

	// Write to buffer
	n, err := w.buf.Write(p)
	if err != nil {
		return n, fmt.Errorf("purgo: buffer write: %w", err)
	}

	w.bytesWritten += int64(n)

	// Flush if buffer exceeds frame size
	if w.buf.Len() >= w.frameSize {
		if err := w.Flush(); err != nil {
			return n, err
		}
	}

	return n, nil
}

// Flush compresses and writes any buffered data.
//
// This should be called when you want to ensure all buffered data is
// written, even if the frame size hasn't been reached.
//
// Returns:
//   - error: Error if compression or write fails
//
// Example:
//
//	writer.Write(data)
//	writer.Flush() // Ensure data is written immediately
func (w *Writer) Flush() error {
	if w.closed {
		return fmt.Errorf("purgo: flush closed writer")
	}

	if w.buf.Len() == 0 {
		// Nothing to flush
		return nil
	}

	// Compress buffered data
	compressed, err := Compress(w.buf.Bytes())
	if err != nil {
		return fmt.Errorf("purgo: compress frame: %w", err)
	}

	// Write compressed frame to underlying writer
	n, err := w.w.Write(compressed)
	if err != nil {
		return fmt.Errorf("purgo: write frame: %w", err)
	}

	if n != len(compressed) {
		return fmt.Errorf("purgo: short write: wrote %d bytes, expected %d", n, len(compressed))
	}

	// Clear buffer for next frame
	w.buf.Reset()
	w.framesWritten++

	return nil
}

// Close flushes any remaining buffered data and closes the writer.
//
// After Close, no further writes are allowed. If the underlying writer
// implements io.Closer, it will also be closed.
//
// Returns:
//   - error: Error if flush or close fails
//
// Example:
//
//	writer, _ := purgo.NewWriter(file)
//	defer writer.Close()
func (w *Writer) Close() error {
	if w.closed {
		return nil // Already closed
	}

	// Flush any remaining data
	if err := w.Flush(); err != nil {
		return err
	}

	w.closed = true

	// Close underlying writer if it implements io.Closer
	if closer, ok := w.w.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			return fmt.Errorf("purgo: close underlying writer: %w", err)
		}
	}

	return nil
}

// BytesWritten returns the total number of uncompressed bytes written.
//
// This is the sum of all data passed to Write(), regardless of how
// many compressed frames have been produced.
//
// Returns:
//   - int64: Total uncompressed bytes written
//
// Example:
//
//	writer.Write(data)
//	fmt.Printf("Wrote %d bytes\n", writer.BytesWritten())
func (w *Writer) BytesWritten() int64 {
	return w.bytesWritten
}

// FramesWritten returns the number of compressed frames written.
//
// Each call to Flush() (explicit or automatic) produces one frame.
//
// Returns:
//   - int: Number of frames written
//
// Example:
//
//	writer.Close()
//	fmt.Printf("Wrote %d frames\n", writer.FramesWritten())
func (w *Writer) FramesWritten() int {
	return w.framesWritten
}
