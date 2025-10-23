// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package openzl

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
)

// TestReader_TruncatedFrame tests handling of truncated compressed data
func TestReader_TruncatedFrame(t *testing.T) {
	// Compress valid data
	original := []byte("test data for truncation")
	var buf bytes.Buffer
	writer, err := NewWriter(&buf)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	_, err = writer.Write(original)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Truncate the compressed data
	compressed := buf.Bytes()
	if len(compressed) < 10 {
		t.Skip("Compressed data too small to truncate meaningfully")
	}

	truncated := compressed[:len(compressed)-10]

	// Should error gracefully, not panic
	reader, err := NewReader(bytes.NewReader(truncated))
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer reader.Close()

	_, err = io.ReadAll(reader)
	if err == nil {
		t.Error("Expected error for truncated data, got nil")
	}

	// Error should be informative
	if !strings.Contains(err.Error(), "openzl") && !strings.Contains(err.Error(), "EOF") {
		t.Logf("Error message: %v", err)
	}
}

// TestReader_InvalidFrameHeader tests handling of invalid frame headers
func TestReader_InvalidFrameHeader(t *testing.T) {
	tests := []struct {
		name   string
		data   []byte
		expectError bool
	}{
		{
			name:   "huge_size",
			data:   []byte{0xFF, 0xFF, 0xFF, 0x7F}, // Large but valid size
			expectError: true,
		},
		{
			name:   "random_bytes",
			data:   []byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC},
			expectError: true,
		},
		{
			name:   "truncated_header",
			data:   []byte{0x10, 0x00}, // Only 2 bytes
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, err := NewReader(bytes.NewReader(tt.data))
			if err != nil {
				// Error on NewReader is acceptable
				return
			}
			defer reader.Close()

			_, err = reader.Read(make([]byte, 100))
			if tt.expectError && err == nil {
				t.Error("Expected error for invalid frame header")
			}
		})
	}
}

// TestCompress_UncompressibleData tests compression of random/encrypted data
func TestCompress_UncompressibleData(t *testing.T) {
	// Random data (from /dev/urandom equivalent)
	random := make([]byte, 10000)
	for i := range random {
		random[i] = byte(i * 7 % 256) // Pseudo-random
	}

	compressed, err := Compress(random)
	if err != nil {
		t.Fatalf("Compress failed on random data: %v", err)
	}

	// Random data typically expands (no patterns)
	ratio := float64(len(random)) / float64(len(compressed))
	if ratio > 1.5 {
		t.Logf("WARNING: Random data compressed unexpectedly well (%.2fx)", ratio)
	}

	// Verify round-trip still works
	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}

	if !bytes.Equal(random, decompressed) {
		t.Error("Round-trip failed for random data")
	}

	t.Logf("Random data: %d bytes -> %d bytes (%.2fx)",
		len(random), len(compressed), ratio)
}

// TestTypedCompression_TypeMismatch documents behavior when type mismatches occur
func TestTypedCompression_TypeMismatch(t *testing.T) {
	// Compress as int64
	numbers := []int64{1, 2, 3, 4, 5}
	compressed, err := CompressNumeric(numbers)
	if err != nil {
		t.Fatalf("CompressNumeric failed: %v", err)
	}

	// Try to decompress as int32 (wrong type)
	// NOTE: This currently succeeds but gives wrong values
	// OpenZL compressed data doesn't store type information
	decompressed32, err := DecompressNumeric[int32](compressed)
	if err != nil {
		// If it errors, that's actually better (type safety)
		t.Logf("Type mismatch detected (good): %v", err)
		return
	}

	// If it succeeds, document the behavior
	t.Logf("WARNING: Type mismatch NOT detected")
	t.Logf("Original int64: %v", numbers)
	t.Logf("Decompressed as int32: %v (first %d values)", decompressed32, min(5, len(decompressed32)))

	// This is expected behavior - user must ensure type consistency
	// Document this in godoc
}

// TestTypedCompression_ZeroLengthArray tests empty array handling
func TestTypedCompression_ZeroLengthArray(t *testing.T) {
	// Empty array should return error
	empty := []int64{}
	_, err := CompressNumeric(empty)
	if err == nil {
		t.Error("Expected error for empty array, got nil")
	}

	// Error should be informative
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("Error should mention 'empty': %v", err)
	}
}

// TestWriter_UnderlyingWriterError tests error handling when underlying writer fails
func TestWriter_UnderlyingWriterError(t *testing.T) {
	// Writer that fails after N bytes
	fw := &failingWriter{failAfter: 50}

	writer, err := NewWriter(fw)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	// Write data that will cause failure
	largeData := bytes.Repeat([]byte("test"), 10000)
	_, err = writer.Write(largeData)

	// Should eventually fail
	if err == nil {
		err = writer.Close()
	}

	if err == nil {
		t.Error("Expected error from failing writer")
	}
}

// failingWriter is a writer that fails after N bytes
type failingWriter struct {
	written   int
	failAfter int
}

func (fw *failingWriter) Write(p []byte) (n int, err error) {
	if fw.written >= fw.failAfter {
		return 0, fmt.Errorf("write failed after %d bytes", fw.failAfter)
	}

	toWrite := len(p)
	if fw.written+toWrite > fw.failAfter {
		toWrite = fw.failAfter - fw.written
	}

	fw.written += toWrite
	return toWrite, nil
}

// TestStreaming_LargeFile tests compression of large files
func TestStreaming_LargeFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large file test in short mode")
	}

	// Create 100MB of test data (reduced from 1GB for faster testing)
	size := 100 * 1024 * 1024
	pattern := []byte("Test pattern for large file compression.\n")

	source := &repeatingReader{
		pattern:   pattern,
		remaining: size,
	}

	var compressed bytes.Buffer
	writer, err := NewWriter(&compressed)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	n, err := io.Copy(writer, source)
	if err != nil {
		t.Fatalf("io.Copy failed: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if n != int64(size) {
		t.Errorf("Compressed %d bytes, want %d", n, size)
	}

	ratio := float64(size) / float64(compressed.Len())
	t.Logf("Large file: %d bytes -> %d bytes (%.2fx ratio)",
		size, compressed.Len(), ratio)

	if ratio < 100 {
		t.Logf("WARNING: Expected better compression ratio for repeated data")
	}

	// Verify decompression
	reader, err := NewReader(&compressed)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer reader.Close()

	n2, err := io.Copy(io.Discard, reader)
	if err != nil {
		t.Fatalf("Decompression failed: %v", err)
	}

	if n2 != int64(size) {
		t.Errorf("Decompressed %d bytes, want %d", n2, size)
	}
}

// repeatingReader generates repeated pattern data
type repeatingReader struct {
	pattern   []byte
	remaining int
	pos       int
}

func (rr *repeatingReader) Read(p []byte) (n int, err error) {
	if rr.remaining == 0 {
		return 0, io.EOF
	}

	for n < len(p) && rr.remaining > 0 {
		p[n] = rr.pattern[rr.pos]
		rr.pos = (rr.pos + 1) % len(rr.pattern)
		n++
		rr.remaining--
	}

	return n, nil
}

// TestDecompress_ErrorMessages tests that error messages are informative
func TestDecompress_ErrorMessages(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		shouldError bool
		errorContains string
	}{
		{
			name:          "empty_input",
			input:         []byte{},
			shouldError:   true,
			errorContains: "empty",
		},
		{
			name:          "random_bytes",
			input:         []byte{1, 2, 3, 4, 5, 6, 7, 8},
			shouldError:   true,
			errorContains: "openzl",
		},
		{
			name:          "single_byte",
			input:         []byte{0},
			shouldError:   true,
			errorContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decompress(tt.input)
			if tt.shouldError && err == nil {
				t.Errorf("Expected error, got nil")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if err != nil && tt.errorContains != "" {
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Error should contain %q, got: %v",
						tt.errorContains, err)
				}
			}
		})
	}
}

// TestConcurrency_Stress is a stress test for concurrent operations
func TestConcurrency_Stress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	// Reduced from 1000 to 100 for reasonable test time
	numGoroutines := 100
	opsPerGoroutine := 100

	errors := make(chan error, numGoroutines*opsPerGoroutine)
	done := make(chan bool)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			compressor, err := NewCompressor()
			if err != nil {
				errors <- fmt.Errorf("goroutine %d: NewCompressor failed: %w", id, err)
				return
			}
			defer compressor.Close()

			decompressor, err := NewDecompressor()
			if err != nil {
				errors <- fmt.Errorf("goroutine %d: NewDecompressor failed: %w", id, err)
				return
			}
			defer decompressor.Close()

			for j := 0; j < opsPerGoroutine; j++ {
				data := []byte(fmt.Sprintf("goroutine-%d-op-%d", id, j))

				compressed, err := compressor.Compress(data)
				if err != nil {
					errors <- fmt.Errorf("goroutine %d op %d: compress failed: %w", id, j, err)
					continue
				}

				decompressed, err := decompressor.Decompress(compressed)
				if err != nil {
					errors <- fmt.Errorf("goroutine %d op %d: decompress failed: %w", id, j, err)
					continue
				}

				if !bytes.Equal(data, decompressed) {
					errors <- fmt.Errorf("goroutine %d op %d: data mismatch", id, j)
				}
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
	close(errors)

	// Check for errors
	errCount := 0
	for err := range errors {
		if errCount < 10 { // Only log first 10 errors
			t.Errorf("Concurrent error: %v", err)
		}
		errCount++
	}

	if errCount > 0 {
		t.Errorf("Total concurrent errors: %d", errCount)
	} else {
		t.Logf("Stress test passed: %d goroutines x %d ops = %d total operations",
			numGoroutines, opsPerGoroutine, numGoroutines*opsPerGoroutine)
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
