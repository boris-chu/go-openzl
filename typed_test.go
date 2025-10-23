// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package openzl

import (
	"testing"
)

func TestCompressNumeric_Int64(t *testing.T) {
	tests := []struct {
		name string
		data []int64
	}{
		{"small sorted", []int64{1, 2, 3, 4, 5}},
		{"small unsorted", []int64{5, 2, 8, 1, 9}},
		{"sorted sequence", []int64{1, 2, 3, 4, 5, 100, 101, 102, 103, 104}},
		{"repeated values", []int64{1, 1, 1, 2, 2, 2, 3, 3, 3}},
		{"large range", make([]int64, 100)},
	}

	// Initialize large range test
	for i := range tests[4].data {
		tests[4].data[i] = int64(i)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Compress
			compressed, err := CompressNumeric(tt.data)
			if err != nil {
				t.Fatalf("CompressNumeric() failed: %v", err)
			}

			// Decompress
			decompressed, err := DecompressNumeric[int64](compressed)
			if err != nil {
				t.Fatalf("DecompressNumeric() failed: %v", err)
			}

			// Verify round-trip
			if len(decompressed) != len(tt.data) {
				t.Fatalf("length mismatch: got %d, want %d", len(decompressed), len(tt.data))
			}

			for i := range tt.data {
				if decompressed[i] != tt.data[i] {
					t.Errorf("mismatch at index %d: got %d, want %d", i, decompressed[i], tt.data[i])
				}
			}

			// Log compression ratio
			uncompressedSize := len(tt.data) * 8 // int64 = 8 bytes
			ratio := float64(uncompressedSize) / float64(len(compressed))
			t.Logf("Compressed %d int64s: %d bytes -> %d bytes (%.2fx ratio)",
				len(tt.data), uncompressedSize, len(compressed), ratio)
		})
	}
}

func TestCompressNumeric_AllTypes(t *testing.T) {
	t.Run("int8", func(t *testing.T) {
		data := []int8{1, 2, 3, 4, 5}
		compressed, err := CompressNumeric(data)
		if err != nil {
			t.Fatal(err)
		}
		decompressed, err := DecompressNumeric[int8](compressed)
		if err != nil {
			t.Fatal(err)
		}
		for i := range data {
			if decompressed[i] != data[i] {
				t.Errorf("mismatch at %d", i)
			}
		}
	})

	t.Run("uint32", func(t *testing.T) {
		data := []uint32{100, 200, 300, 400, 500}
		compressed, err := CompressNumeric(data)
		if err != nil {
			t.Fatal(err)
		}
		decompressed, err := DecompressNumeric[uint32](compressed)
		if err != nil {
			t.Fatal(err)
		}
		for i := range data {
			if decompressed[i] != data[i] {
				t.Errorf("mismatch at %d", i)
			}
		}
	})

	t.Run("float64", func(t *testing.T) {
		data := []float64{1.1, 2.2, 3.3, 4.4, 5.5}
		compressed, err := CompressNumeric(data)
		if err != nil {
			t.Fatal(err)
		}
		decompressed, err := DecompressNumeric[float64](compressed)
		if err != nil {
			t.Fatal(err)
		}
		for i := range data {
			if decompressed[i] != data[i] {
				t.Errorf("mismatch at %d", i)
			}
		}
	})
}

func TestCompressNumeric_Empty(t *testing.T) {
	var data []int64
	_, err := CompressNumeric(data)
	if err != ErrEmptyInput {
		t.Errorf("expected ErrEmptyInput, got: %v", err)
	}
}

func TestDecompressNumeric_Empty(t *testing.T) {
	_, err := DecompressNumeric[int64]([]byte{})
	if err != ErrEmptyInput {
		t.Errorf("expected ErrEmptyInput, got: %v", err)
	}
}

func TestCompressorNumeric(t *testing.T) {
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

	// Test multiple compressions with same context
	testData := [][]int64{
		{1, 2, 3, 4, 5},
		{10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
		{100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110},
	}

	for i, data := range testData {
		// Compress
		compressed, err := CompressorCompressNumeric(compressor, data)
		if err != nil {
			t.Fatalf("compression %d failed: %v", i, err)
		}

		// Decompress
		decompressed, err := DecompressorDecompressNumeric[int64](decompressor, compressed)
		if err != nil {
			t.Fatalf("decompression %d failed: %v", i, err)
		}

		// Verify
		for j := range data {
			if decompressed[j] != data[j] {
				t.Errorf("round-trip %d failed at index %d", i, j)
			}
		}
	}
}

func TestCompressorNumeric_Concurrent(t *testing.T) {
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

	// Test concurrent operations
	const numGoroutines = 10
	const opsPerGoroutine = 50

	errChan := make(chan error, numGoroutines*opsPerGoroutine)
	doneChan := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { doneChan <- true }()

			for j := 0; j < opsPerGoroutine; j++ {
				// Create unique data for each operation
				data := make([]int64, (id*100+j)+1) // +1 to avoid empty
				for k := range data {
					data[k] = int64(k + id*1000)
				}

				// Compress
				compressed, err := CompressorCompressNumeric(compressor, data)
				if err != nil {
					errChan <- err
					return
				}

				// Decompress
				decompressed, err := DecompressorDecompressNumeric[int64](decompressor, compressed)
				if err != nil {
					errChan <- err
					return
				}

				// Verify
				for k := range data {
					if decompressed[k] != data[k] {
						errChan <- err
						return
					}
				}
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-doneChan
	}
	close(errChan)

	// Check for errors
	for err := range errChan {
		t.Errorf("concurrent operation failed: %v", err)
	}
}

// Compare typed vs untyped compression
func TestTypedVsUntypedCompression(t *testing.T) {
	// Create sorted integer sequence (best case for typed compression)
	data := make([]int64, 1000)
	for i := range data {
		data[i] = int64(i)
	}

	// Typed compression
	typedCompressed, err := CompressNumeric(data)
	if err != nil {
		t.Fatalf("typed compression failed: %v", err)
	}

	// Untyped compression (convert to bytes)
	untypedData := make([]byte, len(data)*8)
	for i, v := range data {
		offset := i * 8
		untypedData[offset+0] = byte(v >> 0)
		untypedData[offset+1] = byte(v >> 8)
		untypedData[offset+2] = byte(v >> 16)
		untypedData[offset+3] = byte(v >> 24)
		untypedData[offset+4] = byte(v >> 32)
		untypedData[offset+5] = byte(v >> 40)
		untypedData[offset+6] = byte(v >> 48)
		untypedData[offset+7] = byte(v >> 56)
	}
	untypedCompressed, err := Compress(untypedData)
	if err != nil {
		t.Fatalf("untyped compression failed: %v", err)
	}

	originalSize := len(data) * 8
	typedSize := len(typedCompressed)
	untypedSize := len(untypedCompressed)

	typedRatio := float64(originalSize) / float64(typedSize)
	untypedRatio := float64(originalSize) / float64(untypedSize)
	improvement := (float64(untypedSize) / float64(typedSize)) - 1.0

	t.Logf("Original size: %d bytes", originalSize)
	t.Logf("Typed compression: %d bytes (%.2fx ratio)", typedSize, typedRatio)
	t.Logf("Untyped compression: %d bytes (%.2fx ratio)", untypedSize, untypedRatio)
	t.Logf("Typed compression improvement: %.1f%%", improvement*100)

	// Typed compression should be at least as good as untyped
	if typedSize > untypedSize {
		t.Logf("Warning: Typed compression (%d bytes) is larger than untyped (%d bytes)", typedSize, untypedSize)
	}

	// Verify decompression
	decompressed, err := DecompressNumeric[int64](typedCompressed)
	if err != nil {
		t.Fatalf("typed decompression failed: %v", err)
	}

	for i := range data {
		if decompressed[i] != data[i] {
			t.Errorf("mismatch at index %d: got %d, want %d", i, decompressed[i], data[i])
		}
	}
}

// Helper to convert slice to bytes for comparison
func int64SliceToBytes(data []int64) []byte {
	buf := make([]byte, len(data)*8)
	for i, v := range data {
		offset := i * 8
		buf[offset+0] = byte(v >> 0)
		buf[offset+1] = byte(v >> 8)
		buf[offset+2] = byte(v >> 16)
		buf[offset+3] = byte(v >> 24)
		buf[offset+4] = byte(v >> 32)
		buf[offset+5] = byte(v >> 40)
		buf[offset+6] = byte(v >> 48)
		buf[offset+7] = byte(v >> 56)
	}
	return buf
}

func bytesToInt64Slice(buf []byte) []int64 {
	if len(buf)%8 != 0 {
		return nil
	}
	result := make([]int64, len(buf)/8)
	for i := range result {
		offset := i * 8
		result[i] = int64(buf[offset+0]) |
			int64(buf[offset+1])<<8 |
			int64(buf[offset+2])<<16 |
			int64(buf[offset+3])<<24 |
			int64(buf[offset+4])<<32 |
			int64(buf[offset+5])<<40 |
			int64(buf[offset+6])<<48 |
			int64(buf[offset+7])<<56
	}
	return result
}
