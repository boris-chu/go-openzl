// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package codec

import (
	"encoding/binary"
	"fmt"
	"math"
)

// RangePack compresses numeric data by subtracting the minimum value and packing
// to the narrowest integer type that can represent the resulting range.
//
// Algorithm:
//  1. Find minimum and maximum values in input
//  2. Calculate range = max - min
//  3. Subtract min from all elements
//  4. Pack to narrowest type (uint8/16/32/64) based on range
//
// Example:
//
//	Input:  [1000, 1010, 1050, 1100] (uint16, 100 range)
//	Output: [0, 10, 50, 100] packed as uint8 (saves 50% space)
//
// Use cases:
//   - Timestamps (all > 1,700,000,000) → pack as uint32
//   - IDs with offset (account IDs 1,000,000 - 2,000,000)
//   - Sequential data with base value
type RangePack struct{}

// NewRangePack creates a new RangePack codec.
func NewRangePack() *RangePack {
	return &RangePack{}
}

// ID returns the codec identifier.
func (r *RangePack) ID() ID {
	return IDRangePack
}

// Encode compresses numeric data using range packing.
//
// Input format:
//   - Numeric data (uint8/16/32/64 or int8/16/32/64)
//   - Element width must be 1, 2, 4, or 8 bytes
//
// Output format:
//   - Header: [minValue:8 bytes][maxValue:8 bytes][packedWidth:1 byte]
//   - Data: Packed elements in narrowest width
func (r *RangePack) Encode(dst, src, params []byte) (int, error) {
	if len(src) == 0 {
		return 0, fmt.Errorf("rangepack: empty input")
	}

	// Determine element width from params or infer from length
	elemWidth := 8 // Default to uint64
	if len(params) > 0 {
		elemWidth = int(params[0])
	}

	if elemWidth != 1 && elemWidth != 2 && elemWidth != 4 && elemWidth != 8 {
		return 0, fmt.Errorf("rangepack: invalid element width %d (must be 1, 2, 4, or 8)", elemWidth)
	}

	if len(src)%elemWidth != 0 {
		return 0, fmt.Errorf("rangepack: source length %d not aligned to element width %d", len(src), elemWidth)
	}

	numElements := len(src) / elemWidth

	// Find min and max values
	var minVal, maxVal uint64
	for i := 0; i < numElements; i++ {
		offset := i * elemWidth
		var val uint64
		switch elemWidth {
		case 1:
			val = uint64(src[offset])
		case 2:
			val = uint64(binary.LittleEndian.Uint16(src[offset:]))
		case 4:
			val = uint64(binary.LittleEndian.Uint32(src[offset:]))
		case 8:
			val = binary.LittleEndian.Uint64(src[offset:])
		}

		if i == 0 {
			minVal = val
			maxVal = val
		} else {
			if val < minVal {
				minVal = val
			}
			if val > maxVal {
				maxVal = val
			}
		}
	}

	// Calculate range and determine packed width
	rangeVal := maxVal - minVal
	var packedWidth int
	if rangeVal <= math.MaxUint8 {
		packedWidth = 1
	} else if rangeVal <= math.MaxUint16 {
		packedWidth = 2
	} else if rangeVal <= math.MaxUint32 {
		packedWidth = 4
	} else {
		packedWidth = 8
	}

	// Write header: minValue (8 bytes) + maxValue (8 bytes) + packedWidth (1 byte)
	headerSize := 17
	if len(dst) < headerSize+numElements*packedWidth {
		return 0, fmt.Errorf("rangepack: destination buffer too small")
	}

	outPos := 0
	binary.LittleEndian.PutUint64(dst[outPos:], minVal)
	outPos += 8
	binary.LittleEndian.PutUint64(dst[outPos:], maxVal)
	outPos += 8
	dst[outPos] = byte(packedWidth)
	outPos++

	// Pack elements (subtract min, write in packed width)
	for i := 0; i < numElements; i++ {
		offset := i * elemWidth
		var val uint64
		switch elemWidth {
		case 1:
			val = uint64(src[offset])
		case 2:
			val = uint64(binary.LittleEndian.Uint16(src[offset:]))
		case 4:
			val = uint64(binary.LittleEndian.Uint32(src[offset:]))
		case 8:
			val = binary.LittleEndian.Uint64(src[offset:])
		}

		// Subtract minimum to get packed value
		packedVal := val - minVal

		// Write packed value
		switch packedWidth {
		case 1:
			dst[outPos] = byte(packedVal)
			outPos++
		case 2:
			binary.LittleEndian.PutUint16(dst[outPos:], uint16(packedVal))
			outPos += 2
		case 4:
			binary.LittleEndian.PutUint32(dst[outPos:], uint32(packedVal))
			outPos += 4
		case 8:
			binary.LittleEndian.PutUint64(dst[outPos:], packedVal)
			outPos += 8
		}
	}

	return outPos, nil
}

// Decode decompresses range-packed numeric data.
//
// Input format:
//   - Header: [minValue:8 bytes][maxValue:8 bytes][packedWidth:1 byte]
//   - Data: Packed elements
//
// Output format:
//   - Numeric data in original width (specified by params)
func (r *RangePack) Decode(dst, src, params []byte) (int, error) {
	if len(src) < 17 {
		return 0, fmt.Errorf("rangepack: source too small for header")
	}

	// Read header
	minVal := binary.LittleEndian.Uint64(src[0:8])
	packedWidth := int(src[16])

	if packedWidth != 1 && packedWidth != 2 && packedWidth != 4 && packedWidth != 8 {
		return 0, fmt.Errorf("rangepack: invalid packed width %d", packedWidth)
	}

	// Determine output element width
	outputWidth := 8 // Default to uint64
	if len(params) > 0 {
		outputWidth = int(params[0])
	}

	if outputWidth != 1 && outputWidth != 2 && outputWidth != 4 && outputWidth != 8 {
		return 0, fmt.Errorf("rangepack: invalid output width %d", outputWidth)
	}

	// Calculate number of elements
	dataSize := len(src) - 17
	if dataSize%packedWidth != 0 {
		return 0, fmt.Errorf("rangepack: data size %d not aligned to packed width %d", dataSize, packedWidth)
	}
	numElements := dataSize / packedWidth

	if len(dst) < numElements*outputWidth {
		return 0, fmt.Errorf("rangepack: destination buffer too small")
	}

	// Unpack elements (add min back)
	srcPos := 17
	outPos := 0

	for i := 0; i < numElements; i++ {
		// Read packed value
		var packedVal uint64
		switch packedWidth {
		case 1:
			packedVal = uint64(src[srcPos])
			srcPos++
		case 2:
			packedVal = uint64(binary.LittleEndian.Uint16(src[srcPos:]))
			srcPos += 2
		case 4:
			packedVal = uint64(binary.LittleEndian.Uint32(src[srcPos:]))
			srcPos += 4
		case 8:
			packedVal = binary.LittleEndian.Uint64(src[srcPos:])
			srcPos += 8
		}

		// Add minimum to get original value
		val := packedVal + minVal

		// Write to output in specified width
		switch outputWidth {
		case 1:
			dst[outPos] = byte(val)
			outPos++
		case 2:
			binary.LittleEndian.PutUint16(dst[outPos:], uint16(val))
			outPos += 2
		case 4:
			binary.LittleEndian.PutUint32(dst[outPos:], uint32(val))
			outPos += 4
		case 8:
			binary.LittleEndian.PutUint64(dst[outPos:], val)
			outPos += 8
		}
	}

	return outPos, nil
}

// Name returns the human-readable name of the codec.
func (r *RangePack) Name() string {
	return "RangePack"
}

// PreservesSize returns false since RangePack changes output size.
func (r *RangePack) PreservesSize() bool {
	return false
}

// String returns a human-readable name for the codec.
func (r *RangePack) String() string {
	return "RangePack"
}
