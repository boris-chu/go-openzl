// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package cgo

/*
#include <stdlib.h>
#include <openzl/openzl.h>
#include <openzl/codecs/zl_generic.h>

// Simple graph function that returns the ZL_GRAPH_NUMERIC graph for numeric compression
ZL_GraphID numericGraphFn(ZL_Compressor* compressor) {
    (void)compressor; // unused
    return ZL_GRAPH_NUMERIC;
}

// Helper to get the numeric graph function pointer
ZL_GraphFn getNumericGraphFn() {
    return numericGraphFn;
}
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
// Based on OpenZL's test_generic_clustering.cpp, typed compression requires:
// 1. Creating a ZL_Compressor graph object
// 2. Linking it to the context with ZL_CCtx_refCompressor()
// 3. Then calling ZL_CCtx_compressTypedRef()
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

	// Create a compression graph (required for typed compression)
	// This is what we were missing! Found in test_generic_clustering.cpp
	compressor := C.ZL_Compressor_create()
	if compressor == nil {
		return 0, errors.New("failed to create ZL_Compressor")
	}
	defer C.ZL_Compressor_free(compressor)

	// Initialize the compressor with the numeric graph function
	// This sets up the graph structure needed for typed compression
	result := C.ZL_Compressor_initUsingGraphFn(compressor, C.getNumericGraphFn())
	if C.ZL_isError(result) != 0 {
		return 0, c.getError(result)
	}

	// Reset parameters to clean state before typed compression
	result = C.ZL_CCtx_resetParameters(c.ctx)
	if C.ZL_isError(result) != 0 {
		return 0, c.getError(result)
	}

	// Set format version (required by OpenZL before each compression)
	result = C.ZL_CCtx_setParameter(c.ctx, C.ZL_CParam_formatVersion, C.ZL_MAX_FORMAT_VERSION)
	if C.ZL_isError(result) != 0 {
		return 0, c.getError(result)
	}

	// Link the compression context to the compressor graph
	// This is the critical missing step discovered from OpenZL examples!
	result = C.ZL_CCtx_refCompressor(c.ctx, compressor)
	if C.ZL_isError(result) != 0 {
		return 0, c.getError(result)
	}

	// Compress using typed reference (should now work!)
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
// For typed compression, we must use ZL_DCtx_decompressTyped() instead of
// ZL_DCtx_decompress(). This is the correct way to decompress typed data.
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

	// Output info structure to receive type information
	var outInfo C.ZL_OutputInfo

	// Decompress typed data using the proper typed decompression function
	// This is required for data compressed with ZL_CCtx_compressTypedRef()
	result := C.ZL_DCtx_decompressTyped(
		d.ctx,
		&outInfo,
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
