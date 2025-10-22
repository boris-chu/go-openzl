// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

// Package openzl provides Go bindings for Meta's OpenZL format-aware compression library.
//
// OpenZL delivers high compression ratios while preserving high speed, a level of performance
// that is out of reach for generic compressors. It takes a description of your data and builds
// from it a specialized compressor optimized for your specific format.
//
// # Quick Start
//
// For simple one-shot compression and decompression:
//
//	// Compress data
//	compressed, err := openzl.Compress([]byte("hello world"))
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Decompress data
//	decompressed, err := openzl.Decompress(compressed)
//	if err != nil {
//		log.Fatal(err)
//	}
//
// # Context-Based API
//
// For better performance with repeated operations, use reusable contexts:
//
//	compressor, err := openzl.NewCompressor()
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer compressor.Close()
//
//	dst := make([]byte, compressor.CompressBound(len(src)))
//	n, err := compressor.Compress(dst, src)
//	if err != nil {
//		log.Fatal(err)
//	}
//	compressed := dst[:n]
//
// # Thread Safety
//
// Compressor and Decompressor instances are safe for concurrent use by multiple goroutines.
// Each instance uses a mutex to serialize access to the underlying C context.
//
// # Requirements
//
// This package requires CGO and links against the OpenZL C library. The library will be
// automatically built as part of the Go build process if not already present.
//
// # Platform Support
//
// Currently supported platforms:
//   - Linux (amd64, arm64)
//   - macOS (amd64, arm64)
//   - Windows (amd64) - experimental
//
// # More Information
//
// For more details about OpenZL, see:
//   - GitHub: https://github.com/facebook/openzl
//   - Documentation: http://openzl.org/
//   - Whitepaper: https://arxiv.org/abs/2510.03203
package openzl
