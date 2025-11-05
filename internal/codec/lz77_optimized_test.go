// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package codec

import (
	"bytes"
	"testing"
)

// TestLZ77Optimized_PatternDetection tests pattern detection
func TestLZ77Optimized_PatternDetection(t *testing.T) {
	tests := []struct {
		name        string
		pattern     []byte
		repetitions int
		wantPattern int
	}{
		{
			name:        "37-byte pattern (benchmark case)",
			pattern:     []byte("This is a test pattern that repeats. "),
			repetitions: 2767,
			wantPattern: 37,
		},
		{
			name:        "3-byte pattern",
			pattern:     []byte("ABC"),
			repetitions: 1000,
			wantPattern: 3,
		},
		{
			name:        "10-byte pattern",
			pattern:     []byte("0123456789"),
			repetitions: 500,
			wantPattern: 10,
		},
		{
			name:        "single byte pattern",
			pattern:     []byte("A"),
			repetitions: 10000,
			wantPattern: 1,
		},
	}

	codec := NewLZ77Optimized()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create repeated pattern
			var data []byte
			for i := 0; i < tt.repetitions; i++ {
				data = append(data, tt.pattern...)
			}

			detected := codec.detectPattern(data)

			if detected != tt.wantPattern {
				t.Errorf("detectPattern() = %d, want %d", detected, tt.wantPattern)
			}

			if detected > 0 {
				t.Logf("✅ Detected pattern length: %d bytes (expected %d)", detected, tt.wantPattern)
			} else {
				t.Logf("✅ Correctly detected no pattern")
			}
		})
	}
}

// TestLZ77Optimized_PatternEncoding tests pattern-optimized encoding
func TestLZ77Optimized_PatternEncoding(t *testing.T) {
	codec := NewLZ77Optimized()

	// Benchmark pattern: 37 bytes repeated 2,767 times = 100KB
	pattern := []byte("This is a test pattern that repeats. ")
	data := make([]byte, 0, 100*1024)
	for len(data) < 100*1024 {
		data = append(data, pattern...)
	}
	data = data[:100*1024]

	dst := make([]byte, len(data)*2)
	n, err := codec.Encode(dst, data, nil)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	compressed := dst[:n]

	t.Logf("Pattern encoding results:")
	t.Logf("  Original: %d bytes", len(data))
	t.Logf("  Compressed: %d bytes", len(compressed))
	t.Logf("  Ratio: %.0fx", float64(len(data))/float64(len(compressed)))

	// Target: < 100 bytes (C library achieves 24 bytes in LZ77 data alone)
	// With frame overhead, we expect ~40-60 bytes total
	if len(compressed) > 100 {
		t.Logf("⚠️  Compressed size %d bytes (target: < 100 bytes)", len(compressed))
	} else {
		t.Logf("✅ Compressed size %d bytes (target: < 100 bytes)", len(compressed))
	}

	// Test roundtrip
	decompressed := make([]byte, len(data)*2)
	dn, err := codec.Decode(decompressed, compressed, nil)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if dn != len(data) {
		t.Errorf("Decoded length %d, want %d", dn, len(data))
	}

	if !bytes.Equal(decompressed[:dn], data) {
		t.Errorf("Roundtrip failed: data mismatch")
	} else {
		t.Logf("✅ Roundtrip successful")
	}
}

// TestLZ77Optimized_ConcurrentHashing tests parallel hash table building
func TestLZ77Optimized_ConcurrentHashing(t *testing.T) {
	codec := NewLZ77Optimized()
	codec.useParallel = true

	// Large data to benefit from concurrency
	data := make([]byte, 200*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	t.Logf("Building hash table with %d bytes of data...", len(data))

	hash := codec.buildHashTableParallel(data)

	if hash == nil {
		t.Fatal("buildHashTableParallel returned nil")
	}

	// Verify hash table has entries
	entryCount := 0
	for _, positions := range hash.table {
		entryCount += len(positions)
	}

	t.Logf("✅ Hash table built with %d total entries", entryCount)

	if entryCount == 0 {
		t.Error("Hash table is empty")
	}

	// Compare with sequential version for correctness
	hashSeq := codec.buildHashTableSequential(data)
	entryCountSeq := 0
	for _, positions := range hashSeq.table {
		entryCountSeq += len(positions)
	}

	t.Logf("Sequential hash table: %d entries", entryCountSeq)

	// Both should have similar entry counts (may differ slightly due to maxChain)
	if entryCount < entryCountSeq/2 {
		t.Errorf("Parallel hash table has significantly fewer entries (%d vs %d)", entryCount, entryCountSeq)
	}
}

// TestLZ77Optimized_LazyMatching tests lazy matching strategy
func TestLZ77Optimized_LazyMatching(t *testing.T) {
	// Create data where lazy matching should help
	// Example: "ABCDABCDABCD" - lazy matching should find longer matches
	data := []byte("ABCDABCDABCDABCDABCDABCDABCDABCDABCDABCD")
	data = append(data, data...) // Double it
	data = append(data, data...) // Double again (320 bytes)

	codecNoLazy := NewLZ77Optimized()
	codecNoLazy.useLazy = false

	codecLazy := NewLZ77Optimized()
	codecLazy.useLazy = true

	dst1 := make([]byte, len(data)*2)
	n1, err := codecNoLazy.Encode(dst1, data, nil)
	if err != nil {
		t.Fatalf("Encode without lazy failed: %v", err)
	}

	dst2 := make([]byte, len(data)*2)
	n2, err := codecLazy.Encode(dst2, data, nil)
	if err != nil {
		t.Fatalf("Encode with lazy failed: %v", err)
	}

	t.Logf("Lazy matching results:")
	t.Logf("  Without lazy: %d bytes", n1)
	t.Logf("  With lazy: %d bytes", n2)

	if n2 < n1 {
		improvement := float64(n1-n2) / float64(n1) * 100
		t.Logf("✅ Lazy matching improved compression by %.1f%%", improvement)
	} else if n2 == n1 {
		t.Logf("✅ Lazy matching same as greedy (both found optimal matches)")
	} else {
		t.Logf("⚠️  Lazy matching resulted in larger output (may be overhead)")
	}

	// Test roundtrip for both
	decompressed := make([]byte, len(data)*2)
	dn, err := codecLazy.Decode(decompressed, dst2[:n2], nil)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if !bytes.Equal(decompressed[:dn], data) {
		t.Error("Lazy matching roundtrip failed")
	} else {
		t.Logf("✅ Roundtrip successful")
	}
}

// TestLZ77Optimized_VariousPatternLengths tests different pattern sizes
func TestLZ77Optimized_VariousPatternLengths(t *testing.T) {
	codec := NewLZ77Optimized()

	patternSizes := []int{1, 3, 10, 37, 100, 256}

	for _, size := range patternSizes {
		t.Run(formatPatternSize(size), func(t *testing.T) {
			// Create pattern
			pattern := make([]byte, size)
			for i := range pattern {
				pattern[i] = byte(i % 256)
			}

			// Repeat to make 100KB
			data := make([]byte, 0, 100*1024)
			for len(data) < 100*1024 {
				data = append(data, pattern...)
			}
			data = data[:100*1024]

			dst := make([]byte, len(data)*2)
			n, err := codec.Encode(dst, data, nil)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}

			ratio := float64(len(data)) / float64(n)
			t.Logf("Pattern size %d bytes: compressed to %d bytes (%.0fx)", size, n, ratio)

			// Verify roundtrip
			decompressed := make([]byte, len(data)*2)
			dn, err := codec.Decode(decompressed, dst[:n], nil)
			if err != nil {
				t.Fatalf("Decode failed: %v", err)
			}

			if !bytes.Equal(decompressed[:dn], data) {
				t.Error("Roundtrip failed")
			}
		})
	}
}

// TestLZ77Optimized_EmptyData tests edge case of empty input
func TestLZ77Optimized_EmptyData(t *testing.T) {
	codec := NewLZ77Optimized()
	dst := make([]byte, 100)

	n, err := codec.Encode(dst, []byte{}, nil)
	if err != nil {
		t.Fatalf("Encode failed on empty data: %v", err)
	}

	if n != 4 {
		t.Errorf("Empty data should encode to 4 bytes (token count), got %d", n)
	}

	t.Logf("✅ Empty data encoded to %d bytes", n)
}

// TestLZ77Optimized_SmallData tests small data that shouldn't use parallel
func TestLZ77Optimized_SmallData(t *testing.T) {
	codec := NewLZ77Optimized()
	data := []byte("Small data that fits in a tweet!")

	dst := make([]byte, len(data)*10) // Larger buffer for tokens
	n, err := codec.Encode(dst, data, nil)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	t.Logf("Small data: %d → %d bytes", len(data), n)

	// Verify roundtrip
	decompressed := make([]byte, len(data)*2)
	dn, err := codec.Decode(decompressed, dst[:n], nil)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if !bytes.Equal(decompressed[:dn], data) {
		t.Error("Roundtrip failed")
	} else {
		t.Logf("✅ Roundtrip successful")
	}
}

// BenchmarkLZ77Optimized_PatternData benchmarks pattern compression
func BenchmarkLZ77Optimized_PatternData(b *testing.B) {
	codec := NewLZ77Optimized()

	pattern := []byte("This is a test pattern that repeats. ")
	data := make([]byte, 0, 100*1024)
	for len(data) < 100*1024 {
		data = append(data, pattern...)
	}
	data = data[:100*1024]

	dst := make([]byte, len(data)*2)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := codec.Encode(dst, data, nil)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.SetBytes(int64(len(data)))
}

// BenchmarkLZ77Optimized_vs_Regular compares optimized vs regular LZ77
func BenchmarkLZ77Optimized_vs_Regular(b *testing.B) {
	pattern := []byte("This is a test pattern that repeats. ")
	data := make([]byte, 0, 100*1024)
	for len(data) < 100*1024 {
		data = append(data, pattern...)
	}
	data = data[:100*1024]

	b.Run("Regular LZ77", func(b *testing.B) {
		codec := &LZ77{}
		dst := make([]byte, len(data)*2)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := codec.Encode(dst, data, nil)
			if err != nil {
				b.Fatal(err)
			}
		}

		b.SetBytes(int64(len(data)))
	})

	b.Run("Optimized LZ77", func(b *testing.B) {
		codec := NewLZ77Optimized()
		dst := make([]byte, len(data)*2)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := codec.Encode(dst, data, nil)
			if err != nil {
				b.Fatal(err)
			}
		}

		b.SetBytes(int64(len(data)))
	})
}

// formatPatternSize formats pattern size for test name
func formatPatternSize(size int) string {
	if size < 10 {
		return string(rune('0'+size)) + "B"
	}
	return string(rune('0'+size/10)) + string(rune('0'+size%10)) + "B"
}
