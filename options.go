// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package openzl

// This file will contain configuration options for Compressor.
//
// Note: Phase 2 establishes the options pattern framework.
// Actual option implementations (WithCompressionLevel, WithChecksum, etc.)
// will be added as we discover which OpenZL parameters are available and useful.
//
// For now, the config struct exists but has no fields.
// This allows the API to accept options without breaking changes when we add them.

// Example future options:
//
// WithCompressionLevel sets the compression level (1-9).
// Higher levels provide better compression but are slower.
//
//	func WithCompressionLevel(level int) CompressorOption {
//		return func(cfg *config) error {
//			if level < 1 || level > 9 {
//				return fmt.Errorf("compression level must be 1-9, got %d", level)
//			}
//			cfg.compressionLevel = level
//			return nil
//		}
//	}
//
// WithChecksum enables checksum verification.
//
//	func WithChecksum(enabled bool) CompressorOption {
//		return func(cfg *config) error {
//			cfg.checksum = enabled
//			return nil
//		}
//	}
