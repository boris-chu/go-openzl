// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package codec

import (
	"fmt"
)

// Transpose is a structural transformation that reorganizes multi-byte data
// by separating bytes into distinct streams based on their position.
//
// Instead of storing data in natural byte order, Transpose groups all first
// bytes together, all second bytes together, etc. This exposes byte-level
// patterns that other codecs can exploit.
//
// Example (4 elements × 4 bytes):
//
//	Input (natural layout):
//	  Element 0: [A0 A1 A2 A3]
//	  Element 1: [B0 B1 B2 B3]
//	  Element 2: [C0 C1 C2 C3]
//	  Element 3: [D0 D1 D2 D3]
//	  Memory: A0 A1 A2 A3 | B0 B1 B2 B3 | C0 C1 C2 C3 | D0 D1 D2 D3
//
//	After Transpose:
//	  Byte 0 stream: A0 B0 C0 D0
//	  Byte 1 stream: A1 B1 C1 D1
//	  Byte 2 stream: A2 B2 C2 D2
//	  Byte 3 stream: A3 B3 C3 D3
//	  Memory: A0 B0 C0 D0 | A1 B1 C1 D1 | A2 B2 C2 D2 | A3 B3 C3 D3
//
// Why Transpose?
//
// Multi-byte integers often have predictable high bytes:
//   - Timestamps: high bytes constant (unix epoch range)
//   - Counters: high bytes change slowly
//   - Pointers: high bytes identical (same memory region)
//   - Prices: high bytes similar (same currency range)
//
// After transpose:
//   - High byte streams: constant or slowly changing → RLE/Delta friendly
//   - Low byte streams: varying but sequential → Delta/Bitpack friendly
//   - All streams: skewed distribution → Huffman/FSE friendly
//
// Best for:
//   - Numeric arrays (uint32, uint64, int64)
//   - Timestamps and counters
//   - Pointers and memory addresses
//   - Fixed-point numbers
//   - Color data (RGB/RGBA)
//
// Poor for:
//   - Single-byte types (nothing to transpose)
//   - Random data (no byte-level patterns)
//   - Small datasets (overhead not worth it)
//   - String data (already byte-aligned)
//
// Common Pipelines:
//   - Transpose → RLE (for constant high bytes)
//   - Transpose → Delta (for sequential low bytes)
//   - Transpose → Delta → Bitpack (for small deltas)
//   - Transpose → RLE → Huffman (full pipeline)
//
// Performance:
//   - Encoding: O(n × width), memory-bound (~2-5 GB/s)
//   - Decoding: O(n × width), memory-bound (~2-5 GB/s)
//   - Space: Size-preserving (just rearranges bytes)
//
// OpenZL Integration:
//   - Codec ID: IDTranspose (5)
//   - Size-preserving: Output size = Input size
//   - Params: [width(1 byte)] - element width in bytes
type Transpose struct {
}

// NewTranspose creates a new Transpose codec.
func NewTranspose() *Transpose {
	return &Transpose{}
}

// ID returns the codec identifier.
func (t *Transpose) ID() ID {
	return IDTranspose
}

// Name returns the codec name.
func (t *Transpose) Name() string {
	return "Transpose"
}

// PreservesSize returns true because Transpose only rearranges bytes.
//
// Input size = Output size. Transpose is a structural transformation
// that exposes patterns for other codecs to exploit.
func (t *Transpose) PreservesSize() bool {
	return true
}

// Encode transposes multi-byte elements by separating into byte streams.
//
// Parameters:
//
//	params[0] = width (element width in bytes, e.g., 4 for uint32, 8 for uint64)
//
// Algorithm:
//
//	For each byte position i (0 to width-1):
//	    For each element e (0 to count-1):
//	        output[i*count + e] = input[e*width + i]
//
// Example (width=4, count=3):
//
//	Input:  [A0 A1 A2 A3] [B0 B1 B2 B3] [C0 C1 C2 C3]
//	Output: [A0 B0 C0] [A1 B1 C1] [A2 B2 C2] [A3 B3 C3]
//
// Returns: Number of bytes written (always len(src))
//
//nolint:dupl // Encode and Decode are mirror operations with similar loops
func (t *Transpose) Encode(dst, src, params []byte) (int, error) {
	if len(src) == 0 {
		return 0, nil
	}

	// Parse width from params
	if len(params) < 1 {
		return 0, fmt.Errorf("transpose: missing width parameter")
	}
	width := int(params[0])
	if width == 0 {
		return 0, fmt.Errorf("transpose: width cannot be zero")
	}
	if width == 1 {
		// Width 1: no transpose needed, just copy
		n := copy(dst, src)
		return n, nil
	}

	// Validate input size
	if len(src)%width != 0 {
		return 0, fmt.Errorf("transpose: input size %d not multiple of width %d", len(src), width)
	}

	count := len(src) / width

	// Validate output buffer size
	if len(dst) < len(src) {
		return 0, ErrBufferTooSmall
	}

	// Transpose: for each byte position, gather all elements
	for bytePos := 0; bytePos < width; bytePos++ {
		outBase := bytePos * count
		for elem := 0; elem < count; elem++ {
			srcIdx := elem*width + bytePos
			dstIdx := outBase + elem
			dst[dstIdx] = src[srcIdx]
		}
	}

	return len(src), nil
}

// Decode reverses the transpose operation, restoring natural byte order.
//
// Parameters:
//
//	params[0] = width (element width in bytes)
//
// Algorithm:
//
//	For each element e (0 to count-1):
//	    For each byte position i (0 to width-1):
//	        output[e*width + i] = input[i*count + e]
//
// Example (width=4, count=3):
//
//	Input:  [A0 B0 C0] [A1 B1 C1] [A2 B2 C2] [A3 B3 C3]
//	Output: [A0 A1 A2 A3] [B0 B1 B2 B3] [C0 C1 C2 C3]
//
// Returns: Number of bytes written (always len(src))
//
//nolint:dupl // Encode and Decode are mirror operations with similar loops
func (t *Transpose) Decode(dst, src, params []byte) (int, error) {
	if len(src) == 0 {
		return 0, nil
	}

	// Parse width from params
	if len(params) < 1 {
		return 0, fmt.Errorf("transpose: missing width parameter")
	}
	width := int(params[0])
	if width == 0 {
		return 0, fmt.Errorf("transpose: width cannot be zero")
	}
	if width == 1 {
		// Width 1: no transpose needed, just copy
		n := copy(dst, src)
		return n, nil
	}

	// Validate input size
	if len(src)%width != 0 {
		return 0, fmt.Errorf("transpose: input size %d not multiple of width %d", len(src), width)
	}

	count := len(src) / width

	// Validate output buffer size
	if len(dst) < len(src) {
		return 0, ErrBufferTooSmall
	}

	// Reverse transpose: for each element, gather its bytes
	for elem := 0; elem < count; elem++ {
		outBase := elem * width
		for bytePos := 0; bytePos < width; bytePos++ {
			srcIdx := bytePos*count + elem
			dstIdx := outBase + bytePos
			dst[dstIdx] = src[srcIdx]
		}
	}

	return len(src), nil
}
