// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package codec

import (
	"encoding/binary"
	"fmt"
)

// RLE implements Run-Length Encoding, one of the simplest compression algorithms.
//
// RLE replaces consecutive sequences of identical values (runs) with a single
// value and a count. This is extremely effective for data with long runs of
// repeated values, but can expand data if there are no repetitions.
//
// Best for:
//   - Sparse arrays (many zeros)
//   - Boolean flags with long sequences
//   - Database columns with low cardinality
//   - After quantization or rounding operations
//   - Simple graphics with solid color regions
//
// Poor for:
//   - Random data (causes 2× expansion!)
//   - Alternating values
//   - High-entropy text
//
// Format:
//
//	[num_runs(4)]
//	For each run:
//	  [value(1)] [count(varint)]
//
// Example:
//
//	Input:  [5, 5, 5, 5, 3, 3, 7, 7, 7]
//	Output: 3 runs: (5, count=4) (3, count=2) (7, count=3)
//	Size:   9 bytes → ~9 bytes (depends on varint encoding)
//
// Performance:
//   - Encoding: O(n), very fast (500-1000 MB/s)
//   - Decoding: O(m) where m=output size, even faster (1000-2000 MB/s)
//   - Space: O(1) - constant extra memory
//
// OpenZL Integration:
//   - Codec ID: IDRLE (13)
//   - Size-changing: Output size varies based on number of runs
//   - Pipeline position: After Delta/Transpose, before entropy coding
//
// Common Pipelines:
//   - Delta → RLE (for time-series with plateaus)
//   - Transpose → RLE (for numeric arrays with constant high bytes)
//   - RLE → Huffman (compress run lengths)
type RLE struct {
	minRunLength int // Minimum run length to encode (default: 2)
}

// NewRLE creates a new RLE codec with default parameters.
//
// Default minimum run length is 2, meaning:
//   - Run of 1: emit literal (1 byte)
//   - Run of 2+: emit (value, count) (~2 bytes with varint)
//
// This balances compression ratio vs. expansion risk.
func NewRLE() *RLE {
	return &RLE{
		minRunLength: 2, // Encode runs of 2 or more
	}
}

// ID returns the codec identifier.
func (r *RLE) ID() ID {
	return IDRLE
}

// Name returns the codec name.
func (r *RLE) Name() string {
	return "RLE"
}

// PreservesSize returns false because RLE changes output size.
//
// The output size depends on the number of runs:
//   - Best case: all same value → 1 run → ~5 bytes
//   - Worst case: all different → N runs → ~2N bytes (expansion!)
func (r *RLE) PreservesSize() bool {
	return false
}

// Encode compresses src using run-length encoding.
//
// Format:
//
//	[num_runs(4 bytes)]
//	For each run:
//	  [value(1 byte)] [count(varint, typically 1-2 bytes)]
//
// Algorithm:
//  1. Scan through input finding consecutive identical values
//  2. For runs >= minRunLength: emit (value, count)
//  3. For runs < minRunLength: emit as individual literals
//
// Parameters: unused (RLE has no configurable parameters)
//
// Returns compressed size, or error if buffer too small.
//
// Example:
//
//	Input:  [0, 0, 0, 0, 1, 2, 2, 2]
//	Output: [num_runs=3] (0, 4) (1, 1) (2, 3)
//
//	Breakdown:
//	  - Run of 4 zeros: encoded as (0, 4)
//	  - Single 1: encoded as (1, 1)
//	  - Run of 3 twos: encoded as (2, 3)
func (r *RLE) Encode(dst, src, params []byte) (int, error) {
	if len(src) == 0 {
		// Empty input: write 0 runs
		if len(dst) < 4 {
			return 0, ErrBufferTooSmall
		}
		binary.LittleEndian.PutUint32(dst[0:], 0)
		return 4, nil
	}

	// First pass: count runs to validate buffer size
	runs := r.countRuns(src)

	// Estimate output size: 4 bytes header + runs * (1 byte value + ~1-2 bytes count)
	estimatedSize := 4 + runs*3 // Conservative estimate
	if len(dst) < estimatedSize {
		return 0, ErrBufferTooSmall
	}

	// Second pass: encode runs
	pos := 0
	outPos := 4 // Reserve space for run count
	runCount := 0

	for pos < len(src) {
		// Find run length
		runValue := src[pos]
		runLen := 1
		for pos+runLen < len(src) && src[pos+runLen] == runValue {
			runLen++
		}

		// Encode if run is long enough
		if runLen >= r.minRunLength {
			// Check buffer space: 1 byte value + max 10 bytes varint
			if outPos+11 > len(dst) {
				return 0, ErrBufferTooSmall
			}

			dst[outPos] = runValue
			outPos++
			n := binary.PutUvarint(dst[outPos:], uint64(runLen))
			outPos += n
			runCount++
		} else {
			// Emit literals (runs too short to encode)
			for i := 0; i < runLen; i++ {
				if outPos+11 > len(dst) {
					return 0, ErrBufferTooSmall
				}
				dst[outPos] = runValue
				outPos++
				dst[outPos] = 1 // Count of 1
				outPos++
				runCount++
			}
		}

		pos += runLen
	}

	// Write run count at start
	binary.LittleEndian.PutUint32(dst[0:], uint32(runCount))

	return outPos, nil
}

// Decode decompresses src using run-length decoding.
//
// Format (input):
//
//	[num_runs(4 bytes)]
//	For each run:
//	  [value(1 byte)] [count(varint)]
//
// Algorithm:
//  1. Read number of runs
//  2. For each run: read value and count
//  3. Expand run: write value × count to output
//
// Parameters: unused
//
// Returns decompressed size, or error if buffer too small or invalid data.
//
// Example:
//
//	Input:  [num_runs=2] (5, 4) (3, 2)
//	Output: [5, 5, 5, 5, 3, 3]
func (r *RLE) Decode(dst, src, params []byte) (int, error) {
	if len(src) < 4 {
		return 0, fmt.Errorf("rle: invalid input (need at least 4 bytes for header)")
	}

	numRuns := binary.LittleEndian.Uint32(src[0:4])
	if numRuns == 0 {
		return 0, nil // Empty output
	}

	srcPos := 4
	outPos := 0

	for i := uint32(0); i < numRuns; i++ {
		// Read value
		if srcPos >= len(src) {
			return 0, fmt.Errorf("rle: truncated input at run %d", i)
		}
		value := src[srcPos]
		srcPos++

		// Read count (varint)
		count, n := binary.Uvarint(src[srcPos:])
		if n <= 0 {
			return 0, fmt.Errorf("rle: invalid varint at run %d", i)
		}
		srcPos += n

		// Validate count
		if count == 0 {
			return 0, fmt.Errorf("rle: zero count at run %d", i)
		}
		if count > uint64(len(dst)-outPos) {
			return 0, ErrBufferTooSmall
		}

		// Expand run
		for j := uint64(0); j < count; j++ {
			dst[outPos] = value
			outPos++
		}
	}

	return outPos, nil
}

// countRuns counts the number of runs in the input.
//
// This is used to estimate output buffer size and validate
// that encoding will succeed.
func (r *RLE) countRuns(src []byte) int {
	if len(src) == 0 {
		return 0
	}

	runs := 0
	pos := 0

	for pos < len(src) {
		// Find run length
		runValue := src[pos]
		runLen := 1
		for pos+runLen < len(src) && src[pos+runLen] == runValue {
			runLen++
		}

		// Count runs based on minRunLength threshold
		if runLen >= r.minRunLength {
			runs++
		} else {
			// Short runs encoded as individual literals
			runs += runLen
		}

		pos += runLen
	}

	return runs
}
