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
// 2. Decompress4X/1X: Decode the compressed data using that tree
//
// The dst buffer must be large enough to hold the decompressed output.
// The scratch object is reused across calls to minimize allocations.
//
// Performance:
// - 4X variant: ~1.1-1.4 GB/s on modern CPUs (4 parallel streams)
// - 1X variant: ~283-338 MB/s on modern CPUs (single stream fallback)
//
// This implementation automatically selects 4X for better performance,
// falling back to 1X if 4X is not available (e.g., small data).
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

	// Step 2: Get a stateless decoder (thread-safe, supports both 1X and 4X)
	decoder := scratch.Decoder()

	// Step 3: Try 4X decompression first (4x faster)
	// This splits the data into 4 independent streams for parallel decoding
	var decompressed []byte
	decompressed, err = decoder.Decompress4X(dst, remain)
	if err != nil {
		// 4X failed (might be too small, or 1X encoding)
		// Fall back to 1X decompression
		decompressed, err = decoder.Decompress1X(dst, remain)
		if err != nil {
			return 0, fmt.Errorf("huffman decode failed (both 4X and 1X): %w", err)
		}
	}

	// Verify output fits in destination buffer
	if len(decompressed) > len(dst) {
		return 0, fmt.Errorf("huffman: output size %d exceeds buffer size %d",
			len(decompressed), len(dst))
	}

	// Return length (data is already in dst from Decompress4X/1X)
	return len(decompressed), nil
}

// Encode compresses data using Huffman.
//
// Note: Encoding is not implemented in Phase 3 (decompression only).
// This will be added in Phase 4 when we implement compression.
func (h *Huffman) Encode(dst, src, params []byte) (int, error) {
	if len(src) == 0 {
		return 0, nil
	}

	// Use Klaus Post's Compress1X (single stream)
	// This is simpler and works well for most data
	scratch := &huff0.Scratch{}
	compressed, reused, err := huff0.Compress1X(src, scratch)
	if err != nil {
		return 0, fmt.Errorf("huffman compress1x: %w", err)
	}

	// Check if compressed data fits in destination
	if len(compressed) > len(dst) {
		return 0, ErrBufferTooSmall
	}

	// Copy compressed data to destination
	n := copy(dst, compressed)

	// Note: 'reused' indicates if the scratch buffer was reused
	// We don't need to track this for one-shot compression
	_ = reused

	return n, nil
}

// PreservesSize returns false because Huffman is an entropy coder that changes size.
//
// Huffman compresses data by assigning shorter codes to more frequent symbols,
// typically achieving 1.5-3x compression on text and byte streams.
//
// This is a size-changing codec that requires explicit size metadata.
func (h *Huffman) PreservesSize() bool {
	return false
}
