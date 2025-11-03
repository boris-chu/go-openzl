// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package codec

import (
	"testing"
)

// BenchmarkParseInt_Encode benchmarks ParseInt encoding (text to binary)
func BenchmarkParseInt_Encode(b *testing.B) {
	codec := NewParseInt()

	// 100 integers
	strings := make([]string, 100)
	for i := 0; i < 100; i++ {
		strings[i] = "1000000" // 7 characters
	}
	src := encodeIntStringArray(strings)

	dst := make([]byte, len(strings)*8+100)

	b.ResetTimer()
	b.SetBytes(int64(len(src)))

	for i := 0; i < b.N; i++ {
		_, err := codec.Encode(dst, src, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseInt_Decode benchmarks ParseInt decoding (binary to text)
func BenchmarkParseInt_Decode(b *testing.B) {
	codec := NewParseInt()

	// Prepare binary data
	strings := make([]string, 100)
	for i := 0; i < 100; i++ {
		strings[i] = "1000000"
	}
	src := encodeIntStringArray(strings)

	compressed := make([]byte, len(strings)*8+100)
	n, err := codec.Encode(compressed, src, nil)
	if err != nil {
		b.Fatal(err)
	}
	compressed = compressed[:n]

	dst := make([]byte, len(src)+100)

	b.ResetTimer()
	b.SetBytes(int64(len(src)))

	for i := 0; i < b.N; i++ {
		_, err := codec.Decode(dst, compressed, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseInt_Roundtrip benchmarks full encode+decode
func BenchmarkParseInt_Roundtrip(b *testing.B) {
	codec := NewParseInt()

	strings := make([]string, 100)
	for i := 0; i < 100; i++ {
		strings[i] = "1000000"
	}
	src := encodeIntStringArray(strings)

	compressed := make([]byte, len(strings)*8+100)
	decompressed := make([]byte, len(src)+100)

	b.ResetTimer()
	b.SetBytes(int64(len(src)))

	for i := 0; i < b.N; i++ {
		n, err := codec.Encode(compressed, src, nil)
		if err != nil {
			b.Fatal(err)
		}

		_, err = codec.Decode(decompressed, compressed[:n], nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseInt_SequentialIDs benchmarks parsing sequential IDs
func BenchmarkParseInt_SequentialIDs(b *testing.B) {
	codec := NewParseInt()

	// Sequential IDs (common in CSV)
	strings := make([]string, 100)
	for i := 0; i < 100; i++ {
		strings[i] = "10001"
	}
	src := encodeIntStringArray(strings)

	dst := make([]byte, len(strings)*8+100)

	b.ResetTimer()
	b.SetBytes(int64(len(src)))

	for i := 0; i < b.N; i++ {
		_, err := codec.Encode(dst, src, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}
