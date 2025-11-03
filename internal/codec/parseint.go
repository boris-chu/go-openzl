// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package codec

import (
	"encoding/binary"
	"fmt"
	"strconv"
)

// ParseInt is a codec that parses integer strings to int64 values.
//
// Algorithm:
// - Parses string representations of integers to binary int64
// - Useful for CSV parsing, text-to-binary conversion
// - Typically used as first stage before Delta/ZigZag/Bitpack
//
// Example:
//
//	Input:  ["1000", "1001", "1002", "1003"] (20 bytes as strings)
//	Output: [1000, 1001, 1002, 1003] (32 bytes as int64)
//	Then:   Delta → [1000, 1, 1, 1] → ZigZag → Bitpack → 5 bytes
//	Total:  20 bytes text → 5 bytes compressed (4× compression)
//
// Wire Format (Encode - text to binary):
//
//	Input:
//	  [numIntegers: 4 bytes (uint32)]
//	  For each integer string:
//	    [length: 4 bytes (uint32)]
//	    [data: length bytes] (ASCII digits, optional '-' sign)
//
//	Output:
//	  [numIntegers: 4 bytes (uint32)]
//	  [int64 values: numIntegers * 8 bytes]
//
// Wire Format (Decode - binary to text):
//
//	Input:
//	  [numIntegers: 4 bytes (uint32)]
//	  [int64 values: numIntegers * 8 bytes]
//
//	Output:
//	  [numIntegers: 4 bytes (uint32)]
//	  For each integer:
//	    [length: 4 bytes (uint32)]
//	    [data: length bytes] (ASCII digits)
//
// Use Cases:
// - CSV parsing (convert "123" to binary 123)
// - Text logs with integer IDs
// - Configuration files with numeric values
// - Typically followed by Delta→ZigZag→Bitpack pipeline
type ParseInt struct{}

// NewParseInt creates a new ParseInt codec.
func NewParseInt() *ParseInt {
	return &ParseInt{}
}

// ID returns the codec identifier.
func (p *ParseInt) ID() ID {
	return IDParseInt
}

// Name returns the human-readable name of the codec.
func (p *ParseInt) Name() string {
	return "ParseInt"
}

// PreservesSize returns false since ParseInt changes output size.
func (p *ParseInt) PreservesSize() bool {
	return false
}

// Encode parses integer strings to int64 binary values.
//
// Input format (src):
//
//	[numIntegers: 4 bytes (uint32)]
//	For each integer string:
//	  [length: 4 bytes (uint32)]
//	  [data: length bytes] (ASCII digits, optional '-' sign)
//
// Output format (dst):
//
//	[numIntegers: 4 bytes (uint32)]
//	[int64 values: numIntegers * 8 bytes]
//
// Params: None required
func (p *ParseInt) Encode(dst, src, params []byte) (int, error) {
	if len(src) < 4 {
		return 0, fmt.Errorf("parseint: input too small (need at least 4 bytes for count)")
	}

	// Read number of integers
	numIntegers := binary.LittleEndian.Uint32(src[0:4])
	if numIntegers == 0 {
		// Empty input: just write count
		binary.LittleEndian.PutUint32(dst[0:4], 0)
		return 4, nil
	}

	// Check output buffer size
	outputSize := 4 + int(numIntegers)*8
	if len(dst) < outputSize {
		return 0, ErrBufferTooSmall
	}

	// Write number of integers
	binary.LittleEndian.PutUint32(dst[0:4], numIntegers)

	// Parse each string to int64
	inPos := 4
	outPos := 4

	for i := uint32(0); i < numIntegers; i++ {
		// Read string length
		if inPos+4 > len(src) {
			return 0, fmt.Errorf("parseint: incomplete string %d header", i)
		}
		strLen := binary.LittleEndian.Uint32(src[inPos : inPos+4])
		inPos += 4

		// Read string data
		if inPos+int(strLen) > len(src) {
			return 0, fmt.Errorf("parseint: incomplete string %d data (need %d bytes, have %d)", i, strLen, len(src)-inPos)
		}
		strData := string(src[inPos : inPos+int(strLen)])
		inPos += int(strLen)

		// Parse string to int64
		val, err := strconv.ParseInt(strData, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parseint: failed to parse string %d (%q): %w", i, strData, err)
		}

		// Write int64 value
		binary.LittleEndian.PutUint64(dst[outPos:], uint64(val))
		outPos += 8
	}

	return outPos, nil
}

// Decode converts int64 binary values to integer strings.
//
// Input format (src):
//
//	[numIntegers: 4 bytes (uint32)]
//	[int64 values: numIntegers * 8 bytes]
//
// Output format (dst):
//
//	[numIntegers: 4 bytes (uint32)]
//	For each integer:
//	  [length: 4 bytes (uint32)]
//	  [data: length bytes] (ASCII digits)
//
// Params: None required
func (p *ParseInt) Decode(dst, src, params []byte) (int, error) {
	if len(src) < 4 {
		return 0, fmt.Errorf("parseint: compressed data too small")
	}

	// Read number of integers
	numIntegers := binary.LittleEndian.Uint32(src[0:4])
	if numIntegers == 0 {
		// Empty input: just write count
		binary.LittleEndian.PutUint32(dst[0:4], 0)
		return 4, nil
	}

	// Verify input size
	expectedSize := 4 + int(numIntegers)*8
	if len(src) < expectedSize {
		return 0, fmt.Errorf("parseint: incomplete data (need %d bytes, have %d)", expectedSize, len(src))
	}

	// Write number of integers
	outPos := 0
	if outPos+4 > len(dst) {
		return 0, ErrBufferTooSmall
	}
	binary.LittleEndian.PutUint32(dst[outPos:], numIntegers)
	outPos += 4

	// Convert each int64 to string
	inPos := 4
	for i := uint32(0); i < numIntegers; i++ {
		// Read int64 value
		val := int64(binary.LittleEndian.Uint64(src[inPos:]))
		inPos += 8

		// Convert to string
		strData := strconv.FormatInt(val, 10)

		// Write string length
		if outPos+4 > len(dst) {
			return 0, ErrBufferTooSmall
		}
		binary.LittleEndian.PutUint32(dst[outPos:], uint32(len(strData)))
		outPos += 4

		// Write string data
		if outPos+len(strData) > len(dst) {
			return 0, ErrBufferTooSmall
		}
		copy(dst[outPos:], strData)
		outPos += len(strData)
	}

	return outPos, nil
}

// String returns a human-readable name for the codec.
func (p *ParseInt) String() string {
	return "ParseInt"
}
