// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package cgo

import (
	"fmt"
	"unsafe"
)

// BytesToTypedSlice converts a byte slice to a typed slice.
//
// This function performs an unsafe conversion from bytes to the target type T.
// The byte slice must have a length that is a multiple of the element size,
// and must be properly aligned for type T.
//
// Returns an error if the byte slice length is not a multiple of the element size.
func BytesToTypedSlice[T any](data []byte) ([]T, error) {
	if len(data) == 0 {
		return []T{}, nil
	}

	var zero T
	elementSize := int(unsafe.Sizeof(zero))

	if len(data)%elementSize != 0 {
		return nil, fmt.Errorf("byte slice length %d is not a multiple of element size %d", len(data), elementSize)
	}

	numElements := len(data) / elementSize

	// Create typed slice from bytes using unsafe
	// This is safe because:
	// 1. We verified the size is a multiple of element size
	// 2. Go's slice allocations are properly aligned for any type
	typedSlice := unsafe.Slice((*T)(unsafe.Pointer(&data[0])), numElements)

	// Make a copy to return (so caller doesn't hold reference to input buffer)
	result := make([]T, numElements)
	copy(result, typedSlice)

	return result, nil
}
