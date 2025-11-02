package purgo

import (
	"bytes"
	"testing"
)

// TestCompress_Roundtrip tests basic compression and decompression roundtrip.
func TestCompress_Roundtrip(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"small_text", []byte("hello world")},
		{"empty_string", []byte("")}, // Should error
		{"medium_text", []byte("The quick brown fox jumps over the lazy dog. " +
			"Pack my box with five dozen liquor jugs.")},
		{"binary_data", []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}},
		{"repeated", bytes.Repeat([]byte("A"), 1000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip empty data test (should error)
			if len(tt.data) == 0 {
				_, err := Compress(tt.data)
				if err == nil {
					t.Error("Compress should error on empty data")
				}
				return
			}

			// Compress
			compressed, err := Compress(tt.data)
			if err != nil {
				t.Fatalf("Compress() error = %v", err)
			}

			if len(compressed) == 0 {
				t.Fatal("Compress() returned empty data")
			}

			// Decompress
			decompressed, err := Decompress(compressed)
			if err != nil {
				t.Fatalf("Decompress() error = %v", err)
			}

			// Verify roundtrip
			if !bytes.Equal(decompressed, tt.data) {
				t.Errorf("Roundtrip failed:\noriginal:  %q\nroundtrip: %q", tt.data, decompressed)
			}
		})
	}
}

// TestCompressInt64_Roundtrip tests int64 compression roundtrip.
func TestCompressInt64_Roundtrip(t *testing.T) {
	tests := []struct {
		name string
		data []int64
	}{
		{"sequential", []int64{1, 2, 3, 4, 5}},
		{"sorted", []int64{100, 200, 300, 400, 500}},
		{"timestamps", []int64{1000, 1001, 1002, 1003, 1004}}, // Slowly increasing
		{"small_changes", []int64{100, 101, 99, 100, 102}},    // Small deltas
		{"zeros", []int64{0, 0, 0, 0, 0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Compress
			compressed, err := CompressInt64(tt.data)
			if err != nil {
				t.Fatalf("CompressInt64() error = %v", err)
			}

			// Decompress
			decompressed, err := DecompressInt64(compressed)
			if err != nil {
				t.Fatalf("DecompressInt64() error = %v", err)
			}

			// Verify
			if len(decompressed) != len(tt.data) {
				t.Fatalf("length mismatch: got %d, want %d", len(decompressed), len(tt.data))
			}

			for i := range tt.data {
				if decompressed[i] != tt.data[i] {
					t.Errorf("mismatch at index %d: got %d, want %d", i, decompressed[i], tt.data[i])
				}
			}
		})
	}
}

// TestCompressFloat64_Roundtrip tests float64 compression roundtrip.
func TestCompressFloat64_Roundtrip(t *testing.T) {
	data := []float64{1.0, 2.5, 3.14159, -1.5, 0.0}

	compressed, err := CompressFloat64(data)
	if err != nil {
		t.Fatalf("CompressFloat64() error = %v", err)
	}

	decompressed, err := DecompressFloat64(compressed)
	if err != nil {
		t.Fatalf("DecompressFloat64() error = %v", err)
	}

	if len(decompressed) != len(data) {
		t.Fatalf("length mismatch: got %d, want %d", len(decompressed), len(data))
	}

	for i := range data {
		if decompressed[i] != data[i] {
			t.Errorf("mismatch at index %d: got %f, want %f", i, decompressed[i], data[i])
		}
	}
}

// TestCompressString_Roundtrip tests string compression roundtrip.
func TestCompressString_Roundtrip(t *testing.T) {
	original := "Hello, Pure Go OpenZL compression!"

	compressed, err := CompressString(original)
	if err != nil {
		t.Fatalf("CompressString() error = %v", err)
	}

	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("Decompress() error = %v", err)
	}

	result := string(decompressed)
	if result != original {
		t.Errorf("Roundtrip failed:\noriginal:  %q\nroundtrip: %q", original, result)
	}
}

// BenchmarkCompress benchmarks compression.
func BenchmarkCompress(b *testing.B) {
	data := bytes.Repeat([]byte("test data "), 1000) // 10KB

	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		_, err := Compress(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCompress_Small benchmarks compression of small data.
func BenchmarkCompress_Small(b *testing.B) {
	data := []byte("hello world")

	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		_, err := Compress(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCompressInt64 benchmarks int64 compression.
func BenchmarkCompressInt64(b *testing.B) {
	data := make([]int64, 10000)
	for i := range data {
		data[i] = int64(i)
	}

	b.ResetTimer()
	b.SetBytes(int64(len(data) * 8))

	for i := 0; i < b.N; i++ {
		_, err := CompressInt64(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}
