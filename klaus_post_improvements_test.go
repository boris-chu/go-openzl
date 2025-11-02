// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package openzl_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/boris-chu/go-openzl"
)

// TestCompressTo validates the Klaus Post-inspired user-buffer API
func TestCompressTo(t *testing.T) {
	compressor, err := openzl.NewCompressor()
	if err != nil {
		t.Fatal(err)
	}
	defer compressor.Close()

	data := bytes.Repeat([]byte("OpenZL rocks! "), 100)

	// Pre-allocate buffer (zero allocations in steady state!)
	dst := make([]byte, openzl.CompressBound(len(data)))

	// Compress into user buffer
	n, err := compressor.CompressTo(dst, data)
	if err != nil {
		t.Fatalf("CompressTo failed: %v", err)
	}

	if n == 0 {
		t.Fatal("Compressed size is zero")
	}

	if n > len(dst) {
		t.Fatalf("Compressed size %d exceeds buffer %d", n, len(dst))
	}

	// Verify by decompressing
	decompressed, err := openzl.Decompress(dst[:n])
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}

	if !bytes.Equal(data, decompressed) {
		t.Fatal("Roundtrip failed: data doesn't match")
	}

	t.Logf("✅ CompressTo: Compressed %d bytes to %d bytes (%.2fx ratio)",
		len(data), n, float64(len(data))/float64(n))
}

// TestCompressTo_BufferTooSmall verifies error handling
func TestCompressTo_BufferTooSmall(t *testing.T) {
	compressor, err := openzl.NewCompressor()
	if err != nil {
		t.Fatal(err)
	}
	defer compressor.Close()

	data := []byte("test data")
	dst := make([]byte, 1) // Too small!

	_, err = compressor.CompressTo(dst, data)
	if err == nil {
		t.Fatal("Expected error for too-small buffer, got nil")
	}

	t.Logf("✅ Correctly detected buffer too small: %v", err)
}

// BenchmarkCompress_BufferPooling benchmarks the new buffer pooling
func BenchmarkCompress_BufferPooling(b *testing.B) {
	compressor, _ := openzl.NewCompressor()
	defer compressor.Close()

	data := bytes.Repeat([]byte("data"), 250) // 1KB

	b.Run("WithPooling", func(b *testing.B) {
		b.SetBytes(int64(len(data)))
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := compressor.Compress(data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkCompressTo_ZeroAlloc demonstrates zero-allocation compression
func BenchmarkCompressTo_ZeroAlloc(b *testing.B) {
	compressor, _ := openzl.NewCompressor()
	defer compressor.Close()

	data := bytes.Repeat([]byte("data"), 250) // 1KB

	// Pre-allocate buffer once
	dst := make([]byte, openzl.CompressBound(len(data)))

	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		n, err := compressor.CompressTo(dst, data)
		if err != nil {
			b.Fatal(err)
		}
		if n == 0 {
			b.Fatal("Zero bytes compressed")
		}
	}

	// Should see 0 allocs/op in steady state!
}

// Example_compressToZeroAlloc demonstrates the zero-allocation pattern
func Example_compressToZeroAlloc() {
	compressor, _ := openzl.NewCompressor()
	defer compressor.Close()

	// Pre-allocate buffer once for maximum message size
	maxSize := 1024 * 1024 // 1MB
	dst := make([]byte, openzl.CompressBound(maxSize))

	// Process many messages with zero allocations!
	messages := [][]byte{
		[]byte("First message"),
		[]byte("Second message"),
		[]byte("Third message"),
	}

	for i, msg := range messages {
		n, err := compressor.CompressTo(dst, msg)
		if err != nil {
			panic(err)
		}

		// Use dst[:n] - no allocation!
		fmt.Printf("Message %d compressed to %d bytes\n", i, n)
		_ = dst[:n] // In real code, send this over network/write to file
	}

	// Output:
	// Message 0 compressed to 45 bytes
	// Message 1 compressed to 46 bytes
	// Message 2 compressed to 45 bytes
}
