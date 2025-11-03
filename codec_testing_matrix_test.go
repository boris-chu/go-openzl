// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

//go:build cgo

package openzl

import (
	"testing"

	"github.com/boris-chu/go-openzl/purgo"
)

// TestAllCodecs_vs_CLibrary tests all Pure-Go codecs against C library
//
// This provides a comprehensive comparison matrix showing which codecs
// we've implemented and how they perform vs C library.
func TestAllCodecs_vs_CLibrary(t *testing.T) {
	tests := []struct {
		name     string
		dataFunc func() []byte
		codecs   []string
	}{
		{
			name: "True repetition (all zeros)",
			dataFunc: func() []byte {
				return make([]byte, 100*1024)
			},
			codecs: []string{"RLE", "Constant"},
		},
		{
			name: "Sequential integers (1,2,3...)",
			dataFunc: func() []byte {
				data := make([]byte, 100*1024)
				for i := range data {
					data[i] = byte(i % 256)
				}
				return data
			},
			codecs: []string{"Delta", "ZigZag", "Bitpack"},
		},
		{
			name: "Pattern repetition",
			dataFunc: func() []byte {
				pattern := []byte("This is a test pattern that repeats. ")
				data := make([]byte, 0, 100*1024)
				for len(data) < 100*1024 {
					data = append(data, pattern...)
				}
				return data[:100*1024]
			},
			codecs: []string{"LZ77", "FSE", "Huffman"},
		},
		{
			name: "Transposed data (column-major)",
			dataFunc: func() []byte {
				// Simulate 1000 uint64s that would benefit from transpose
				data := make([]byte, 8*1000)
				for i := 0; i < 1000; i++ {
					// Pattern: high bytes similar, low bytes vary
					data[i*8+0] = byte(i % 256)
					data[i*8+1] = byte(i % 256)
					data[i*8+2] = 0xAA
					data[i*8+3] = 0xBB
					data[i*8+4] = 0xCC
					data[i*8+5] = 0xDD
					data[i*8+6] = 0xEE
					data[i*8+7] = 0xFF
				}
				return data
			},
			codecs: []string{"Transpose"},
		},
	}

	t.Logf("=== Codec Testing Matrix: Pure-Go vs C Library ===\n")
	t.Logf("%-35s %15s %10s %15s %10s %10s %20s",
		"Test Case", "C Lib Size", "C Ratio", "Pure-Go Size", "PG Ratio", "Gap", "Applicable Codecs")
	t.Logf("%s", "=========================================================================================================")

	for _, tt := range tests {
		data := tt.dataFunc()

		// C Library
		cCompressed, err := Compress(data)
		if err != nil {
			t.Logf("%-35s %15s %10s %15s %10s %10s %20s",
				tt.name, "ERROR", "-", "-", "-", "-", "-")
			continue
		}
		cRatio := float64(len(data)) / float64(len(cCompressed))

		// Pure-Go
		pgCompressed, err := purgo.CompressSmart(data)
		if err != nil {
			t.Logf("%-35s %12d bytes %9.0fx %15s %10s %10s %20s",
				tt.name, len(cCompressed), cRatio, "ERROR", "-", "-", "-")
			continue
		}
		pgRatio := float64(len(data)) / float64(len(pgCompressed))

		gap := float64(len(pgCompressed)) / float64(len(cCompressed))
		codecList := ""
		for i, c := range tt.codecs {
			if i > 0 {
				codecList += ", "
			}
			codecList += c
		}

		t.Logf("%-35s %12d bytes %9.0fx %12d bytes %9.0fx %9.2fx %20s",
			tt.name, len(cCompressed), cRatio, len(pgCompressed), pgRatio, gap, codecList)
	}

	t.Logf("\n=== Codec Implementation Status ===")
	t.Logf("\nImplemented Pure-Go Codecs:")
	t.Logf("  ✅ RLE - Run-Length Encoding")
	t.Logf("  ✅ Delta - Delta encoding (SIMD-optimized)")
	t.Logf("  ✅ ZigZag - ZigZag encoding for signed integers")
	t.Logf("  ✅ Bitpack - Bit packing")
	t.Logf("  ✅ Constant - Constant value detection")
	t.Logf("  ✅ Transpose - Byte transposition")
	t.Logf("  ✅ LZ77 - Dictionary compression (via Klaus Post)")
	t.Logf("  ✅ Huffman - Entropy coding (via Klaus Post)")
	t.Logf("  ✅ FSE - Finite State Entropy (via Klaus Post)")
	t.Logf("  ✅ Identity - Pass-through codec")

	t.Logf("\nC Library Codecs Not Yet Tested:")
	t.Logf("  ⚠️  ROLZ - Reduced Offset LZ")
	t.Logf("  ⚠️  Field LZ - Field-aware LZ")
	t.Logf("  ⚠️  Zstd - Zstandard codec")
	t.Logf("  ⚠️  Quantize - Quantization codec")
	t.Logf("  ⚠️  Float Deconstruct - Float decomposition")
	t.Logf("  ⚠️  And ~20 more specialized codecs")

	t.Logf("\n=== Key Findings ===")
	t.Logf("Run this test to see which codecs are competitive and which need work!")
}

// TestIndividualCodec_RLE tests RLE specifically
func TestIndividualCodec_RLE(t *testing.T) {
	t.Skip("RLE testing integrated into main comparison - see TestCLibrary_vs_PureGo_TrueRepetition")
}

// TestIndividualCodec_Delta tests Delta encoding
func TestIndividualCodec_Delta(t *testing.T) {
	// Sequential data - ideal for Delta
	data := make([]byte, 10000)
	for i := range data {
		data[i] = byte(i % 256)
	}

	t.Logf("=== Delta Codec Test ===")
	t.Logf("Input: Sequential bytes 0-255 repeated, %d bytes", len(data))

	// C Library
	cCompressed, err := Compress(data)
	if err != nil {
		t.Fatalf("C library Compress failed: %v", err)
	}
	cRatio := float64(len(data)) / float64(len(cCompressed))

	// Pure-Go (CompressSmart will choose codec)
	pgCompressed, err := purgo.CompressSmart(data)
	if err != nil {
		t.Fatalf("Pure-Go CompressSmart failed: %v", err)
	}
	pgRatio := float64(len(data)) / float64(len(pgCompressed))

	t.Logf("C Library: %d bytes (%.0fx)", len(cCompressed), cRatio)
	t.Logf("Pure-Go:   %d bytes (%.0fx)", len(pgCompressed), pgRatio)

	if len(pgCompressed) > len(cCompressed) {
		gap := float64(len(pgCompressed)) / float64(len(cCompressed))
		t.Logf("Gap: %.2fx larger", gap)
	} else {
		t.Logf("✅ Pure-Go matches or beats C library!")
	}
}

// TestIndividualCodec_Transpose tests Transpose codec
func TestIndividualCodec_Transpose(t *testing.T) {
	// Column-major data that benefits from transpose
	const numInts = 1000
	data := make([]byte, numInts*8)

	// Create pattern where high bytes are constant
	for i := 0; i < numInts; i++ {
		data[i*8+0] = byte(i % 256)
		data[i*8+1] = byte(i / 256)
		data[i*8+2] = 0xAA
		data[i*8+3] = 0xBB
		data[i*8+4] = 0xCC
		data[i*8+5] = 0xDD
		data[i*8+6] = 0xEE
		data[i*8+7] = 0xFF
	}

	t.Logf("=== Transpose Codec Test ===")
	t.Logf("Input: %d uint64s with constant high bytes, %d bytes", numInts, len(data))

	cCompressed, err := Compress(data)
	if err != nil {
		t.Fatalf("C library Compress failed: %v", err)
	}
	cRatio := float64(len(data)) / float64(len(cCompressed))

	pgCompressed, err := purgo.CompressSmart(data)
	if err != nil {
		t.Fatalf("Pure-Go CompressSmart failed: %v", err)
	}
	pgRatio := float64(len(data)) / float64(len(pgCompressed))

	t.Logf("C Library: %d bytes (%.0fx)", len(cCompressed), cRatio)
	t.Logf("Pure-Go:   %d bytes (%.0fx)", len(pgCompressed), pgRatio)

	if len(pgCompressed) > len(cCompressed) {
		gap := float64(len(pgCompressed)) / float64(len(cCompressed))
		t.Logf("Gap: %.2fx larger", gap)
	} else {
		t.Logf("✅ Pure-Go matches or beats C library!")
	}
}
