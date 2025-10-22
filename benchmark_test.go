// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package openzl

import (
	"bytes"
	"testing"
)

// Benchmark data sets
var (
	benchSmallText  = []byte("hello world")
	benchMediumText = bytes.Repeat([]byte("The quick brown fox jumps over the lazy dog. "), 100)
	benchLargeText  = bytes.Repeat([]byte("Lorem ipsum dolor sit amet. "), 1000)
)

// One-shot API benchmarks (Phase 1)

func BenchmarkCompress(b *testing.B) {
	data := benchSmallText
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Compress(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecompress(b *testing.B) {
	data := benchSmallText
	compressed, err := Compress(data)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Decompress(compressed)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Context API benchmarks (Phase 2)

func BenchmarkCompressorCompress(b *testing.B) {
	compressor, err := NewCompressor()
	if err != nil {
		b.Fatal(err)
	}
	defer compressor.Close()

	data := benchSmallText
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := compressor.Compress(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecompressorDecompress(b *testing.B) {
	compressor, err := NewCompressor()
	if err != nil {
		b.Fatal(err)
	}
	defer compressor.Close()

	decompressor, err := NewDecompressor()
	if err != nil {
		b.Fatal(err)
	}
	defer decompressor.Close()

	data := benchSmallText
	compressed, err := compressor.Compress(data)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := decompressor.Decompress(compressed)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Size comparison benchmarks

func BenchmarkCompressor_SmallData(b *testing.B) {
	compressor, err := NewCompressor()
	if err != nil {
		b.Fatal(err)
	}
	defer compressor.Close()

	data := benchSmallText
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := compressor.Compress(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompressor_MediumData(b *testing.B) {
	compressor, err := NewCompressor()
	if err != nil {
		b.Fatal(err)
	}
	defer compressor.Close()

	data := benchMediumText
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := compressor.Compress(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompressor_LargeData(b *testing.B) {
	compressor, err := NewCompressor()
	if err != nil {
		b.Fatal(err)
	}
	defer compressor.Close()

	data := benchLargeText
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := compressor.Compress(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecompressor_SmallData(b *testing.B) {
	compressor, err := NewCompressor()
	if err != nil {
		b.Fatal(err)
	}
	defer compressor.Close()

	decompressor, err := NewDecompressor()
	if err != nil {
		b.Fatal(err)
	}
	defer decompressor.Close()

	compressed, err := compressor.Compress(benchSmallText)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := decompressor.Decompress(compressed)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecompressor_MediumData(b *testing.B) {
	compressor, err := NewCompressor()
	if err != nil {
		b.Fatal(err)
	}
	defer compressor.Close()

	decompressor, err := NewDecompressor()
	if err != nil {
		b.Fatal(err)
	}
	defer decompressor.Close()

	compressed, err := compressor.Compress(benchMediumText)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := decompressor.Decompress(compressed)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecompressor_LargeData(b *testing.B) {
	compressor, err := NewCompressor()
	if err != nil {
		b.Fatal(err)
	}
	defer compressor.Close()

	decompressor, err := NewDecompressor()
	if err != nil {
		b.Fatal(err)
	}
	defer decompressor.Close()

	compressed, err := compressor.Compress(benchLargeText)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := decompressor.Decompress(compressed)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Parallel benchmarks to test concurrent performance

func BenchmarkCompressorParallel(b *testing.B) {
	compressor, err := NewCompressor()
	if err != nil {
		b.Fatal(err)
	}
	defer compressor.Close()

	data := benchSmallText
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := compressor.Compress(data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkDecompressorParallel(b *testing.B) {
	compressor, err := NewCompressor()
	if err != nil {
		b.Fatal(err)
	}
	defer compressor.Close()

	decompressor, err := NewDecompressor()
	if err != nil {
		b.Fatal(err)
	}
	defer decompressor.Close()

	compressed, err := compressor.Compress(benchSmallText)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := decompressor.Decompress(compressed)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
