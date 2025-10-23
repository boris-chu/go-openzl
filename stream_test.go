// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package openzl

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestWriterReader_Simple(t *testing.T) {
	// Compress data using Writer
	var buf bytes.Buffer
	writer, err := NewWriter(&buf)
	if err != nil {
		t.Fatalf("NewWriter() failed: %v", err)
	}

	original := []byte("Hello, OpenZL streaming!")
	n, err := writer.Write(original)
	if err != nil {
		t.Fatalf("Write() failed: %v", err)
	}
	if n != len(original) {
		t.Errorf("Write() wrote %d bytes, want %d", n, len(original))
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	// Decompress using Reader
	reader, err := NewReader(&buf)
	if err != nil {
		t.Fatalf("NewReader() failed: %v", err)
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() failed: %v", err)
	}

	if !bytes.Equal(decompressed, original) {
		t.Errorf("Decompressed data mismatch:\ngot:  %q\nwant: %q", decompressed, original)
	}
}

func TestWriterReader_LargeData(t *testing.T) {
	// Create large test data (multiple frames)
	original := bytes.Repeat([]byte("0123456789"), 10000) // 100KB

	var buf bytes.Buffer
	writer, err := NewWriter(&buf)
	if err != nil {
		t.Fatalf("NewWriter() failed: %v", err)
	}

	n, err := writer.Write(original)
	if err != nil {
		t.Fatalf("Write() failed: %v", err)
	}
	if n != len(original) {
		t.Errorf("Write() wrote %d bytes, want %d", n, len(original))
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	t.Logf("Compressed %d bytes to %d bytes (%.2fx ratio)",
		len(original), buf.Len(), float64(len(original))/float64(buf.Len()))

	// Decompress
	reader, err := NewReader(&buf)
	if err != nil {
		t.Fatalf("NewReader() failed: %v", err)
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() failed: %v", err)
	}

	if !bytes.Equal(decompressed, original) {
		t.Errorf("Decompressed data length mismatch: got %d, want %d",
			len(decompressed), len(original))
	}
}

func TestWriterReader_MultipleWrites(t *testing.T) {
	var buf bytes.Buffer
	writer, err := NewWriter(&buf)
	if err != nil {
		t.Fatalf("NewWriter() failed: %v", err)
	}

	// Write multiple chunks
	chunks := []string{
		"First chunk",
		"Second chunk",
		"Third chunk",
	}

	var original bytes.Buffer
	for _, chunk := range chunks {
		n, err := writer.Write([]byte(chunk))
		if err != nil {
			t.Fatalf("Write() failed: %v", err)
		}
		if n != len(chunk) {
			t.Errorf("Write() wrote %d bytes, want %d", n, len(chunk))
		}
		original.WriteString(chunk)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	// Decompress
	reader, err := NewReader(&buf)
	if err != nil {
		t.Fatalf("NewReader() failed: %v", err)
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() failed: %v", err)
	}

	if !bytes.Equal(decompressed, original.Bytes()) {
		t.Errorf("Decompressed data mismatch:\ngot:  %q\nwant: %q",
			decompressed, original.Bytes())
	}
}

func TestWriterReader_IoCopy(t *testing.T) {
	original := []byte(strings.Repeat("Hello, streaming world! ", 1000))

	// Compress using io.Copy
	var compressed bytes.Buffer
	writer, err := NewWriter(&compressed)
	if err != nil {
		t.Fatalf("NewWriter() failed: %v", err)
	}

	source := bytes.NewReader(original)
	n, err := io.Copy(writer, source)
	if err != nil {
		t.Fatalf("io.Copy() to writer failed: %v", err)
	}
	if n != int64(len(original)) {
		t.Errorf("io.Copy() copied %d bytes, want %d", n, len(original))
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() failed: %v", err)
	}

	t.Logf("Compressed %d bytes to %d bytes using io.Copy (%.2fx ratio)",
		len(original), compressed.Len(), float64(len(original))/float64(compressed.Len()))

	// Decompress using io.Copy
	reader, err := NewReader(&compressed)
	if err != nil {
		t.Fatalf("NewReader() failed: %v", err)
	}
	defer reader.Close()

	var decompressed bytes.Buffer
	n, err = io.Copy(&decompressed, reader)
	if err != nil {
		t.Fatalf("io.Copy() from reader failed: %v", err)
	}
	if n != int64(len(original)) {
		t.Errorf("io.Copy() copied %d bytes, want %d", n, len(original))
	}

	if !bytes.Equal(decompressed.Bytes(), original) {
		t.Errorf("Decompressed data length mismatch: got %d, want %d",
			decompressed.Len(), len(original))
	}
}

func TestWriter_EmptyWrite(t *testing.T) {
	var buf bytes.Buffer
	writer, err := NewWriter(&buf)
	if err != nil {
		t.Fatalf("NewWriter() failed: %v", err)
	}

	// Write empty data
	n, err := writer.Write([]byte{})
	if err != nil {
		t.Fatalf("Write() failed: %v", err)
	}
	if n != 0 {
		t.Errorf("Write() wrote %d bytes, want 0", n)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	// Should only have end-of-stream marker
	if buf.Len() != 4 {
		t.Errorf("Compressed size = %d, want 4 (end marker only)", buf.Len())
	}
}

func TestWriter_FrameSize(t *testing.T) {
	original := bytes.Repeat([]byte("test"), 1000)

	tests := []struct {
		name      string
		frameSize int
		wantError bool
	}{
		{"Default", DefaultFrameSize, false},
		{"Small (4KB)", 4 * 1024, false},
		{"Large (1MB)", 1024 * 1024, false},
		{"Too small", 1024, true},
		{"Too large", 2 * 1024 * 1024, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			writer, err := NewWriter(&buf, WithFrameSize(tt.frameSize))
			if tt.wantError {
				if err == nil {
					t.Errorf("NewWriter() succeeded, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("NewWriter() failed: %v", err)
			}
			defer writer.Close()

			_, err = writer.Write(original)
			if err != nil {
				t.Fatalf("Write() failed: %v", err)
			}

			if err := writer.Close(); err != nil {
				t.Fatalf("Close() failed: %v", err)
			}

			// Verify decompression works
			reader, err := NewReader(&buf)
			if err != nil {
				t.Fatalf("NewReader() failed: %v", err)
			}
			defer reader.Close()

			decompressed, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("ReadAll() failed: %v", err)
			}

			if !bytes.Equal(decompressed, original) {
				t.Errorf("Decompressed data mismatch")
			}
		})
	}
}

func TestWriter_ClosedWriter(t *testing.T) {
	var buf bytes.Buffer
	writer, err := NewWriter(&buf)
	if err != nil {
		t.Fatalf("NewWriter() failed: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	// Writing to closed writer should error
	_, err = writer.Write([]byte("test"))
	if err == nil {
		t.Errorf("Write() to closed writer succeeded, want error")
	}

	// Closing again should be safe
	if err := writer.Close(); err != nil {
		t.Errorf("second Close() failed: %v", err)
	}
}

func TestReader_ClosedReader(t *testing.T) {
	// Create some compressed data
	var buf bytes.Buffer
	writer, _ := NewWriter(&buf)
	writer.Write([]byte("test"))
	writer.Close()

	// Create reader and close it
	reader, err := NewReader(&buf)
	if err != nil {
		t.Fatalf("NewReader() failed: %v", err)
	}

	if err := reader.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	// Reading from closed reader should error
	var p [10]byte
	_, err = reader.Read(p[:])
	if err == nil {
		t.Errorf("Read() from closed reader succeeded, want error")
	}

	// Closing again should be safe
	if err := reader.Close(); err != nil {
		t.Errorf("second Close() failed: %v", err)
	}
}

func TestWriter_Reset(t *testing.T) {
	// Write to first buffer
	var buf1 bytes.Buffer
	writer, err := NewWriter(&buf1)
	if err != nil {
		t.Fatalf("NewWriter() failed: %v", err)
	}

	data1 := []byte("First data")
	writer.Write(data1)
	writer.Close()

	// Reset and write to second buffer
	var buf2 bytes.Buffer
	if err := writer.Reset(&buf2); err != nil {
		t.Fatalf("Reset() failed: %v", err)
	}

	data2 := []byte("Second data")
	writer.Write(data2)
	writer.Close()

	// Verify both buffers
	reader1, _ := NewReader(&buf1)
	result1, _ := io.ReadAll(reader1)
	reader1.Close()

	if !bytes.Equal(result1, data1) {
		t.Errorf("First buffer data mismatch")
	}

	reader2, _ := NewReader(&buf2)
	result2, _ := io.ReadAll(reader2)
	reader2.Close()

	if !bytes.Equal(result2, data2) {
		t.Errorf("Second buffer data mismatch")
	}
}

func TestReader_Reset(t *testing.T) {
	// Create two compressed buffers
	var buf1, buf2 bytes.Buffer

	writer1, _ := NewWriter(&buf1)
	data1 := []byte("First data")
	writer1.Write(data1)
	writer1.Close()

	writer2, _ := NewWriter(&buf2)
	data2 := []byte("Second data")
	writer2.Write(data2)
	writer2.Close()

	// Read first buffer
	reader, err := NewReader(&buf1)
	if err != nil {
		t.Fatalf("NewReader() failed: %v", err)
	}

	result1, _ := io.ReadAll(reader)
	reader.Close()

	if !bytes.Equal(result1, data1) {
		t.Errorf("First buffer data mismatch")
	}

	// Reset and read second buffer
	if err := reader.Reset(&buf2); err != nil {
		t.Fatalf("Reset() failed: %v", err)
	}

	result2, _ := io.ReadAll(reader)
	reader.Close()

	if !bytes.Equal(result2, data2) {
		t.Errorf("Second buffer data mismatch")
	}
}

func TestWriterReader_NilWriter(t *testing.T) {
	_, err := NewWriter(nil)
	if err == nil {
		t.Errorf("NewWriter(nil) succeeded, want error")
	}
}

func TestWriterReader_NilReader(t *testing.T) {
	_, err := NewReader(nil)
	if err == nil {
		t.Errorf("NewReader(nil) succeeded, want error")
	}
}
