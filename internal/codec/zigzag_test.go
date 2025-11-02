// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package codec

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestZigZag_Decode8(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte // ZigZag encoded
		expected []int8 // Original signed values
	}{
		{
			name:     "basic mapping",
			input:    []byte{0, 1, 2, 3, 4, 5},
			expected: []int8{0, -1, 1, -2, 2, -3},
		},
		{
			name:     "all positive",
			input:    []byte{0, 2, 4, 6, 8},
			expected: []int8{0, 1, 2, 3, 4},
		},
		{
			name:     "all negative",
			input:    []byte{1, 3, 5, 7, 9},
			expected: []int8{-1, -2, -3, -4, -5},
		},
		{
			name:     "extreme values",
			input:    []byte{0, 255, 254, 1},
			expected: []int8{0, -128, 127, -1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codec := NewZigZag(1)
			dst := make([]byte, len(tt.input))
			params := []byte{1} // 1-byte elements

			n, err := codec.Decode(dst, tt.input, params)
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}

			if n != len(tt.input) {
				t.Errorf("Decode() returned %d bytes, want %d", n, len(tt.input))
			}

			// Convert to int8 for comparison
			result := make([]int8, len(dst))
			for i, b := range dst {
				result[i] = int8(b)
			}

			if !equalInt8(result, tt.expected) {
				t.Errorf("Decode() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestZigZag_Decode16(t *testing.T) {
	codec := NewZigZag(2)
	params := []byte{2}

	// Test values: 0, -1, 1, -2, 2
	// ZigZag:      0,  1, 2,  3, 4
	input := make([]byte, 10)
	binary.LittleEndian.PutUint16(input[0:], 0)
	binary.LittleEndian.PutUint16(input[2:], 1)
	binary.LittleEndian.PutUint16(input[4:], 2)
	binary.LittleEndian.PutUint16(input[6:], 3)
	binary.LittleEndian.PutUint16(input[8:], 4)

	dst := make([]byte, len(input))
	n, err := codec.Decode(dst, input, params)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if n != len(input) {
		t.Errorf("Decode() returned %d bytes, want %d", n, len(input))
	}

	expected := []int16{0, -1, 1, -2, 2}
	for i, exp := range expected {
		offset := i * 2
		got := int16(binary.LittleEndian.Uint16(dst[offset:]))
		if got != exp {
			t.Errorf("Element %d: got %d, want %d", i, got, exp)
		}
	}
}

func TestZigZag_Decode32(t *testing.T) {
	codec := NewZigZag(4)
	params := []byte{4}

	// Test values: 0, -1, 1, -2, 2, 1000, -1000
	// ZigZag:      0,  1, 2,  3, 4, 2000,  1999
	input := make([]byte, 28)
	binary.LittleEndian.PutUint32(input[0:], 0)
	binary.LittleEndian.PutUint32(input[4:], 1)
	binary.LittleEndian.PutUint32(input[8:], 2)
	binary.LittleEndian.PutUint32(input[12:], 3)
	binary.LittleEndian.PutUint32(input[16:], 4)
	binary.LittleEndian.PutUint32(input[20:], 2000)
	binary.LittleEndian.PutUint32(input[24:], 1999)

	dst := make([]byte, len(input))
	n, err := codec.Decode(dst, input, params)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if n != len(input) {
		t.Errorf("Decode() returned %d bytes, want %d", n, len(input))
	}

	expected := []int32{0, -1, 1, -2, 2, 1000, -1000}
	for i, exp := range expected {
		offset := i * 4
		got := int32(binary.LittleEndian.Uint32(dst[offset:]))
		if got != exp {
			t.Errorf("Element %d: got %d, want %d", i, got, exp)
		}
	}
}

func TestZigZag_Decode64(t *testing.T) {
	codec := NewZigZag(8)
	params := []byte{8}

	// Test values: 0, -1, 1, -2, 2
	// ZigZag:      0,  1, 2,  3, 4
	input := make([]byte, 40)
	binary.LittleEndian.PutUint64(input[0:], 0)
	binary.LittleEndian.PutUint64(input[8:], 1)
	binary.LittleEndian.PutUint64(input[16:], 2)
	binary.LittleEndian.PutUint64(input[24:], 3)
	binary.LittleEndian.PutUint64(input[32:], 4)

	dst := make([]byte, len(input))
	n, err := codec.Decode(dst, input, params)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if n != len(input) {
		t.Errorf("Decode() returned %d bytes, want %d", n, len(input))
	}

	expected := []int64{0, -1, 1, -2, 2}
	for i, exp := range expected {
		offset := i * 8
		got := int64(binary.LittleEndian.Uint64(dst[offset:]))
		if got != exp {
			t.Errorf("Element %d: got %d, want %d", i, got, exp)
		}
	}
}

func TestZigZag_Encode8(t *testing.T) {
	codec := NewZigZag(1)
	params := []byte{1}

	// Input: 0, -1, 1, -2, 2, -3
	input := []byte{0, 255, 1, 254, 2, 253} // byte representation of int8
	dst := make([]byte, len(input))

	n, err := codec.Encode(dst, input, params)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	if n != len(input) {
		t.Errorf("Encode() returned %d bytes, want %d", n, len(input))
	}

	expected := []byte{0, 1, 2, 3, 4, 5}
	if !bytes.Equal(dst, expected) {
		t.Errorf("Encode() = %v, want %v", dst, expected)
	}
}

func TestZigZag_Roundtrip(t *testing.T) {
	tests := []struct {
		name        string
		elementSize int
		data        []byte
	}{
		{
			name:        "1-byte elements",
			elementSize: 1,
			data:        []byte{0, 255, 1, 254, 2, 253}, // 0, -1, 1, -2, 2, -3 as int8
		},
		{
			name:        "2-byte elements",
			elementSize: 2,
			data:        int16ToBytes([]int16{0, -1, 1, -100, 100, -1000, 1000}),
		},
		{
			name:        "4-byte elements",
			elementSize: 4,
			data:        int32ToBytes([]int32{0, -1, 1, -1000, 1000, -100000, 100000}),
		},
		{
			name:        "8-byte elements",
			elementSize: 8,
			data:        int64ToBytes([]int64{0, -1, 1, -1000000, 1000000}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codec := NewZigZag(tt.elementSize)
			params := []byte{byte(tt.elementSize)}

			// Encode
			encoded := make([]byte, len(tt.data))
			n, err := codec.Encode(encoded, tt.data, params)
			if err != nil {
				t.Fatalf("Encode() error = %v", err)
			}

			// Decode
			decoded := make([]byte, len(tt.data))
			n2, err := codec.Decode(decoded, encoded[:n], params)
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}

			if n2 != len(tt.data) {
				t.Errorf("Decode() returned %d bytes, want %d", n2, len(tt.data))
			}

			if !bytes.Equal(decoded[:n2], tt.data) {
				t.Errorf("Roundtrip failed: got %v, want %v", decoded[:n2], tt.data)
			}
		})
	}
}

func TestZigZag_EmptyData(t *testing.T) {
	codec := NewZigZag(4)
	params := []byte{4}

	dst := make([]byte, 0)
	src := make([]byte, 0)

	n, err := codec.Decode(dst, src, params)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if n != 0 {
		t.Errorf("Decode() returned %d bytes, want 0", n)
	}
}

func TestZigZag_BufferTooSmall(t *testing.T) {
	codec := NewZigZag(4)
	params := []byte{4}

	src := make([]byte, 8)
	dst := make([]byte, 4) // Too small

	_, err := codec.Decode(dst, src, params)
	if err != ErrBufferTooSmall {
		t.Errorf("Decode() error = %v, want ErrBufferTooSmall", err)
	}
}

func TestZigZag_InvalidElementSize(t *testing.T) {
	codec := NewZigZag(4)

	tests := []struct {
		name   string
		params []byte
	}{
		{"zero", []byte{0}},
		{"three", []byte{3}},
		{"five", []byte{5}},
		{"seven", []byte{7}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := make([]byte, 8)
			dst := make([]byte, 8)

			_, err := codec.Decode(dst, src, tt.params)
			if err == nil {
				t.Error("Decode() expected error for invalid element size, got nil")
			}
		})
	}
}

func TestZigZag_UnalignedInput(t *testing.T) {
	codec := NewZigZag(4)
	params := []byte{4}

	// 7 bytes, not divisible by 4
	src := make([]byte, 7)
	dst := make([]byte, 8)

	_, err := codec.Decode(dst, src, params)
	if err == nil {
		t.Error("Decode() expected error for unaligned input, got nil")
	}
}

func TestZigZag_Metadata(t *testing.T) {
	codec := NewZigZag(4)

	if codec.ID() != IDZigZag {
		t.Errorf("ID() = %v, want %v", codec.ID(), IDZigZag)
	}

	if codec.Name() != nameZigZag {
		t.Errorf("Name() = %q, want %q", codec.Name(), nameZigZag)
	}
}

func TestZigZag_DeltaPipeline(t *testing.T) {
	// Test realistic pipeline: Delta → ZigZag
	// This is how they work together in real compression

	// Original timestamps (increasing)
	timestamps := []int32{1000, 1005, 1003, 1008, 1004, 1010}

	// Step 1: Delta encode
	deltaCodec := NewDelta(4)
	timestampBytes := int32ToBytes(timestamps)
	deltaEncoded := make([]byte, len(timestampBytes))
	params := []byte{4}

	n, err := deltaCodec.Encode(deltaEncoded, timestampBytes, params)
	if err != nil {
		t.Fatalf("Delta encode error: %v", err)
	}

	// After delta: [1000, 5, -2, 5, -4, 6]
	// Note the negative deltas!

	// Step 2: ZigZag encode
	zigzagCodec := NewZigZag(4)
	zigzagEncoded := make([]byte, n)

	n2, err := zigzagCodec.Encode(zigzagEncoded, deltaEncoded[:n], params)
	if err != nil {
		t.Fatalf("ZigZag encode error: %v", err)
	}

	// After zigzag: [2000, 10, 3, 10, 7, 12]
	// All positive, small numbers! Perfect for varint compression!

	// Step 3: Decode pipeline (reverse order)
	// ZigZag decode
	zigzagDecoded := make([]byte, n2)
	n3, err := zigzagCodec.Decode(zigzagDecoded, zigzagEncoded[:n2], params)
	if err != nil {
		t.Fatalf("ZigZag decode error: %v", err)
	}

	// Delta decode
	deltaDecoded := make([]byte, n3)
	n4, err := deltaCodec.Decode(deltaDecoded, zigzagDecoded[:n3], params)
	if err != nil {
		t.Fatalf("Delta decode error: %v", err)
	}

	// Should match original
	if !bytes.Equal(deltaDecoded[:n4], timestampBytes) {
		originalVals := bytesToInt32(timestampBytes)
		decodedVals := bytesToInt32(deltaDecoded[:n4])
		t.Errorf("Pipeline roundtrip failed:\nOriginal: %v\nDecoded:  %v", originalVals, decodedVals)
	}

	t.Logf("✓ Delta + ZigZag pipeline works!")
	t.Logf("  Original: %v", timestamps)
	t.Logf("  After delta: %v", bytesToInt32(deltaEncoded[:n]))
	t.Logf("  After zigzag: %v", bytesToUint32(zigzagEncoded[:n2]))
	t.Logf("  Size: %d bytes → %d bytes (delta) → %d bytes (zigzag)",
		len(timestampBytes), n, n2)
}

// Helper functions

func equalInt8(a, b []int8) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func int16ToBytes(data []int16) []byte {
	buf := make([]byte, len(data)*2)
	for i, v := range data {
		binary.LittleEndian.PutUint16(buf[i*2:], uint16(v))
	}
	return buf
}

func int32ToBytes(data []int32) []byte {
	buf := make([]byte, len(data)*4)
	for i, v := range data {
		binary.LittleEndian.PutUint32(buf[i*4:], uint32(v))
	}
	return buf
}

func int64ToBytes(data []int64) []byte {
	buf := make([]byte, len(data)*8)
	for i, v := range data {
		binary.LittleEndian.PutUint64(buf[i*8:], uint64(v))
	}
	return buf
}

func bytesToInt32(buf []byte) []int32 {
	result := make([]int32, len(buf)/4)
	for i := range result {
		result[i] = int32(binary.LittleEndian.Uint32(buf[i*4:]))
	}
	return result
}

func bytesToUint32(buf []byte) []uint32 {
	result := make([]uint32, len(buf)/4)
	for i := range result {
		result[i] = binary.LittleEndian.Uint32(buf[i*4:])
	}
	return result
}

func BenchmarkZigZag_Decode32(b *testing.B) {
	codec := NewZigZag(4)
	params := []byte{4}

	// Create test data (1MB of int32s)
	const size = 1024 * 1024 / 4
	src := make([]byte, size*4)
	for i := 0; i < size; i++ {
		// Mix of positive and negative for realistic data
		val := int32(i%1000) - 500
		zigzag := uint32((val << 1) ^ (val >> 31))
		binary.LittleEndian.PutUint32(src[i*4:], zigzag)
	}

	dst := make([]byte, len(src))

	b.SetBytes(int64(len(src)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = codec.Decode(dst, src, params)
	}
}

func BenchmarkZigZag_Encode32(b *testing.B) {
	codec := NewZigZag(4)
	params := []byte{4}

	// Create test data (1MB of int32s)
	const size = 1024 * 1024 / 4
	src := make([]byte, size*4)
	for i := 0; i < size; i++ {
		val := int32(i%1000) - 500
		binary.LittleEndian.PutUint32(src[i*4:], uint32(val))
	}

	dst := make([]byte, len(src))

	b.SetBytes(int64(len(src)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = codec.Encode(dst, src, params)
	}
}
