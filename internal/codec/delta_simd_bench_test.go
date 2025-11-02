// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package codec

import (
	"encoding/binary"
	"testing"
)

// BenchmarkDeltaDecode64_SIMD benchmarks SIMD-optimized Delta decoding for int64
func BenchmarkDeltaDecode64_SIMD(b *testing.B) {
	sizes := []int{100, 1000, 10000, 100000}

	for _, size := range sizes {
		b.Run(formatSize(size), func(b *testing.B) {
			// Create test data: sequential timestamps (ideal for Delta)
			deltas := make([]byte, size*8)
			for i := 0; i < size; i++ {
				// Small deltas (typical for timestamps)
				binary.LittleEndian.PutUint64(deltas[i*8:], uint64(i%100+1))
			}

			codec := NewDelta(8)
			dst := make([]byte, size*8)

			b.SetBytes(int64(size * 8))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := codec.Decode(dst, deltas, nil)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkDeltaDecode64_Scalar benchmarks scalar Delta decoding for comparison
func BenchmarkDeltaDecode64_Scalar(b *testing.B) {
	sizes := []int{100, 1000, 10000, 100000}

	for _, size := range sizes {
		b.Run(formatSize(size), func(b *testing.B) {
			deltas := make([]byte, size*8)
			for i := 0; i < size; i++ {
				binary.LittleEndian.PutUint64(deltas[i*8:], uint64(i%100+1))
			}

			codec := NewDelta(8)
			dst := make([]byte, size*8)

			b.SetBytes(int64(size * 8))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Force scalar version
				_, err := codec.decode64(dst, deltas, size)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkDeltaDecode32_SIMD benchmarks SIMD-optimized Delta decoding for int32
func BenchmarkDeltaDecode32_SIMD(b *testing.B) {
	sizes := []int{100, 1000, 10000, 100000}

	for _, size := range sizes {
		b.Run(formatSize(size), func(b *testing.B) {
			deltas := make([]byte, size*4)
			for i := 0; i < size; i++ {
				binary.LittleEndian.PutUint32(deltas[i*4:], uint32(i%100+1))
			}

			codec := NewDelta(4)
			dst := make([]byte, size*4)

			b.SetBytes(int64(size * 4))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := codec.Decode(dst, deltas, nil)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkDeltaDecode32_Scalar benchmarks scalar Delta decoding for int32
func BenchmarkDeltaDecode32_Scalar(b *testing.B) {
	sizes := []int{100, 1000, 10000, 100000}

	for _, size := range sizes {
		b.Run(formatSize(size), func(b *testing.B) {
			deltas := make([]byte, size*4)
			for i := 0; i < size; i++ {
				binary.LittleEndian.PutUint32(deltas[i*4:], uint32(i%100+1))
			}

			codec := NewDelta(4)
			dst := make([]byte, size*4)

			b.SetBytes(int64(size * 4))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := codec.decode32(dst, deltas, size)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkDeltaEncode64_SIMD benchmarks SIMD-optimized Delta encoding for int64
func BenchmarkDeltaEncode64_SIMD(b *testing.B) {
	sizes := []int{100, 1000, 10000, 100000}

	for _, size := range sizes {
		b.Run(formatSize(size), func(b *testing.B) {
			// Create sequential data
			values := make([]byte, size*8)
			for i := 0; i < size; i++ {
				binary.LittleEndian.PutUint64(values[i*8:], uint64(1000+i*5))
			}

			codec := NewDelta(8)
			dst := make([]byte, size*8)

			b.SetBytes(int64(size * 8))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := codec.Encode(dst, values, nil)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkDeltaEncode64_Scalar benchmarks scalar Delta encoding for int64
func BenchmarkDeltaEncode64_Scalar(b *testing.B) {
	sizes := []int{100, 1000, 10000, 100000}

	for _, size := range sizes {
		b.Run(formatSize(size), func(b *testing.B) {
			values := make([]byte, size*8)
			for i := 0; i < size; i++ {
				binary.LittleEndian.PutUint64(values[i*8:], uint64(1000+i*5))
			}

			codec := NewDelta(8)
			dst := make([]byte, size*8)

			b.SetBytes(int64(size * 8))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := codec.encode64(dst, values, size)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// formatSize formats benchmark size labels
func formatSize(n int) string {
	if n >= 1000 {
		return string(rune('0'+n/1000)) + "K"
	}
	return string(rune('0' + n/100))
}
