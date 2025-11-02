package codec

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestDelta_Decode8 tests 1-byte delta decoding
func TestDelta_Decode8(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte // Delta-encoded
		expected []byte // Original values
	}{
		{
			name:     "simple sequence",
			input:    []byte{100, 2, 3, 253, 5}, // 100, +2, +3, -3 (253=two's complement), +5
			expected: []byte{100, 102, 105, 102, 107},
		},
		{
			name:     "constant value",
			input:    []byte{42, 0, 0, 0, 0},
			expected: []byte{42, 42, 42, 42, 42},
		},
		{
			name:     "incrementing by 1",
			input:    []byte{0, 1, 1, 1, 1},
			expected: []byte{0, 1, 2, 3, 4},
		},
		{
			name:     "single value",
			input:    []byte{255},
			expected: []byte{255},
		},
		{
			name:     "overflow wrap",
			input:    []byte{250, 10}, // 250 + 10 = 260 & 0xFF = 4
			expected: []byte{250, 4},
		},
	}

	codec := NewDelta(1)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst := make([]byte, len(tt.expected))
			n, err := codec.Decode(dst, tt.input, []byte{1})
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}

			if n != len(tt.expected) {
				t.Errorf("Decode() returned %d bytes, want %d", n, len(tt.expected))
			}

			if !bytes.Equal(dst[:n], tt.expected) {
				t.Errorf("Decode() = %v, want %v", dst[:n], tt.expected)
			}
		})
	}
}

// TestDelta_Decode16 tests 2-byte delta decoding
func TestDelta_Decode16(t *testing.T) {
	// Create test data: [1000, 1002, 1005, 1003, 1008]
	input := make([]byte, 10)
	binary.LittleEndian.PutUint16(input[0:], 1000)  // First value
	binary.LittleEndian.PutUint16(input[2:], 2)     // +2
	binary.LittleEndian.PutUint16(input[4:], 3)     // +3
	binary.LittleEndian.PutUint16(input[6:], 65534) // -2 (two's complement)
	binary.LittleEndian.PutUint16(input[8:], 5)     // +5

	codec := NewDelta(2)
	dst := make([]byte, 10)
	n, err := codec.Decode(dst, input, []byte{2})
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if n != 10 {
		t.Errorf("Decode() returned %d bytes, want 10", n)
	}

	// Verify decoded values
	expected := []uint16{1000, 1002, 1005, 1003, 1008}
	for i, want := range expected {
		got := binary.LittleEndian.Uint16(dst[i*2:])
		if got != want {
			t.Errorf("Decode() value[%d] = %d, want %d", i, got, want)
		}
	}
}

// TestDelta_Decode32 tests 4-byte delta decoding
func TestDelta_Decode32(t *testing.T) {
	input := make([]byte, 20)
	binary.LittleEndian.PutUint32(input[0:], 1000000)
	binary.LittleEndian.PutUint32(input[4:], 100)
	binary.LittleEndian.PutUint32(input[8:], 200)
	binary.LittleEndian.PutUint32(input[12:], 0xFFFFFFF6) // -10 (two's complement)
	binary.LittleEndian.PutUint32(input[16:], 500)

	codec := NewDelta(4)
	dst := make([]byte, 20)
	_, err := codec.Decode(dst, input, []byte{4})
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	expected := []uint32{1000000, 1000100, 1000300, 1000290, 1000790}
	for i, want := range expected {
		got := binary.LittleEndian.Uint32(dst[i*4:])
		if got != want {
			t.Errorf("Decode() value[%d] = %d, want %d", i, got, want)
		}
	}
}

// TestDelta_Decode64 tests 8-byte delta decoding
func TestDelta_Decode64(t *testing.T) {
	input := make([]byte, 40)
	binary.LittleEndian.PutUint64(input[0:], 1000000000)
	binary.LittleEndian.PutUint64(input[8:], 1000)
	binary.LittleEndian.PutUint64(input[16:], 2000)
	binary.LittleEndian.PutUint64(input[24:], 0xFFFFFFFFFFFFFC18) // -1000 (two's complement)
	binary.LittleEndian.PutUint64(input[32:], 5000)

	codec := NewDelta(8)
	dst := make([]byte, 40)
	_, err := codec.Decode(dst, input, []byte{8})
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	expected := []uint64{1000000000, 1000001000, 1000003000, 1000002000, 1000007000}
	for i, want := range expected {
		got := binary.LittleEndian.Uint64(dst[i*8:])
		if got != want {
			t.Errorf("Decode() value[%d] = %d, want %d", i, got, want)
		}
	}
}

// TestDelta_Encode8 tests 1-byte delta encoding
func TestDelta_Encode8(t *testing.T) {
	input := []byte{100, 102, 105, 102, 107}
	expected := []byte{100, 2, 3, 253, 5} // 253 = -3 in two's complement

	codec := NewDelta(1)
	dst := make([]byte, len(input))
	n, err := codec.Encode(dst, input, []byte{1})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	if n != len(expected) {
		t.Errorf("Encode() returned %d bytes, want %d", n, len(expected))
	}

	if !bytes.Equal(dst[:n], expected) {
		t.Errorf("Encode() = %v, want %v", dst[:n], expected)
	}
}

// TestDelta_Roundtrip tests encode->decode roundtrip
func TestDelta_Roundtrip(t *testing.T) {
	tests := []struct {
		name        string
		elementSize int
		data        []byte
	}{
		{
			name:        "1-byte elements",
			elementSize: 1,
			data:        []byte{50, 55, 60, 58, 65, 70},
		},
		{
			name:        "2-byte elements",
			elementSize: 2,
			data:        makeUint16Bytes([]uint16{1000, 1005, 1010, 1008, 1015}),
		},
		{
			name:        "4-byte elements",
			elementSize: 4,
			data:        makeUint32Bytes([]uint32{100000, 100050, 100100, 100090, 100150}),
		},
		{
			name:        "8-byte elements",
			elementSize: 8,
			data:        makeUint64Bytes([]uint64{1000000000, 1000001000, 1000003000, 1000002000}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codec := NewDelta(tt.elementSize)
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

			// Verify
			if n2 != len(tt.data) {
				t.Errorf("Decoded %d bytes, want %d", n2, len(tt.data))
			}

			if !bytes.Equal(decoded[:n2], tt.data) {
				t.Errorf("Roundtrip failed: decoded data doesn't match original")
			}
		})
	}
}

// TestDelta_EmptyData tests handling of empty input
func TestDelta_EmptyData(t *testing.T) {
	codec := NewDelta(8)

	dst := make([]byte, 0)
	n, err := codec.Decode(dst, []byte{}, []byte{8})
	if err != nil {
		t.Errorf("Decode() error = %v", err)
	}
	if n != 0 {
		t.Errorf("Decode() returned %d bytes, want 0", n)
	}

	n, err = codec.Encode(dst, []byte{}, []byte{8})
	if err != nil {
		t.Errorf("Encode() error = %v", err)
	}
	if n != 0 {
		t.Errorf("Encode() returned %d bytes, want 0", n)
	}
}

// TestDelta_BufferTooSmall tests error handling
func TestDelta_BufferTooSmall(t *testing.T) {
	codec := NewDelta(4)
	input := make([]byte, 20) // 5 elements

	// Too small buffer
	dst := make([]byte, 10) // Only 2.5 elements worth
	_, err := codec.Decode(dst, input, []byte{4})
	if err != ErrBufferTooSmall {
		t.Errorf("Decode() error = %v, want ErrBufferTooSmall", err)
	}
}

// TestDelta_InvalidElementSize tests error handling for invalid sizes
func TestDelta_InvalidElementSize(t *testing.T) {
	codec := NewDelta(3) // Invalid size
	input := []byte{1, 2, 3}
	dst := make([]byte, 10)

	_, err := codec.Decode(dst, input, []byte{3})
	if err == nil {
		t.Error("Decode() should error for invalid element size")
	}

	_, err = codec.Encode(dst, input, []byte{3})
	if err == nil {
		t.Error("Encode() should error for invalid element size")
	}
}

// TestDelta_UnalignedInput tests error handling for misaligned input
func TestDelta_UnalignedInput(t *testing.T) {
	codec := NewDelta(4)
	input := []byte{1, 2, 3} // 3 bytes, not aligned to 4
	dst := make([]byte, 10)

	_, err := codec.Decode(dst, input, []byte{4})
	if err == nil {
		t.Error("Decode() should error for unaligned input")
	}
}

// TestDelta_Metadata tests codec metadata
func TestDelta_Metadata(t *testing.T) {
	codec := NewDelta(8)

	if codec.ID() != IDDelta {
		t.Errorf("ID() = %v, want %v", codec.ID(), IDDelta)
	}

	if codec.Name() != "Delta" {
		t.Errorf("Name() = %q, want %q", codec.Name(), "Delta")
	}
}

// TestDelta_TimeSeries tests realistic time series data
func TestDelta_TimeSeries(t *testing.T) {
	// Simulate timestamps increasing by ~1000ms each time
	timestamps := []uint64{
		1700000000000,
		1700000001000,
		1700000002100, // Slight variation
		1700000003000,
		1700000004200,
	}

	input := makeUint64Bytes(timestamps)
	codec := NewDelta(8)

	// Encode
	encoded := make([]byte, len(input))
	n, err := codec.Encode(encoded, input, []byte{8})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	t.Logf("Original size: %d bytes", len(input))
	t.Logf("Encoded size: %d bytes (deltas compress better!)", n)

	// Decode
	decoded := make([]byte, len(input))
	n2, err := codec.Decode(decoded, encoded[:n], []byte{8})
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	// Verify
	if !bytes.Equal(decoded[:n2], input) {
		t.Error("Time series roundtrip failed")
	}
}

// Helper functions

func makeUint16Bytes(values []uint16) []byte {
	data := make([]byte, len(values)*2)
	for i, v := range values {
		binary.LittleEndian.PutUint16(data[i*2:], v)
	}
	return data
}

func makeUint32Bytes(values []uint32) []byte {
	data := make([]byte, len(values)*4)
	for i, v := range values {
		binary.LittleEndian.PutUint32(data[i*4:], v)
	}
	return data
}

func makeUint64Bytes(values []uint64) []byte {
	data := make([]byte, len(values)*8)
	for i, v := range values {
		binary.LittleEndian.PutUint64(data[i*8:], v)
	}
	return data
}

// Benchmarks

func BenchmarkDelta_Decode(b *testing.B) {
	sizes := []struct {
		name        string
		numElements int
		elementSize int
	}{
		{"8bit-100", 100, 1},
		{"16bit-100", 100, 2},
		{"32bit-100", 100, 4},
		{"64bit-100", 100, 8},
		{"64bit-1000", 1000, 8},
	}

	for _, sz := range sizes {
		b.Run(sz.name, func(b *testing.B) {
			codec := NewDelta(sz.elementSize)
			input := make([]byte, sz.numElements*sz.elementSize)
			dst := make([]byte, sz.numElements*sz.elementSize)
			params := []byte{byte(sz.elementSize)}

			b.SetBytes(int64(len(input)))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := codec.Decode(dst, input, params)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkDelta_Encode(b *testing.B) {
	sizes := []struct {
		name        string
		numElements int
		elementSize int
	}{
		{"8bit-100", 100, 1},
		{"64bit-100", 100, 8},
		{"64bit-1000", 1000, 8},
	}

	for _, sz := range sizes {
		b.Run(sz.name, func(b *testing.B) {
			codec := NewDelta(sz.elementSize)
			input := make([]byte, sz.numElements*sz.elementSize)
			dst := make([]byte, sz.numElements*sz.elementSize)
			params := []byte{byte(sz.elementSize)}

			b.SetBytes(int64(len(input)))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := codec.Encode(dst, input, params)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
