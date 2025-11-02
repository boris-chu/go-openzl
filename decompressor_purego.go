//go:build !cgo
// +build !cgo

// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package openzl

import "fmt"

// Decompressor provides a reusable decompression context with thread safety.
//
// Note: Decompressor requires CGO. In Pure Go builds (CGO_ENABLED=0), this type
// is not available. For Pure Go decompression, use the one-shot Decompress function
// or the streaming purgo.Reader.
type Decompressor struct{}

// NewDecompressor creates a new reusable Decompressor.
//
// Note: This function is not available in Pure Go builds. Use the one-shot Decompress
// function or purgo.Reader instead.
func NewDecompressor() (*Decompressor, error) {
	return nil, fmt.Errorf("Decompressor requires CGO (use Decompress or purgo.Reader instead, or build with CGO_ENABLED=1)")
}

// Decompress decompresses OpenZL-compressed data using the reusable decompression context.
//
// Note: This function is not available in Pure Go builds. Use the one-shot Decompress
// function or purgo.Reader instead.
func (d *Decompressor) Decompress(src []byte) ([]byte, error) {
	return nil, fmt.Errorf("Decompressor requires CGO (use Decompress or purgo.Reader instead, or build with CGO_ENABLED=1)")
}

// Close releases the underlying decompression context and frees associated memory.
//
// Note: This is a no-op in Pure Go builds.
func (d *Decompressor) Close() error {
	return nil
}
