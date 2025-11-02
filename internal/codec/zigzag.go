// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package codec

import (
	"encoding/binary"
	"fmt"
)

// ZigZag codec encodes signed integers as unsigned integers, making small
// negative numbers compress efficiently when combined with varints.
//
// ZigZag mapping:
//
//	Signed:  0  -1   1  -2   2  -3   3  -4   4
//	ZigZag:  0   1   2   3   4   5   6   7   8
//
// The formula interleaves positive and negative values:
//   - Non-negative → Even ZigZag values
//   - Negative     → Odd ZigZag values
//
// This makes small negative numbers (like -1, -2) encode compactly as 1, 3
// instead of requiring 5 bytes in two's complement representation.
type ZigZag struct {
	elementSize int // 1, 2, 4, or 8 bytes
}

// NewZigZag creates a new ZigZag codec with the specified element size.
// Element size determines how many bytes each integer uses:
//   - 1 byte:  int8  (−128 to 127)
//   - 2 bytes: int16 (−32,768 to 32,767)
//   - 4 bytes: int32 (−2B to 2B)
//   - 8 bytes: int64 (−9Q to 9Q)
func NewZigZag(elementSize int) *ZigZag {
	return &ZigZag{elementSize: elementSize}
}

// ID returns the codec identifier
func (c *ZigZag) ID() ID {
	return IDZigZag
}

// Name returns the codec name
func (c *ZigZag) Name() string {
	return nameZigZag
}

// Decode converts ZigZag-encoded unsigned integers back to signed integers.
//
// Input:  Buffer of ZigZag-encoded unsigned integers (bytes)
// Output: Original signed integers (same size as input)
// Params: Element size (1, 2, 4, or 8 bytes) - if empty, uses default
//
// Example (4-byte elements):
//
//	Input:  [0, 1, 2, 3, 4]  (ZigZag encoded)
//	Output: [0, -1, 1, -2, 2] (original signed values)
func (c *ZigZag) Decode(dst, src, params []byte) (int, error) {
	// Determine element size
	elementSize := c.elementSize
	if len(params) > 0 {
		elementSize = int(params[0])
	}

	// Validate element size
	if elementSize != 1 && elementSize != 2 && elementSize != 4 && elementSize != 8 {
		return 0, fmt.Errorf("zigzag: invalid element size %d (must be 1, 2, 4, or 8)", elementSize)
	}

	// Validate input size
	if len(src)%elementSize != 0 {
		return 0, fmt.Errorf("zigzag: input size %d not aligned to element size %d", len(src), elementSize)
	}

	numElements := len(src) / elementSize

	// Validate output buffer
	if len(dst) < len(src) {
		return 0, ErrBufferTooSmall
	}

	// Decode based on element size
	switch elementSize {
	case 1:
		return c.decode8(dst, src, numElements)
	case 2:
		return c.decode16(dst, src, numElements)
	case 4:
		return c.decode32(dst, src, numElements)
	case 8:
		return c.decode64(dst, src, numElements)
	default:
		return 0, fmt.Errorf("zigzag: unsupported element size %d", elementSize)
	}
}

// decode8 decodes 1-byte elements (int8)
func (c *ZigZag) decode8(dst, src []byte, numElements int) (int, error) {
	for i := 0; i < numElements; i++ {
		zigzag := src[i]
		// Decode: (n >> 1) ^ -(n & 1)
		value := int8((zigzag >> 1) ^ (-(zigzag & 1)))
		dst[i] = byte(value)
	}
	return numElements, nil
}

// decode16 decodes 2-byte elements (int16)
func (c *ZigZag) decode16(dst, src []byte, numElements int) (int, error) {
	for i := 0; i < numElements; i++ {
		offset := i * 2
		zigzag := binary.LittleEndian.Uint16(src[offset:])
		// Decode: (n >> 1) ^ -(n & 1)
		value := int16((zigzag >> 1) ^ (-(zigzag & 1)))
		binary.LittleEndian.PutUint16(dst[offset:], uint16(value))
	}
	return numElements * 2, nil
}

// decode32 decodes 4-byte elements (int32)
func (c *ZigZag) decode32(dst, src []byte, numElements int) (int, error) {
	for i := 0; i < numElements; i++ {
		offset := i * 4
		zigzag := binary.LittleEndian.Uint32(src[offset:])
		// Decode: (n >> 1) ^ -(n & 1)
		value := int32((zigzag >> 1) ^ (-(zigzag & 1)))
		binary.LittleEndian.PutUint32(dst[offset:], uint32(value))
	}
	return numElements * 4, nil
}

// decode64 decodes 8-byte elements (int64)
func (c *ZigZag) decode64(dst, src []byte, numElements int) (int, error) {
	for i := 0; i < numElements; i++ {
		offset := i * 8
		zigzag := binary.LittleEndian.Uint64(src[offset:])
		// Decode: (n >> 1) ^ -(n & 1)
		value := int64((zigzag >> 1) ^ (-(zigzag & 1)))
		binary.LittleEndian.PutUint64(dst[offset:], uint64(value))
	}
	return numElements * 8, nil
}

// Encode converts signed integers to ZigZag-encoded unsigned integers.
//
// Input:  Buffer of signed integers (bytes)
// Output: ZigZag-encoded unsigned integers (same size as input)
// Params: Element size (1, 2, 4, or 8 bytes) - if empty, uses default
//
// Example (4-byte elements):
//
//	Input:  [0, -1, 1, -2, 2] (signed values)
//	Output: [0, 1, 2, 3, 4]  (ZigZag encoded)
func (c *ZigZag) Encode(dst, src, params []byte) (int, error) {
	// Determine element size
	elementSize := c.elementSize
	if len(params) > 0 {
		elementSize = int(params[0])
	}

	// Validate element size
	if elementSize != 1 && elementSize != 2 && elementSize != 4 && elementSize != 8 {
		return 0, fmt.Errorf("zigzag: invalid element size %d (must be 1, 2, 4, or 8)", elementSize)
	}

	// Validate input size
	if len(src)%elementSize != 0 {
		return 0, fmt.Errorf("zigzag: input size %d not aligned to element size %d", len(src), elementSize)
	}

	numElements := len(src) / elementSize

	// Validate output buffer
	if len(dst) < len(src) {
		return 0, ErrBufferTooSmall
	}

	// Encode based on element size
	switch elementSize {
	case 1:
		return c.encode8(dst, src, numElements)
	case 2:
		return c.encode16(dst, src, numElements)
	case 4:
		return c.encode32(dst, src, numElements)
	case 8:
		return c.encode64(dst, src, numElements)
	default:
		return 0, fmt.Errorf("zigzag: unsupported element size %d", elementSize)
	}
}

// encode8 encodes 1-byte elements (int8)
func (c *ZigZag) encode8(dst, src []byte, numElements int) (int, error) {
	for i := 0; i < numElements; i++ {
		value := int8(src[i])
		// Encode: (n << 1) ^ (n >> 7) for int8 (shift by 7, not 31)
		zigzag := uint8((value << 1) ^ (value >> 7))
		dst[i] = zigzag
	}
	return numElements, nil
}

// encode16 encodes 2-byte elements (int16)
func (c *ZigZag) encode16(dst, src []byte, numElements int) (int, error) {
	for i := 0; i < numElements; i++ {
		offset := i * 2
		value := int16(binary.LittleEndian.Uint16(src[offset:]))
		// Encode: (n << 1) ^ (n >> 15) for int16
		zigzag := uint16((value << 1) ^ (value >> 15))
		binary.LittleEndian.PutUint16(dst[offset:], zigzag)
	}
	return numElements * 2, nil
}

// encode32 encodes 4-byte elements (int32)
func (c *ZigZag) encode32(dst, src []byte, numElements int) (int, error) {
	for i := 0; i < numElements; i++ {
		offset := i * 4
		value := int32(binary.LittleEndian.Uint32(src[offset:]))
		// Encode: (n << 1) ^ (n >> 31) for int32
		zigzag := uint32((value << 1) ^ (value >> 31))
		binary.LittleEndian.PutUint32(dst[offset:], zigzag)
	}
	return numElements * 4, nil
}

// encode64 encodes 8-byte elements (int64)
func (c *ZigZag) encode64(dst, src []byte, numElements int) (int, error) {
	for i := 0; i < numElements; i++ {
		offset := i * 8
		value := int64(binary.LittleEndian.Uint64(src[offset:]))
		// Encode: (n << 1) ^ (n >> 63) for int64
		zigzag := uint64((value << 1) ^ (value >> 63))
		binary.LittleEndian.PutUint64(dst[offset:], zigzag)
	}
	return numElements * 8, nil
}

// PreservesSize returns true because ZigZag always produces output
// of the same size as its input.
//
// ZigZag re-encodes signed integers as unsigned but maintains the same
// number of elements and element size. For example, 100 int32s → 100 uint32s.
func (c *ZigZag) PreservesSize() bool {
	return true
}
