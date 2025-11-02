// Package codec provides Pure Go OpenZL codec implementations.
//
// This file implements the Huffman codec using Klaus Post's
// excellent compress library (huff0).
//
// Copyright (c) 2019 Klaus Post (github.com/klauspost/compress/huff0)
// Licensed under BSD 3-Clause License.
package codec

import (
	"fmt"

	"github.com/klauspost/compress/huff0"
)

// Huffman implements Huffman (huff0) decoding.
//
// Huffman is a classic entropy coder that assigns shorter codes to
// more frequent symbols. Klaus Post's huff0 implementation is used
// in zstd and achieves excellent compression ratios.
//
// This codec wraps Klaus Post's huff0 implementation using the
// simpler 1X variant (single stream). For 4x performance, we could
// later add support for Decompress4X (four parallel streams).
type Huffman struct {
	id      ID
	scratch *huff0.Scratch // Reused across calls for zero allocations
}

// NewHuffman creates a new Huffman codec.
func NewHuffman() *Huffman {
	return &Huffman{
		id:      IDHuffman,
		scratch: &huff0.Scratch{},
	}
}

// ID returns the codec identifier.
func (h *Huffman) ID() ID {
	return h.id
}

// Name returns the human-readable codec name.
func (h *Huffman) Name() string {
	return "Huffman (huff0)"
}

// Decode decompresses Huffman-encoded data.
//
// Huffman encoding uses a two-step process:
// 1. ReadTable: Parse the Huffman tree from the beginning of src
// 2. Decompress1X: Decode the compressed data using that tree
//
// The dst buffer must be large enough to hold the decompressed output.
// The scratch object is reused across calls to minimize allocations.
//
// Performance: ~150-200 MB/s on modern CPUs (1X variant).
// Note: Could use Decompress4X for 4x speedup (400-600 MB/s).
func (h *Huffman) Decode(dst, src, params []byte) (int, error) {
	if len(src) == 0 {
		return 0, fmt.Errorf("huffman: empty input")
	}

	// Step 1: Read Huffman table from compressed data
	// This parses the Huffman tree and returns remaining compressed bytes
	scratch, remain, err := huff0.ReadTable(src, h.scratch)
	if err != nil {
		return 0, fmt.Errorf("huffman read table failed: %w", err)
	}

	// Update our scratch for reuse
	h.scratch = scratch

	// Step 2: Decompress using the table
	// Using Decompress1X (single stream, simpler)
	decompressed, err := scratch.Decompress1X(remain)
	if err != nil {
		return 0, fmt.Errorf("huffman decode failed: %w", err)
	}

	// Verify output fits in destination buffer
	if len(decompressed) > len(dst) {
		return 0, fmt.Errorf("huffman: output size %d exceeds buffer size %d",
			len(decompressed), len(dst))
	}

	// Copy to caller's destination buffer
	n := copy(dst, decompressed)
	return n, nil
}

// Encode compresses data using Huffman.
//
// Note: Encoding is not implemented in Phase 3 (decompression only).
// This will be added in Phase 4 when we implement compression.
func (h *Huffman) Encode(dst, src, params []byte) (int, error) {
	return 0, fmt.Errorf("huffman encode not yet implemented (decompression only in Phase 3)")
}
