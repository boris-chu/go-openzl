//go:build !cgo
// +build !cgo

// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package openzl

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/boris-chu/go-openzl/purgo"
)

// Numeric is a constraint that permits all numeric types that OpenZL supports.
// OpenZL supports numeric types with widths of 1, 2, 4, and 8 bytes.
type Numeric interface {
	int8 | uint8 | int16 | uint16 | int32 | uint32 | int64 | uint64 | float32 | float64
}

// CompressNumeric compresses a slice of numeric values using Pure Go OpenZL encoder.
//
// The Pure Go implementation uses the Identity codec. For better compression
// ratios with Delta, ZigZag, and entropy coding, build with CGO_ENABLED=1.
//
// Example:
//
//	numbers := []int64{1, 2, 3, 4, 5}
//	compressed, err := openzl.CompressNumeric(numbers)
//	if err != nil {
//		log.Fatal(err)
//	}
func CompressNumeric[T Numeric](data []T) ([]byte, error) {
	if len(data) == 0 {
		return nil, ErrEmptyInput
	}

	// Convert typed slice to bytes
	buf := new(bytes.Buffer)
	for _, val := range data {
		if err := binary.Write(buf, binary.LittleEndian, val); err != nil {
			return nil, fmt.Errorf("write element: %w", err)
		}
	}

	// Compress the bytes
	result, err := purgo.Compress(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("compress: %w", err)
	}
	return result, nil
}

// DecompressNumeric decompresses data that was compressed with CompressNumeric.
//
// This function uses the Pure Go decoder when CGO is disabled.
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

	// Use Pure Go decoder from purgo package
	// Decompress to raw bytes first
	rawBytes, err := purgo.Decompress(compressed)
	if err != nil {
		return nil, fmt.Errorf("decompress: %w", err)
	}

	// Convert bytes to typed slice
	var elemSize int
	var dummy T
	switch any(dummy).(type) {
	case int8, uint8:
		elemSize = 1
	case int16, uint16:
		elemSize = 2
	case int32, uint32, float32:
		elemSize = 4
	case int64, uint64, float64:
		elemSize = 8
	default:
		return nil, fmt.Errorf("unsupported type")
	}

	// Verify size is multiple of element size
	if len(rawBytes)%elemSize != 0 {
		return nil, fmt.Errorf("decompressed size %d not multiple of element size %d", len(rawBytes), elemSize)
	}

	// Convert bytes to typed slice
	count := len(rawBytes) / elemSize
	result := make([]T, count)

	reader := bytes.NewReader(rawBytes)
	for i := 0; i < count; i++ {
		if err := binary.Read(reader, binary.LittleEndian, &result[i]); err != nil {
			return nil, fmt.Errorf("read element at index %d failed: %w", i, err)
		}
	}

	return result, nil
}

// CompressorCompressNumeric compresses a slice of numeric values using a reusable compression context.
//
// Note: Compression requires CGO. Build with CGO_ENABLED=1 to use this function.
func CompressorCompressNumeric[T Numeric](c *Compressor, data []T) ([]byte, error) {
	return nil, fmt.Errorf("typed compression requires CGO (build with CGO_ENABLED=1)")
}

// DecompressorDecompressNumeric decompresses numeric data using a reusable decompression context.
//
// Note: This function is not available in Pure Go builds because Decompressor requires CGO.
// For Pure Go decompression, use the one-shot DecompressNumeric function instead.
func DecompressorDecompressNumeric[T Numeric](d *Decompressor, compressed []byte) ([]T, error) {
	return nil, fmt.Errorf("Decompressor requires CGO (use DecompressNumeric instead, or build with CGO_ENABLED=1)")
}
