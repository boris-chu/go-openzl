package purgo

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

// Tests for NewReader

func TestNewReader_Create(t *testing.T) {
	data := []byte("test data")
	compressed := createCompressed(t, data)

	reader, err := NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	if reader == nil {
		t.Fatal("NewReader returned nil reader")
	}

	// Verify reader is not initialized yet
	if reader.initialized {
		t.Error("reader should not be initialized on creation")
	}
}

// Tests for Read()

func TestReader_ReadSmallData(t *testing.T) {
	original := []byte("hello world")
	compressed := createCompressed(t, original)

	reader, err := NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	// Read all data at once
	result := make([]byte, len(original))
	n, err := reader.Read(result)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if n != len(original) {
		t.Errorf("Read returned %d bytes, want %d", n, len(original))
	}

	if !bytes.Equal(result, original) {
		t.Errorf("Read data mismatch:\nwant: %q\ngot:  %q", original, result)
	}

	// Next read should return EOF
	n, err = reader.Read(result)
	if err != io.EOF {
		t.Errorf("second Read should return EOF, got: %v", err)
	}
	if n != 0 {
		t.Errorf("second Read returned %d bytes, want 0", n)
	}
}

func TestReader_ReadIncrementally(t *testing.T) {
	original := []byte("The quick brown fox jumps over the lazy dog")
	compressed := createCompressed(t, original)

	reader, err := NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	// Read in small chunks
	var result []byte
	buffer := make([]byte, 10) // 10 bytes at a time

	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			result = append(result, buffer[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}
	}

	if !bytes.Equal(result, original) {
		t.Errorf("Incremental read mismatch:\nwant: %q\ngot:  %q", original, result)
	}
}

func TestReader_ReadWithIOCopy(t *testing.T) {
	original := []byte("Test data for io.Copy")
	compressed := createCompressed(t, original)

	reader, err := NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	// Use io.Copy to read all data
	var output bytes.Buffer
	n, err := io.Copy(&output, reader)
	if err != nil {
		t.Fatalf("io.Copy failed: %v", err)
	}

	if n != int64(len(original)) {
		t.Errorf("io.Copy copied %d bytes, want %d", n, len(original))
	}

	if !bytes.Equal(output.Bytes(), original) {
		t.Errorf("io.Copy result mismatch:\nwant: %q\ngot:  %q", original, output.Bytes())
	}
}

func TestReader_ReadEmpty(t *testing.T) {
	original := []byte{}
	compressed := createCompressed(t, original)

	reader, err := NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	buffer := make([]byte, 10)
	n, err := reader.Read(buffer)
	if err != io.EOF {
		t.Errorf("Read on empty data should return EOF, got: %v", err)
	}
	if n != 0 {
		t.Errorf("Read on empty data returned %d bytes, want 0", n)
	}
}

func TestReader_ReadLargeData(t *testing.T) {
	// Create 100KB of data
	original := bytes.Repeat([]byte("OpenZL streaming test. "), 4500)
	compressed := createCompressed(t, original)

	reader, err := NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	// Read in 1KB chunks
	var result []byte
	buffer := make([]byte, 1024)

	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			result = append(result, buffer[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}
	}

	if !bytes.Equal(result, original) {
		t.Errorf("Large data read mismatch: got %d bytes, want %d bytes", len(result), len(original))
	}
}

func TestReader_MultipleEOFReads(t *testing.T) {
	original := []byte("test")
	compressed := createCompressed(t, original)

	reader, err := NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	// Read all data
	result := make([]byte, len(original))
	_, err = reader.Read(result)
	if err != nil {
		t.Fatalf("First Read failed: %v", err)
	}

	// Multiple EOF reads should all return EOF
	buffer := make([]byte, 10)
	for i := 0; i < 3; i++ {
		n, err := reader.Read(buffer)
		if err != io.EOF {
			t.Errorf("Read #%d after EOF should return EOF, got: %v", i+2, err)
		}
		if n != 0 {
			t.Errorf("Read #%d after EOF returned %d bytes, want 0", i+2, n)
		}
	}
}

// Error handling tests

func TestReader_EmptyInput(t *testing.T) {
	reader, err := NewReader(bytes.NewReader([]byte{}))
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	buffer := make([]byte, 10)
	_, err = reader.Read(buffer)
	if err == nil {
		t.Error("Read on empty input should fail")
	}
	if !strings.Contains(err.Error(), "empty input") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestReader_InvalidData(t *testing.T) {
	// Random invalid data
	invalid := []byte{0x00, 0x01, 0x02, 0x03, 0x04}
	reader, err := NewReader(bytes.NewReader(invalid))
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	buffer := make([]byte, 10)
	_, err = reader.Read(buffer)
	if err == nil {
		t.Error("Read on invalid data should fail")
	}
}

func TestReader_ReadAfterError(t *testing.T) {
	// Invalid data that will cause parse error
	invalid := []byte{0x00, 0x01, 0x02}
	reader, err := NewReader(bytes.NewReader(invalid))
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	buffer := make([]byte, 10)

	// First read should fail
	_, err = reader.Read(buffer)
	if err == nil {
		t.Fatal("first Read should fail on invalid data")
	}

	// Second read should also return an error (may be same or different)
	_, err2 := reader.Read(buffer)
	if err2 == nil {
		t.Error("second Read should also return error")
	}
	// Note: The second error might differ from the first (e.g., returning stored error
	// vs EOF from buffer), but both should indicate failure
}

// Tests for Close()

func TestReader_Close(t *testing.T) {
	original := []byte("test data")
	compressed := createCompressed(t, original)

	reader, err := NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	// Close should succeed (bytes.Reader doesn't implement Close, so no-op)
	err = reader.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestReader_CloseWithCloser(t *testing.T) {
	original := []byte("test data")
	compressed := createCompressed(t, original)

	// Create a ReadCloser
	rc := io.NopCloser(bytes.NewReader(compressed))

	reader, err := NewReader(rc)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	// Close should succeed
	err = reader.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

// Integration tests

func TestReader_IntegrationWithTypedAPI(t *testing.T) {
	// Create some int64 data
	original := []int64{1, 2, 3, 4, 5}
	compressed := createCompressedInt64(t, original)

	// First, verify typed API works
	result1, err := DecompressInt64(compressed)
	if err != nil {
		t.Fatalf("DecompressInt64 failed: %v", err)
	}

	// Now use streaming API to get raw bytes
	reader, err := NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	var rawBytes []byte
	buffer := make([]byte, 1024)
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			rawBytes = append(rawBytes, buffer[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}
	}

	// Verify byte count
	expectedBytes := len(original) * 8 // 8 bytes per int64
	if len(rawBytes) != expectedBytes {
		t.Errorf("got %d bytes, want %d bytes", len(rawBytes), expectedBytes)
	}

	// Convert raw bytes back to int64 for comparison
	result2 := make([]int64, len(original))
	for i := 0; i < len(original); i++ {
		offset := i * 8
		result2[i] = int64(rawBytes[offset]) |
			int64(rawBytes[offset+1])<<8 |
			int64(rawBytes[offset+2])<<16 |
			int64(rawBytes[offset+3])<<24 |
			int64(rawBytes[offset+4])<<32 |
			int64(rawBytes[offset+5])<<40 |
			int64(rawBytes[offset+6])<<48 |
			int64(rawBytes[offset+7])<<56
	}

	// Verify results match
	for i := range original {
		if result1[i] != result2[i] {
			t.Errorf("mismatch at index %d: typed=%d, streaming=%d", i, result1[i], result2[i])
		}
	}
}

// Benchmarks

func BenchmarkReader_SmallData(b *testing.B) {
	original := []byte("hello world")
	compressed := createCompressed(&testing.T{}, original)

	b.ResetTimer()
	b.SetBytes(int64(len(original)))

	for i := 0; i < b.N; i++ {
		reader, _ := NewReader(bytes.NewReader(compressed))
		io.Copy(io.Discard, reader)
	}
}

func BenchmarkReader_LargeData(b *testing.B) {
	// 10KB of data
	original := bytes.Repeat([]byte("OpenZL benchmark test data. "), 370)
	compressed := createCompressed(&testing.T{}, original)

	b.ResetTimer()
	b.SetBytes(int64(len(original)))

	for i := 0; i < b.N; i++ {
		reader, _ := NewReader(bytes.NewReader(compressed))
		io.Copy(io.Discard, reader)
	}
}

func BenchmarkReader_IncrementalRead(b *testing.B) {
	// 10KB of data
	original := bytes.Repeat([]byte("Incremental read benchmark. "), 370)
	compressed := createCompressed(&testing.T{}, original)

	b.ResetTimer()
	b.SetBytes(int64(len(original)))

	for i := 0; i < b.N; i++ {
		reader, _ := NewReader(bytes.NewReader(compressed))
		buffer := make([]byte, 512)
		for {
			_, err := reader.Read(buffer)
			if err == io.EOF {
				break
			}
		}
	}
}
