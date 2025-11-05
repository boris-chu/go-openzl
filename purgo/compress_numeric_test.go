// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package purgo

import (
	"bytes"
	"encoding/binary"
	"testing"
	"unsafe"
)

// TestCompressSmart_Int64Timestamps tests Transpose→RLE pipeline on timestamps
func TestCompressSmart_Int64Timestamps(t *testing.T) {
	// Create realistic int64 timestamps (Unix milliseconds)
	// 2025-01-01: ~1,735,000,000,000 ms
	data := make([]byte, 8*100)
	baseTimestamp := uint64(1735000000000)

	for i := 0; i < 100; i++ {
		// Timestamps incrementing by 1 second (1000 ms)
		timestamp := baseTimestamp + uint64(i*1000)
		binary.LittleEndian.PutUint64(data[i*8:], timestamp)
	}

	// Test compression
	compressed, err := CompressSmart(data)
	if err != nil {
		t.Fatalf("CompressSmart failed: %v", err)
	}

	// Should achieve excellent compression (timestamps have constant high bytes)
	// Expected: 10-100× compression ratio
	ratio := float64(len(data)) / float64(len(compressed))
	if ratio < 5.0 {
		t.Errorf("CompressSmart achieved only %.2f× compression on timestamps (expected >5×)", ratio)
	}

	t.Logf("Int64 timestamps: %d bytes → %d bytes (%.2f× compression)",
		len(data), len(compressed), ratio)

	// Verify roundtrip
	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("Decompression failed: %v", err)
	}

	if !bytes.Equal(decompressed, data) {
		t.Errorf("Roundtrip mismatch")
	}
}

// TestCompressSmart_Int32IDs tests Transpose→RLE on sequential 32-bit IDs
func TestCompressSmart_Int32IDs(t *testing.T) {
	// Create array of sequential int32 IDs
	data := make([]byte, 4*200)

	for i := 0; i < 200; i++ {
		id := uint32(1000000 + i) // IDs from 1,000,000 to 1,000,199
		binary.LittleEndian.PutUint32(data[i*4:], id)
	}

	compressed, err := CompressSmart(data)
	if err != nil {
		t.Fatalf("CompressSmart failed: %v", err)
	}

	ratio := float64(len(data)) / float64(len(compressed))
	if ratio < 5.0 {
		t.Errorf("CompressSmart achieved only %.2f× compression on IDs (expected >5×)", ratio)
	}

	t.Logf("Int32 IDs: %d bytes → %d bytes (%.2f× compression)",
		len(data), len(compressed), ratio)

	// Verify roundtrip
	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("Decompression failed: %v", err)
	}

	if !bytes.Equal(decompressed, data) {
		t.Errorf("Roundtrip mismatch")
	}
}

// TestCompressSmart_Int16SensorData tests Transpose→RLE on 16-bit sensor readings
func TestCompressSmart_Int16SensorData(t *testing.T) {
	// Create array of int16 sensor readings (simulating temperature, audio, etc.)
	data := make([]byte, 2*500)

	baseValue := uint16(30000) // ~30,000 range (typical for sensors)
	for i := 0; i < 500; i++ {
		// Slowly varying sensor readings
		value := baseValue + uint16(i/10)
		binary.LittleEndian.PutUint16(data[i*2:], value)
	}

	compressed, err := CompressSmart(data)
	if err != nil {
		t.Fatalf("CompressSmart failed: %v", err)
	}

	ratio := float64(len(data)) / float64(len(compressed))
	if ratio < 3.0 {
		t.Errorf("CompressSmart achieved only %.2f× compression on sensor data (expected >3×)", ratio)
	}

	t.Logf("Int16 sensor data: %d bytes → %d bytes (%.2f× compression)",
		len(data), len(compressed), ratio)

	// Verify roundtrip
	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("Decompression failed: %v", err)
	}

	if !bytes.Equal(decompressed, data) {
		t.Errorf("Roundtrip mismatch")
	}
}

// TestCompressSmart_Float64Prices tests numeric detection on float64 data
func TestCompressSmart_Float64Prices(t *testing.T) {
	// Create array of float64 prices (similar magnitudes)
	data := make([]byte, 8*100)
	prices := []float64{
		100.50, 101.25, 99.75, 102.00, 100.00,
		103.50, 98.25, 104.75, 100.50, 105.00,
	}

	for i := 0; i < 100; i++ {
		price := prices[i%len(prices)]
		bits := float64ToBits(price)
		binary.LittleEndian.PutUint64(data[i*8:], bits)
	}

	compressed, err := CompressSmart(data)
	if err != nil {
		t.Fatalf("CompressSmart failed: %v", err)
	}

	ratio := float64(len(data)) / float64(len(compressed))

	t.Logf("Float64 prices: %d bytes → %d bytes (%.2f× compression)",
		len(data), len(compressed), ratio)

	// Verify roundtrip
	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("Decompression failed: %v", err)
	}

	if !bytes.Equal(decompressed, data) {
		t.Errorf("Roundtrip mismatch")
	}
}

// TestCompressSmart_MixedData verifies fallback on non-numeric data
func TestCompressSmart_MixedData(t *testing.T) {
	// Mix of numeric and text data (should NOT trigger transpose)
	data := []byte("timestamp:1735000000000,value:100.5,name:sensor_01,timestamp:1735000001000,value:101.2")

	compressed, err := CompressSmart(data)
	if err != nil {
		t.Fatalf("CompressSmart failed: %v", err)
	}

	ratio := float64(len(data)) / float64(len(compressed))

	t.Logf("Mixed data: %d bytes → %d bytes (%.2f× compression)",
		len(data), len(compressed), ratio)

	// Should use LZ77 (not transpose) due to text patterns
	// Verify it doesn't crash and achieves some compression
	if ratio < 1.5 {
		t.Logf("Warning: Low compression ratio %.2f× on mixed data (expected LZ77 to work better)", ratio)
	}

	// Verify roundtrip
	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("Decompression failed: %v", err)
	}

	if !bytes.Equal(decompressed, data) {
		t.Errorf("Roundtrip mismatch")
	}
}

// TestDetectNumericWidth tests numeric width detection
func TestDetectNumericWidth(t *testing.T) {
	t.Run("Int64Array", func(t *testing.T) {
		data := make([]byte, 8*50)
		base := uint64(1735000000000)
		for i := 0; i < 50; i++ {
			binary.LittleEndian.PutUint64(data[i*8:], base+uint64(i*1000))
		}

		width := detectNumericWidth(data)
		if width != 8 {
			t.Errorf("Expected width 8 for int64, got %d", width)
		}
	})

	t.Run("Int32Array", func(t *testing.T) {
		data := make([]byte, 4*50)
		for i := 0; i < 50; i++ {
			binary.LittleEndian.PutUint32(data[i*4:], uint32(1000000+i))
		}

		width := detectNumericWidth(data)
		if width != 4 {
			t.Errorf("Expected width 4 for int32, got %d", width)
		}
	})

	t.Run("Int16Array", func(t *testing.T) {
		data := make([]byte, 2*50)
		for i := 0; i < 50; i++ {
			binary.LittleEndian.PutUint16(data[i*2:], uint16(30000+i))
		}

		width := detectNumericWidth(data)
		if width != 2 {
			t.Errorf("Expected width 2 for int16, got %d", width)
		}
	})

	t.Run("TextData", func(t *testing.T) {
		data := []byte("Hello, World! This is plain text data.")

		width := detectNumericWidth(data)
		if width != 0 {
			t.Errorf("Expected width 0 for text, got %d", width)
		}
	})

	t.Run("TooSmall", func(t *testing.T) {
		// Less than 4 elements
		data := []byte{1, 2, 3, 4, 5, 6, 7}

		width := detectNumericWidth(data)
		if width != 0 {
			t.Errorf("Expected width 0 for small data, got %d", width)
		}
	})

	t.Run("RandomData", func(t *testing.T) {
		// Random bytes (not numeric pattern)
		data := []byte{
			0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0,
			0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88,
			0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x99,
			0x21, 0x43, 0x65, 0x87, 0xA9, 0xCB, 0xED, 0x0F,
		}

		width := detectNumericWidth(data)
		if width != 0 {
			t.Errorf("Expected width 0 for random data, got %d", width)
		}
	})
}

// TestCalculateBytePositionEntropy tests entropy calculation
func TestCalculateBytePositionEntropy(t *testing.T) {
	t.Run("AllSame", func(t *testing.T) {
		// All zeros in byte position 0
		data := make([]byte, 8*10)
		// First byte of each element is 0
		for i := 0; i < 10; i++ {
			data[i*8] = 0
			// Fill other bytes with varying data
			for j := 1; j < 8; j++ {
				data[i*8+j] = byte(i + j)
			}
		}

		entropy := calculateBytePositionEntropy(data, 8, 0, 10)
		if entropy != 0.0 {
			t.Errorf("Expected entropy 0.0 for all-same byte, got %.2f", entropy)
		}
	})

	t.Run("HighVariation", func(t *testing.T) {
		// Varying data in byte position 7 (last byte)
		data := make([]byte, 8*10)
		for i := 0; i < 10; i++ {
			// First bytes constant
			data[i*8] = 0x42
			// Last byte varies
			data[i*8+7] = byte(i)
		}

		entropy := calculateBytePositionEntropy(data, 8, 7, 10)
		if entropy < 2.0 {
			t.Errorf("Expected entropy >2.0 for varying byte, got %.2f", entropy)
		}
	})
}

// float64ToBits converts float64 to uint64 bits (helper for testing)
func float64ToBits(f float64) uint64 {
	return binary.LittleEndian.Uint64((*[8]byte)(unsafe.Pointer(&f))[:])
}
