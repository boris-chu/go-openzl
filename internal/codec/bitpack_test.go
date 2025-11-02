package codec

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestBitpack_Metadata verifies the codec metadata
func TestBitpack_Metadata(t *testing.T) {
	codec := NewBitpack(4)

	if codec.ID() != IDBitpack {
		t.Errorf("ID() = %v, want %v", codec.ID(), IDBitpack)
	}

	if codec.Name() != "Bitpack" {
		t.Errorf("Name() = %q, want %q", codec.Name(), "Bitpack")
	}
}

// TestBitpack_Roundtrip_SmallValues tests compression of small values
func TestBitpack_Roundtrip_SmallValues(t *testing.T) {
	tests := []struct {
		name         string
		values       []uint32
		expectedBits int
	}{
		{
			name:         "single_bit",
			values:       []uint32{0, 1, 0, 1, 0},
			expectedBits: 1,
		},
		{
			name:         "three_bits",
			values:       []uint32{5, 2, 7, 1, 4},
			expectedBits: 3,
		},
		{
			name:         "four_bits",
			values:       []uint32{15, 2, 9, 1, 14},
			expectedBits: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codec := NewBitpack(4)

			// Convert to bytes
			src := make([]byte, len(tt.values)*4)
			for i, v := range tt.values {
				binary.LittleEndian.PutUint32(src[i*4:], v)
			}

			// Encode
			encoded := make([]byte, len(src)*2)
			n, err := codec.Encode(encoded, src, nil)
			if err != nil {
				t.Fatalf("Encode() error = %v", err)
			}
			encoded = encoded[:n]

			// Check bits used
			bitsUsed := int(encoded[1])
			if bitsUsed != tt.expectedBits {
				t.Errorf("Expected %d bits, got %d bits", tt.expectedBits, bitsUsed)
			}

			// Calculate compression ratio
			originalSize := len(src)
			compressedSize := len(encoded)
			ratio := float64(originalSize) / float64(compressedSize)

			t.Logf("Original: %d bytes, Compressed: %d bytes, Ratio: %.2fx",
				originalSize, compressedSize, ratio)

			// Decode
			decoded := make([]byte, len(src))
			n, err = codec.Decode(decoded, encoded, nil)
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}

			// Verify
			if !bytes.Equal(src, decoded[:n]) {
				t.Errorf("Roundtrip failed: decoded != original")
			}
		})
	}
}

// TestBitpack_AllZeros tests handling of all-zero data
func TestBitpack_AllZeros(t *testing.T) {
	codec := NewBitpack(4)

	// Create all zeros
	src := make([]byte, 100*4) // 100 uint32s

	// Encode
	encoded := make([]byte, len(src))
	n, err := codec.Encode(encoded, src, nil)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	encoded = encoded[:n]

	// Should use 0 bits (special case)
	bitsUsed := int(encoded[1])
	if bitsUsed != 0 {
		t.Errorf("All zeros should use 0 bits, got %d", bitsUsed)
	}

	// Should be tiny (just header)
	if len(encoded) != 6 {
		t.Errorf("All zeros should compress to 6 bytes (header), got %d", len(encoded))
	}

	t.Logf("100 zeros: %d bytes → %d bytes (%.2fx)", len(src), len(encoded), float64(len(src))/float64(len(encoded)))

	// Decode
	decoded := make([]byte, len(src))
	n, err = codec.Decode(decoded, encoded, nil)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	// Verify
	if !bytes.Equal(src, decoded[:n]) {
		t.Errorf("Roundtrip failed for all zeros")
	}
}

// TestBitpack_DifferentElementSizes tests all element sizes
func TestBitpack_DifferentElementSizes(t *testing.T) {
	tests := []struct {
		name        string
		elementSize int
		value       uint64
		count       int
	}{
		{"1-byte", 1, 7, 10},
		{"2-byte", 2, 255, 10},
		{"4-byte", 4, 65535, 10},
		{"8-byte", 8, 1000000, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codec := NewBitpack(tt.elementSize)

			// Create source data
			src := make([]byte, tt.count*tt.elementSize)
			for i := 0; i < tt.count; i++ {
				offset := i * tt.elementSize
				switch tt.elementSize {
				case 1:
					src[offset] = byte(tt.value)
				case 2:
					binary.LittleEndian.PutUint16(src[offset:], uint16(tt.value))
				case 4:
					binary.LittleEndian.PutUint32(src[offset:], uint32(tt.value))
				case 8:
					binary.LittleEndian.PutUint64(src[offset:], tt.value)
				}
			}

			// Encode
			encoded := make([]byte, len(src)*2)
			n, err := codec.Encode(encoded, src, nil)
			if err != nil {
				t.Fatalf("Encode() error = %v", err)
			}
			encoded = encoded[:n]

			// Decode
			decoded := make([]byte, len(src))
			n, err = codec.Decode(decoded, encoded, nil)
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}

			// Verify
			if !bytes.Equal(src, decoded[:n]) {
				t.Errorf("Roundtrip failed for element size %d", tt.elementSize)
			}

			t.Logf("Element size %d: %d → %d bytes (%.2f%%)",
				tt.elementSize, len(src), len(encoded), 100.0*float64(len(encoded))/float64(len(src)))
		})
	}
}

// TestBitpack_AfterDeltaZigZag tests realistic pipeline: Delta → ZigZag → Bitpack
func TestBitpack_AfterDeltaZigZag(t *testing.T) {
	// Original timestamps (increasing with small deltas)
	timestamps := []int32{1000, 1005, 1003, 1008, 1004, 1010}

	// Convert to bytes
	src := make([]byte, len(timestamps)*4)
	for i, v := range timestamps {
		binary.LittleEndian.PutUint32(src[i*4:], uint32(v))
	}

	t.Logf("Original: %v", timestamps)
	originalSize := len(src)

	// Step 1: Delta encode
	deltaCodec := NewDelta(4)
	deltaDst := make([]byte, len(src))
	n, err := deltaCodec.Encode(deltaDst, src, nil)
	if err != nil {
		t.Fatalf("Delta encode error: %v", err)
	}
	deltaDst = deltaDst[:n]

	// Extract delta values for logging
	deltas := make([]int32, len(timestamps))
	for i := 0; i < len(timestamps); i++ {
		deltas[i] = int32(binary.LittleEndian.Uint32(deltaDst[i*4:]))
	}
	t.Logf("After Delta: %v", deltas)

	// Step 2: ZigZag encode
	zigzagCodec := NewZigZag(4)
	zigzagDst := make([]byte, len(deltaDst))
	n, err = zigzagCodec.Encode(zigzagDst, deltaDst, nil)
	if err != nil {
		t.Fatalf("ZigZag encode error: %v", err)
	}
	zigzagDst = zigzagDst[:n]

	// Extract zigzag values for logging
	zigzags := make([]uint32, len(timestamps))
	for i := 0; i < len(timestamps); i++ {
		zigzags[i] = binary.LittleEndian.Uint32(zigzagDst[i*4:])
	}
	t.Logf("After ZigZag: %v", zigzags)

	// Step 3: Bitpack
	bitpackCodec := NewBitpack(4)
	bitpackDst := make([]byte, len(zigzagDst)*2)
	n, err = bitpackCodec.Encode(bitpackDst, zigzagDst, nil)
	if err != nil {
		t.Fatalf("Bitpack encode error: %v", err)
	}
	bitpackDst = bitpackDst[:n]

	finalSize := len(bitpackDst)
	ratio := float64(originalSize) / float64(finalSize)

	t.Logf("Final compressed size: %d bytes", finalSize)
	t.Logf("Compression ratio: %.2fx (from %d to %d bytes)", ratio, originalSize, finalSize)
	t.Logf("Bits used for packing: %d", bitpackDst[1])

	// Decode pipeline (reverse order)
	// Step 3 reverse: Unpack
	unpacked := make([]byte, len(zigzagDst))
	n, err = bitpackCodec.Decode(unpacked, bitpackDst, nil)
	if err != nil {
		t.Fatalf("Bitpack decode error: %v", err)
	}

	// Step 2 reverse: ZigZag decode
	unzigzagged := make([]byte, len(deltaDst))
	n, err = zigzagCodec.Decode(unzigzagged, unpacked[:n], nil)
	if err != nil {
		t.Fatalf("ZigZag decode error: %v", err)
	}

	// Step 1 reverse: Delta decode
	final := make([]byte, len(src))
	n, err = deltaCodec.Decode(final, unzigzagged[:n], nil)
	if err != nil {
		t.Fatalf("Delta decode error: %v", err)
	}

	// Verify roundtrip
	if !bytes.Equal(src, final[:n]) {
		t.Errorf("Pipeline roundtrip failed")
	}

	t.Logf("✓ Delta → ZigZag → Bitpack pipeline works!")
}

// TestBitpack_LargeData tests bitpacking on larger datasets
func TestBitpack_LargeData(t *testing.T) {
	codec := NewBitpack(4)

	// Create 1000 small values (0-15, should use 4 bits each)
	src := make([]byte, 1000*4)
	for i := 0; i < 1000; i++ {
		binary.LittleEndian.PutUint32(src[i*4:], uint32(i%16))
	}

	// Encode
	encoded := make([]byte, len(src))
	n, err := codec.Encode(encoded, src, nil)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	encoded = encoded[:n]

	ratio := float64(len(src)) / float64(len(encoded))
	t.Logf("1000 values (0-15): %d bytes → %d bytes (%.2fx compression)",
		len(src), len(encoded), ratio)

	// Should achieve significant compression
	if ratio < 2.0 {
		t.Errorf("Expected at least 2x compression, got %.2fx", ratio)
	}

	// Decode
	decoded := make([]byte, len(src))
	n, err = codec.Decode(decoded, encoded, nil)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	// Verify
	if !bytes.Equal(src, decoded[:n]) {
		t.Errorf("Roundtrip failed for large data")
	}
}

// TestBitpack_EmptyInput tests handling of empty input
func TestBitpack_EmptyInput(t *testing.T) {
	codec := NewBitpack(4)

	src := []byte{}
	dst := make([]byte, 100)

	_, err := codec.Encode(dst, src, nil)
	if err == nil {
		t.Errorf("Expected error for empty input, got nil")
	}
}

// TestBitpack_BufferTooSmall tests error handling for small buffers
func TestBitpack_BufferTooSmall(t *testing.T) {
	codec := NewBitpack(4)

	src := make([]byte, 10*4)
	for i := 0; i < 10; i++ {
		binary.LittleEndian.PutUint32(src[i*4:], uint32(i))
	}

	// Encode first
	encoded := make([]byte, len(src)*2)
	n, err := codec.Encode(encoded, src, nil)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	encoded = encoded[:n]

	// Try to decode with too small buffer
	tooSmall := make([]byte, len(src)/2)
	_, err = codec.Decode(tooSmall, encoded, nil)
	if err == nil {
		t.Errorf("Expected error for too small decode buffer, got nil")
	}
}

// TestBitpack_InvalidElementSize tests error handling for invalid sizes
func TestBitpack_InvalidElementSize(t *testing.T) {
	invalidSizes := []int{0, 3, 5, 7, 16}

	for _, size := range invalidSizes {
		t.Run(string(rune(size)), func(t *testing.T) {
			codec := NewBitpack(size)

			src := make([]byte, 10*4) // Some data
			dst := make([]byte, len(src))

			_, err := codec.Encode(dst, src, []byte{byte(size)})
			if err == nil {
				t.Errorf("Expected error for invalid element size %d, got nil", size)
			}
		})
	}
}

// BenchmarkBitpack_Encode benchmarks bitpack encoding
func BenchmarkBitpack_Encode(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(string(rune(size)), func(b *testing.B) {
			codec := NewBitpack(4)

			// Create data with small values (good for bitpacking)
			src := make([]byte, size*4)
			for i := 0; i < size; i++ {
				binary.LittleEndian.PutUint32(src[i*4:], uint32(i%256))
			}

			dst := make([]byte, len(src)*2)

			b.SetBytes(int64(len(src)))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := codec.Encode(dst, src, nil)
				if err != nil {
					b.Fatalf("Encode() error = %v", err)
				}
			}
		})
	}
}

// BenchmarkBitpack_Decode benchmarks bitpack decoding
func BenchmarkBitpack_Decode(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(string(rune(size)), func(b *testing.B) {
			codec := NewBitpack(4)

			// Create and encode data
			src := make([]byte, size*4)
			for i := 0; i < size; i++ {
				binary.LittleEndian.PutUint32(src[i*4:], uint32(i%256))
			}

			encoded := make([]byte, len(src)*2)
			n, err := codec.Encode(encoded, src, nil)
			if err != nil {
				b.Fatalf("Encode() error = %v", err)
			}
			encoded = encoded[:n]

			decoded := make([]byte, len(src))

			b.SetBytes(int64(len(src)))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := codec.Decode(decoded, encoded, nil)
				if err != nil {
					b.Fatalf("Decode() error = %v", err)
				}
			}
		})
	}
}
