// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package openzl_test

import (
	"bytes"
	"testing"

	"github.com/borischu/go-openzl"
)

func TestCompressDecompress(t *testing.T) {
	t.Skip("TODO: Implement in Phase 1")

	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "simple text",
			input: []byte("hello world"),
		},
		{
			name:  "repeated data",
			input: bytes.Repeat([]byte("test"), 100),
		},
		{
			name:  "binary data",
			input: []byte{0x00, 0x01, 0x02, 0xFF, 0xFE},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Compress
			compressed, err := openzl.Compress(tt.input)
			if err != nil {
				t.Fatalf("Compress() error = %v", err)
			}

			t.Logf("Original: %d bytes, Compressed: %d bytes, Ratio: %.2f",
				len(tt.input), len(compressed),
				float64(len(tt.input))/float64(len(compressed)))

			// Decompress
			decompressed, err := openzl.Decompress(compressed)
			if err != nil {
				t.Fatalf("Decompress() error = %v", err)
			}

			// Verify
			if !bytes.Equal(tt.input, decompressed) {
				t.Errorf("Decompressed data doesn't match original")
			}
		})
	}
}

func TestCompressEmpty(t *testing.T) {
	t.Skip("TODO: Implement in Phase 1")

	_, err := openzl.Compress([]byte{})
	if err == nil {
		t.Error("Expected error for empty input")
	}
}

func TestDecompressCorrupted(t *testing.T) {
	t.Skip("TODO: Implement in Phase 1")

	corrupted := []byte{0x00, 0x01, 0x02, 0x03}
	_, err := openzl.Decompress(corrupted)
	if err == nil {
		t.Error("Expected error for corrupted data")
	}
}

func BenchmarkCompress(b *testing.B) {
	b.Skip("TODO: Implement in Phase 1")

	data := bytes.Repeat([]byte("benchmark test data "), 100)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := openzl.Compress(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecompress(b *testing.B) {
	b.Skip("TODO: Implement in Phase 1")

	data := bytes.Repeat([]byte("benchmark test data "), 100)
	compressed, err := openzl.Compress(data)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := openzl.Decompress(compressed)
		if err != nil {
			b.Fatal(err)
		}
	}
}
