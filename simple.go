// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package openzl

// Compress compresses the input data using OpenZL with default settings.
// It returns the compressed data or an error.
//
// This is a simple one-shot compression function suitable for occasional use.
// For better performance with repeated operations, use the Compressor type.
//
// Example:
//
//	data := []byte("hello world")
//	compressed, err := openzl.Compress(data)
//	if err != nil {
//		log.Fatal(err)
//	}
func Compress(src []byte) ([]byte, error) {
	// TODO: Implement in Phase 1
	return nil, ErrNotImplemented
}

// Decompress decompresses OpenZL-compressed data.
// It returns the decompressed data or an error.
//
// This is a simple one-shot decompression function suitable for occasional use.
// For better performance with repeated operations, use the Decompressor type.
//
// Example:
//
//	decompressed, err := openzl.Decompress(compressed)
//	if err != nil {
//		log.Fatal(err)
//	}
func Decompress(src []byte) ([]byte, error) {
	// TODO: Implement in Phase 1
	return nil, ErrNotImplemented
}

// ErrNotImplemented is returned by functions not yet implemented
var ErrNotImplemented = &NotImplementedError{}

// NotImplementedError indicates that a feature is not yet implemented
type NotImplementedError struct {
	Feature string
}

func (e *NotImplementedError) Error() string {
	if e.Feature != "" {
		return "openzl: " + e.Feature + " not yet implemented"
	}
	return "openzl: not yet implemented"
}
