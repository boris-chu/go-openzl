//go:build !cgo
// +build !cgo

// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package openzl

import "fmt"

// Compressor provides a reusable compression context with thread safety.
//
// Note: Compressor requires CGO. In Pure Go builds (CGO_ENABLED=0), this type
// is not available. For Pure Go decompression, use the one-shot Decompress function
// or the streaming purgo.Reader.
type Compressor struct{}

// CompressorOption configures a Compressor during creation.
type CompressorOption func(*config) error

// config holds the configuration options for Compressor.
type config struct{}

// NewCompressor creates a new reusable Compressor with optional configuration.
//
// Note: Compression requires CGO. Build with CGO_ENABLED=1 to use this function.
func NewCompressor(opts ...CompressorOption) (*Compressor, error) {
	return nil, fmt.Errorf("Compressor requires CGO (build with CGO_ENABLED=1)")
}

// Compress compresses the input data using the reusable compression context.
//
// Note: Compression requires CGO. Build with CGO_ENABLED=1 to use this function.
func (c *Compressor) Compress(src []byte) ([]byte, error) {
	return nil, fmt.Errorf("Compressor requires CGO (build with CGO_ENABLED=1)")
}

// CompressTo compresses src into dst, returning the number of bytes written.
//
// Note: Compression requires CGO. Build with CGO_ENABLED=1 to use this function.
func (c *Compressor) CompressTo(dst, src []byte) (int, error) {
	return 0, fmt.Errorf("Compressor requires CGO (build with CGO_ENABLED=1)")
}

// Close releases the underlying compression context and frees associated memory.
//
// Note: This is a no-op in Pure Go builds.
func (c *Compressor) Close() error {
	return nil
}
