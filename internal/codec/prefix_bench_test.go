// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package codec

import (
	"testing"
)

// BenchmarkPrefix_Encode benchmarks Prefix encoding
func BenchmarkPrefix_Encode(b *testing.B) {
	codec := NewPrefix()

	// 100 URLs with common base
	strings := make([]string, 100)
	for i := 0; i < 100; i++ {
		strings[i] = "https://api.example.com/v1/users/" + string(rune('a'+i%26))
	}
	src := encodeStringArray(strings)

	dst := make([]byte, len(src)+1000)

	b.ResetTimer()
	b.SetBytes(int64(len(src)))

	for i := 0; i < b.N; i++ {
		_, err := codec.Encode(dst, src, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPrefix_Decode benchmarks Prefix decoding
func BenchmarkPrefix_Decode(b *testing.B) {
	codec := NewPrefix()

	// Prepare compressed data
	strings := make([]string, 100)
	for i := 0; i < 100; i++ {
		strings[i] = "https://api.example.com/v1/users/" + string(rune('a'+i%26))
	}
	src := encodeStringArray(strings)

	compressed := make([]byte, len(src)+1000)
	n, err := codec.Encode(compressed, src, nil)
	if err != nil {
		b.Fatal(err)
	}
	compressed = compressed[:n]

	dst := make([]byte, len(src)+1000)

	b.ResetTimer()
	b.SetBytes(int64(len(src)))

	for i := 0; i < b.N; i++ {
		_, err := codec.Decode(dst, compressed, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPrefix_Roundtrip benchmarks full encode+decode
func BenchmarkPrefix_Roundtrip(b *testing.B) {
	codec := NewPrefix()

	strings := make([]string, 100)
	for i := 0; i < 100; i++ {
		strings[i] = "https://api.example.com/v1/users/" + string(rune('a'+i%26))
	}
	src := encodeStringArray(strings)

	compressed := make([]byte, len(src)+1000)
	decompressed := make([]byte, len(src)+1000)

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

// BenchmarkPrefix_URLList benchmarks compression of URL list
func BenchmarkPrefix_URLList(b *testing.B) {
	codec := NewPrefix()

	urls := []string{
		"https://api.example.com/v1/users",
		"https://api.example.com/v1/posts",
		"https://api.example.com/v1/comments",
		"https://api.example.com/v2/users",
		"https://api.example.com/v2/posts",
	}
	src := encodeStringArray(urls)

	dst := make([]byte, len(src)+100)

	b.ResetTimer()
	b.SetBytes(int64(len(src)))

	for i := 0; i < b.N; i++ {
		_, err := codec.Encode(dst, src, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}
