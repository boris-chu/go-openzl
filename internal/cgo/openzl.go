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

// CCtx wraps ZL_CCtx compression context
type CCtx struct {
	ctx *C.ZL_CCtx
}

// NewCCtx creates a compression context
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

// Free releases the compression context
func (c *CCtx) Free() {
	if c.ctx != nil {
		C.ZL_CCtx_free(c.ctx)
		c.ctx = nil
	}
}

// Compress compresses src into dst
// Returns the number of bytes written or an error
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

// getError translates C error to Go error
func (c *CCtx) getError(result C.ZL_Report) error {
	errCode := C.ZL_errorCode(result)
	errName := C.GoString(C.ZL_ErrorCode_toString(errCode))
	return fmt.Errorf("openzl: %s", errName)
}

// DCtx wraps ZL_DCtx decompression context
type DCtx struct {
	ctx *C.ZL_DCtx
}

// NewDCtx creates a decompression context
func NewDCtx() (*DCtx, error) {
	ctx := C.ZL_DCtx_create()
	if ctx == nil {
		return nil, errors.New("failed to create decompression context")
	}
	return &DCtx{ctx: ctx}, nil
}

// Free releases the decompression context
func (d *DCtx) Free() {
	if d.ctx != nil {
		C.ZL_DCtx_free(d.ctx)
		d.ctx = nil
	}
}

// Decompress decompresses src into dst
// Returns the number of bytes written or an error
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

// getError translates C error to Go error
func (d *DCtx) getError(result C.ZL_Report) error {
	errCode := C.ZL_errorCode(result)
	errName := C.GoString(C.ZL_ErrorCode_toString(errCode))
	return fmt.Errorf("openzl: %s", errName)
}

// GetDecompressedSize returns the size needed for decompression
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

// CompressBound returns upper bound for compressed size
func CompressBound(srcSize int) int {
	return int(C.ZL_compressBound(C.size_t(srcSize)))
}
