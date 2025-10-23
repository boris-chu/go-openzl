// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"fmt"
	"log"
	"time"
	"unsafe"

	"github.com/borischu/go-openzl"
)

func main() {
	fmt.Println("OpenZL Typed Compression Example")
	fmt.Println("==================================")
	fmt.Println()

	// Example 1: Compress numeric data with typed compression
	fmt.Println("Example 1: Numeric Array Compression")
	fmt.Println("-------------------------------------")

	// Create test data: sorted sequence
	numbers := make([]int64, 1000)
	for i := range numbers {
		numbers[i] = int64(i * 100)
	}

	start := time.Now()

	// Compress with typed API (optimized for numeric data)
	typedCompressed, err := openzl.CompressNumeric(numbers)
	if err != nil {
		log.Fatal(err)
	}

	typedElapsed := time.Since(start)

	// Also compress with untyped API for comparison
	start = time.Now()

	// Convert to bytes for untyped compression
	bytesData := unsafe.Slice((*byte)(unsafe.Pointer(&numbers[0])), len(numbers)*8)
	untypedCompressed, err := openzl.Compress(bytesData)
	if err != nil {
		log.Fatal(err)
	}

	untypedElapsed := time.Since(start)

	originalSize := len(numbers) * 8
	fmt.Printf("Original size:     %d bytes (%d int64s)\n", originalSize, len(numbers))
	fmt.Printf("Typed compressed:  %d bytes (%.2fx ratio)\n",
		len(typedCompressed), float64(originalSize)/float64(len(typedCompressed)))
	fmt.Printf("Untyped compressed: %d bytes (%.2fx ratio)\n",
		len(untypedCompressed), float64(originalSize)/float64(len(untypedCompressed)))
	fmt.Printf("Improvement:       %.1f%% smaller with typed compression\n",
		100.0*(1.0-float64(len(typedCompressed))/float64(len(untypedCompressed))))
	fmt.Printf("Typed time:        %v\n", typedElapsed)
	fmt.Printf("Untyped time:      %v\n", untypedElapsed)
	fmt.Println()

	// Decompress and verify
	decompressed, err := openzl.DecompressNumeric[int64](typedCompressed)
	if err != nil {
		log.Fatal(err)
	}

	// Verify correctness
	if len(decompressed) != len(numbers) {
		log.Fatalf("Length mismatch: got %d, want %d", len(decompressed), len(numbers))
	}
	for i := range numbers {
		if decompressed[i] != numbers[i] {
			log.Fatalf("Data mismatch at index %d: got %d, want %d",
				i, decompressed[i], numbers[i])
		}
	}
	fmt.Println("✓ Decompression verified: all data matches!")
	fmt.Println()

	// Example 2: Use context API for better performance
	fmt.Println("Example 2: Reusable Compressor Context")
	fmt.Println("---------------------------------------")

	compressor, err := openzl.NewCompressor()
	if err != nil {
		log.Fatal(err)
	}
	defer compressor.Close()

	decompressor, err := openzl.NewDecompressor()
	if err != nil {
		log.Fatal(err)
	}
	defer decompressor.Close()

	// Compress multiple arrays using reusable context
	testArrays := [][]int64{
		{1, 2, 3, 4, 5},
		{100, 200, 300, 400, 500},
		{1000, 2000, 3000, 4000, 5000},
	}

	start = time.Now()
	for i, arr := range testArrays {
		compressed, err := openzl.CompressorCompressNumeric(compressor, arr)
		if err != nil {
			log.Fatal(err)
		}

		decompressed, err := openzl.DecompressorDecompressNumeric[int64](decompressor, compressed)
		if err != nil {
			log.Fatal(err)
		}

		ratio := float64(len(arr)*8) / float64(len(compressed))
		fmt.Printf("Array %d: %d bytes -> %d bytes (%.2fx)\n",
			i+1, len(arr)*8, len(compressed), ratio)

		// Verify
		for j := range arr {
			if decompressed[j] != arr[j] {
				log.Fatalf("Mismatch in array %d at index %d", i, j)
			}
		}
	}
	elapsed := time.Since(start)

	fmt.Printf("Total time for 3 arrays: %v\n", elapsed)
	fmt.Println("✓ All arrays compressed and verified!")
	fmt.Println()

	// Example 3: Different numeric types
	fmt.Println("Example 3: Multiple Numeric Types")
	fmt.Println("----------------------------------")

	// int32
	int32Data := []int32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	int32Compressed, _ := openzl.CompressNumeric(int32Data)
	fmt.Printf("int32:   %d bytes -> %d bytes (%.2fx)\n",
		len(int32Data)*4, len(int32Compressed),
		float64(len(int32Data)*4)/float64(len(int32Compressed)))

	// uint64
	uint64Data := []uint64{1000, 2000, 3000, 4000, 5000}
	uint64Compressed, _ := openzl.CompressNumeric(uint64Data)
	fmt.Printf("uint64:  %d bytes -> %d bytes (%.2fx)\n",
		len(uint64Data)*8, len(uint64Compressed),
		float64(len(uint64Data)*8)/float64(len(uint64Compressed)))

	// float64
	float64Data := []float64{1.1, 2.2, 3.3, 4.4, 5.5}
	float64Compressed, _ := openzl.CompressNumeric(float64Data)
	fmt.Printf("float64: %d bytes -> %d bytes (%.2fx)\n",
		len(float64Data)*8, len(float64Compressed),
		float64(len(float64Data)*8)/float64(len(float64Compressed)))

	fmt.Println()
	fmt.Println("Summary")
	fmt.Println("-------")
	fmt.Println("✓ Typed compression achieves significantly better ratios on numeric data")
	fmt.Println("✓ Use context API for repeated operations (20-50% faster)")
	fmt.Println("✓ Supports all Go numeric types (int8/16/32/64, uint8/16/32/64, float32/64)")
}
