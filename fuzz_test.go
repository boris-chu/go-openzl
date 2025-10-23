// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package openzl

import (
	"bytes"
	"testing"
)

// FuzzCompress tests the one-shot Compress function with random inputs
func FuzzCompress(f *testing.F) {
	// Seed corpus with interesting inputs
	f.Add([]byte("Hello, World!"))
	f.Add([]byte(""))
	f.Add([]byte{0, 1, 2, 3, 4, 5})
	f.Add(bytes.Repeat([]byte("A"), 1000))
	f.Add(bytes.Repeat([]byte{0}, 10000))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Skip empty data (expected to return error)
		if len(data) == 0 {
			return
		}

		// Compress should not panic
		compressed, err := Compress(data)
		if err != nil {
			// Compression can fail for valid reasons (e.g., data too large)
			return
		}

		// Decompression should not panic and should return original data
		decompressed, err := Decompress(compressed)
		if err != nil {
			t.Fatalf("Decompress failed after successful compress: %v", err)
		}

		if !bytes.Equal(data, decompressed) {
			t.Fatalf("Round-trip failed: len(original)=%d, len(decompressed)=%d",
				len(data), len(decompressed))
		}
	})
}

// FuzzCompressor tests the Compressor type with random inputs
func FuzzCompressor(f *testing.F) {
	// Seed corpus
	f.Add([]byte("Test data"))
	f.Add(bytes.Repeat([]byte("X"), 5000))
	f.Add([]byte{255, 254, 253, 252})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 {
			return
		}

		compressor, err := NewCompressor()
		if err != nil {
			t.Fatalf("NewCompressor failed: %v", err)
		}
		defer compressor.Close()

		compressed, err := compressor.Compress(data)
		if err != nil {
			return
		}

		decompressor, err := NewDecompressor()
		if err != nil {
			t.Fatalf("NewDecompressor failed: %v", err)
		}
		defer decompressor.Close()

		decompressed, err := decompressor.Decompress(compressed)
		if err != nil {
			t.Fatalf("Decompress failed: %v", err)
		}

		if !bytes.Equal(data, decompressed) {
			t.Fatalf("Round-trip failed")
		}
	})
}

// FuzzNumericInt64 tests typed compression with int64 slices
func FuzzNumericInt64(f *testing.F) {
	// Seed with interesting patterns
	f.Add([]byte{1, 0, 0, 0, 0, 0, 0, 0}) // Single int64
	f.Add(bytes.Repeat([]byte{1, 0, 0, 0, 0, 0, 0, 0}, 10)) // Repeated value

	f.Fuzz(func(t *testing.T, data []byte) {
		// Must be multiple of 8 bytes for int64
		if len(data) == 0 || len(data)%8 != 0 {
			return
		}

		// Convert bytes to []int64
		numInts := len(data) / 8
		numbers := make([]int64, numInts)
		for i := 0; i < numInts; i++ {
			offset := i * 8
			numbers[i] = int64(data[offset]) |
				int64(data[offset+1])<<8 |
				int64(data[offset+2])<<16 |
				int64(data[offset+3])<<24 |
				int64(data[offset+4])<<32 |
				int64(data[offset+5])<<40 |
				int64(data[offset+6])<<48 |
				int64(data[offset+7])<<56
		}

		// Compress
		compressed, err := CompressNumeric(numbers)
		if err != nil {
			return
		}

		// Decompress
		decompressed, err := DecompressNumeric[int64](compressed)
		if err != nil {
			t.Fatalf("DecompressNumeric failed: %v", err)
		}

		// Verify
		if len(decompressed) != len(numbers) {
			t.Fatalf("Length mismatch: got %d, want %d", len(decompressed), len(numbers))
		}

		for i := range numbers {
			if decompressed[i] != numbers[i] {
				t.Fatalf("Data mismatch at index %d: got %d, want %d",
					i, decompressed[i], numbers[i])
			}
		}
	})
}

// FuzzWriter tests the streaming Writer with random inputs
func FuzzWriter(f *testing.F) {
	// Seed corpus
	f.Add([]byte("Streaming data"))
	f.Add(bytes.Repeat([]byte("S"), 100000)) // Large data

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 {
			return
		}

		// Compress with Writer
		var compressed bytes.Buffer
		writer, err := NewWriter(&compressed)
		if err != nil {
			t.Fatalf("NewWriter failed: %v", err)
		}

		_, err = writer.Write(data)
		if err != nil {
			writer.Close()
			return
		}

		if err := writer.Close(); err != nil {
			t.Fatalf("Writer.Close failed: %v", err)
		}

		// Decompress with Reader
		reader, err := NewReader(&compressed)
		if err != nil {
			t.Fatalf("NewReader failed: %v", err)
		}
		defer reader.Close()

		decompressed := make([]byte, len(data))
		n, err := reader.Read(decompressed)
		if err != nil && n != len(data) {
			t.Fatalf("Read failed: %v (read %d bytes)", err, n)
		}

		if !bytes.Equal(data, decompressed[:n]) {
			t.Fatalf("Streaming round-trip failed")
		}
	})
}

// FuzzDecompress tests decompression with random (potentially corrupted) inputs
func FuzzDecompress(f *testing.F) {
	// Seed with some valid compressed data
	validCompressed, _ := Compress([]byte("Hello"))
	f.Add(validCompressed)
	f.Add([]byte{0, 0, 0, 0}) // Minimal invalid data
	f.Add(bytes.Repeat([]byte{255}, 100))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Decompress should not panic, even on invalid input
		// It should return an error instead
		_, err := Decompress(data)
		// We expect most random data to fail decompression
		// The important thing is that it doesn't panic
		_ = err
	})
}
