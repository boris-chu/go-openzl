// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package openzl

import (
	"bytes"
	"sync"
	"testing"
)

func TestCompressor(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"simple text", []byte("hello world")},
		{"repeated data", bytes.Repeat([]byte("a"), 400)},
		{"binary data", []byte{0x00, 0xFF, 0xAA, 0x55, 0x42}},
		{"empty string", []byte("")},
		{"unicode", []byte("Hello, 世界! 🌍")},
		{"large data", bytes.Repeat([]byte("The quick brown fox jumps over the lazy dog. "), 1000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip empty test case for compression
			if len(tt.data) == 0 {
				compressor, err := NewCompressor()
				if err != nil {
					t.Fatalf("NewCompressor() failed: %v", err)
				}
				defer compressor.Close()

				_, err = compressor.Compress(tt.data)
				if err != ErrEmptyInput {
					t.Errorf("expected ErrEmptyInput for empty data, got: %v", err)
				}
				return
			}

			// Create compressor
			compressor, err := NewCompressor()
			if err != nil {
				t.Fatalf("NewCompressor() failed: %v", err)
			}
			defer compressor.Close()

			// Compress
			compressed, err := compressor.Compress(tt.data)
			if err != nil {
				t.Fatalf("Compress() failed: %v", err)
			}

			// Verify compressed data is not empty
			if len(compressed) == 0 {
				t.Fatal("compressed data is empty")
			}

			// Create decompressor
			decompressor, err := NewDecompressor()
			if err != nil {
				t.Fatalf("NewDecompressor() failed: %v", err)
			}
			defer decompressor.Close()

			// Decompress
			decompressed, err := decompressor.Decompress(compressed)
			if err != nil {
				t.Fatalf("Decompress() failed: %v", err)
			}

			// Verify round-trip
			if !bytes.Equal(tt.data, decompressed) {
				t.Errorf("round-trip failed:\noriginal: %v\ndecompressed: %v", tt.data, decompressed)
			}

			// Log compression ratio for informational purposes
			ratio := float64(len(tt.data)) / float64(len(compressed))
			t.Logf("Original: %d bytes, Compressed: %d bytes, Ratio: %.2f", len(tt.data), len(compressed), ratio)
		})
	}
}

func TestCompressorReuse(t *testing.T) {
	compressor, err := NewCompressor()
	if err != nil {
		t.Fatalf("NewCompressor() failed: %v", err)
	}
	defer compressor.Close()

	decompressor, err := NewDecompressor()
	if err != nil {
		t.Fatalf("NewDecompressor() failed: %v", err)
	}
	defer decompressor.Close()

	// Test multiple compressions with the same context
	testData := [][]byte{
		[]byte("first compression"),
		[]byte("second compression with different data"),
		bytes.Repeat([]byte("repeated "), 100),
		[]byte("final compression"),
	}

	for i, data := range testData {
		// Compress
		compressed, err := compressor.Compress(data)
		if err != nil {
			t.Fatalf("compression %d failed: %v", i, err)
		}

		// Decompress
		decompressed, err := decompressor.Decompress(compressed)
		if err != nil {
			t.Fatalf("decompression %d failed: %v", i, err)
		}

		// Verify
		if !bytes.Equal(data, decompressed) {
			t.Errorf("round-trip %d failed", i)
		}
	}
}

func TestCompressorConcurrent(t *testing.T) {
	compressor, err := NewCompressor()
	if err != nil {
		t.Fatalf("NewCompressor() failed: %v", err)
	}
	defer compressor.Close()

	decompressor, err := NewDecompressor()
	if err != nil {
		t.Fatalf("NewDecompressor() failed: %v", err)
	}
	defer decompressor.Close()

	// Number of concurrent goroutines
	const numGoroutines = 10
	const opsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Track errors
	errChan := make(chan error, numGoroutines*opsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < opsPerGoroutine; j++ {
				// Create unique data for each operation (ensure at least 1 repeat)
				data := bytes.Repeat([]byte("goroutine "), id*100+j+1)

				// Compress
				compressed, err := compressor.Compress(data)
				if err != nil {
					errChan <- err
					return
				}

				// Decompress
				decompressed, err := decompressor.Decompress(compressed)
				if err != nil {
					errChan <- err
					return
				}

				// Verify
				if !bytes.Equal(data, decompressed) {
					errChan <- err
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Check for any errors
	for err := range errChan {
		t.Errorf("concurrent operation failed: %v", err)
	}
}

func TestCompressorClose(t *testing.T) {
	compressor, err := NewCompressor()
	if err != nil {
		t.Fatalf("NewCompressor() failed: %v", err)
	}

	// Test that Close can be called multiple times safely
	if err := compressor.Close(); err != nil {
		t.Errorf("first Close() failed: %v", err)
	}

	if err := compressor.Close(); err != nil {
		t.Errorf("second Close() failed: %v", err)
	}
}

func TestDecompressorClose(t *testing.T) {
	decompressor, err := NewDecompressor()
	if err != nil {
		t.Fatalf("NewDecompressor() failed: %v", err)
	}

	// Test that Close can be called multiple times safely
	if err := decompressor.Close(); err != nil {
		t.Errorf("first Close() failed: %v", err)
	}

	if err := decompressor.Close(); err != nil {
		t.Errorf("second Close() failed: %v", err)
	}
}

func TestDecompressorEmpty(t *testing.T) {
	decompressor, err := NewDecompressor()
	if err != nil {
		t.Fatalf("NewDecompressor() failed: %v", err)
	}
	defer decompressor.Close()

	_, err = decompressor.Decompress([]byte{})
	if err != ErrEmptyInput {
		t.Errorf("expected ErrEmptyInput for empty data, got: %v", err)
	}
}

func TestDecompressorCorrupted(t *testing.T) {
	decompressor, err := NewDecompressor()
	if err != nil {
		t.Fatalf("NewDecompressor() failed: %v", err)
	}
	defer decompressor.Close()

	// Try to decompress random garbage data
	corruptedData := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE, 0xBA, 0xBE}
	_, err = decompressor.Decompress(corruptedData)
	if err == nil {
		t.Error("expected error when decompressing corrupted data, got nil")
	}
}
