// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

//go:build !amd64 || purego

package codec

// decode64SIMD falls back to scalar implementation on non-amd64 platforms.
// Note: These SIMD stubs are intentionally unused - scalar is faster for Delta.
// They're kept for reference and future assembly implementations.
//
//nolint:unused
func (c *Delta) decode64SIMD(dst, src []byte, numElements int) (int, error) {
	return c.decode64(dst, src, numElements)
}

//nolint:unused
func (c *Delta) decode32SIMD(dst, src []byte, numElements int) (int, error) {
	return c.decode32(dst, src, numElements)
}

//nolint:unused
func (c *Delta) encode64SIMD(dst, src []byte, numElements int) (int, error) {
	return c.encode64(dst, src, numElements)
}

//nolint:unused
func (c *Delta) encode32SIMD(dst, src []byte, numElements int) (int, error) {
	return c.encode32(dst, src, numElements)
}
