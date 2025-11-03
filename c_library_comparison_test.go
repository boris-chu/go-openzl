// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

//go:build cgo

package openzl

import (
	"fmt"
	"testing"

	"github.com/boris-chu/go-openzl/purgo"
)

// generateRepeatedDataHelper creates test data matching benchmark patterns
func generateRepeatedDataHelper(size int) []byte {
	pattern := []byte("This is a test pattern that repeats. ")
	data := make([]byte, 0, size)
	for len(data) < size {
		data = append(data, pattern...)
	}
	return data[:size]
}

// TestCLibrary_vs_PureGo_TrueRepetition compares C library and Pure-Go on truly repeated data
func TestCLibrary_vs_PureGo_TrueRepetition(t *testing.T) {
	// Test data: 100KB of all 'A' (TRUE repetition)
	data := make([]byte, 100*1024)
	for i := range data {
		data[i] = 'A'
	}

	t.Logf("=== Test Case: True Repetition (100KB of 'A') ===\n")

	// Test 1: C Library (CGO)
	cCompressed, err := Compress(data)
	if err != nil {
		t.Fatalf("C library Compress failed: %v", err)
	}
	cRatio := float64(len(data)) / float64(len(cCompressed))

	// Verify decompression
	cDecompressed, err := Decompress(cCompressed)
	if err != nil {
		t.Fatalf("C library Decompress failed: %v", err)
	}
	if len(cDecompressed) != len(data) {
		t.Fatalf("C library round-trip size mismatch")
	}

	t.Logf("C Library:")
	t.Logf("  Compressed: %d bytes", len(cCompressed))
	t.Logf("  Ratio: %.0fx", cRatio)

	// Test 2: Pure-Go
	pureGoCompressed, err := purgo.CompressSmart(data)
	if err != nil {
		t.Fatalf("Pure-Go CompressSmart failed: %v", err)
	}
	pureGoRatio := float64(len(data)) / float64(len(pureGoCompressed))

	// Verify decompression
	pureGoDecompressed, err := purgo.Decompress(pureGoCompressed)
	if err != nil {
		t.Fatalf("Pure-Go Decompress failed: %v", err)
	}
	if len(pureGoDecompressed) != len(data) {
		t.Fatalf("Pure-Go round-trip size mismatch")
	}

	t.Logf("\nPure-Go:")
	t.Logf("  Compressed: %d bytes", len(pureGoCompressed))
	t.Logf("  Ratio: %.0fx", pureGoRatio)

	// Comparison
	t.Logf("\n=== Comparison ===")
	t.Logf("C Library: %d bytes (%.0fx)", len(cCompressed), cRatio)
	t.Logf("Pure-Go:   %d bytes (%.0fx)", len(pureGoCompressed), pureGoRatio)

	if len(cCompressed) < len(pureGoCompressed) {
		gap := float64(len(pureGoCompressed)) / float64(len(cCompressed))
		t.Logf("Gap: Pure-Go is %.2fx larger than C library", gap)
		t.Logf("Pure-Go overhead: %d bytes (frame format)", len(pureGoCompressed)-len(cCompressed))
	} else {
		t.Logf("✅ Pure-Go matches or beats C library!")
	}

	// Theoretical perfect
	perfectRLE := 8 // 4 header + 1 value + 3 varint
	t.Logf("\nTheoretical perfect RLE: %d bytes (%.0fx)", perfectRLE, float64(len(data))/float64(perfectRLE))
}

// TestCLibrary_vs_PureGo_PatternRepetition compares on pattern repetition
func TestCLibrary_vs_PureGo_PatternRepetition(t *testing.T) {
	// Test data: 100KB of pattern repetition (EXACT benchmark data)
	data := generateRepeatedDataHelper(100 * 1024)

	t.Logf("=== Test Case: Pattern Repetition (100KB) ===")
	t.Logf("Pattern: \"This is a test pattern that repeats. \" (37 bytes)\n")

	// Test 1: C Library
	cCompressed, err := Compress(data)
	if err != nil {
		t.Fatalf("C library Compress failed: %v", err)
	}
	cRatio := float64(len(data)) / float64(len(cCompressed))

	// Verify decompression
	cDecompressed, err := Decompress(cCompressed)
	if err != nil {
		t.Fatalf("C library Decompress failed: %v", err)
	}
	if len(cDecompressed) != len(data) {
		t.Fatalf("C library round-trip size mismatch")
	}

	t.Logf("C Library:")
	t.Logf("  Compressed: %d bytes", len(cCompressed))
	t.Logf("  Ratio: %.0fx", cRatio)

	// Test 2: Pure-Go
	pureGoCompressed, err := purgo.CompressSmart(data)
	if err != nil {
		t.Fatalf("Pure-Go CompressSmart failed: %v", err)
	}
	pureGoRatio := float64(len(data)) / float64(len(pureGoCompressed))

	// Verify decompression
	pureGoDecompressed, err := purgo.Decompress(pureGoCompressed)
	if err != nil {
		t.Fatalf("Pure-Go Decompress failed: %v", err)
	}
	if len(pureGoDecompressed) != len(data) {
		t.Fatalf("Pure-Go round-trip size mismatch")
	}

	t.Logf("\nPure-Go:")
	t.Logf("  Compressed: %d bytes", len(pureGoCompressed))
	t.Logf("  Ratio: %.0fx", pureGoRatio)

	// Comparison
	t.Logf("\n=== Comparison ===")
	t.Logf("C Library: %d bytes (%.0fx)", len(cCompressed), cRatio)
	t.Logf("Pure-Go:   %d bytes (%.0fx)", len(pureGoCompressed), pureGoRatio)

	if len(cCompressed) < len(pureGoCompressed) {
		gap := float64(len(pureGoCompressed)) / float64(len(cCompressed))
		t.Logf("Gap: Pure-Go is %.2fx larger than C library", gap)

		// Estimate without frame overhead (Pure-Go has ~120 bytes)
		pureGoDataOnly := len(pureGoCompressed) - 120
		if pureGoDataOnly > 0 {
			adjustedGap := float64(pureGoDataOnly) / float64(len(cCompressed))
			t.Logf("Without frame overhead: %.2fx larger", adjustedGap)
		}
	} else {
		t.Logf("✅ Pure-Go matches or beats C library!")
	}
}

// TestCLibrary_vs_PureGo_AllZeros tests with all zeros
func TestCLibrary_vs_PureGo_AllZeros(t *testing.T) {
	// Test data: 100KB of all zeros
	data := make([]byte, 100*1024)

	t.Logf("=== Test Case: All Zeros (100KB) ===\n")

	// Test 1: C Library
	cCompressed, err := Compress(data)
	if err != nil {
		t.Fatalf("C library Compress failed: %v", err)
	}
	cRatio := float64(len(data)) / float64(len(cCompressed))

	t.Logf("C Library:")
	t.Logf("  Compressed: %d bytes", len(cCompressed))
	t.Logf("  Ratio: %.0fx", cRatio)

	// Test 2: Pure-Go
	pureGoCompressed, err := purgo.CompressSmart(data)
	if err != nil {
		t.Fatalf("Pure-Go CompressSmart failed: %v", err)
	}
	pureGoRatio := float64(len(data)) / float64(len(pureGoCompressed))

	t.Logf("\nPure-Go:")
	t.Logf("  Compressed: %d bytes", len(pureGoCompressed))
	t.Logf("  Ratio: %.0fx", pureGoRatio)

	// Comparison
	t.Logf("\n=== Comparison ===")
	t.Logf("C Library: %d bytes (%.0fx)", len(cCompressed), cRatio)
	t.Logf("Pure-Go:   %d bytes (%.0fx)", len(pureGoCompressed), pureGoRatio)

	perfectRLE := 8
	t.Logf("Perfect RLE: %d bytes (%.0fx)", perfectRLE, float64(len(data))/float64(perfectRLE))

	if len(cCompressed) < len(pureGoCompressed) {
		gap := float64(len(pureGoCompressed)) / float64(len(cCompressed))
		t.Logf("Gap: Pure-Go is %.2fx larger", gap)
	} else {
		t.Logf("✅ Pure-Go matches or beats C library!")
	}
}

// TestCLibrary_vs_PureGo_Summary prints comprehensive comparison
func TestCLibrary_vs_PureGo_Summary(t *testing.T) {
	tests := []struct {
		name     string
		dataFunc func() []byte
	}{
		{
			name: "All zeros (100KB)",
			dataFunc: func() []byte {
				return make([]byte, 100*1024)
			},
		},
		{
			name: "All 'A' (100KB)",
			dataFunc: func() []byte {
				data := make([]byte, 100*1024)
				for i := range data {
					data[i] = 'A'
				}
				return data
			},
		},
		{
			name: "Pattern repetition (benchmark)",
			dataFunc: func() []byte {
				return generateRepeatedDataHelper(100 * 1024)
			},
		},
	}

	t.Logf("=== C Library vs Pure-Go Comprehensive Comparison ===\n")
	t.Logf("%-35s %15s %10s %15s %10s %10s", "Test Case", "C Lib Size", "C Ratio", "Pure-Go Size", "PG Ratio", "Gap")
	t.Logf("%s", "==================================================================================================")

	for _, tt := range tests {
		data := tt.dataFunc()

		// C Library
		cCompressed, err := Compress(data)
		if err != nil {
			t.Logf("%-35s %15s %10s", tt.name, "ERROR", err.Error())
			continue
		}
		cRatio := float64(len(data)) / float64(len(cCompressed))

		// Pure-Go
		pgCompressed, err := purgo.CompressSmart(data)
		if err != nil {
			t.Logf("%-35s %15s %10.0fx %15s", tt.name, fmt.Sprintf("%d bytes", len(cCompressed)), cRatio, "ERROR")
			continue
		}
		pgRatio := float64(len(data)) / float64(len(pgCompressed))

		gap := float64(len(pgCompressed)) / float64(len(cCompressed))
		t.Logf("%-35s %12d bytes %9.0fx %12d bytes %9.0fx %9.2fx",
			tt.name, len(cCompressed), cRatio, len(pgCompressed), pgRatio, gap)
	}

	t.Logf("\n=== Key Findings ===")
	t.Logf("This table shows the ACTUAL performance gap between C library and Pure-Go")
	t.Logf("on identical test data, allowing fair apples-to-apples comparison.")
}
