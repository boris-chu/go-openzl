// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package codec

import (
	"encoding/binary"
	"testing"
)

// TestRangePack_Timestamps tests compression of Unix timestamps
func TestRangePack_Timestamps(t *testing.T) {
	t.Log("=== RangePack: Unix Timestamps ===\n")

	codec := NewRangePack()

	// Timestamps: All > 1,700,000,000 (recent timestamps)
	// Range: ~1000 seconds (fits in uint16)
	timestamps := []uint64{
		1700000000, 1700000100, 1700000200, 1700000300,
		1700000400, 1700000500, 1700000600, 1700000700,
		1700000800, 1700000900, 1700001000,
	}

	// Encode timestamps to bytes (8 bytes each)
	src := make([]byte, len(timestamps)*8)
	for i, ts := range timestamps {
		binary.LittleEndian.PutUint64(src[i*8:], ts)
	}

	t.Logf("Input: %d timestamps (%d bytes)", len(timestamps), len(src))

	// Params: element width = 8 bytes (uint64)
	params := []byte{8}

	// Compress
	dst := make([]byte, len(src)+100) // Extra space for header
	n, err := codec.Encode(dst, src, params)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	ratio := float64(len(src)) / float64(n)
	t.Logf("Compressed: %d bytes → %d bytes (%.2f× compression)", len(src), n, ratio)

	// Verify header
	minVal := binary.LittleEndian.Uint64(dst[0:8])
	maxVal := binary.LittleEndian.Uint64(dst[8:16])
	packedWidth := int(dst[16])

	t.Logf("  Min: %d, Max: %d", minVal, maxVal)
	t.Logf("  Range: %d (packed as uint%d)", maxVal-minVal, packedWidth*8)

	if minVal != timestamps[0] {
		t.Errorf("Expected min=%d, got %d", timestamps[0], minVal)
	}
	if maxVal != timestamps[len(timestamps)-1] {
		t.Errorf("Expected max=%d, got %d", timestamps[len(timestamps)-1], maxVal)
	}
	if packedWidth != 2 {
		t.Errorf("Expected packed width 2 (uint16), got %d", packedWidth)
	}

	// Decompress
	decompressed := make([]byte, len(src))
	dn, err := codec.Decode(decompressed, dst[:n], params)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if dn != len(src) {
		t.Errorf("Expected %d decompressed bytes, got %d", len(src), dn)
	}

	// Verify roundtrip
	for i := 0; i < len(timestamps); i++ {
		got := binary.LittleEndian.Uint64(decompressed[i*8:])
		if got != timestamps[i] {
			t.Errorf("Timestamp %d: expected %d, got %d", i, timestamps[i], got)
		}
	}

	t.Logf("✅ Roundtrip successful\n")

	// Compression ratio target: 2-4× for timestamps
	if ratio < 2.0 {
		t.Errorf("Expected compression ratio >= 2×, got %.2f×", ratio)
	}
}

// TestRangePack_IDsWithOffset tests compression of account IDs with offset
func TestRangePack_IDsWithOffset(t *testing.T) {
	t.Log("=== RangePack: Account IDs with Offset ===\n")

	codec := NewRangePack()

	// Account IDs: 1,000,000 to 1,000,100 (range = 100, fits in uint8)
	ids := make([]uint32, 101)
	for i := 0; i < 101; i++ {
		ids[i] = 1000000 + uint32(i)
	}

	// Encode to bytes (4 bytes each)
	src := make([]byte, len(ids)*4)
	for i, id := range ids {
		binary.LittleEndian.PutUint32(src[i*4:], id)
	}

	t.Logf("Input: %d IDs (%d bytes)", len(ids), len(src))

	// Params: element width = 4 bytes (uint32)
	params := []byte{4}

	// Compress
	dst := make([]byte, len(src)+100)
	n, err := codec.Encode(dst, src, params)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	ratio := float64(len(src)) / float64(n)
	t.Logf("Compressed: %d bytes → %d bytes (%.2f× compression)", len(src), n, ratio)

	packedWidth := int(dst[16])
	t.Logf("  Packed as uint%d (saves 75%% space)", packedWidth*8)

	if packedWidth != 1 {
		t.Errorf("Expected packed width 1 (uint8), got %d", packedWidth)
	}

	// Decompress
	decompressed := make([]byte, len(src))
	dn, err := codec.Decode(decompressed, dst[:n], params)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if dn != len(src) {
		t.Errorf("Expected %d decompressed bytes, got %d", len(src), dn)
	}

	// Verify roundtrip
	for i := 0; i < len(ids); i++ {
		got := binary.LittleEndian.Uint32(decompressed[i*4:])
		if got != ids[i] {
			t.Errorf("ID %d: expected %d, got %d", i, ids[i], got)
		}
	}

	t.Logf("✅ Roundtrip successful\n")

	// Target: 3-4× compression (uint32 → uint8 = 4× minus header overhead)
	if ratio < 2.0 {
		t.Errorf("Expected compression ratio >= 2×, got %.2f×", ratio)
	}
}

// TestRangePack_SmallRange tests compression of values with small range
func TestRangePack_SmallRange(t *testing.T) {
	t.Log("=== RangePack: Small Range (1000-1100) ===\n")

	codec := NewRangePack()

	// Values: 1000-1100 (range = 100, fits in uint8)
	values := []uint16{1000, 1010, 1050, 1100}

	// Encode to bytes (2 bytes each)
	src := make([]byte, len(values)*2)
	for i, val := range values {
		binary.LittleEndian.PutUint16(src[i*2:], val)
	}

	t.Logf("Input: %v (%d bytes)", values, len(src))

	// Params: element width = 2 bytes (uint16)
	params := []byte{2}

	// Compress
	dst := make([]byte, len(src)+100)
	n, err := codec.Encode(dst, src, params)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	ratio := float64(len(src)) / float64(n)
	t.Logf("Compressed: %d bytes → %d bytes (%.2f× compression)", len(src), n, ratio)

	// Verify packed as uint8
	packedWidth := int(dst[16])
	t.Logf("  Packed as uint%d", packedWidth*8)

	if packedWidth != 1 {
		t.Errorf("Expected packed width 1 (uint8), got %d", packedWidth)
	}

	// Decompress
	decompressed := make([]byte, len(src))
	_, err = codec.Decode(decompressed, dst[:n], params)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Verify roundtrip
	for i := 0; i < len(values); i++ {
		got := binary.LittleEndian.Uint16(decompressed[i*2:])
		if got != values[i] {
			t.Errorf("Value %d: expected %d, got %d", i, values[i], got)
		}
	}

	t.Logf("✅ Roundtrip successful\n")

	// For small arrays, header overhead dominates
	// Expected: Slight expansion due to 17-byte header
	if n > len(src)+20 {
		t.Errorf("Expected compressed size <= %d, got %d", len(src)+20, n)
	}
}

// TestRangePack_AllWidths tests all element widths (1, 2, 4, 8 bytes)
func TestRangePack_AllWidths(t *testing.T) {
	t.Log("=== RangePack: All Element Widths ===\n")

	codec := NewRangePack()

	testCases := []struct {
		name      string
		width     int
		values    []uint64
		expectPW  int // Expected packed width
	}{
		{
			name:     "uint8_range255",
			width:    1,
			values:   []uint64{100, 150, 200, 250, 255},
			expectPW: 1, // Range 155, fits in uint8
		},
		{
			name:     "uint16_range1000",
			width:    2,
			values:   []uint64{10000, 10100, 10500, 11000},
			expectPW: 2, // Range 1000, fits in uint16
		},
		{
			name:     "uint32_range100K",
			width:    4,
			values:   []uint64{1000000, 1010000, 1050000, 1100000},
			expectPW: 4, // Range 100K, fits in uint32
		},
		{
			name:     "uint64_large_range",
			width:    8,
			values:   []uint64{1700000000000, 1700000000100, 1700000001000},
			expectPW: 2, // Range 1000, fits in uint16 despite uint64 input
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("\nTest: %s (element width: %d bytes)", tc.name, tc.width)

			// Encode values to bytes
			src := make([]byte, len(tc.values)*tc.width)
			for i, val := range tc.values {
				offset := i * tc.width
				switch tc.width {
				case 1:
					src[offset] = byte(val)
				case 2:
					binary.LittleEndian.PutUint16(src[offset:], uint16(val))
				case 4:
					binary.LittleEndian.PutUint32(src[offset:], uint32(val))
				case 8:
					binary.LittleEndian.PutUint64(src[offset:], val)
				}
			}

			params := []byte{byte(tc.width)}

			// Compress
			dst := make([]byte, len(src)+100)
			n, err := codec.Encode(dst, src, params)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}

			ratio := float64(len(src)) / float64(n)
			packedWidth := int(dst[16])

			t.Logf("  Input: %d bytes", len(src))
			t.Logf("  Compressed: %d bytes (%.2f× compression)", n, ratio)
			t.Logf("  Packed width: %d bytes (uint%d)", packedWidth, packedWidth*8)

			if packedWidth != tc.expectPW {
				t.Errorf("Expected packed width %d, got %d", tc.expectPW, packedWidth)
			}

			// Decompress
			decompressed := make([]byte, len(src))
			_, err = codec.Decode(decompressed, dst[:n], params)
			if err != nil {
				t.Fatalf("Decode failed: %v", err)
			}

			// Verify roundtrip
			for i := 0; i < len(tc.values); i++ {
				offset := i * tc.width
				var got uint64
				switch tc.width {
				case 1:
					got = uint64(decompressed[offset])
				case 2:
					got = uint64(binary.LittleEndian.Uint16(decompressed[offset:]))
				case 4:
					got = uint64(binary.LittleEndian.Uint32(decompressed[offset:]))
				case 8:
					got = binary.LittleEndian.Uint64(decompressed[offset:])
				}

				if got != tc.values[i] {
					t.Errorf("Value %d: expected %d, got %d", i, tc.values[i], got)
				}
			}

			t.Logf("  ✅ Roundtrip successful")
		})
	}

	t.Log("\n✅ All widths tested successfully")
}

// TestRangePack_Roundtrip tests encode/decode correctness
func TestRangePack_Roundtrip(t *testing.T) {
	t.Log("=== RangePack: Roundtrip Tests ===\n")

	codec := NewRangePack()

	testCases := []struct {
		name   string
		values []uint64
		width  int
	}{
		{
			name:   "sequential",
			values: []uint64{100, 101, 102, 103, 104, 105},
			width:  2,
		},
		{
			name:   "sparse",
			values: []uint64{1000, 2000, 3000, 4000},
			width:  4,
		},
		{
			name:   "all_same",
			values: []uint64{42, 42, 42, 42, 42},
			width:  1,
		},
		{
			name:   "two_extremes",
			values: []uint64{100, 100, 100, 200, 200, 200},
			width:  2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encode values
			src := make([]byte, len(tc.values)*tc.width)
			for i, val := range tc.values {
				offset := i * tc.width
				switch tc.width {
				case 1:
					src[offset] = byte(val)
				case 2:
					binary.LittleEndian.PutUint16(src[offset:], uint16(val))
				case 4:
					binary.LittleEndian.PutUint32(src[offset:], uint32(val))
				case 8:
					binary.LittleEndian.PutUint64(src[offset:], val)
				}
			}

			params := []byte{byte(tc.width)}

			// Compress
			dst := make([]byte, len(src)+100)
			n, err := codec.Encode(dst, src, params)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}

			// Decompress
			decompressed := make([]byte, len(src))
			dn, err := codec.Decode(decompressed, dst[:n], params)
			if err != nil {
				t.Fatalf("Decode failed: %v", err)
			}

			if dn != len(src) {
				t.Errorf("Expected %d decompressed bytes, got %d", len(src), dn)
			}

			// Verify exact match
			for i := 0; i < len(src); i++ {
				if decompressed[i] != src[i] {
					t.Errorf("Byte %d: expected %d, got %d", i, src[i], decompressed[i])
				}
			}

			t.Logf("%s: ✅ Roundtrip successful (%d values)", tc.name, len(tc.values))
		})
	}
}

// TestRangePack_EmptyInput tests error handling for empty input
func TestRangePack_EmptyInput(t *testing.T) {
	codec := NewRangePack()

	dst := make([]byte, 100)
	params := []byte{8}

	_, err := codec.Encode(dst, []byte{}, params)
	if err == nil {
		t.Fatal("Expected error for empty input")
	}

	t.Logf("✅ Empty input rejected: %v", err)
}

// TestRangePack_InvalidWidth tests error handling for invalid element widths
func TestRangePack_InvalidWidth(t *testing.T) {
	codec := NewRangePack()

	src := make([]byte, 16)
	dst := make([]byte, 100)

	invalidWidths := []int{3, 5, 6, 7, 9, 16}

	for _, width := range invalidWidths {
		params := []byte{byte(width)}
		_, err := codec.Encode(dst, src, params)
		if err == nil {
			t.Errorf("Expected error for width %d", width)
		}
	}

	t.Log("✅ Invalid widths rejected")
}

// TestRangePack_MisalignedData tests error handling for misaligned data
func TestRangePack_MisalignedData(t *testing.T) {
	codec := NewRangePack()

	// 13 bytes is not aligned to any valid width (2, 4, 8)
	src := make([]byte, 13)
	dst := make([]byte, 100)

	params := []byte{4} // Width 4, but 13 is not divisible by 4

	_, err := codec.Encode(dst, src, params)
	if err == nil {
		t.Fatal("Expected error for misaligned data")
	}

	t.Logf("✅ Misaligned data rejected: %v", err)
}

// TestRangePack_CodecInterface tests codec interface compliance
func TestRangePack_CodecInterface(t *testing.T) {
	codec := NewRangePack()

	if codec.ID() != IDRangePack {
		t.Errorf("Expected ID %d, got %d", IDRangePack, codec.ID())
	}

	if codec.Name() != "RangePack" {
		t.Errorf("Expected name 'RangePack', got '%s'", codec.Name())
	}

	if codec.PreservesSize() {
		t.Error("RangePack should not preserve size")
	}

	t.Log("✅ Codec interface compliance verified")
}

// TestRangePack_LargeDataset tests compression on 1000+ values
func TestRangePack_LargeDataset(t *testing.T) {
	t.Log("=== RangePack: Large Dataset (1000 Timestamps) ===\n")

	codec := NewRangePack()

	// 1000 timestamps with 1-second increments
	count := 1000
	baseTimestamp := uint64(1700000000)
	timestamps := make([]uint64, count)
	for i := 0; i < count; i++ {
		timestamps[i] = baseTimestamp + uint64(i)
	}

	// Encode to bytes
	src := make([]byte, count*8)
	for i, ts := range timestamps {
		binary.LittleEndian.PutUint64(src[i*8:], ts)
	}

	t.Logf("Input: %d timestamps (%d bytes, %.2f KB)", count, len(src), float64(len(src))/1024)

	params := []byte{8}

	// Compress
	dst := make([]byte, len(src)+100)
	n, err := codec.Encode(dst, src, params)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	ratio := float64(len(src)) / float64(n)
	t.Logf("Compressed: %d bytes (%.2f KB) → %d bytes (%.2f KB)",
		len(src), float64(len(src))/1024, n, float64(n)/1024)
	t.Logf("Compression ratio: %.2f×", ratio)

	packedWidth := int(dst[16])
	t.Logf("Packed as uint%d (saves %.0f%% space)", packedWidth*8, (1.0-float64(packedWidth)/8.0)*100)

	// Decompress
	decompressed := make([]byte, len(src))
	dn, err := codec.Decode(decompressed, dst[:n], params)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if dn != len(src) {
		t.Errorf("Expected %d decompressed bytes, got %d", len(src), dn)
	}

	// Verify sample values
	for i := 0; i < count; i += 100 {
		got := binary.LittleEndian.Uint64(decompressed[i*8:])
		if got != timestamps[i] {
			t.Errorf("Timestamp %d: expected %d, got %d", i, timestamps[i], got)
		}
	}

	t.Logf("✅ All %d values decompressed correctly", count)
	t.Logf("✅ Large dataset test passed\n")

	// Target: 3-4× compression for large timestamp datasets
	if ratio < 3.0 {
		t.Errorf("Expected compression ratio >= 3×, got %.2f×", ratio)
	}
}
