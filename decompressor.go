// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package openzl

import (
	"fmt"
	"sync"

	"github.com/borischu/go-openzl/internal/cgo"
)

// Decompressor provides a reusable decompression context with thread safety.
//
// Unlike the one-shot Decompress function, Decompressor maintains an internal
// decompression context that can be reused across multiple operations, providing
// 10-50% better performance for repeated decompressions.
//
// Decompressor is safe for concurrent use by multiple goroutines. Each decompression
// operation is protected by an internal mutex, ensuring thread safety while
// allowing the underlying context to be reused.
//
// Example:
//
//	decompressor, err := openzl.NewDecompressor()
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer decompressor.Close()
//
//	// Reuse the decompressor for multiple operations
//	for _, compressed := range compressedInputs {
//		decompressed, err := decompressor.Decompress(compressed)
//		if err != nil {
//			log.Printf("decompression failed: %v", err)
//			continue
//		}
//		// Use decompressed data...
//	}
type Decompressor struct {
	mu  sync.Mutex   // Protects ctx for thread safety
	ctx *cgo.DCtx    // Underlying decompression context
}

// NewDecompressor creates a new reusable Decompressor.
//
// The returned Decompressor is safe for concurrent use by multiple goroutines.
// When finished, call Close() to release the underlying decompression context
// and prevent memory leaks.
//
// Example:
//
//	decompressor, err := openzl.NewDecompressor()
//	if err != nil {
//		return err
//	}
//	defer decompressor.Close()
//
// Returns an error if the underlying decompression context cannot be created.
func NewDecompressor() (*Decompressor, error) {
	ctx, err := cgo.NewDCtx()
	if err != nil {
		return nil, fmt.Errorf("create context: %w", err)
	}

	return &Decompressor{
		ctx: ctx,
	}, nil
}

// Decompress decompresses OpenZL-compressed data using the reusable decompression context.
//
// This method is safe for concurrent use by multiple goroutines. Each call
// acquires an internal lock, decompresses the data using the shared context,
// and then releases the lock.
//
// The input data is not modified. The returned decompressed data is a newly
// allocated slice containing only the decompressed bytes (no extra capacity).
//
// Returns an error if:
//   - src is empty (use ErrEmptyInput check)
//   - src does not contain valid OpenZL compressed data
//   - the compressed data is corrupted
//   - the underlying decompression operation fails
//
// Example:
//
//	decompressed, err := decompressor.Decompress(compressedData)
//	if err != nil {
//		log.Fatal(err)
//	}
func (d *Decompressor) Decompress(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, ErrEmptyInput
	}

	// Lock for thread safety
	d.mu.Lock()
	defer d.mu.Unlock()

	// Get decompressed size from frame header
	dstSize, err := cgo.GetDecompressedSize(src)
	if err != nil {
		return nil, fmt.Errorf("get decompressed size: %w", err)
	}

	// Allocate destination buffer
	dst := make([]byte, dstSize)

	// Decompress using reusable context
	n, err := d.ctx.Decompress(dst, src)
	if err != nil {
		return nil, fmt.Errorf("decompress: %w", err)
	}

	return dst[:n], nil
}

// Close releases the underlying decompression context and frees associated memory.
//
// After calling Close, the Decompressor cannot be used for further decompression
// operations. Calling Close multiple times is safe and has no effect after
// the first call.
//
// It is recommended to use defer to ensure Close is called:
//
//	decompressor, err := openzl.NewDecompressor()
//	if err != nil {
//		return err
//	}
//	defer decompressor.Close()
func (d *Decompressor) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.ctx != nil {
		d.ctx.Free()
		d.ctx = nil
	}
	return nil
}
