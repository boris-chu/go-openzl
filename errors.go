// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package openzl

import "errors"

var (
	// ErrEmptyInput indicates that the input buffer is empty
	ErrEmptyInput = errors.New("openzl: empty input")

	// ErrBufferTooSmall indicates that the destination buffer is too small
	ErrBufferTooSmall = errors.New("openzl: buffer too small")

	// ErrCorruptedData indicates that the compressed data is corrupted
	ErrCorruptedData = errors.New("openzl: corrupted data")

	// ErrInvalidParameter indicates an invalid parameter was passed
	ErrInvalidParameter = errors.New("openzl: invalid parameter")

	// ErrContextClosed indicates that the context has been closed
	ErrContextClosed = errors.New("openzl: context closed")

	// ErrOutOfMemory indicates that memory allocation failed
	ErrOutOfMemory = errors.New("openzl: out of memory")
)
