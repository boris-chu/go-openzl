// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package codec

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestConstant_Decode8(t *testing.T) {
	tests := []struct {
		name     string
		constant byte
		count    int
	}{
		{"zeros", 0, 10},
		{"ones", 1, 10},
		{"value 42", 42, 100},
		{"max value", 255, 50},
		{"single", 5, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codec := NewConstant(1)
			params := []byte{1}

			// Source: just the constant value
			src := []byte{tt.constant}

			// Destination: count repetitions
			dst := make([]byte, tt.count)

			n, err := codec.Decode(dst, src, params)
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}

			if n != tt.count {
				t.Errorf("Decode() returned %d bytes, want %d", n, tt.count)
			}

			// Verify all values are the constant
			for i, val := range dst {
				if val != tt.constant {
					t.Errorf("Element %d = %d, want %d", i, val, tt.constant)
					break
				}
			}
		})
	}
}

func TestConstant_Decode32(t *testing.T) {
	codec := NewConstant(4)
	params := []byte{4}

	tests := []struct {
		name     string
		constant uint32
		count    int
	}{
		{"zeros", 0, 10},
		{"value 1000", 1000, 100},
		{"max uint32", 0xFFFFFFFF, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Source: just the constant value (4 bytes)
			src := make([]byte, 4)
			binary.LittleEndian.PutUint32(src, tt.constant)

			// Destination: count repetitions
			dst := make([]byte, tt.count*4)

			n, err := codec.Decode(dst, src, params)
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}

			if n != tt.count*4 {
				t.Errorf("Decode() returned %d bytes, want %d", n, tt.count*4)
			}

			// Verify all values are the constant
			for i := 0; i < tt.count; i++ {
				val := binary.LittleEndian.Uint32(dst[i*4:])
				if val != tt.constant {
					t.Errorf("Element %d = %d, want %d", i, val, tt.constant)
					break
				}
			}
		})
	}
}

func TestConstant_Encode8(t *testing.T) {
	codec := NewConstant(1)
	params := []byte{1}

	tests := []struct {
		name     string
		constant byte
		count    int
	}{
		{"zeros", 0, 10},
		{"ones", 1, 10},
		{"value 42", 42, 100},
		{"single", 5, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create input with constant value repeated
			src := bytes.Repeat([]byte{tt.constant}, tt.count)

			// Destination: should be just one byte
			dst := make([]byte, 1)

			n, err := codec.Encode(dst, src, params)
			if err != nil {
				t.Fatalf("Encode() error = %v", err)
			}

			if n != 1 {
				t.Errorf("Encode() returned %d bytes, want 1", n)
			}

			if dst[0] != tt.constant {
				t.Errorf("Encode() = %d, want %d", dst[0], tt.constant)
			}
		})
	}
}

func TestConstant_Encode32(t *testing.T) {
	codec := NewConstant(4)
	params := []byte{4}

	constant := uint32(12345)
	count := 100

	// Create input with constant value repeated
	src := make([]byte, count*4)
	for i := 0; i < count; i++ {
		binary.LittleEndian.PutUint32(src[i*4:], constant)
	}

	// Destination: should be just 4 bytes
	dst := make([]byte, 4)

	n, err := codec.Encode(dst, src, params)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	if n != 4 {
		t.Errorf("Encode() returned %d bytes, want 4", n)
	}

	result := binary.LittleEndian.Uint32(dst)
	if result != constant {
		t.Errorf("Encode() = %d, want %d", result, constant)
	}
}

func TestConstant_EncodeNonConstant(t *testing.T) {
	codec := NewConstant(4)
	params := []byte{4}

	// Create input with varying values (NOT constant)
	src := make([]byte, 20)
	binary.LittleEndian.PutUint32(src[0:], 1)
	binary.LittleEndian.PutUint32(src[4:], 1)
	binary.LittleEndian.PutUint32(src[8:], 2) // Different!
	binary.LittleEndian.PutUint32(src[12:], 1)
	binary.LittleEndian.PutUint32(src[16:], 1)

	dst := make([]byte, 4)

	_, err := codec.Encode(dst, src, params)
	if err == nil {
		t.Error("Encode() expected error for non-constant data, got nil")
	}

	if err != nil && !bytes.Contains([]byte(err.Error()), []byte("not all values are identical")) {
		t.Errorf("Encode() error = %v, want error about non-identical values", err)
	}
}

func TestConstant_Roundtrip(t *testing.T) {
	tests := []struct {
		name        string
		elementSize int
		constant    []byte
		count       int
	}{
		{
			name:        "1-byte",
			elementSize: 1,
			constant:    []byte{42},
			count:       100,
		},
		{
			name:        "2-byte",
			elementSize: 2,
			constant:    uint16ToBytes([]uint16{12345}),
			count:       100,
		},
		{
			name:        "4-byte",
			elementSize: 4,
			constant:    uint32ToBytes([]uint32{123456789}),
			count:       100,
		},
		{
			name:        "8-byte",
			elementSize: 8,
			constant:    uint64ToBytes([]uint64{123456789012345}),
			count:       100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codec := NewConstant(tt.elementSize)
			params := []byte{byte(tt.elementSize)}

			// Create original data (constant value repeated)
			original := bytes.Repeat(tt.constant, tt.count)

			// Encode
			encoded := make([]byte, tt.elementSize)
			n, err := codec.Encode(encoded, original, params)
			if err != nil {
				t.Fatalf("Encode() error = %v", err)
			}

			if n != tt.elementSize {
				t.Errorf("Encode() returned %d bytes, want %d", n, tt.elementSize)
			}

			// Verify encoded is just the constant
			if !bytes.Equal(encoded[:n], tt.constant) {
				t.Errorf("Encode() = %v, want %v", encoded[:n], tt.constant)
			}

			// Decode
			decoded := make([]byte, len(original))
			n2, err := codec.Decode(decoded, encoded[:n], params)
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}

			if n2 != len(original) {
				t.Errorf("Decode() returned %d bytes, want %d", n2, len(original))
			}

			// Verify roundtrip
			if !bytes.Equal(decoded[:n2], original) {
				t.Errorf("Roundtrip failed")
			}

			// Log compression ratio
			ratio := float64(len(original)) / float64(n)
			t.Logf("Compression: %d bytes → %d bytes (%.1f:1 ratio)",
				len(original), n, ratio)
		})
	}
}

func TestConstant_EmptyData(t *testing.T) {
	codec := NewConstant(4)
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

	// Encode empty data
	n, err = codec.Encode(dst, src, params)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	if n != 0 {
		t.Errorf("Encode() returned %d bytes, want 0", n)
	}
}

func TestConstant_SingleElement(t *testing.T) {
	codec := NewConstant(4)
	params := []byte{4}

	value := uint32(42)
	src := make([]byte, 4)
	binary.LittleEndian.PutUint32(src, value)

	// Encode single element (always constant)
	dst := make([]byte, 4)
	n, err := codec.Encode(dst, src, params)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	if n != 4 {
		t.Errorf("Encode() returned %d bytes, want 4", n)
	}

	if !bytes.Equal(dst[:n], src) {
		t.Errorf("Single element encode failed")
	}
}

func TestConstant_BufferTooSmall(t *testing.T) {
	codec := NewConstant(4)
	params := []byte{4}

	src := make([]byte, 100*4) // 100 elements
	for i := 0; i < 100; i++ {
		binary.LittleEndian.PutUint32(src[i*4:], 42)
	}

	// Dst too small for encoding
	dst := make([]byte, 2) // Need 4 bytes

	_, err := codec.Encode(dst, src, params)
	if err != ErrBufferTooSmall {
		t.Errorf("Encode() error = %v, want ErrBufferTooSmall", err)
	}
}

func TestConstant_InvalidElementSize(t *testing.T) {
	codec := NewConstant(4)

	tests := []struct {
		name   string
		params []byte
	}{
		{"zero", []byte{0}},
		{"three", []byte{3}},
		{"five", []byte{5}},
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

func TestConstant_UnalignedInput(t *testing.T) {
	codec := NewConstant(4)
	params := []byte{4}

	// Source not aligned (7 bytes, not divisible by 4)
	src := make([]byte, 7)
	dst := make([]byte, 8)

	_, err := codec.Encode(dst, src, params)
	if err == nil {
		t.Error("Encode() expected error for unaligned input, got nil")
	}

	// Destination not aligned
	src = make([]byte, 4)
	dst = make([]byte, 7)

	_, err = codec.Decode(dst, src, params)
	if err == nil {
		t.Error("Decode() expected error for unaligned output, got nil")
	}
}

func TestConstant_WrongSourceSize(t *testing.T) {
	codec := NewConstant(4)
	params := []byte{4}

	// Source must be exactly 4 bytes for decoding
	tests := []struct {
		name string
		size int
	}{
		{"too small", 2},
		{"too large", 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := make([]byte, tt.size)
			dst := make([]byte, 16)

			_, err := codec.Decode(dst, src, params)
			if err == nil {
				t.Error("Decode() expected error for wrong source size, got nil")
			}
		})
	}
}

func TestConstant_Metadata(t *testing.T) {
	codec := NewConstant(4)

	if codec.ID() != IDConstant {
		t.Errorf("ID() = %v, want %v", codec.ID(), IDConstant)
	}

	if codec.Name() != nameConstant {
		t.Errorf("Name() = %q, want %q", codec.Name(), nameConstant)
	}
}

func TestConstant_ExtremeCompression(t *testing.T) {
	// Test extreme case: 1 million identical values
	codec := NewConstant(4)
	params := []byte{4}

	constant := uint32(42)
	count := 1000000

	// Create 1M identical values (4 MB)
	src := make([]byte, count*4)
	for i := 0; i < count; i++ {
		binary.LittleEndian.PutUint32(src[i*4:], constant)
	}

	// Encode to just 4 bytes
	dst := make([]byte, 4)
	n, err := codec.Encode(dst, src, params)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	if n != 4 {
		t.Errorf("Encode() returned %d bytes, want 4", n)
	}

	ratio := float64(len(src)) / float64(n)
	t.Logf("Extreme compression: %d bytes → %d bytes (%.0f:1 ratio)",
		len(src), n, ratio)

	if ratio < 999999 {
		t.Errorf("Compression ratio %.0f:1 is too low, want at least 999999:1", ratio)
	}

	// Decode back
	decoded := make([]byte, len(src))
	_, err = codec.Decode(decoded, dst[:n], params)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	// Verify a sample (checking all 1M would be slow)
	samples := []int{0, count / 4, count / 2, count * 3 / 4, count - 1}
	for _, idx := range samples {
		val := binary.LittleEndian.Uint32(decoded[idx*4:])
		if val != constant {
			t.Errorf("Sample %d = %d, want %d", idx, val, constant)
		}
	}

	t.Logf("✓ Successfully decoded 1M identical values")
}

// Helper functions

func uint16ToBytes(data []uint16) []byte {
	buf := make([]byte, len(data)*2)
	for i, v := range data {
		binary.LittleEndian.PutUint16(buf[i*2:], v)
	}
	return buf
}

func uint32ToBytes(data []uint32) []byte {
	buf := make([]byte, len(data)*4)
	for i, v := range data {
		binary.LittleEndian.PutUint32(buf[i*4:], v)
	}
	return buf
}

func uint64ToBytes(data []uint64) []byte {
	buf := make([]byte, len(data)*8)
	for i, v := range data {
		binary.LittleEndian.PutUint64(buf[i*8:], v)
	}
	return buf
}

func BenchmarkConstant_Decode(b *testing.B) {
	codec := NewConstant(4)
	params := []byte{4}

	// Source: single constant value
	constant := uint32(42)
	src := make([]byte, 4)
	binary.LittleEndian.PutUint32(src, constant)

	// Destination: 1MB of repeated values
	const size = 1024 * 1024 / 4
	dst := make([]byte, size*4)

	b.SetBytes(int64(len(dst)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = codec.Decode(dst, src, params)
	}
}

func BenchmarkConstant_Encode(b *testing.B) {
	codec := NewConstant(4)
	params := []byte{4}

	// Source: 1MB of identical values
	const size = 1024 * 1024 / 4
	src := make([]byte, size*4)
	constant := uint32(42)
	for i := 0; i < size; i++ {
		binary.LittleEndian.PutUint32(src[i*4:], constant)
	}

	dst := make([]byte, 4)

	b.SetBytes(int64(len(src)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = codec.Encode(dst, src, params)
	}
}
