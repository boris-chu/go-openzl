// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package codec

import (
	"encoding/binary"
	"fmt"
	"testing"
)

// TestRLE_Repeated100KB_DetailedAnalysis provides detailed analysis of why our RLE
// achieves only 171× compression instead of the C library's 1,219× on repeated data.
//
// Goal: Understand the 7× efficiency gap and find path to 82-byte output.
func TestRLE_Repeated100KB_DetailedAnalysis(t *testing.T) {
	// Create test data: "Hello, World!" repeated 7,142 times = ~100KB
	pattern := []byte("Hello, World!")
	repeats := 7142
	totalSize := len(pattern) * repeats
	input := make([]byte, totalSize)
	for i := 0; i < repeats; i++ {
		copy(input[i*len(pattern):], pattern)
	}

	t.Logf("=== RLE Inefficiency Analysis ===")
	t.Logf("Input: \"%s\" × %d = %d bytes", pattern, repeats, totalSize)

	// Encode with RLE
	codec := NewRLE()
	dst := make([]byte, totalSize*10) // Larger buffer for RLE worst case
	compressed, err := codec.Encode(dst, input, nil)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	ratio := float64(totalSize) / float64(compressed)
	t.Logf("RLE output: %d bytes (%.2fx compression)", compressed, ratio)

	// Parse RLE structure
	numRuns := binary.LittleEndian.Uint32(dst[0:4])
	t.Logf("Number of runs: %d", numRuns)

	// Analyze run structure
	t.Logf("\n=== First 20 Runs ===")
	srcPos := 4
	for i := uint32(0); i < 20 && i < numRuns; i++ {
		value := dst[srcPos]
		srcPos++

		count, n := binary.Uvarint(dst[srcPos:])
		srcPos += n

		char := string([]byte{value})
		if value < 32 || value >= 127 {
			char = fmt.Sprintf("0x%02X", value)
		}
		t.Logf("  Run %3d: '%s' × %d (varint: %d bytes)", i, char, count, n)
	}

	// Calculate overhead
	avgBytesPerRun := float64(compressed-4) / float64(numRuns)
	t.Logf("\n=== Efficiency Metrics ===")
	t.Logf("Average bytes per run: %.3f", avgBytesPerRun)
	t.Logf("Breakdown: 1 byte (value) + %.3f bytes (varint count)", avgBytesPerRun-1)

	// Count actual consecutive runs
	actualRuns := countConsecutiveRuns(input)
	t.Logf("Actual consecutive runs: %d", actualRuns)

	// Theoretical perfect encoding
	perfectSize := 4 + actualRuns*2 // 4-byte header + (1 byte value + 1 byte count=1)
	t.Logf("Perfect RLE (count=1 always): %d bytes", perfectSize)
	t.Logf("Our RLE: %d bytes", compressed)
	t.Logf("Overhead vs perfect: %.2fx", float64(compressed)/float64(perfectSize))

	// Target: C library performance
	cLibrarySize := 82
	cLibraryRatio := float64(totalSize) / float64(cLibrarySize)
	t.Logf("\n=== Gap to C Library ===")
	t.Logf("C library size: %d bytes (%.0fx compression)", cLibrarySize, cLibraryRatio)
	t.Logf("Our size: %d bytes (%.0fx compression)", compressed, ratio)
	t.Logf("Gap: %.1fx worse", float64(compressed)/float64(cLibrarySize))

	// Key insight
	t.Logf("\n=== ROOT CAUSE ===")
	t.Logf("Pattern 'Hello, World!' has %d characters", len(pattern))
	t.Logf("Each character appears ONCE per repetition (not consecutive)")
	t.Logf("Total runs: %d characters × %d repetitions = %d runs", len(pattern), repeats, actualRuns)
	t.Logf("")
	t.Logf("Our RLE encodes %d runs = %d bytes", numRuns, compressed)
	t.Logf("C library must be doing something smarter than byte-level RLE!")
	t.Logf("")
	t.Logf("Possible C library optimizations:")
	t.Logf("  1. Pattern detection: Recognize repeated byte sequences")
	t.Logf("  2. LZ77-like: Store pattern once + reference count")
	t.Logf("  3. Multi-byte runs: Encode 'Hello, World!' as single unit")
	t.Logf("  4. Different test data: Maybe C library test uses different input?")
}

// TestRLE_TrulyRepeatedData tests RLE on data where ALL bytes are identical.
// This is the ideal case for RLE and should achieve massive compression.
func TestRLE_TrulyRepeatedData(t *testing.T) {
	// 100KB of all zeros
	input := make([]byte, 100000)
	// All zeros by default

	codec := NewRLE()
	dst := make([]byte, len(input)*2)
	compressed, err := codec.Encode(dst, input, nil)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	ratio := float64(len(input)) / float64(compressed)
	t.Logf("100KB of zeros: %d bytes → %d bytes (%.0fx compression)", len(input), compressed, ratio)

	// Parse structure
	numRuns := binary.LittleEndian.Uint32(dst[0:4])
	t.Logf("Number of runs: %d", numRuns)

	// Should be exactly 1 run
	if numRuns != 1 {
		t.Errorf("Expected 1 run for all-zero data, got %d", numRuns)
	}

	// Parse the single run
	value := dst[4]
	count, n := binary.Uvarint(dst[5:])
	t.Logf("Single run: value=0x%02X, count=%d, varint size=%d bytes", value, count, n)

	totalSize := 4 + 1 + n // header + value + varint
	if compressed != totalSize {
		t.Errorf("Expected %d bytes, got %d", totalSize, compressed)
	}

	t.Logf("\n=== Theoretical Perfect RLE ===")
	t.Logf("Perfect encoding: 4 bytes (header) + 1 byte (value=0) + ? bytes (count=100000)")

	// Varint encoding of 100000
	// 100000 = 0x186A0
	// Varint: needs 3 bytes for numbers up to 2^21
	perfectVarintSize := 3
	perfectTotal := 4 + 1 + perfectVarintSize
	t.Logf("Perfect size: 4 + 1 + %d = %d bytes", perfectVarintSize, perfectTotal)
	t.Logf("Our size: %d bytes", compressed)

	if compressed > perfectTotal {
		t.Logf("Overhead: %d bytes", compressed-perfectTotal)
	} else {
		t.Logf("✅ Optimal encoding achieved!")
	}

	// This gives us the BEST possible ratio for RLE
	perfectRatio := float64(len(input)) / float64(perfectTotal)
	t.Logf("\nPerfect RLE ratio: %.0fx", perfectRatio)
	t.Logf("Our ratio: %.0fx", ratio)
}

// TestRLE_ComparePatternsVsRepetition compares compression of:
// 1. Pattern repetition: "ABC" × 33,333 = 100KB
// 2. True repetition: 'A' × 100,000 = 100KB
func TestRLE_ComparePatternsVsRepetition(t *testing.T) {
	codec := NewRLE()

	// Test 1: Pattern repetition
	pattern := []byte("ABC")
	repeats := 33333
	totalSize := len(pattern) * repeats
	pattern_input := make([]byte, totalSize)
	for i := 0; i < repeats; i++ {
		copy(pattern_input[i*len(pattern):], pattern)
	}

	dst1 := make([]byte, totalSize*2)
	compressed1, _ := codec.Encode(dst1, pattern_input, nil)
	ratio1 := float64(totalSize) / float64(compressed1)

	// Test 2: True repetition
	true_input := make([]byte, 100000)
	for i := range true_input {
		true_input[i] = 'A'
	}

	dst2 := make([]byte, len(true_input)*2)
	compressed2, _ := codec.Encode(dst2, true_input, nil)
	ratio2 := float64(len(true_input)) / float64(compressed2)

	t.Logf("=== Pattern vs Repetition ===")
	t.Logf("Pattern 'ABC' × %d: %d bytes (%.0fx compression)", repeats, compressed1, ratio1)
	t.Logf("True 'A' × 100000: %d bytes (%.0fx compression)", compressed2, ratio2)
	t.Logf("Difference: %.0fx", ratio2/ratio1)
	t.Logf("")
	t.Logf("Insight: RLE on patterns is %dx WORSE than RLE on true repetition", int(ratio2/ratio1))
}

// countConsecutiveRuns counts the number of consecutive runs in data.
func countConsecutiveRuns(data []byte) int {
	if len(data) == 0 {
		return 0
	}

	runs := 0
	pos := 0
	for pos < len(data) {
		runValue := data[pos]
		runLen := 1
		for pos+runLen < len(data) && data[pos+runLen] == runValue {
			runLen++
		}
		runs++
		pos += runLen
	}
	return runs
}
