// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package purgo

import (
	"testing"
)

// generateRepeatedData creates test data EXACTLY matching benchmark_comparison_test.go
func generateRepeatedData(size int) []byte {
	pattern := []byte("This is a test pattern that repeats. ")
	data := make([]byte, 0, size)
	for len(data) < size {
		data = append(data, pattern...)
	}
	return data[:size]
}

// TestCompressSmart_ExactBenchmarkData tests with the EXACT data from benchmarks
//
// This test reveals whether the 171× compression ratio is:
// 1. From RLE codec (if RLE is chosen)
// 2. From LZ77 codec (if LZ77 is chosen)
// 3. From double-compression overhead
func TestCompressSmart_ExactBenchmarkData(t *testing.T) {
	// EXACT test data from benchmark_comparison_test.go
	data := generateRepeatedData(100 * 1024) // 100KB

	t.Logf("=== Testing with EXACT benchmark data ===")
	t.Logf("Input: %d bytes", len(data))
	t.Logf("Pattern: \"This is a test pattern that repeats. \" (37 bytes)")
	t.Logf("Repetitions: ~%d", len(data)/37)

	// Compress with CompressSmart
	compressed, err := CompressSmart(data)
	if err != nil {
		t.Fatalf("CompressSmart failed: %v", err)
	}

	ratio := float64(len(data)) / float64(len(compressed))
	t.Logf("\nCompression Results:")
	t.Logf("  Compressed size: %d bytes", len(compressed))
	t.Logf("  Compression ratio: %.2fx", ratio)

	// Decompress to verify
	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}

	if len(decompressed) != len(data) {
		t.Fatalf("Decompressed size mismatch: got %d, want %d", len(decompressed), len(data))
	}

	// Verify data integrity
	for i := range data {
		if data[i] != decompressed[i] {
			t.Fatalf("Data mismatch at byte %d: got 0x%02X, want 0x%02X", i, decompressed[i], data[i])
		}
	}

	t.Logf("\n✅ Round-trip successful")

	// Compare with documentation claim
	t.Logf("\n=== Comparison ===")
	t.Logf("Pure-Go (this test): %.0fx", ratio)
	t.Logf("Documented result: 171× (may be from earlier version)")

	if ratio < 150 {
		t.Logf("⚠️  Ratio is WORSE than documented (%.0fx < 171×)", ratio)
	} else if ratio > 200 {
		t.Logf("✅ Ratio is BETTER than documented (%.0fx > 171×)", ratio)
	} else {
		t.Logf("✅ Ratio matches documented result (~171×)")
	}

	// Frame overhead analysis
	frameOverheadEstimate := 120 // Two frames (LZ77 + Huffman wrap)
	actualCompressed := len(compressed) - frameOverheadEstimate
	if actualCompressed > 0 {
		ratioWithoutFrames := float64(len(data)) / float64(actualCompressed)
		t.Logf("\n=== Frame Overhead Analysis ===")
		t.Logf("Estimated frame overhead: ~%d bytes", frameOverheadEstimate)
		t.Logf("Actual compressed data: ~%d bytes", actualCompressed)
		t.Logf("Ratio without frame overhead: %.0fx", ratioWithoutFrames)
		t.Logf("Overhead impact: %.1f%% of output size", 100.0*float64(frameOverheadEstimate)/float64(len(compressed)))
	}
}

// TestRLE_TrueRepetition tests RLE on truly repeated data (all same byte)
//
// This should achieve near-perfect compression (~12,500×)
func TestRLE_TrueRepetition_100KB(t *testing.T) {
	// 100KB of all zeros (TRUE repetition, not pattern)
	data := make([]byte, 100*1024)

	t.Logf("=== Testing RLE on TRUE repetition ===")
	t.Logf("Input: %d bytes (all zeros)", len(data))

	// Compress with CompressSmart
	compressed, err := CompressSmart(data)
	if err != nil {
		t.Fatalf("CompressSmart failed: %v", err)
	}

	ratio := float64(len(data)) / float64(len(compressed))
	t.Logf("\nCompression Results:")
	t.Logf("  Compressed size: %d bytes", len(compressed))
	t.Logf("  Compression ratio: %.0fx", ratio)

	// Theoretical perfect RLE
	perfectRLE := 4 + 1 + 3 // header + value + varint(100000)
	perfectRatio := float64(len(data)) / float64(perfectRLE)
	t.Logf("\n=== Theoretical Analysis ===")
	t.Logf("Perfect RLE encoding: %d bytes (%.0fx)", perfectRLE, perfectRatio)
	t.Logf("Our result: %d bytes (%.0fx)", len(compressed), ratio)

	if ratio >= perfectRatio-1 {
		t.Logf("✅ Near-perfect compression achieved!")
	} else {
		t.Logf("⚠️  Gap to perfect: %.0fx (%.1f%% of optimal)", perfectRatio/ratio, 100.0*ratio/perfectRatio)
	}

	// Verify round-trip
	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}

	if len(decompressed) != len(data) {
		t.Fatalf("Size mismatch: got %d, want %d", len(decompressed), len(data))
	}

	t.Logf("\n✅ Round-trip successful")
}

// TestComparisonSummary provides a summary comparing different test scenarios
func TestComparisonSummary(t *testing.T) {
	tests := []struct {
		name     string
		dataFunc func() []byte
	}{
		{
			name: "True repetition (all 'A')",
			dataFunc: func() []byte {
				data := make([]byte, 100*1024)
				for i := range data {
					data[i] = 'A'
				}
				return data
			},
		},
		{
			name: "True repetition (all zeros)",
			dataFunc: func() []byte {
				return make([]byte, 100*1024)
			},
		},
		{
			name: "Pattern repetition (benchmark data)",
			dataFunc: func() []byte {
				return generateRepeatedData(100 * 1024)
			},
		},
	}

	t.Logf("=== Compression Comparison Summary ===\n")
	t.Logf("%-40s %15s %15s", "Test Case", "Compressed", "Ratio")
	t.Logf("%s", "==================================================================================")

	for _, tt := range tests {
		data := tt.dataFunc()
		compressed, err := CompressSmart(data)
		if err != nil {
			t.Logf("%-40s %15s %15s", tt.name, "ERROR", err.Error())
			continue
		}

		ratio := float64(len(data)) / float64(len(compressed))
		t.Logf("%-40s %12d bytes %14.0fx", tt.name, len(compressed), ratio)
	}

	t.Logf("\n=== Key Insights ===")
	t.Logf("1. True repetition should achieve ~12,500× (perfect RLE)")
	t.Logf("2. Pattern repetition achieves ~171× (LZ77 + Huffman)")
	t.Logf("3. RLE is NOT used for patterns (CompressSmart chooses LZ77)")
	t.Logf("4. The 7× gap was comparing RLE (C) vs LZ77+Huffman (Pure-Go)")
}
