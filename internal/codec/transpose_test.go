// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package codec

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestTranspose_Metadata verifies Transpose codec metadata.
func TestTranspose_Metadata(t *testing.T) {
	codec := NewTranspose()

	if codec.ID() != IDTranspose {
		t.Errorf("ID() = %v, want %v", codec.ID(), IDTranspose)
	}

	if !codec.PreservesSize() {
		t.Error("PreservesSize() = false, want true (Transpose preserves size)")
	}
}

// TestTranspose_EmptyData verifies empty input handling.
func TestTranspose_EmptyData(t *testing.T) {
	codec := NewTranspose()
	params := []byte{4} // width=4

	dst := make([]byte, 1024)
	n, err := codec.Encode(dst, []byte{}, params)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	if n != 0 {
		t.Errorf("Empty data should encode to 0 bytes, got %d", n)
	}

	// Decode
	output := make([]byte, 0)
	n, err = codec.Decode(output, dst[:n], params)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if n != 0 {
		t.Errorf("Decompressed %d bytes, expected 0", n)
	}
}

// TestTranspose_Width1 verifies width=1 (single-byte) is no-op.
func TestTranspose_Width1(t *testing.T) {
	codec := NewTranspose()
	params := []byte{1} // width=1

	input := []byte{1, 2, 3, 4, 5}
	dst := make([]byte, len(input))

	// Encode (should be no-op)
	n, err := codec.Encode(dst, input, params)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	if !bytes.Equal(dst[:n], input) {
		t.Errorf("Width=1 should be no-op:\nGot:  %v\nWant: %v", dst[:n], input)
	}

	// Decode (should also be no-op)
	output := make([]byte, len(input))
	n, err = codec.Decode(output, dst[:n], params)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if !bytes.Equal(output[:n], input) {
		t.Errorf("Roundtrip failed for width=1")
	}
}

// TestTranspose_Width2 verifies uint16 transposition.
func TestTranspose_Width2(t *testing.T) {
	codec := NewTranspose()
	params := []byte{2} // width=2 (uint16)

	// Input: 4 uint16 values in little-endian
	// 0x1234, 0x5678, 0x9ABC, 0xDEF0
	input := []byte{
		0x34, 0x12, // 0x1234
		0x78, 0x56, // 0x5678
		0xBC, 0x9A, // 0x9ABC
		0xF0, 0xDE, // 0xDEF0
	}

	dst := make([]byte, len(input))
	n, err := codec.Encode(dst, input, params)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	// Expected after transpose:
	// Byte 0: 0x34 0x78 0xBC 0xF0  (low bytes)
	// Byte 1: 0x12 0x56 0x9A 0xDE  (high bytes)
	expected := []byte{
		0x34, 0x78, 0xBC, 0xF0, // Byte position 0
		0x12, 0x56, 0x9A, 0xDE, // Byte position 1
	}

	if !bytes.Equal(dst[:n], expected) {
		t.Errorf("Transpose failed:\nGot:  %02x\nWant: %02x", dst[:n], expected)
	}

	// Roundtrip verification
	output := make([]byte, len(input))
	n, err = codec.Decode(output, dst[:n], params)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if !bytes.Equal(output[:n], input) {
		t.Errorf("Roundtrip failed:\nGot:  %02x\nWant: %02x", output[:n], input)
	}
}

// TestTranspose_Width4 verifies uint32 transposition.
func TestTranspose_Width4(t *testing.T) {
	codec := NewTranspose()
	params := []byte{4} // width=4 (uint32)

	// Input: 3 uint32 values
	// 0x12345678, 0x12345679, 0x1234567A
	input := make([]byte, 12)
	binary.LittleEndian.PutUint32(input[0:], 0x12345678)
	binary.LittleEndian.PutUint32(input[4:], 0x12345679)
	binary.LittleEndian.PutUint32(input[8:], 0x1234567A)

	dst := make([]byte, len(input))
	n, err := codec.Encode(dst, input, params)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	// Expected: high bytes should be identical
	// Byte 0: 78 79 7A (varying)
	// Byte 1: 56 56 56 (identical)
	// Byte 2: 34 34 34 (identical)
	// Byte 3: 12 12 12 (identical)

	// Verify high bytes are grouped
	if dst[3] != 0x56 || dst[4] != 0x56 || dst[5] != 0x56 {
		t.Errorf("High byte 1 not grouped correctly: %02x", dst[3:6])
	}
	if dst[6] != 0x34 || dst[7] != 0x34 || dst[8] != 0x34 {
		t.Errorf("High byte 2 not grouped correctly: %02x", dst[6:9])
	}
	if dst[9] != 0x12 || dst[10] != 0x12 || dst[11] != 0x12 {
		t.Errorf("High byte 3 not grouped correctly: %02x", dst[9:12])
	}

	// Roundtrip verification
	output := make([]byte, len(input))
	n, err = codec.Decode(output, dst[:n], params)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if !bytes.Equal(output[:n], input) {
		t.Errorf("Roundtrip failed")
	}
}

// TestTranspose_Width8 verifies uint64 transposition (timestamps).
func TestTranspose_Width8(t *testing.T) {
	codec := NewTranspose()
	params := []byte{8} // width=8 (uint64)

	// Input: 4 timestamps (unix epoch, seconds)
	// All have same high bytes (in 2021 range)
	timestamps := []uint64{
		1609459200, // 2021-01-01 00:00:00
		1609459201, // 2021-01-01 00:00:01
		1609459202, // 2021-01-01 00:00:02
		1609459203, // 2021-01-01 00:00:03
	}

	input := make([]byte, 32)
	for i, ts := range timestamps {
		binary.LittleEndian.PutUint64(input[i*8:], ts)
	}

	dst := make([]byte, len(input))
	n, err := codec.Encode(dst, input, params)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	t.Logf("Transposed timestamp bytes:")
	for bytePos := 0; bytePos < 8; bytePos++ {
		stream := dst[bytePos*4 : bytePos*4+4]
		t.Logf("  Byte %d: %02x", bytePos, stream)
	}

	// Verify high bytes (bytes 4-7) are identical across all timestamps
	for bytePos := 4; bytePos < 8; bytePos++ {
		base := bytePos * 4
		first := dst[base]
		for i := 1; i < 4; i++ {
			if dst[base+i] != first {
				t.Errorf("Byte position %d not constant: %02x", bytePos, dst[base:base+4])
			}
		}
		t.Logf("  ✅ Byte %d constant: 0x%02x (RLE-friendly!)", bytePos, first)
	}

	// Roundtrip verification
	output := make([]byte, len(input))
	n, err = codec.Decode(output, dst[:n], params)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if !bytes.Equal(output[:n], input) {
		t.Errorf("Roundtrip failed")
	}
}

// TestTranspose_InvalidWidth verifies error handling for invalid widths.
func TestTranspose_InvalidWidth(t *testing.T) {
	codec := NewTranspose()

	tests := []struct {
		name   string
		params []byte
		input  []byte
	}{
		{"no params", []byte{}, []byte{1, 2, 3, 4}},
		{"zero width", []byte{0}, []byte{1, 2, 3, 4}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst := make([]byte, 1024)
			_, err := codec.Encode(dst, tt.input, tt.params)
			if err == nil {
				t.Error("Encode with invalid width should return error")
			}
			t.Logf("Correctly rejected: %v", err)
		})
	}
}

// TestTranspose_InvalidSize verifies error handling for misaligned input.
func TestTranspose_InvalidSize(t *testing.T) {
	codec := NewTranspose()
	params := []byte{4} // width=4

	// Input size not multiple of width
	input := []byte{1, 2, 3, 4, 5, 6, 7} // 7 bytes, not multiple of 4

	dst := make([]byte, 1024)
	_, err := codec.Encode(dst, input, params)
	if err == nil {
		t.Error("Encode with misaligned size should return error")
	}
	t.Logf("Correctly rejected misaligned input: %v", err)
}

// TestTranspose_BufferTooSmall verifies error handling for small buffers.
func TestTranspose_BufferTooSmall(t *testing.T) {
	codec := NewTranspose()
	params := []byte{4} // width=4

	input := []byte{1, 2, 3, 4, 5, 6, 7, 8}

	// Buffer too small
	tooSmall := make([]byte, 4)
	_, err := codec.Encode(tooSmall, input, params)
	if err != ErrBufferTooSmall {
		t.Errorf("Encode with small buffer: got error %v, want %v", err, ErrBufferTooSmall)
	}

	// Encode normally
	dst := make([]byte, len(input))
	n, err := codec.Encode(dst, input, params)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	// Try to decode with buffer too small
	_, err = codec.Decode(tooSmall, dst[:n], params)
	if err != ErrBufferTooSmall {
		t.Errorf("Decode with small buffer: got error %v, want %v", err, ErrBufferTooSmall)
	}
}

// TestTranspose_LargeArray verifies handling of large datasets.
func TestTranspose_LargeArray(t *testing.T) {
	codec := NewTranspose()
	params := []byte{8} // width=8 (uint64)

	// 10,000 uint64 values
	count := 10000
	input := make([]byte, count*8)
	for i := 0; i < count; i++ {
		binary.LittleEndian.PutUint64(input[i*8:], uint64(1000000+i))
	}

	dst := make([]byte, len(input))
	n, err := codec.Encode(dst, input, params)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	if n != len(input) {
		t.Errorf("Encode size %d != input size %d", n, len(input))
	}

	// Verify size preservation
	t.Logf("Large array: %d bytes → %d bytes (size preserved)", len(input), n)

	// Roundtrip verification
	output := make([]byte, len(input))
	n, err = codec.Decode(output, dst[:n], params)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if !bytes.Equal(output[:n], input) {
		t.Errorf("Roundtrip failed for large array")
	}
}

// TestTranspose_PatternsExposed verifies that transpose exposes byte patterns.
func TestTranspose_PatternsExposed(t *testing.T) {
	codec := NewTranspose()
	params := []byte{4} // width=4

	// Create data with predictable high bytes
	count := 100
	input := make([]byte, count*4)
	for i := 0; i < count; i++ {
		// Values like 0x12340000, 0x12340001, 0x12340002, ...
		// High bytes constant, low bytes sequential
		val := uint32(0x12340000 + i)
		binary.LittleEndian.PutUint32(input[i*4:], val)
	}

	dst := make([]byte, len(input))
	n, err := codec.Encode(dst, input, params)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	// After transpose, check byte streams
	// Byte 0: 00 01 02 03 ... (sequential - Delta friendly!)
	// Byte 1: 00 00 00 00 ... (constant - RLE heaven!)
	// Byte 2: 34 34 34 34 ... (constant - RLE heaven!)
	// Byte 3: 12 12 12 12 ... (constant - RLE heaven!)

	// Count unique values in each byte stream
	for bytePos := 0; bytePos < 4; bytePos++ {
		unique := make(map[byte]bool)
		base := bytePos * count
		for i := 0; i < count; i++ {
			unique[dst[base+i]] = true
		}
		t.Logf("Byte position %d: %d unique values", bytePos, len(unique))

		// High bytes should have only 1 unique value
		if bytePos > 0 && len(unique) > 1 {
			t.Errorf("Byte position %d should be constant, got %d unique values", bytePos, len(unique))
		}
	}

	// Roundtrip verification
	output := make([]byte, len(input))
	n, err = codec.Decode(output, dst[:n], params)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if !bytes.Equal(output[:n], input) {
		t.Errorf("Roundtrip failed")
	}
}

// BenchmarkTranspose_Encode benchmarks Transpose encoding.
func BenchmarkTranspose_Encode(t *testing.B) {
	codec := NewTranspose()
	params := []byte{8} // width=8 (uint64)

	// 4096 uint64 values
	input := make([]byte, 4096*8)
	for i := 0; i < 4096; i++ {
		binary.LittleEndian.PutUint64(input[i*8:], uint64(1000000+i))
	}

	dst := make([]byte, len(input))

	t.ResetTimer()
	t.SetBytes(int64(len(input)))

	for i := 0; i < t.N; i++ {
		_, err := codec.Encode(dst, input, params)
		if err != nil {
			t.Fatal(err)
		}
	}
}

// BenchmarkTranspose_Decode benchmarks Transpose decoding.
func BenchmarkTranspose_Decode(t *testing.B) {
	codec := NewTranspose()
	params := []byte{8} // width=8 (uint64)

	// Create and encode test data
	input := make([]byte, 4096*8)
	for i := 0; i < 4096; i++ {
		binary.LittleEndian.PutUint64(input[i*8:], uint64(1000000+i))
	}

	transposed := make([]byte, len(input))
	codec.Encode(transposed, input, params)

	output := make([]byte, len(input))

	t.ResetTimer()
	t.SetBytes(int64(len(input)))

	for i := 0; i < t.N; i++ {
		_, err := codec.Decode(output, transposed, params)
		if err != nil {
			t.Fatal(err)
		}
	}
}
