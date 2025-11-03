// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package codec

import (
	"encoding/binary"
	"testing"
)

// BenchmarkRangePack_Encode benchmarks RangePack encoding
func BenchmarkRangePack_Encode(b *testing.B) {
	codec := NewRangePack()

	// 1000 timestamps with 1-second increments
	count := 1000
	baseTimestamp := uint64(1700000000)
	src := make([]byte, count*8)
	for i := 0; i < count; i++ {
		binary.LittleEndian.PutUint64(src[i*8:], baseTimestamp+uint64(i))
	}

	params := []byte{8}
	dst := make([]byte, len(src)+100)

	b.ResetTimer()
	b.SetBytes(int64(len(src)))

	for i := 0; i < b.N; i++ {
		_, err := codec.Encode(dst, src, params)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRangePack_Decode benchmarks RangePack decoding
func BenchmarkRangePack_Decode(b *testing.B) {
	codec := NewRangePack()

	// Prepare compressed data
	count := 1000
	baseTimestamp := uint64(1700000000)
	src := make([]byte, count*8)
	for i := 0; i < count; i++ {
		binary.LittleEndian.PutUint64(src[i*8:], baseTimestamp+uint64(i))
	}

	params := []byte{8}
	compressed := make([]byte, len(src)+100)
	n, err := codec.Encode(compressed, src, params)
	if err != nil {
		b.Fatal(err)
	}
	compressed = compressed[:n]

	dst := make([]byte, len(src))

	b.ResetTimer()
	b.SetBytes(int64(len(src)))

	for i := 0; i < b.N; i++ {
		_, err := codec.Decode(dst, compressed, params)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRangePack_Roundtrip benchmarks full encode+decode
func BenchmarkRangePack_Roundtrip(b *testing.B) {
	codec := NewRangePack()

	count := 1000
	baseTimestamp := uint64(1700000000)
	src := make([]byte, count*8)
	for i := 0; i < count; i++ {
		binary.LittleEndian.PutUint64(src[i*8:], baseTimestamp+uint64(i))
	}

	params := []byte{8}
	compressed := make([]byte, len(src)+100)
	decompressed := make([]byte, len(src))

	b.ResetTimer()
	b.SetBytes(int64(len(src)))

	for i := 0; i < b.N; i++ {
		n, err := codec.Encode(compressed, src, params)
		if err != nil {
			b.Fatal(err)
		}

		_, err = codec.Decode(decompressed, compressed[:n], params)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRangePack_IDs benchmarks compression of account IDs
func BenchmarkRangePack_IDs(b *testing.B) {
	codec := NewRangePack()

	// 1000 account IDs: 1,000,000 to 1,001,000 (range fits in uint16)
	count := 1000
	src := make([]byte, count*4)
	for i := 0; i < count; i++ {
		binary.LittleEndian.PutUint32(src[i*4:], 1000000+uint32(i))
	}

	params := []byte{4}
	dst := make([]byte, len(src)+100)

	b.ResetTimer()
	b.SetBytes(int64(len(src)))

	for i := 0; i < b.N; i++ {
		_, err := codec.Encode(dst, src, params)
		if err != nil {
			b.Fatal(err)
		}
	}
}
