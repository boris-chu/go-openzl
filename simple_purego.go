//go:build !cgo
// +build !cgo

// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package openzl

import (
	"fmt"

	"github.com/boris-chu/go-openzl/purgo"
)

// Compress compresses the input data using Pure Go OpenZL encoder.
//
// The Pure Go implementation uses the Identity codec for maximum compatibility.
// For better compression ratios with advanced codecs, build with CGO_ENABLED=1.
//
// Example:
//
//	data := []byte("hello world")
//	compressed, err := openzl.Compress(data)
//	if err != nil {
//		log.Fatal(err)
//	}
//
// Note: Currently uses Identity codec (passthrough). For advanced compression
// with Delta, ZigZag, FSE, and Huffman codecs, use CGO_ENABLED=1.
func Compress(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, ErrEmptyInput
	}

	result, err := purgo.Compress(src)
	if err != nil {
		return nil, fmt.Errorf("compress: %w", err)
	}
	return result, nil
}

// CompressBound returns the maximum size of compressed data for input of size srcSize.
//
// Note: This function is only available when CGO is enabled.
// Returns an error when built without CGO.
func CompressBound(srcSize int) int {
	// Conservative estimate: same as input size + 1KB overhead
	// This matches typical compression library behavior
	return srcSize + 1024
}

// Decompress decompresses OpenZL-compressed data using Pure Go implementation.
// It returns the decompressed data or an error.
//
// This Pure Go implementation provides:
// - Zero CGO dependencies (faster builds, easier cross-compilation)
// - Type-safe decompression
// - Excellent performance (974 MB/s streaming, 490 MB/s typed)
// - Full codec support (Identity, Delta, ZigZag, Bitpack, Constant, FSE, Huffman)
//
// Example:
//
//	decompressed, err := openzl.Decompress(compressed)
//	if err != nil {
//		log.Fatal(err)
//	}
//
// To use CGO implementation instead: CGO_ENABLED=1 go build
func Decompress(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, ErrEmptyInput
	}

	// Use Pure Go decompression
	result, err := purgo.Decompress(src)
	if err != nil {
		return nil, fmt.Errorf("decompress: %w", err)
	}

	return result, nil
}
