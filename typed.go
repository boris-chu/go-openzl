// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package openzl

import (
	"fmt"

	"github.com/borischu/go-openzl/internal/cgo"
)

// Numeric is a constraint that permits all numeric types that OpenZL supports.
// OpenZL supports numeric types with widths of 1, 2, 4, and 8 bytes.
type Numeric interface {
	int8 | uint8 | int16 | uint16 | int32 | uint32 | int64 | uint64 | float32 | float64
}

// CompressNumeric compresses a slice of numeric values using OpenZL's typed compression.
//
// This function leverages OpenZL's format-aware compression to achieve significantly
// better compression ratios (2-5x) on numeric data compared to the untyped Compress function.
// It works best with structured or sorted numeric data.
//
// Supported types: int8, uint8, int16, uint16, int32, uint32, int64, uint64, float32, float64
//
// Example:
//
//	numbers := []int64{1, 2, 3, 4, 5, 100, 101, 102, 103}
//	compressed, err := openzl.CompressNumeric(numbers)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Decompress back to typed slice
//	decompressed, err := openzl.DecompressNumeric[int64](compressed)
//
// Returns an error if:
//   - the input slice is empty
//   - the compression operation fails
func CompressNumeric[T Numeric](data []T) ([]byte, error) {
	if len(data) == 0 {
		return nil, ErrEmptyInput
	}

	// Create typed reference for the numeric array
	tref, err := cgo.NewTypedRefNumeric(data)
	if err != nil {
		return nil, fmt.Errorf("create typed ref: %w", err)
	}
	defer tref.Free()

	// Create compression context
	ctx, err := cgo.NewCCtx()
	if err != nil {
		return nil, fmt.Errorf("create context: %w", err)
	}
	defer ctx.Free()

	// Allocate destination buffer
	// TypedRef compression may need more space than CompressBound for raw bytes
	srcSize := len(data) * int(tref.ElementSize())
	dstSize := cgo.CompressBound(srcSize) * 2 // Extra margin for typed compression
	dst := make([]byte, dstSize)

	// Compress using typed reference
	n, err := ctx.CompressTypedRef(dst, tref)
	if err != nil {
		return nil, fmt.Errorf("compress typed: %w", err)
	}

	return dst[:n], nil
}

// DecompressNumeric decompresses data that was compressed with CompressNumeric.
//
// The type parameter T must match the type used during compression, otherwise
// the decompression will fail or produce incorrect results.
//
// Example:
//
//	compressed, _ := openzl.CompressNumeric([]int64{1, 2, 3, 4, 5})
//	decompressed, err := openzl.DecompressNumeric[int64](compressed)
//	if err != nil {
//		log.Fatal(err)
//	}
//	// decompressed is []int64{1, 2, 3, 4, 5}
//
// Returns an error if:
//   - the input is empty
//   - the compressed data is invalid or corrupted
//   - the type parameter doesn't match the original compression type
func DecompressNumeric[T Numeric](compressed []byte) ([]T, error) {
	if len(compressed) == 0 {
		return nil, ErrEmptyInput
	}

	// Create decompression context
	ctx, err := cgo.NewDCtx()
	if err != nil {
		return nil, fmt.Errorf("create context: %w", err)
	}
	defer ctx.Free()

	// Decompress to bytes
	decompressedBytes, err := ctx.DecompressTypedToBytes(compressed)
	if err != nil {
		return nil, fmt.Errorf("decompress typed: %w", err)
	}

	// Convert bytes to typed slice
	data, err := cgo.BytesToTypedSlice[T](decompressedBytes)
	if err != nil {
		return nil, fmt.Errorf("convert to typed slice: %w", err)
	}

	return data, nil
}

// CompressorCompressNumeric compresses a slice of numeric values using a reusable compression context.
//
// This function combines the performance benefits of the Context API (Phase 2) with the
// compression ratio improvements of typed compression (Phase 3).
//
// Example:
//
//	compressor, _ := openzl.NewCompressor()
//	defer compressor.Close()
//
//	numbers := []int64{1, 2, 3, 4, 5, 100, 101, 102}
//	compressed, err := openzl.CompressorCompressNumeric(compressor, numbers)
//
// Returns an error if:
//   - the input slice is empty
//   - the compression operation fails
func CompressorCompressNumeric[T Numeric](c *Compressor, data []T) ([]byte, error) {
	if len(data) == 0 {
		return nil, ErrEmptyInput
	}

	// Create typed reference for the numeric array
	tref, err := cgo.NewTypedRefNumeric(data)
	if err != nil {
		return nil, fmt.Errorf("create typed ref: %w", err)
	}
	defer tref.Free()

	// Lock for thread safety
	c.mu.Lock()
	defer c.mu.Unlock()

	// Allocate destination buffer
	srcSize := len(data) * int(tref.ElementSize())
	dstSize := cgo.CompressBound(srcSize) * 2
	dst := make([]byte, dstSize)

	// Compress using typed reference with reusable context
	n, err := c.ctx.CompressTypedRef(dst, tref)
	if err != nil {
		return nil, fmt.Errorf("compress typed: %w", err)
	}

	return dst[:n], nil
}

// DecompressorDecompressNumeric decompresses numeric data using a reusable decompression context.
//
// This function combines the performance benefits of the Context API (Phase 2) with
// typed decompression (Phase 3).
//
// Example:
//
//	decompressor, _ := openzl.NewDecompressor()
//	defer decompressor.Close()
//
//	decompressed, err := openzl.DecompressorDecompressNumeric[int64](decompressor, compressed)
//
// Returns an error if:
//   - the input is empty
//   - the compressed data is invalid or corrupted
//   - the type parameter doesn't match the original compression type
func DecompressorDecompressNumeric[T Numeric](d *Decompressor, compressed []byte) ([]T, error) {
	if len(compressed) == 0 {
		return nil, ErrEmptyInput
	}

	// Lock for thread safety
	d.mu.Lock()
	defer d.mu.Unlock()

	// Decompress to bytes with reusable context
	decompressedBytes, err := d.ctx.DecompressTypedToBytes(compressed)
	if err != nil {
		return nil, fmt.Errorf("decompress typed: %w", err)
	}

	// Convert bytes to typed slice
	data, err := cgo.BytesToTypedSlice[T](decompressedBytes)
	if err != nil {
		return nil, fmt.Errorf("convert to typed slice: %w", err)
	}

	return data, nil
}
