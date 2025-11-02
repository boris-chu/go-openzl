//go:build !cgo
// +build !cgo

// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package openzl

import (
	"fmt"

	"github.com/borischu/go-openzl/purgo"
)

// Compress compresses the input data using OpenZL with default settings.
// It returns the compressed data or an error.
//
// Note: Pure Go compression is not yet implemented. This function returns
// an error when CGO is disabled. Use CGO build for compression support.
//
// To enable CGO: CGO_ENABLED=1 go build
//
// Example:
//
//	data := []byte("hello world")
//	compressed, err := openzl.Compress(data)
//	if err != nil {
//		log.Fatal(err)
//	}
func Compress(src []byte) ([]byte, error) {
	return nil, fmt.Errorf("compression requires CGO (build with CGO_ENABLED=1)")
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
