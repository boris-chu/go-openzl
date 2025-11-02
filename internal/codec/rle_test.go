// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package codec

import (
	"bytes"
	"testing"
)

// TestRLE_Metadata verifies RLE codec metadata.
func TestRLE_Metadata(t *testing.T) {
	codec := NewRLE()

	if codec.ID() != IDRLE {
		t.Errorf("ID() = %v, want %v", codec.ID(), IDRLE)
	}

	if codec.PreservesSize() {
		t.Error("PreservesSize() = true, want false (RLE changes size)")
	}
}

// TestRLE_EmptyData verifies empty input handling.
func TestRLE_EmptyData(t *testing.T) {
	codec := NewRLE()

	dst := make([]byte, 1024)
	compressed, err := codec.Encode(dst, []byte{}, nil)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	if compressed != 4 {
		t.Errorf("Empty data should encode to 4 bytes (num_runs=0), got %d", compressed)
	}

	// Decompress
	output := make([]byte, 0)
	n, err := codec.Decode(output, dst[:compressed], nil)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if n != 0 {
		t.Errorf("Decompressed %d bytes, expected 0", n)
	}
}

// TestRLE_SingleValue verifies all-same-value encoding (best case).
func TestRLE_SingleValue(t *testing.T) {
	codec := NewRLE()

	// 100 identical values
	input := make([]byte, 100)
	for i := range input {
		input[i] = 0x42 // 'B'
	}

	dst := make([]byte, 1024)
	compressed, err := codec.Encode(dst, input, nil)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	ratio := float64(len(input)) / float64(compressed)
	t.Logf("Single value: %d bytes → %d bytes (%.2fx compression)",
		len(input), compressed, ratio)

	// Should achieve excellent compression (1 run)
	if ratio < 5.0 {
		t.Errorf("Expected at least 5× compression for single value, got %.2fx", ratio)
	}

	// Decompress and verify
	output := make([]byte, len(input))
	n, err := codec.Decode(output, dst[:compressed], nil)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if !bytes.Equal(output[:n], input) {
		t.Error("Roundtrip failed: output != input")
	}
}

// TestRLE_LongRuns verifies encoding of multiple long runs.
func TestRLE_LongRuns(t *testing.T) {
	codec := NewRLE()

	// Pattern: 50 zeros, 50 ones, 50 twos
	input := make([]byte, 150)
	for i := 0; i < 50; i++ {
		input[i] = 0
	}
	for i := 50; i < 100; i++ {
		input[i] = 1
	}
	for i := 100; i < 150; i++ {
		input[i] = 2
	}

	dst := make([]byte, 1024)
	compressed, err := codec.Encode(dst, input, nil)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	ratio := float64(len(input)) / float64(compressed)
	t.Logf("Long runs: %d bytes → %d bytes (%.2fx compression)",
		len(input), compressed, ratio)

	// Should compress well (only 3 runs)
	if ratio < 5.0 {
		t.Errorf("Expected at least 5× compression for long runs, got %.2fx", ratio)
	}

	// Decompress and verify
	output := make([]byte, len(input))
	n, err := codec.Decode(output, dst[:compressed], nil)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if !bytes.Equal(output[:n], input) {
		t.Errorf("Roundtrip failed")
	}
}

// TestRLE_ShortRuns verifies handling of runs at the minRunLength threshold.
func TestRLE_ShortRuns(t *testing.T) {
	codec := NewRLE()

	// Pattern with 2-element runs (exactly at threshold)
	input := []byte{1, 1, 2, 2, 3, 3, 4, 4}

	dst := make([]byte, 1024)
	compressed, err := codec.Encode(dst, input, nil)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	ratio := float64(len(input)) / float64(compressed)
	t.Logf("Short runs: %d bytes → %d bytes (%.2fx)",
		len(input), compressed, ratio)

	// Roundtrip verification
	output := make([]byte, len(input))
	n, err := codec.Decode(output, dst[:compressed], nil)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if !bytes.Equal(output[:n], input) {
		t.Errorf("Roundtrip failed:\nGot:  %v\nWant: %v", output[:n], input)
	}
}

// TestRLE_NoRuns verifies worst-case behavior (all different values).
func TestRLE_NoRuns(t *testing.T) {
	codec := NewRLE()

	// All different values
	input := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	dst := make([]byte, 1024)
	compressed, err := codec.Encode(dst, input, nil)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	ratio := float64(len(input)) / float64(compressed)
	t.Logf("No runs: %d bytes → %d bytes (%.2fx - EXPANSION)",
		len(input), compressed, ratio)

	// This should expand (worst case for RLE)
	if ratio > 1.0 {
		t.Logf("Warning: No-run data should expand, but got %.2fx compression", ratio)
	}

	// Roundtrip verification (even though it expanded)
	output := make([]byte, len(input))
	n, err := codec.Decode(output, dst[:compressed], nil)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if !bytes.Equal(output[:n], input) {
		t.Errorf("Roundtrip failed")
	}
}

// TestRLE_Alternating verifies pathological case (alternating values).
func TestRLE_Alternating(t *testing.T) {
	codec := NewRLE()

	// Worst case: alternating pattern
	input := make([]byte, 20)
	for i := range input {
		input[i] = byte(i % 2) // 0, 1, 0, 1, ...
	}

	dst := make([]byte, 1024)
	compressed, err := codec.Encode(dst, input, nil)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	ratio := float64(len(input)) / float64(compressed)
	t.Logf("Alternating: %d bytes → %d bytes (%.2fx - SEVERE EXPANSION)",
		len(input), compressed, ratio)

	// This is the worst case - should expand significantly
	if ratio > 0.8 {
		t.Errorf("Alternating pattern should cause expansion, got %.2fx", ratio)
	}

	// Roundtrip verification
	output := make([]byte, len(input))
	n, err := codec.Decode(output, dst[:compressed], nil)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if !bytes.Equal(output[:n], input) {
		t.Errorf("Roundtrip failed")
	}
}

// TestRLE_SparseArray verifies compression of sparse arrays (many zeros).
func TestRLE_SparseArray(t *testing.T) {
	codec := NewRLE()

	// Sparse: mostly zeros with occasional non-zero values
	input := make([]byte, 100)
	input[10] = 1
	input[50] = 2
	input[90] = 3
	// Rest are zeros

	dst := make([]byte, 1024)
	compressed, err := codec.Encode(dst, input, nil)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	ratio := float64(len(input)) / float64(compressed)
	t.Logf("Sparse array: %d bytes → %d bytes (%.2fx compression)",
		len(input), compressed, ratio)

	// Should compress well (long runs of zeros)
	if ratio < 2.0 {
		t.Errorf("Expected at least 2× compression for sparse array, got %.2fx", ratio)
	}

	// Roundtrip verification
	output := make([]byte, len(input))
	n, err := codec.Decode(output, dst[:compressed], nil)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if !bytes.Equal(output[:n], input) {
		t.Errorf("Roundtrip failed")
	}
}

// TestRLE_BooleanFlags verifies boolean-like data (0s and 1s).
func TestRLE_BooleanFlags(t *testing.T) {
	codec := NewRLE()

	// Boolean flags: true×20, false×30, true×10
	input := make([]byte, 60)
	for i := 0; i < 20; i++ {
		input[i] = 1 // true
	}
	for i := 20; i < 50; i++ {
		input[i] = 0 // false
	}
	for i := 50; i < 60; i++ {
		input[i] = 1 // true
	}

	dst := make([]byte, 1024)
	compressed, err := codec.Encode(dst, input, nil)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	ratio := float64(len(input)) / float64(compressed)
	t.Logf("Boolean flags: %d bytes → %d bytes (%.2fx compression)",
		len(input), compressed, ratio)

	// Should compress well (only 3 runs)
	if ratio < 3.0 {
		t.Errorf("Expected at least 3× compression for boolean flags, got %.2fx", ratio)
	}

	// Roundtrip verification
	output := make([]byte, len(input))
	n, err := codec.Decode(output, dst[:compressed], nil)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if !bytes.Equal(output[:n], input) {
		t.Errorf("Roundtrip failed")
	}
}

// TestRLE_BufferTooSmall verifies error handling for small buffers.
func TestRLE_BufferTooSmall(t *testing.T) {
	codec := NewRLE()

	input := []byte{1, 1, 1, 1, 1}

	// Buffer way too small
	tooSmall := make([]byte, 2)
	_, err := codec.Encode(tooSmall, input, nil)
	if err != ErrBufferTooSmall {
		t.Errorf("Encode with tiny buffer: got error %v, want %v", err, ErrBufferTooSmall)
	}

	// Encode normally
	dst := make([]byte, 1024)
	compressed, err := codec.Encode(dst, input, nil)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	// Try to decode with buffer too small
	_, err = codec.Decode(tooSmall, dst[:compressed], nil)
	if err != ErrBufferTooSmall {
		t.Errorf("Decode with small buffer: got error %v, want %v", err, ErrBufferTooSmall)
	}
}

// TestRLE_InvalidInput verifies error handling for corrupted data.
func TestRLE_InvalidInput(t *testing.T) {
	codec := NewRLE()

	tests := []struct {
		name  string
		input []byte
	}{
		{"truncated header", []byte{0x01}},
		{"truncated run", []byte{0x01, 0x00, 0x00, 0x00, 0x42}}, // num_runs=1 but no count
		{"invalid varint", []byte{0x01, 0x00, 0x00, 0x00, 0x42, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := make([]byte, 1024)
			_, err := codec.Decode(output, tt.input, nil)
			if err == nil {
				t.Error("Decode with invalid input should return error")
			}
			t.Logf("Correctly rejected invalid input: %v", err)
		})
	}
}

// TestRLE_LargeRun verifies handling of very large run counts.
func TestRLE_LargeRun(t *testing.T) {
	codec := NewRLE()

	// 10,000 identical values
	input := make([]byte, 10000)
	for i := range input {
		input[i] = 0x55
	}

	dst := make([]byte, len(input)*2)
	compressed, err := codec.Encode(dst, input, nil)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	ratio := float64(len(input)) / float64(compressed)
	t.Logf("Large run: %d bytes → %d bytes (%.2fx compression)",
		len(input), compressed, ratio)

	// Should achieve massive compression
	if ratio < 100.0 {
		t.Errorf("Expected at least 100× compression for large run, got %.2fx", ratio)
	}

	// Roundtrip verification
	output := make([]byte, len(input))
	n, err := codec.Decode(output, dst[:compressed], nil)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if !bytes.Equal(output[:n], input) {
		t.Errorf("Roundtrip failed")
	}
}

// BenchmarkRLE_Encode benchmarks RLE encoding.
func BenchmarkRLE_Encode(t *testing.B) {
	codec := NewRLE()

	// Create data with good runs (50% compression typical)
	input := make([]byte, 4096)
	for i := 0; i < len(input); i += 4 {
		val := byte(i / 4)
		input[i] = val
		input[i+1] = val
		input[i+2] = val
		input[i+3] = val
	}

	dst := make([]byte, len(input)*2)

	t.ResetTimer()
	t.SetBytes(int64(len(input)))

	for i := 0; i < t.N; i++ {
		_, err := codec.Encode(dst, input, nil)
		if err != nil {
			t.Fatal(err)
		}
	}
}

// BenchmarkRLE_Decode benchmarks RLE decoding.
func BenchmarkRLE_Decode(t *testing.B) {
	codec := NewRLE()

	// Create and encode test data
	input := make([]byte, 4096)
	for i := 0; i < len(input); i += 4 {
		val := byte(i / 4)
		input[i] = val
		input[i+1] = val
		input[i+2] = val
		input[i+3] = val
	}

	compressed := make([]byte, len(input)*2)
	compressedSize, _ := codec.Encode(compressed, input, nil)
	compressed = compressed[:compressedSize]

	output := make([]byte, len(input))

	t.ResetTimer()
	t.SetBytes(int64(len(input)))

	for i := 0; i < t.N; i++ {
		_, err := codec.Decode(output, compressed, nil)
		if err != nil {
			t.Fatal(err)
		}
	}
}
