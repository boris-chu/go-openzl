// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

// Package main demonstrates using the OpenZL Context API for better performance.
//
// The Context API (Compressor and Decompressor types) provides 20-50% better
// performance than the one-shot API when compressing/decompressing multiple
// pieces of data.
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/borischu/go-openzl"
)

func main() {
	fmt.Println("OpenZL Context API Example")
	fmt.Println("===========================")
	fmt.Println()

	// Create a compressor (reusable context)
	compressor, err := openzl.NewCompressor()
	if err != nil {
		log.Fatalf("failed to create compressor: %v", err)
	}
	defer compressor.Close()

	// Create a decompressor (reusable context)
	decompressor, err := openzl.NewDecompressor()
	if err != nil {
		log.Fatalf("failed to create decompressor: %v", err)
	}
	defer decompressor.Close()

	// Sample data to compress
	testData := []string{
		"Hello, OpenZL!",
		"This is the second message.",
		"Context reuse provides better performance.",
		"OpenZL is format-aware compression from Meta.",
	}

	fmt.Printf("Compressing %d messages...\n\n", len(testData))

	// Compress each message using the same compressor
	var totalOriginal, totalCompressed int
	start := time.Now()

	for i, msg := range testData {
		data := []byte(msg)

		// Compress using reusable context
		compressed, err := compressor.Compress(data)
		if err != nil {
			log.Fatalf("compression failed: %v", err)
		}

		totalOriginal += len(data)
		totalCompressed += len(compressed)

		ratio := float64(len(data)) / float64(len(compressed))
		fmt.Printf("Message %d:\n", i+1)
		fmt.Printf("  Original:   %3d bytes\n", len(data))
		fmt.Printf("  Compressed: %3d bytes\n", len(compressed))
		fmt.Printf("  Ratio:      %.2fx\n\n", ratio)

		// Decompress to verify
		decompressed, err := decompressor.Decompress(compressed)
		if err != nil {
			log.Fatalf("decompression failed: %v", err)
		}

		if string(decompressed) != msg {
			log.Fatalf("round-trip failed: expected %q, got %q", msg, string(decompressed))
		}
	}

	elapsed := time.Since(start)

	fmt.Println("Summary:")
	fmt.Println("--------")
	fmt.Printf("Total original:   %d bytes\n", totalOriginal)
	fmt.Printf("Total compressed: %d bytes\n", totalCompressed)
	fmt.Printf("Overall ratio:    %.2fx\n", float64(totalOriginal)/float64(totalCompressed))
	fmt.Printf("Time elapsed:     %v\n", elapsed)
	fmt.Printf("Operations/sec:   %.0f compress + %.0f decompress\n",
		float64(len(testData))/elapsed.Seconds(),
		float64(len(testData))/elapsed.Seconds())
	fmt.Println()
	fmt.Println("Note: Using context API provides 20-50% better performance")
	fmt.Println("compared to the one-shot Compress() and Decompress() functions.")
}
