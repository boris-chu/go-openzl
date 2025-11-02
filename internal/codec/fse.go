// Package codec provides Pure Go OpenZL codec implementations.
//
// This file implements the FSE (Finite State Entropy) codec using
// Klaus Post's excellent compress library.
//
// Copyright (c) 2019 Klaus Post (github.com/klauspost/compress/fse)
// Licensed under BSD 3-Clause License.
package codec

import (
	"fmt"

	"github.com/klauspost/compress/fse"
)

// FSE implements Finite State Entropy (FSE/tANS) decoding.
//
// FSE is a modern entropy coder that achieves near-optimal compression
// ratios with excellent performance. It's used as the primary entropy
// coder in zstd and OpenZL.
//
// This codec wraps Klaus Post's FSE implementation, which has been
// proven to match or exceed C reference implementation performance.
type FSE struct {
	id      ID
	scratch *fse.Scratch // Reused across calls for zero allocations
}

// NewFSE creates a new FSE codec.
func NewFSE() *FSE {
	return &FSE{
		id:      IDFSE,
		scratch: &fse.Scratch{},
	}
}

// ID returns the codec identifier.
func (f *FSE) ID() ID {
	return f.id
}

// Name returns the human-readable codec name.
func (f *FSE) Name() string {
	return "FSE (Finite State Entropy)"
}

// Decode decompresses FSE-encoded data.
//
// The dst buffer must be large enough to hold the decompressed output.
// The scratch object is reused across calls to minimize allocations.
//
// Performance: ~200-300 MB/s on modern CPUs (Klaus Post benchmarks).
func (f *FSE) Decode(dst, src, params []byte) (int, error) {
	if len(src) == 0 {
		return 0, fmt.Errorf("fse: empty input")
	}

	// Set decompression limit to dst buffer size to prevent decompression bombs
	f.scratch.DecompressLimit = len(dst)

	// Clear output buffer to avoid reusing previous data
	// (Klaus Post's library may reuse scratch.Out if set)
	f.scratch.Out = nil

	// Decompress using Klaus Post's FSE implementation
	decompressed, err := fse.Decompress(src, f.scratch)
	if err != nil {
		return 0, fmt.Errorf("fse decode failed: %w", err)
	}

	// Verify output fits in destination buffer
	if len(decompressed) > len(dst) {
		return 0, fmt.Errorf("fse: output size %d exceeds buffer size %d", len(decompressed), len(dst))
	}

	// Copy to caller's destination buffer
	n := copy(dst, decompressed)
	return n, nil
}

// Encode compresses data using FSE.
//
// Note: Encoding is not implemented in Phase 3 (decompression only).
// This will be added in Phase 4 when we implement compression.
func (f *FSE) Encode(dst, src, params []byte) (int, error) {
	if len(src) == 0 {
		return 0, nil
	}

	// Use Klaus Post's FSE compression
	scratch := &fse.Scratch{}
	compressed, err := fse.Compress(src, scratch)
	if err != nil {
		return 0, fmt.Errorf("fse compress: %w", err)
	}

	// Check if compressed data fits in destination
	if len(compressed) > len(dst) {
		return 0, ErrBufferTooSmall
	}

	// Copy compressed data to destination
	n := copy(dst, compressed)

	return n, nil
}

// PreservesSize returns false because FSE is an entropy coder that changes size.
//
// FSE compresses data by using fewer bits for more frequent symbols,
// typically achieving 1.5-3x compression on byte streams.
//
// This is a size-changing codec that requires explicit size metadata.
func (f *FSE) PreservesSize() bool {
	return false
}
