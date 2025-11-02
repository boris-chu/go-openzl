package purgo

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

// TestNewWriter_Success tests successful Writer creation.
func TestNewWriter_Success(t *testing.T) {
	buf := new(bytes.Buffer)
	writer, err := NewWriter(buf)
	if err != nil {
		t.Fatalf("NewWriter() failed: %v", err)
	}
	if writer == nil {
		t.Fatal("NewWriter() returned nil writer")
	}
	if writer.frameSize != DefaultFrameSize {
		t.Errorf("Default frame size = %d, want %d", writer.frameSize, DefaultFrameSize)
	}
	if writer.BytesWritten() != 0 {
		t.Errorf("Initial BytesWritten() = %d, want 0", writer.BytesWritten())
	}
	if writer.FramesWritten() != 0 {
		t.Errorf("Initial FramesWritten() = %d, want 0", writer.FramesWritten())
	}
}

// TestNewWriter_NilWriter tests that NewWriter rejects nil writer.
func TestNewWriter_NilWriter(t *testing.T) {
	_, err := NewWriter(nil)
	if err == nil {
		t.Error("NewWriter(nil) should return error")
	}
}

// TestWriter_WriteSmallData tests writing data smaller than frame size.
func TestWriter_WriteSmallData(t *testing.T) {
	buf := new(bytes.Buffer)
	writer, _ := NewWriter(buf)
	defer writer.Close()

	data := []byte("hello world")
	n, err := writer.Write(data)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if n != len(data) {
		t.Errorf("Write() n = %d, want %d", n, len(data))
	}

	// Data should be buffered, not yet written
	if buf.Len() > 0 {
		t.Error("Data written before Flush/Close")
	}

	// Close should flush the data
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// Now compressed data should be written
	if buf.Len() == 0 {
		t.Error("No data written after Close()")
	}
}

// TestWriter_Roundtrip tests write→read roundtrip.
func TestWriter_Roundtrip(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"small", []byte("hello world")},
		{"medium", []byte(strings.Repeat("test data ", 100))},
		{"repeated", bytes.Repeat([]byte("A"), 1000)},
		{"binary", []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Compress
			buf := new(bytes.Buffer)
			writer, _ := NewWriter(buf)

			n, err := writer.Write(tt.data)
			if err != nil {
				t.Fatalf("Write() error = %v", err)
			}
			if n != len(tt.data) {
				t.Errorf("Write() n = %d, want %d", n, len(tt.data))
			}

			if err := writer.Close(); err != nil {
				t.Fatalf("Close() error = %v", err)
			}

			// Decompress
			reader, _ := NewReader(buf)
			decompressed, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("ReadAll() error = %v", err)
			}

			// Verify
			if !bytes.Equal(decompressed, tt.data) {
				t.Errorf("Roundtrip mismatch:\nwant: %q\ngot:  %q", tt.data, decompressed)
			}
		})
	}
}

// TestWriter_LargeData tests writing data larger than frame size.
func TestWriter_LargeData(t *testing.T) {
	buf := new(bytes.Buffer)
	// Use small frame size to test auto-flush behavior
	writer, _ := NewWriter(buf, WithFrameSize(512))
	defer writer.Close()

	// Write varied data that exceeds frame size
	// Use varied data to avoid compression making it too small
	chunk := []byte("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ!@#$%^&*()_+-=[]{}|;:,.<>?/~` ")

	// Write 10 chunks (~1000 bytes total, > 512 frame size)
	// This will trigger auto-flush
	for i := 0; i < 10; i++ {
		n, err := writer.Write(chunk)
		if err != nil {
			t.Fatalf("Write() error = %v", err)
		}
		if n != len(chunk) {
			t.Errorf("Write() n = %d, want %d", n, len(chunk))
		}
	}

	// At this point, at least one frame should have been auto-flushed
	if writer.FramesWritten() < 1 {
		t.Error("Expected at least one auto-flushed frame")
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// Verify bytes written counter is correct
	expectedBytes := int64(len(chunk) * 10)
	if writer.BytesWritten() != expectedBytes {
		t.Errorf("BytesWritten() = %d, want %d", writer.BytesWritten(), expectedBytes)
	}
}

// TestWriter_IncrementalWrites tests multiple small writes.
func TestWriter_IncrementalWrites(t *testing.T) {
	buf := new(bytes.Buffer)
	writer, _ := NewWriter(buf)
	defer writer.Close()

	chunks := [][]byte{
		[]byte("chunk 1 "),
		[]byte("chunk 2 "),
		[]byte("chunk 3"),
	}

	for _, chunk := range chunks {
		if _, err := writer.Write(chunk); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// Verify all chunks were written
	expected := bytes.Join(chunks, nil)
	reader, _ := NewReader(buf)
	decompressed, _ := io.ReadAll(reader)

	if !bytes.Equal(decompressed, expected) {
		t.Errorf("Incremental writes mismatch:\nwant: %q\ngot:  %q", expected, decompressed)
	}
}

// TestWriter_Flush tests explicit Flush().
func TestWriter_Flush(t *testing.T) {
	buf := new(bytes.Buffer)
	writer, _ := NewWriter(buf)
	defer writer.Close()

	// Write small data (less than frame size)
	data := []byte("test data")
	writer.Write(data)

	// Data should still be buffered
	if buf.Len() > 0 {
		t.Error("Data written before Flush()")
	}

	// Explicit flush
	if err := writer.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	// Now data should be written
	if buf.Len() == 0 {
		t.Error("No data written after Flush()")
	}

	// Verify frame was written
	if writer.FramesWritten() != 1 {
		t.Errorf("FramesWritten() = %d, want 1", writer.FramesWritten())
	}
}

// TestWriter_MultipleFlush tests multiple Flush() calls.
func TestWriter_MultipleFlush(t *testing.T) {
	buf := new(bytes.Buffer)
	writer, _ := NewWriter(buf)
	defer writer.Close()

	// Write and flush multiple times
	for i := 0; i < 3; i++ {
		data := []byte("data chunk ")
		writer.Write(data)
		if err := writer.Flush(); err != nil {
			t.Fatalf("Flush() error = %v", err)
		}
	}

	if writer.FramesWritten() != 3 {
		t.Errorf("FramesWritten() = %d, want 3", writer.FramesWritten())
	}
}

// TestWriter_EmptyFlush tests flushing with no buffered data.
func TestWriter_EmptyFlush(t *testing.T) {
	buf := new(bytes.Buffer)
	writer, _ := NewWriter(buf)
	defer writer.Close()

	// Flush with no data should be a no-op
	if err := writer.Flush(); err != nil {
		t.Errorf("Flush() with no data error = %v", err)
	}

	if writer.FramesWritten() != 0 {
		t.Errorf("FramesWritten() = %d, want 0", writer.FramesWritten())
	}
}

// TestWriter_Close tests Close() behavior.
func TestWriter_Close(t *testing.T) {
	buf := new(bytes.Buffer)
	writer, _ := NewWriter(buf)

	data := []byte("test")
	writer.Write(data)

	// Close should flush remaining data
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// Second close should be no-op
	if err := writer.Close(); err != nil {
		t.Errorf("Second Close() error = %v", err)
	}

	// Write after close should error
	_, err := writer.Write([]byte("after close"))
	if err == nil {
		t.Error("Write() after Close() should return error")
	}
}

// TestWriter_WriteAfterClose tests that writes after Close fail.
func TestWriter_WriteAfterClose(t *testing.T) {
	buf := new(bytes.Buffer)
	writer, _ := NewWriter(buf)
	writer.Close()

	_, err := writer.Write([]byte("data"))
	if err == nil {
		t.Error("Write() after Close() should return error")
	}
}

// TestWriter_FlushAfterClose tests that flush after Close fails.
func TestWriter_FlushAfterClose(t *testing.T) {
	buf := new(bytes.Buffer)
	writer, _ := NewWriter(buf)
	writer.Close()

	err := writer.Flush()
	if err == nil {
		t.Error("Flush() after Close() should return error")
	}
}

// TestWriter_BytesWritten tests BytesWritten() counter.
func TestWriter_BytesWritten(t *testing.T) {
	buf := new(bytes.Buffer)
	writer, _ := NewWriter(buf)
	defer writer.Close()

	totalBytes := int64(0)
	chunks := [][]byte{
		[]byte("chunk 1"),
		[]byte("chunk 2"),
		[]byte("chunk 3"),
	}

	for _, chunk := range chunks {
		writer.Write(chunk)
		totalBytes += int64(len(chunk))

		if writer.BytesWritten() != totalBytes {
			t.Errorf("BytesWritten() = %d, want %d", writer.BytesWritten(), totalBytes)
		}
	}
}

// TestWriter_WithFrameSize tests custom frame size option.
func TestWriter_WithFrameSize(t *testing.T) {
	buf := new(bytes.Buffer)
	customSize := 512
	writer, _ := NewWriter(buf, WithFrameSize(customSize))
	defer writer.Close()

	if writer.frameSize != customSize {
		t.Errorf("frameSize = %d, want %d", writer.frameSize, customSize)
	}

	// Write data exceeding custom frame size
	data := bytes.Repeat([]byte("x"), 600)
	writer.Write(data)

	// Should have auto-flushed
	if writer.FramesWritten() < 1 {
		t.Error("Frame should have been auto-flushed")
	}
}

// TestWriter_ZeroFrameSize tests that invalid frame sizes are rejected.
func TestWriter_ZeroFrameSize(t *testing.T) {
	buf := new(bytes.Buffer)
	writer, _ := NewWriter(buf, WithFrameSize(0))
	defer writer.Close()

	// Should use default frame size
	if writer.frameSize != DefaultFrameSize {
		t.Errorf("frameSize = %d, want default %d", writer.frameSize, DefaultFrameSize)
	}
}

// TestWriter_NegativeFrameSize tests that negative frame sizes are rejected.
func TestWriter_NegativeFrameSize(t *testing.T) {
	buf := new(bytes.Buffer)
	writer, _ := NewWriter(buf, WithFrameSize(-100))
	defer writer.Close()

	// Should use default frame size
	if writer.frameSize != DefaultFrameSize {
		t.Errorf("frameSize = %d, want default %d", writer.frameSize, DefaultFrameSize)
	}
}

// TestWriter_ioCopy tests using io.Copy with Writer.
func TestWriter_ioCopy(t *testing.T) {
	// Source data
	data := []byte(strings.Repeat("io.Copy test data ", 1000))
	source := bytes.NewReader(data)

	// Compress using io.Copy
	buf := new(bytes.Buffer)
	writer, _ := NewWriter(buf)

	n, err := io.Copy(writer, source)
	if err != nil {
		t.Fatalf("io.Copy() error = %v", err)
	}
	if n != int64(len(data)) {
		t.Errorf("io.Copy() n = %d, want %d", n, len(data))
	}

	writer.Close()

	// Decompress and verify
	reader, _ := NewReader(buf)
	decompressed, _ := io.ReadAll(reader)

	if !bytes.Equal(decompressed, data) {
		t.Error("io.Copy roundtrip mismatch")
	}
}

// TestWriter_CloserInterface tests that underlying Closer is called.
func TestWriter_CloserInterface(t *testing.T) {
	closeCounter := &closeCounter{Buffer: new(bytes.Buffer)}
	writer, _ := NewWriter(closeCounter)

	writer.Write([]byte("test"))
	writer.Close()

	if closeCounter.closeCalls != 1 {
		t.Errorf("Close() called %d times, want 1", closeCounter.closeCalls)
	}
}

// closeCounter wraps bytes.Buffer with a Close counter
type closeCounter struct {
	*bytes.Buffer
	closeCalls int
}

func (c *closeCounter) Close() error {
	c.closeCalls++
	return nil
}

// TestWriter_UnderlyingWriteError tests error propagation from underlying writer.
func TestWriter_UnderlyingWriteError(t *testing.T) {
	errWriter := &errorWriter{err: errors.New("write error")}
	writer, _ := NewWriter(errWriter)

	writer.Write([]byte("data"))
	err := writer.Flush()

	if err == nil {
		t.Error("Flush() should return error from underlying writer")
	}
}

// errorWriter always returns an error on Write
type errorWriter struct {
	err error
}

func (e *errorWriter) Write(p []byte) (int, error) {
	return 0, e.err
}

// TestWriter_EmptyWrite tests writing empty data.
func TestWriter_EmptyWrite(t *testing.T) {
	buf := new(bytes.Buffer)
	writer, _ := NewWriter(buf)
	defer writer.Close()

	n, err := writer.Write([]byte{})
	if err != nil {
		t.Errorf("Write([]) error = %v", err)
	}
	if n != 0 {
		t.Errorf("Write([]) n = %d, want 0", n)
	}
}

// Benchmarks

// BenchmarkWriter_SmallData benchmarks compressing small data chunks.
func BenchmarkWriter_SmallData(b *testing.B) {
	data := []byte(strings.Repeat("benchmark test ", 64)) // ~1KB
	buf := new(bytes.Buffer)

	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		buf.Reset()
		writer, _ := NewWriter(buf)
		writer.Write(data)
		writer.Close()
	}
}

// BenchmarkWriter_LargeData benchmarks compressing large data chunks.
func BenchmarkWriter_LargeData(b *testing.B) {
	data := []byte(strings.Repeat("benchmark test data ", 51200)) // ~1MB
	buf := new(bytes.Buffer)

	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		buf.Reset()
		writer, _ := NewWriter(buf)
		writer.Write(data)
		writer.Close()
	}
}

// BenchmarkWriter_IncrementalWrites benchmarks many small writes.
func BenchmarkWriter_IncrementalWrites(b *testing.B) {
	chunk := []byte("small chunk ")
	buf := new(bytes.Buffer)

	b.ResetTimer()
	b.SetBytes(int64(len(chunk)))

	for i := 0; i < b.N; i++ {
		writer, _ := NewWriter(buf)
		writer.Write(chunk)
		writer.Close()
		buf.Reset()
	}
}

// BenchmarkWriter_Throughput benchmarks streaming throughput with io.Copy.
func BenchmarkWriter_Throughput(b *testing.B) {
	// 10MB of varied data
	data := bytes.Repeat([]byte("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ!@#$%^&*()"), 100000)

	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		buf := new(bytes.Buffer)
		writer, _ := NewWriter(buf)
		source := bytes.NewReader(data)
		io.Copy(writer, source)
		writer.Close()
	}
}
