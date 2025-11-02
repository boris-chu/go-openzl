//go:build cgo
// +build cgo

// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package openzl

import (
	"fmt"

	"github.com/boris-chu/go-openzl/internal/cgo"
)

// Compress compresses the input data using OpenZL with default settings.
// It returns the compressed data or an error.
//
// This is a simple one-shot compression function suitable for occasional use.
// For better performance with repeated operations, use the Compressor type.
//
// Example:
//
//	data := []byte("hello world")
//	compressed, err := openzl.Compress(data)
//	if err != nil {
//		log.Fatal(err)
//	}
func Compress(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, ErrEmptyInput
	}

	// Create compression context
	ctx, err := cgo.NewCCtx()
	if err != nil {
		return nil, fmt.Errorf("create context: %w", err)
	}
	defer ctx.Free()

	// Allocate destination buffer
	dstSize := cgo.CompressBound(len(src))
	dst := make([]byte, dstSize)

	// Compress
	n, err := ctx.Compress(dst, src)
	if err != nil {
		return nil, fmt.Errorf("compress: %w", err)
	}

	return dst[:n], nil
}

// CompressBound returns the maximum size of compressed data for input of size srcSize.
//
// Use this function to pre-allocate buffers for CompressTo():
//
//	dst := make([]byte, openzl.CompressBound(len(src)))
//	n, err := compressor.CompressTo(dst, src)
//	compressed := dst[:n]
func CompressBound(srcSize int) int {
	return cgo.CompressBound(srcSize)
}

// Decompress decompresses OpenZL-compressed data.
// It returns the decompressed data or an error.
//
// This is a simple one-shot decompression function suitable for occasional use.
// For better performance with repeated operations, use the Decompressor type.
//
// Example:
//
//	decompressed, err := openzl.Decompress(compressed)
//	if err != nil {
//		log.Fatal(err)
//	}
func Decompress(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, ErrEmptyInput
	}

	// Get decompressed size
	dstSize, err := cgo.GetDecompressedSize(src)
	if err != nil {
		return nil, fmt.Errorf("get decompressed size: %w", err)
	}

	// Allocate destination buffer
	dst := make([]byte, dstSize)

	// Create decompression context
	ctx, err := cgo.NewDCtx()
	if err != nil {
		return nil, fmt.Errorf("create context: %w", err)
	}
	defer ctx.Free()

	// Decompress
	n, err := ctx.Decompress(dst, src)
	if err != nil {
		return nil, fmt.Errorf("decompress: %w", err)
	}

	return dst[:n], nil
}
