// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package cgo

/*
#cgo CFLAGS: -I${SRCDIR}/../../vendor/openzl/include
#cgo LDFLAGS: ${SRCDIR}/../../vendor/openzl/lib/libopenzl.a ${SRCDIR}/../../vendor/openzl/lib/libzstd.a -lm -lpthread
#include <stdlib.h>
#include <openzl/openzl.h>
*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"
)

// CCtx wraps the OpenZL C compression context (ZL_CCtx).
//
// This type provides a thin Go wrapper around the underlying C compression
// context, handling memory management and error translation.
//
// The context must be freed with Free() when no longer needed to avoid
// memory leaks.
type CCtx struct {
	ctx *C.ZL_CCtx // Underlying OpenZL compression context
}

// NewCCtx creates a new compression context.
//
// The returned context is configured with OpenZL's maximum format version
// to ensure compatibility with all features. The context must be freed with
// Free() when no longer needed.
//
// Returns an error if the underlying C context creation fails or if
// the format version cannot be set.
func NewCCtx() (*CCtx, error) {
	ctx := C.ZL_CCtx_create()
	if ctx == nil {
		return nil, errors.New("failed to create compression context")
	}

	// Set default format version (required by OpenZL)
	result := C.ZL_CCtx_setParameter(ctx, C.ZL_CParam_formatVersion, C.ZL_MAX_FORMAT_VERSION)
	if C.ZL_isError(result) != 0 {
		C.ZL_CCtx_free(ctx)
		errCode := C.ZL_errorCode(result)
		errName := C.GoString(C.ZL_ErrorCode_toString(errCode))
		return nil, fmt.Errorf("set format version: %s", errName)
	}

	return &CCtx{ctx: ctx}, nil
}

// Free releases the compression context and frees the underlying C memory.
//
// After calling Free, the context cannot be used for further operations.
// Calling Free multiple times is safe and has no effect after the first call.
func (c *CCtx) Free() {
	if c.ctx != nil {
		C.ZL_CCtx_free(c.ctx)
		c.ctx = nil
	}
}

// Compress compresses src into dst using the OpenZL C API.
//
// The dst buffer must be large enough to hold the compressed data.
// Use CompressBound to determine the required buffer size.
//
// This method directly calls ZL_CCtx_compress from the OpenZL C library,
// passing Go slice pointers to C using unsafe.Pointer. Both src and dst
// must be non-empty.
//
// Returns the number of bytes written to dst on success, or an error if:
//   - src or dst is empty
//   - dst is too small to hold the compressed data
//   - the underlying C compression fails
func (c *CCtx) Compress(dst, src []byte) (int, error) {
	if len(src) == 0 {
		return 0, errors.New("empty input")
	}
	if len(dst) == 0 {
		return 0, errors.New("empty destination buffer")
	}

	result := C.ZL_CCtx_compress(
		c.ctx,
		unsafe.Pointer(&dst[0]),
		C.size_t(len(dst)),
		unsafe.Pointer(&src[0]),
		C.size_t(len(src)),
	)

	if C.ZL_isError(result) != 0 {
		return 0, c.getError(result)
	}

	return int(C.ZL_validResult(result)), nil
}

// getError translates an OpenZL C error Result into a Go error.
//
// OpenZL uses a Result type (ZL_Report) that can contain either a value
// or an error code. This method extracts the error code and converts it
// to a human-readable error message using OpenZL's error string function.
func (c *CCtx) getError(result C.ZL_Report) error {
	errCode := C.ZL_errorCode(result)
	errName := C.GoString(C.ZL_ErrorCode_toString(errCode))
	return fmt.Errorf("openzl: %s", errName)
}

// DCtx wraps the OpenZL C decompression context (ZL_DCtx).
//
// This type provides a thin Go wrapper around the underlying C decompression
// context, handling memory management and error translation.
//
// The context must be freed with Free() when no longer needed to avoid
// memory leaks.
type DCtx struct {
	ctx *C.ZL_DCtx // Underlying OpenZL decompression context
}

// NewDCtx creates a new decompression context.
//
// The returned context can be reused for multiple decompression operations.
// The context must be freed with Free() when no longer needed.
//
// Returns an error if the underlying C context creation fails.
func NewDCtx() (*DCtx, error) {
	ctx := C.ZL_DCtx_create()
	if ctx == nil {
		return nil, errors.New("failed to create decompression context")
	}
	return &DCtx{ctx: ctx}, nil
}

// Free releases the decompression context and frees the underlying C memory.
//
// After calling Free, the context cannot be used for further operations.
// Calling Free multiple times is safe and has no effect after the first call.
func (d *DCtx) Free() {
	if d.ctx != nil {
		C.ZL_DCtx_free(d.ctx)
		d.ctx = nil
	}
}

// Decompress decompresses src into dst using the OpenZL C API.
//
// The dst buffer must be large enough to hold the decompressed data.
// Use GetDecompressedSize to determine the required buffer size before
// calling this method.
//
// This method directly calls ZL_DCtx_decompress from the OpenZL C library,
// passing Go slice pointers to C using unsafe.Pointer. Both src and dst
// must be non-empty.
//
// Returns the number of bytes written to dst on success, or an error if:
//   - src or dst is empty
//   - dst is too small to hold the decompressed data
//   - src contains invalid or corrupted compressed data
//   - the underlying C decompression fails
func (d *DCtx) Decompress(dst, src []byte) (int, error) {
	if len(src) == 0 {
		return 0, errors.New("empty input")
	}
	if len(dst) == 0 {
		return 0, errors.New("empty destination buffer")
	}

	result := C.ZL_DCtx_decompress(
		d.ctx,
		unsafe.Pointer(&dst[0]),
		C.size_t(len(dst)),
		unsafe.Pointer(&src[0]),
		C.size_t(len(src)),
	)

	if C.ZL_isError(result) != 0 {
		return 0, d.getError(result)
	}

	return int(C.ZL_validResult(result)), nil
}

// getError translates an OpenZL C error Result into a Go error.
//
// OpenZL uses a Result type (ZL_Report) that can contain either a value
// or an error code. This method extracts the error code and converts it
// to a human-readable error message using OpenZL's error string function.
func (d *DCtx) getError(result C.ZL_Report) error {
	errCode := C.ZL_errorCode(result)
	errName := C.GoString(C.ZL_ErrorCode_toString(errCode))
	return fmt.Errorf("openzl: %s", errName)
}

// GetDecompressedSize returns the size needed to decompress the given compressed data.
//
// This function reads the OpenZL frame header from src to determine the
// decompressed size without actually decompressing the data. Use this to
// allocate an appropriately-sized buffer before calling Decompress.
//
// Returns an error if:
//   - src is empty
//   - src does not contain a valid OpenZL compressed frame
//   - the frame header is corrupted
func GetDecompressedSize(src []byte) (int, error) {
	if len(src) == 0 {
		return 0, errors.New("empty input")
	}

	result := C.ZL_getDecompressedSize(
		unsafe.Pointer(&src[0]),
		C.size_t(len(src)),
	)

	if C.ZL_isError(result) != 0 {
		errCode := C.ZL_errorCode(result)
		errName := C.GoString(C.ZL_ErrorCode_toString(errCode))
		return 0, fmt.Errorf("openzl: %s", errName)
	}

	return int(C.ZL_validResult(result)), nil
}

// CompressBound returns the maximum possible compressed size for input of the given size.
//
// This function provides a conservative upper bound for buffer allocation.
// The actual compressed size is typically much smaller (often 50-90% less),
// but this guarantees that a buffer of this size will always be sufficient.
//
// Use this to allocate destination buffers for compression:
//
//	dst := make([]byte, cgo.CompressBound(len(src)))
//	n, err := ctx.Compress(dst, src)
//	compressed := dst[:n]
func CompressBound(srcSize int) int {
	return int(C.ZL_compressBound(C.size_t(srcSize)))
}
