package openzl

import (
	"bytes"
	"compress/gzip"
	"testing"

	"github.com/klauspost/compress/zstd"
)

// Benchmark data generators
func generateRepeatedData(size int) []byte {
	pattern := []byte("This is a test pattern that repeats. ")
	data := make([]byte, 0, size)
	for len(data) < size {
		data = append(data, pattern...)
	}
	return data[:size]
}

func generateMixedData(size int) []byte {
	// Mix of repeated and varied data
	data := make([]byte, size)
	for i := range data {
		if i%100 < 50 {
			// Repeated pattern
			data[i] = byte(i % 10)
		} else {
			// More varied
			data[i] = byte((i * 7) % 256)
		}
	}
	return data
}

func generateTextData(size int) []byte {
	// Simulate text with spaces and common words
	words := []string{"the", "quick", "brown", "fox", "jumps", "over", "lazy", "dog"}
	var buf bytes.Buffer
	for buf.Len() < size {
		for _, word := range words {
			buf.WriteString(word)
			buf.WriteByte(' ')
			if buf.Len() >= size {
				break
			}
		}
	}
	return buf.Bytes()[:size]
}

// ============================================================================
// Small Data Benchmarks (1KB)
// ============================================================================

func BenchmarkSmall_OpenZL(b *testing.B) {
	data := generateRepeatedData(1024)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		compressed, err := Compress(data)
		if err != nil {
			b.Fatal(err)
		}
		_, err = Decompress(compressed)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSmall_Gzip(b *testing.B) {
	data := generateRepeatedData(1024)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		_, err := w.Write(data)
		if err != nil {
			b.Fatal(err)
		}
		w.Close()

		r, err := gzip.NewReader(&buf)
		if err != nil {
			b.Fatal(err)
		}
		var out bytes.Buffer
		_, err = out.ReadFrom(r)
		if err != nil {
			b.Fatal(err)
		}
		r.Close()
	}
}

func BenchmarkSmall_Zstd(b *testing.B) {
	data := generateRepeatedData(1024)
	encoder, _ := zstd.NewWriter(nil)
	decoder, _ := zstd.NewReader(nil)
	defer encoder.Close()
	defer decoder.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		compressed := encoder.EncodeAll(data, nil)
		_, err := decoder.DecodeAll(compressed, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ============================================================================
// Medium Data Benchmarks (100KB)
// ============================================================================

func BenchmarkMedium_OpenZL(b *testing.B) {
	data := generateRepeatedData(100 * 1024)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		compressed, err := Compress(data)
		if err != nil {
			b.Fatal(err)
		}
		_, err = Decompress(compressed)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMedium_Gzip(b *testing.B) {
	data := generateRepeatedData(100 * 1024)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		w.Write(data)
		w.Close()

		r, _ := gzip.NewReader(&buf)
		var out bytes.Buffer
		out.ReadFrom(r)
		r.Close()
	}
}

func BenchmarkMedium_Zstd(b *testing.B) {
	data := generateRepeatedData(100 * 1024)
	encoder, _ := zstd.NewWriter(nil)
	decoder, _ := zstd.NewReader(nil)
	defer encoder.Close()
	defer decoder.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		compressed := encoder.EncodeAll(data, nil)
		decoder.DecodeAll(compressed, nil)
	}
}

// ============================================================================
// Large Data Benchmarks (1MB)
// ============================================================================

func BenchmarkLarge_OpenZL(b *testing.B) {
	data := generateRepeatedData(1024 * 1024)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		compressed, err := Compress(data)
		if err != nil {
			b.Fatal(err)
		}
		_, err = Decompress(compressed)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLarge_Gzip(b *testing.B) {
	data := generateRepeatedData(1024 * 1024)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		w.Write(data)
		w.Close()

		r, _ := gzip.NewReader(&buf)
		var out bytes.Buffer
		out.ReadFrom(r)
		r.Close()
	}
}

func BenchmarkLarge_Zstd(b *testing.B) {
	data := generateRepeatedData(1024 * 1024)
	encoder, _ := zstd.NewWriter(nil)
	decoder, _ := zstd.NewReader(nil)
	defer encoder.Close()
	defer decoder.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		compressed := encoder.EncodeAll(data, nil)
		decoder.DecodeAll(compressed, nil)
	}
}

// ============================================================================
// Compression Ratio Benchmarks
// ============================================================================

func BenchmarkRatio_Repeated_OpenZL(b *testing.B) {
	data := generateRepeatedData(100 * 1024)
	compressed, _ := Compress(data)
	ratio := float64(len(data)) / float64(len(compressed))
	b.ReportMetric(ratio, "ratio")
	b.ReportMetric(float64(len(compressed)), "compressed_bytes")
}

func BenchmarkRatio_Repeated_Gzip(b *testing.B) {
	data := generateRepeatedData(100 * 1024)
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	w.Write(data)
	w.Close()
	ratio := float64(len(data)) / float64(buf.Len())
	b.ReportMetric(ratio, "ratio")
	b.ReportMetric(float64(buf.Len()), "compressed_bytes")
}

func BenchmarkRatio_Repeated_Zstd(b *testing.B) {
	data := generateRepeatedData(100 * 1024)
	encoder, _ := zstd.NewWriter(nil)
	defer encoder.Close()
	compressed := encoder.EncodeAll(data, nil)
	ratio := float64(len(data)) / float64(len(compressed))
	b.ReportMetric(ratio, "ratio")
	b.ReportMetric(float64(len(compressed)), "compressed_bytes")
}

func BenchmarkRatio_Mixed_OpenZL(b *testing.B) {
	data := generateMixedData(100 * 1024)
	compressed, _ := Compress(data)
	ratio := float64(len(data)) / float64(len(compressed))
	b.ReportMetric(ratio, "ratio")
	b.ReportMetric(float64(len(compressed)), "compressed_bytes")
}

func BenchmarkRatio_Mixed_Gzip(b *testing.B) {
	data := generateMixedData(100 * 1024)
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	w.Write(data)
	w.Close()
	ratio := float64(len(data)) / float64(buf.Len())
	b.ReportMetric(ratio, "ratio")
	b.ReportMetric(float64(buf.Len()), "compressed_bytes")
}

func BenchmarkRatio_Mixed_Zstd(b *testing.B) {
	data := generateMixedData(100 * 1024)
	encoder, _ := zstd.NewWriter(nil)
	defer encoder.Close()
	compressed := encoder.EncodeAll(data, nil)
	ratio := float64(len(data)) / float64(len(compressed))
	b.ReportMetric(ratio, "ratio")
	b.ReportMetric(float64(len(compressed)), "compressed_bytes")
}

func BenchmarkRatio_Text_OpenZL(b *testing.B) {
	data := generateTextData(100 * 1024)
	compressed, _ := Compress(data)
	ratio := float64(len(data)) / float64(len(compressed))
	b.ReportMetric(ratio, "ratio")
	b.ReportMetric(float64(len(compressed)), "compressed_bytes")
}

func BenchmarkRatio_Text_Gzip(b *testing.B) {
	data := generateTextData(100 * 1024)
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	w.Write(data)
	w.Close()
	ratio := float64(len(data)) / float64(buf.Len())
	b.ReportMetric(ratio, "ratio")
	b.ReportMetric(float64(buf.Len()), "compressed_bytes")
}

func BenchmarkRatio_Text_Zstd(b *testing.B) {
	data := generateTextData(100 * 1024)
	encoder, _ := zstd.NewWriter(nil)
	defer encoder.Close()
	compressed := encoder.EncodeAll(data, nil)
	ratio := float64(len(data)) / float64(len(compressed))
	b.ReportMetric(ratio, "ratio")
	b.ReportMetric(float64(len(compressed)), "compressed_bytes")
}

// ============================================================================
// Compress-Only Benchmarks
// ============================================================================

func BenchmarkCompressOnly_OpenZL(b *testing.B) {
	data := generateRepeatedData(100 * 1024)
	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		_, err := Compress(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompressOnly_Gzip(b *testing.B) {
	data := generateRepeatedData(100 * 1024)
	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		w.Write(data)
		w.Close()
	}
}

func BenchmarkCompressOnly_Zstd(b *testing.B) {
	data := generateRepeatedData(100 * 1024)
	encoder, _ := zstd.NewWriter(nil)
	defer encoder.Close()

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		encoder.EncodeAll(data, nil)
	}
}

// ============================================================================
// Decompress-Only Benchmarks
// ============================================================================

func BenchmarkDecompressOnly_OpenZL(b *testing.B) {
	data := generateRepeatedData(100 * 1024)
	compressed, _ := Compress(data)

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		_, err := Decompress(compressed)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecompressOnly_Gzip(b *testing.B) {
	data := generateRepeatedData(100 * 1024)
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	w.Write(data)
	w.Close()
	compressed := buf.Bytes()

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		r, _ := gzip.NewReader(bytes.NewReader(compressed))
		var out bytes.Buffer
		out.ReadFrom(r)
		r.Close()
	}
}

func BenchmarkDecompressOnly_Zstd(b *testing.B) {
	data := generateRepeatedData(100 * 1024)
	encoder, _ := zstd.NewWriter(nil)
	compressed := encoder.EncodeAll(data, nil)
	encoder.Close()

	decoder, _ := zstd.NewReader(nil)
	defer decoder.Close()

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		decoder.DecodeAll(compressed, nil)
	}
}

// ============================================================================
// Typed Numeric Compression Benchmarks (OpenZL advantage)
// ============================================================================

func BenchmarkNumeric_Int64_OpenZL(b *testing.B) {
	// Sequential int64 array - OpenZL's strength
	data := make([]int64, 1000)
	for i := range data {
		data[i] = int64(i * 10)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		compressed, err := CompressNumeric(data)
		if err != nil {
			b.Fatal(err)
		}
		_, err = DecompressNumeric[int64](compressed)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNumeric_Int64_AsBytes_Gzip(b *testing.B) {
	// Convert int64 to bytes for gzip (typical approach)
	data := make([]int64, 1000)
	for i := range data {
		data[i] = int64(i * 10)
	}

	// Convert to bytes
	byteData := make([]byte, len(data)*8)
	for i, v := range data {
		offset := i * 8
		byteData[offset] = byte(v)
		byteData[offset+1] = byte(v >> 8)
		byteData[offset+2] = byte(v >> 16)
		byteData[offset+3] = byte(v >> 24)
		byteData[offset+4] = byte(v >> 32)
		byteData[offset+5] = byte(v >> 40)
		byteData[offset+6] = byte(v >> 48)
		byteData[offset+7] = byte(v >> 56)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		w.Write(byteData)
		w.Close()

		r, _ := gzip.NewReader(&buf)
		var out bytes.Buffer
		out.ReadFrom(r)
		r.Close()
	}
}

func BenchmarkNumeric_Int64_AsBytes_Zstd(b *testing.B) {
	data := make([]int64, 1000)
	for i := range data {
		data[i] = int64(i * 10)
	}

	byteData := make([]byte, len(data)*8)
	for i, v := range data {
		offset := i * 8
		byteData[offset] = byte(v)
		byteData[offset+1] = byte(v >> 8)
		byteData[offset+2] = byte(v >> 16)
		byteData[offset+3] = byte(v >> 24)
		byteData[offset+4] = byte(v >> 32)
		byteData[offset+5] = byte(v >> 40)
		byteData[offset+6] = byte(v >> 48)
		byteData[offset+7] = byte(v >> 56)
	}

	encoder, _ := zstd.NewWriter(nil)
	decoder, _ := zstd.NewReader(nil)
	defer encoder.Close()
	defer decoder.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		compressed := encoder.EncodeAll(byteData, nil)
		decoder.DecodeAll(compressed, nil)
	}
}
