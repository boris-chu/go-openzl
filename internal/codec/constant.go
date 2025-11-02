// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package codec

import (
	"bytes"
	"fmt"
)

// Constant codec handles data where all values are identical.
//
// Instead of storing N identical values, it stores just one value and
// repeats it during decompression. This achieves extreme compression ratios
// (often 100:1 or higher) for constant data.
//
// Use cases:
//   - Sensor data during offline periods (all zeros)
//   - Boolean flags (mostly false)
//   - Padding bytes in structs
//   - Default values in configurations
//   - Sparse arrays with mostly identical values
//
// Example:
//
//	Input:  [5, 5, 5, 5, 5, 5, 5, 5] (8 identical int32s = 32 bytes)
//	Encoded: [5] (1 int32 = 4 bytes)
//	Ratio: 8:1 compression
//
// For 1000 identical values, the ratio is 1000:1!
type Constant struct {
	elementSize int // 1, 2, 4, or 8 bytes
}

// NewConstant creates a new Constant codec with the specified element size.
// Element size determines how many bytes each value uses:
//   - 1 byte:  uint8/int8
//   - 2 bytes: uint16/int16
//   - 4 bytes: uint32/int32
//   - 8 bytes: uint64/int64
func NewConstant(elementSize int) *Constant {
	return &Constant{elementSize: elementSize}
}

// ID returns the codec identifier
func (c *Constant) ID() ID {
	return IDConstant
}

// Name returns the codec name
func (c *Constant) Name() string {
	return nameConstant
}

// Decode fills the output buffer with a constant value repeated.
//
// Input:  Single constant value (elementSize bytes)
// Output: That value repeated to fill dst
// Params: Element size (1, 2, 4, or 8 bytes) - if empty, uses default
//
// The number of repetitions is calculated from the output buffer size:
//
//	numElements = len(dst) / elementSize
//
// Example:
//
//	Input:  [5] (one 4-byte int32)
//	Output: [5, 5, 5, 5, 5, 5, 5, 5] (8 repetitions)
//	dst is 32 bytes → 32/4 = 8 elements
func (c *Constant) Decode(dst, src, params []byte) (int, error) {
	// Determine element size
	elementSize := c.elementSize
	if len(params) > 0 {
		elementSize = int(params[0])
	}

	// Validate element size
	if elementSize != 1 && elementSize != 2 && elementSize != 4 && elementSize != 8 {
		return 0, fmt.Errorf("constant: invalid element size %d (must be 1, 2, 4, or 8)", elementSize)
	}

	// Empty data is valid (no-op)
	if len(src) == 0 {
		return 0, nil
	}

	// Source must be exactly one element
	if len(src) != elementSize {
		return 0, fmt.Errorf("constant: source size %d does not match element size %d", len(src), elementSize)
	}

	// Validate output buffer alignment
	if len(dst)%elementSize != 0 {
		return 0, fmt.Errorf("constant: output size %d not aligned to element size %d", len(dst), elementSize)
	}

	numElements := len(dst) / elementSize

	// Extract the constant value
	constantValue := src[0:elementSize]

	// Fill output buffer with repeated value
	// For small element sizes, use a fast loop
	// For larger buffers, this is memory-bandwidth limited (~10-20 GB/s)
	for i := 0; i < numElements; i++ {
		offset := i * elementSize
		copy(dst[offset:offset+elementSize], constantValue)
	}

	return len(dst), nil
}

// Encode compresses data by storing only one value if all values are identical.
//
// Input:  Buffer of identical values
// Output: Single constant value (elementSize bytes)
// Params: Element size (1, 2, 4, or 8 bytes) - if empty, uses default
//
// Returns an error if not all values are identical.
//
// Example:
//
//	Input:  [5, 5, 5, 5, 5, 5, 5, 5] (8 identical int32s)
//	Output: [5] (one int32)
//	Size: 32 bytes → 4 bytes (8:1 compression)
func (c *Constant) Encode(dst, src, params []byte) (int, error) {
	// Determine element size
	elementSize := c.elementSize
	if len(params) > 0 {
		elementSize = int(params[0])
	}

	// Validate element size
	if elementSize != 1 && elementSize != 2 && elementSize != 4 && elementSize != 8 {
		return 0, fmt.Errorf("constant: invalid element size %d (must be 1, 2, 4, or 8)", elementSize)
	}

	// Empty data is valid (no-op)
	if len(src) == 0 {
		return 0, nil
	}

	// Validate input size alignment
	if len(src)%elementSize != 0 {
		return 0, fmt.Errorf("constant: input size %d not aligned to element size %d", len(src), elementSize)
	}

	numElements := len(src) / elementSize

	// Validate output buffer (must fit at least one element)
	if len(dst) < elementSize {
		return 0, ErrBufferTooSmall
	}

	// Single element is always constant
	if numElements == 1 {
		copy(dst, src[0:elementSize])
		return elementSize, nil
	}

	// Get the first value (the constant)
	constantValue := src[0:elementSize]

	// Verify all subsequent values match
	for i := 1; i < numElements; i++ {
		offset := i * elementSize
		element := src[offset : offset+elementSize]

		if !bytes.Equal(element, constantValue) {
			return 0, fmt.Errorf("constant: not all values are identical (element %d differs)", i)
		}
	}

	// All values are identical - output just the first one
	copy(dst, constantValue)
	return elementSize, nil
}

// PreservesSize returns false because Constant changes size.
//
// During compression, it reduces N elements to 1 element.
// During decompression, it expands 1 element to N elements.
//
// This is a size-changing codec that requires explicit size metadata.
func (c *Constant) PreservesSize() bool {
	return false
}
