// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package cgo

/*
#include <stdlib.h>
#include <openzl/openzl.h>
*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"
)

// TypedRef wraps the OpenZL ZL_TypedRef for typed compression.
//
// TypedRef represents a typed reference to data that OpenZL can compress
// in a format-aware manner. This allows for significantly better compression
// ratios (2-5x) on structured data compared to untyped byte compression.
//
// The TypedRef must be freed with Free() when no longer needed.
type TypedRef struct {
	ref         *C.ZL_TypedRef // Underlying OpenZL typed reference
	elementSize int            // Size of each element in bytes
}

// NewTypedRefNumeric creates a TypedRef for a numeric array.
//
// This function creates a TypedRef that references the provided numeric slice.
// OpenZL will use format-aware compression optimized for numeric data.
//
// Supported element sizes: 1, 2, 4, 8 bytes (int8, int16, int32, int64, etc.)
//
// The data slice must remain valid for the lifetime of the TypedRef.
//
// Returns an error if:
//   - data is empty
//   - element size is not supported
//   - TypedRef creation fails
func NewTypedRefNumeric[T any](data []T) (*TypedRef, error) {
	if len(data) == 0 {
		return nil, errors.New("empty data slice")
	}

	var zero T
	elementSize := int(unsafe.Sizeof(zero))

	// OpenZL supports widths of 1, 2, 4, and 8 bytes
	if elementSize != 1 && elementSize != 2 && elementSize != 4 && elementSize != 8 {
		return nil, fmt.Errorf("unsupported element size: %d (must be 1, 2, 4, or 8)", elementSize)
	}

	// Create TypedRef using OpenZL's numeric array API
	ref := C.ZL_TypedRef_createNumeric(
		unsafe.Pointer(&data[0]),
		C.size_t(elementSize),
		C.size_t(len(data)),
	)

	if ref == nil {
		return nil, errors.New("failed to create TypedRef")
	}

	return &TypedRef{
		ref:         ref,
		elementSize: elementSize,
	}, nil
}

// ElementSize returns the size of each element in bytes.
func (t *TypedRef) ElementSize() int {
	return t.elementSize
}

// Free releases the TypedRef and frees the underlying C memory.
//
// After calling Free, the TypedRef cannot be used for further operations.
// Calling Free multiple times is safe and has no effect after the first call.
func (t *TypedRef) Free() {
	if t.ref != nil {
		C.ZL_TypedRef_free(t.ref)
		t.ref = nil
	}
}

// CompressTypedRef compresses data using a TypedRef for format-aware compression.
//
// This method uses OpenZL's typed compression API which achieves significantly
// better compression ratios (2-5x) on structured data compared to untyped compression.
//
// The dst buffer must be large enough to hold the compressed data.
// Use CompressBound(srcSize) * 2 for a safe buffer size with typed compression.
//
// Returns the number of bytes written to dst on success, or an error if:
//   - dst is empty
//   - dst is too small to hold the compressed data
//   - the underlying C compression fails
func (c *CCtx) CompressTypedRef(dst []byte, tref *TypedRef) (int, error) {
	if len(dst) == 0 {
		return 0, errors.New("empty destination buffer")
	}
	if tref == nil || tref.ref == nil {
		return 0, errors.New("nil TypedRef")
	}

	// Set format version (required by OpenZL before each compression)
	result := C.ZL_CCtx_setParameter(c.ctx, C.ZL_CParam_formatVersion, C.ZL_MAX_FORMAT_VERSION)
	if C.ZL_isError(result) != 0 {
		return 0, c.getError(result)
	}

	// Compress using typed reference
	result = C.ZL_CCtx_compressTypedRef(
		c.ctx,
		unsafe.Pointer(&dst[0]),
		C.size_t(len(dst)),
		tref.ref,
	)

	if C.ZL_isError(result) != 0 {
		return 0, c.getError(result)
	}

	return int(C.ZL_validResult(result)), nil
}

// DecompressTypedToBytes decompresses data that was compressed with typed compression.
//
// This method decompresses data compressed with OpenZL's typed API and returns
// the result as a byte slice. The caller is responsible for converting the bytes
// to the appropriate typed slice.
//
// Returns the decompressed data as bytes, or an error if:
//   - src is empty
//   - src does not contain valid OpenZL compressed data
//   - the decompression operation fails
func (d *DCtx) DecompressTypedToBytes(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, errors.New("empty input")
	}

	// Get decompressed size from frame header
	dstSize, err := GetDecompressedSize(src)
	if err != nil {
		return nil, fmt.Errorf("get decompressed size: %w", err)
	}

	// Allocate byte buffer for decompression
	dstBytes := make([]byte, dstSize)

	// Decompress to byte buffer
	result := C.ZL_DCtx_decompress(
		d.ctx,
		unsafe.Pointer(&dstBytes[0]),
		C.size_t(len(dstBytes)),
		unsafe.Pointer(&src[0]),
		C.size_t(len(src)),
	)

	if C.ZL_isError(result) != 0 {
		return nil, d.getError(result)
	}

	n := int(C.ZL_validResult(result))
	return dstBytes[:n], nil
}
