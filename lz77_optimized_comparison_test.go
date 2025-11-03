// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

//go:build cgo

package openzl

import (
	"testing"

	"github.com/boris-chu/go-openzl/internal/codec"
)

// TestLZ77Optimized_vs_CLibrary_PatternData compares optimized LZ77 against C library
func TestLZ77Optimized_vs_CLibrary_PatternData(t *testing.T) {
	// Benchmark pattern: 37 bytes repeated 2,767 times = 100KB
	pattern := []byte("This is a test pattern that repeats. ")
	data := make([]byte, 0, 100*1024)
	for len(data) < 100*1024 {
		data = append(data, pattern...)
	}
	data = data[:100*1024]

	t.Logf("=== LZ77 Optimized vs C Library Comparison ===")
	t.Logf("Test: Pattern repetition (37-byte pattern × 2,767)")
	t.Logf("Input size: %d bytes (100KB)", len(data))
	t.Logf("")

	// C Library
	cCompressed, err := Compress(data)
	if err != nil {
		t.Fatalf("C library Compress failed: %v", err)
	}
	cRatio := float64(len(data)) / float64(len(cCompressed))

	t.Logf("C Library Results:")
	t.Logf("  Compressed size: %d bytes", len(cCompressed))
	t.Logf("  Compression ratio: %.0fx", cRatio)
	t.Logf("")

	// LZ77 Optimized (codec only, no frame)
	lz77Opt := codec.NewLZ77Optimized()
	lz77Dst := make([]byte, len(data)*2)
	lz77Size, err := lz77Opt.Encode(lz77Dst, data, nil)
	if err != nil {
		t.Fatalf("LZ77Optimized Encode failed: %v", err)
	}
	lz77Ratio := float64(len(data)) / float64(lz77Size)

	t.Logf("LZ77Optimized Results (codec only):")
	t.Logf("  Compressed size: %d bytes", lz77Size)
	t.Logf("  Compression ratio: %.0fx", lz77Ratio)
	t.Logf("")

	// Gap analysis
	gap := float64(lz77Size) / float64(len(cCompressed))
	t.Logf("Performance Gap:")
	t.Logf("  LZ77Optimized / C Library: %.2fx", gap)

	if gap <= 1.2 {
		t.Logf("  Status: EXCELLENT (within 20%% of C library)")
	} else if gap <= 2.0 {
		t.Logf("  Status: GOOD (within 2x of C library)")
	} else if gap <= 5.0 {
		t.Logf("  Status: ACCEPTABLE (within 5x of C library)")
	} else {
		t.Logf("  Status: NEEDS WORK (>5x C library)")
	}
	t.Logf("")

	// Compare with original LZ77 from docs
	t.Logf("Historical Comparison:")
	t.Logf("  Original LZ77 (from docs): 454 bytes")
	t.Logf("  LZ77Optimized: %d bytes", lz77Size)
	if lz77Size < 454 {
		improvement := float64(454-lz77Size) / 454.0 * 100
		t.Logf("  Improvement: %.1f%% smaller (%.1fx better)", improvement, 454.0/float64(lz77Size))
	}
	t.Logf("")

	// Verify roundtrip
	decompressed := make([]byte, len(data)*2)
	dn, err := lz77Opt.Decode(decompressed, lz77Dst[:lz77Size], nil)
	if err != nil {
		t.Fatalf("LZ77Optimized Decode failed: %v", err)
	}

	if dn != len(data) {
		t.Errorf("Decoded size %d, want %d", dn, len(data))
	}

	// Verify data integrity
	mismatch := false
	for i := 0; i < len(data); i++ {
		if decompressed[i] != data[i] {
			mismatch = true
			break
		}
	}

	if mismatch {
		t.Error("Roundtrip failed: data mismatch")
	} else {
		t.Logf("Roundtrip successful: data integrity verified")
	}
}

// TestLZ77Optimized_MultiplePatternSizes tests various pattern sizes
func TestLZ77Optimized_MultiplePatternSizes(t *testing.T) {
	lz77Opt := codec.NewLZ77Optimized()

	tests := []struct {
		name        string
		patternSize int
		dataSize    int
	}{
		{"Small pattern (3 bytes)", 3, 10 * 1024},
		{"Medium pattern (37 bytes)", 37, 100 * 1024},
		{"Large pattern (100 bytes)", 100, 100 * 1024},
	}

	t.Logf("=== LZ77Optimized Pattern Size Comparison ===")
	t.Logf("%-30s %15s %15s %10s %15s %10s", "Pattern Type", "C Library", "LZ77Optimized", "Gap", "vs Original", "Status")
	t.Logf("%s", "================================================================================================================")

	for _, tt := range tests {
		// Create pattern
		pattern := make([]byte, tt.patternSize)
		for i := range pattern {
			pattern[i] = byte(i % 256)
		}

		// Repeat to target size
		data := make([]byte, 0, tt.dataSize)
		for len(data) < tt.dataSize {
			data = append(data, pattern...)
		}
		data = data[:tt.dataSize]

		// C Library
		cCompressed, err := Compress(data)
		if err != nil {
			t.Logf("%-30s %15s %15s %10s %15s %10s", tt.name, "ERROR", "-", "-", "-", "-")
			continue
		}

		// LZ77Optimized
		dst := make([]byte, len(data)*2)
		lz77Size, err := lz77Opt.Encode(dst, data, nil)
		if err != nil {
			t.Logf("%-30s %12d bytes %15s %10s %15s %10s", tt.name, len(cCompressed), "ERROR", "-", "-", "-")
			continue
		}

		gap := float64(lz77Size) / float64(len(cCompressed))
		status := "Needs work"
		if gap <= 1.2 {
			status = "Excellent"
		} else if gap <= 2.0 {
			status = "Good"
		} else if gap <= 5.0 {
			status = "OK"
		}

		// Compare with theoretical worst case (454 bytes for 100KB pattern)
		vsOriginal := 454.0 / float64(lz77Size)

		t.Logf("%-30s %12d bytes %12d bytes %9.2fx %14.1fx %15s",
			tt.name, len(cCompressed), lz77Size, gap, vsOriginal, status)
	}
}

// BenchmarkLZ77Optimized_vs_CLibrary benchmarks both implementations
func BenchmarkLZ77Optimized_vs_CLibrary(b *testing.B) {
	pattern := []byte("This is a test pattern that repeats. ")
	data := make([]byte, 0, 100*1024)
	for len(data) < 100*1024 {
		data = append(data, pattern...)
	}
	data = data[:100*1024]

	b.Run("C Library", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := Compress(data)
			if err != nil {
				b.Fatal(err)
			}
		}

		b.SetBytes(int64(len(data)))
	})

	b.Run("LZ77Optimized", func(b *testing.B) {
		lz77Opt := codec.NewLZ77Optimized()
		dst := make([]byte, len(data)*2)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := lz77Opt.Encode(dst, data, nil)
			if err != nil {
				b.Fatal(err)
			}
		}

		b.SetBytes(int64(len(data)))
	})
}
