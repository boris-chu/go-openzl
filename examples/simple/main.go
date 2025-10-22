// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"fmt"
	"log"

	"github.com/borischu/go-openzl"
)

func main() {
	// Original data
	data := []byte("Hello, OpenZL! This is a simple compression example.")

	fmt.Printf("Original data: %s\n", data)
	fmt.Printf("Original size: %d bytes\n\n", len(data))

	// Compress
	compressed, err := openzl.Compress(data)
	if err != nil {
		log.Fatalf("Compression failed: %v", err)
	}

	fmt.Printf("Compressed size: %d bytes\n", len(compressed))
	fmt.Printf("Compression ratio: %.2f:1\n\n", float64(len(data))/float64(len(compressed)))

	// Decompress
	decompressed, err := openzl.Decompress(compressed)
	if err != nil {
		log.Fatalf("Decompression failed: %v", err)
	}

	fmt.Printf("Decompressed data: %s\n", decompressed)
	fmt.Printf("Decompressed size: %d bytes\n", len(decompressed))

	// Verify
	if string(data) == string(decompressed) {
		fmt.Println("\n✓ Round-trip successful!")
	} else {
		log.Fatal("✗ Round-trip failed - data doesn't match!")
	}
}
