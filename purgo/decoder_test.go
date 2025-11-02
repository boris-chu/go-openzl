package purgo

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/borischu/go-openzl/internal/codec"
	"github.com/borischu/go-openzl/internal/frame"
	"github.com/borischu/go-openzl/internal/graph"
)

// Test helper to create OpenZL compressed data
func createCompressed(t *testing.T, rawBytes []byte) []byte {
	t.Helper()

	g := &graph.Graph{
		Nodes: []*graph.Node{
			{CodecID: codec.IDIdentity, Params: nil, Inputs: nil},
		},
		Outputs: []int{0},
	}

	graphBytes, err := graph.EncodeGraph(g)
	if err != nil {
		t.Fatalf("failed to encode graph: %v", err)
	}
	payload := append(graphBytes, rawBytes...)

	f := &frame.Frame{
		Header: &frame.Header{
			Magic:   frame.MagicNumberBase + 21,
			Version: 21,
			Flags:   0,
		},
		Outputs: []*frame.Output{
			{
				Type:             frame.TypeSerial,
				DecompressedSize: uint64(len(rawBytes)),
			},
		},
		Payload: payload,
	}

	// For testing, we manually serialize the frame
	// In a real scenario, this would use a proper frame writer
	frameBuf := new(bytes.Buffer)

	// Write magic number (little-endian)
	magic := f.Header.Magic
	frameBuf.WriteByte(byte(magic))
	frameBuf.WriteByte(byte(magic >> 8))
	frameBuf.WriteByte(byte(magic >> 16))
	frameBuf.WriteByte(byte(magic >> 24))

	// Write flags
	frameBuf.WriteByte(byte(f.Header.Flags))

	// Write token1 (nbOutputs in lower 4 bits)
	token1 := byte(len(f.Outputs))
	frameBuf.WriteByte(token1)

	// Write output size (varint)
	// Note: OpenZL stores size as (actual_size + 1), so 0 size = varint 1
	if len(f.Outputs) > 0 {
		writeVarintTest(frameBuf, f.Outputs[0].DecompressedSize+1)
	}

	// Write payload
	frameBuf.Write(f.Payload)

	return frameBuf.Bytes()
}

// writeVarintTest writes a LEB128 varint (for test helpers)
func writeVarintTest(buf *bytes.Buffer, value uint64) {
	for {
		b := byte(value & 0x7F)
		value >>= 7
		if value != 0 {
			b |= 0x80
		}
		buf.WriteByte(b)
		if value == 0 {
			break
		}
	}
}

// Test helpers for typed data

func createCompressedInt64(t *testing.T, data []int64) []byte {
	t.Helper()
	buf := new(bytes.Buffer)
	for _, val := range data {
		binary.Write(buf, binary.LittleEndian, val)
	}
	return createCompressed(t, buf.Bytes())
}

func createCompressedFloat64(t *testing.T, data []float64) []byte {
	t.Helper()
	buf := new(bytes.Buffer)
	for _, val := range data {
		binary.Write(buf, binary.LittleEndian, val)
	}
	return createCompressed(t, buf.Bytes())
}

func createCompressedInt32(t *testing.T, data []int32) []byte {
	t.Helper()
	buf := new(bytes.Buffer)
	for _, val := range data {
		binary.Write(buf, binary.LittleEndian, val)
	}
	return createCompressed(t, buf.Bytes())
}

// Tests for Decompress (general-purpose decompression)

func TestDecompress_Empty(t *testing.T) {
	_, err := Decompress([]byte{})
	if err == nil {
		t.Error("expected error on empty input")
	}
}

func TestDecompress_SmallData(t *testing.T) {
	original := []byte("hello world")
	compressed := createCompressed(t, original)

	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}

	if !bytes.Equal(decompressed, original) {
		t.Errorf("decompressed data mismatch:\nwant: %v\ngot:  %v", original, decompressed)
	}
}

func TestDecompress_LargeData(t *testing.T) {
	// Create 1MB of data
	original := make([]byte, 1024*1024)
	for i := range original {
		original[i] = byte(i % 256)
	}

	compressed := createCompressed(t, original)

	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}

	if !bytes.Equal(decompressed, original) {
		t.Error("decompressed large data mismatch")
	}
}

// Tests for DecompressInt64

func TestDecompressInt64_Basic(t *testing.T) {
	original := []int64{1, 2, 3, 4, 5, -1, -2, -3}
	compressed := createCompressedInt64(t, original)

	result, err := DecompressInt64(compressed)
	if err != nil {
		t.Fatalf("DecompressInt64 failed: %v", err)
	}

	if len(result) != len(original) {
		t.Fatalf("length mismatch: want %d, got %d", len(original), len(result))
	}

	for i, val := range result {
		if val != original[i] {
			t.Errorf("value mismatch at index %d: want %d, got %d", i, original[i], val)
		}
	}
}

func TestDecompressInt64_Empty(t *testing.T) {
	original := []int64{}
	compressed := createCompressedInt64(t, original)

	result, err := DecompressInt64(compressed)
	if err != nil {
		t.Fatalf("DecompressInt64 failed: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty slice, got length %d", len(result))
	}
}

func TestDecompressInt64_LargeArray(t *testing.T) {
	// 10,000 int64 values
	original := make([]int64, 10000)
	for i := range original {
		original[i] = int64(i * 1000)
	}

	compressed := createCompressedInt64(t, original)

	result, err := DecompressInt64(compressed)
	if err != nil {
		t.Fatalf("DecompressInt64 failed: %v", err)
	}

	if len(result) != len(original) {
		t.Fatalf("length mismatch: want %d, got %d", len(original), len(result))
	}

	for i, val := range result {
		if val != original[i] {
			t.Errorf("value mismatch at index %d: want %d, got %d", i, original[i], val)
		}
	}
}

func TestDecompressInt64_AlignmentError(t *testing.T) {
	// Create data that's not a multiple of 8 bytes (should fail)
	rawBytes := []byte{1, 2, 3, 4, 5} // 5 bytes, not multiple of 8
	compressed := createCompressed(t, rawBytes)

	_, err := DecompressInt64(compressed)
	if err == nil {
		t.Error("expected error on misaligned data")
	}
}

// Tests for DecompressFloat64

func TestDecompressFloat64_Basic(t *testing.T) {
	original := []float64{1.1, 2.2, 3.3, -4.4, -5.5}
	compressed := createCompressedFloat64(t, original)

	result, err := DecompressFloat64(compressed)
	if err != nil {
		t.Fatalf("DecompressFloat64 failed: %v", err)
	}

	if len(result) != len(original) {
		t.Fatalf("length mismatch: want %d, got %d", len(original), len(result))
	}

	for i, val := range result {
		if val != original[i] {
			t.Errorf("value mismatch at index %d: want %f, got %f", i, original[i], val)
		}
	}
}

func TestDecompressFloat64_SpecialValues(t *testing.T) {
	// Test special float64 values
	original := []float64{
		0.0,
		-0.0,
		1.7976931348623157e+308, // Max float64
		2.2250738585072014e-308, // Min positive float64
	}

	compressed := createCompressedFloat64(t, original)

	result, err := DecompressFloat64(compressed)
	if err != nil {
		t.Fatalf("DecompressFloat64 failed: %v", err)
	}

	if len(result) != len(original) {
		t.Fatalf("length mismatch: want %d, got %d", len(original), len(result))
	}

	for i, val := range result {
		if val != original[i] {
			t.Errorf("value mismatch at index %d: want %e, got %e", i, original[i], val)
		}
	}
}

// Tests for DecompressInt32

func TestDecompressInt32_Basic(t *testing.T) {
	original := []int32{100, 200, -300, 400, -500}
	compressed := createCompressedInt32(t, original)

	result, err := DecompressInt32(compressed)
	if err != nil {
		t.Fatalf("DecompressInt32 failed: %v", err)
	}

	if len(result) != len(original) {
		t.Fatalf("length mismatch: want %d, got %d", len(original), len(result))
	}

	for i, val := range result {
		if val != original[i] {
			t.Errorf("value mismatch at index %d: want %d, got %d", i, original[i], val)
		}
	}
}

// Tests for DecompressUint64

func TestDecompressUint64_Basic(t *testing.T) {
	original := []uint64{0, 1, 100, 1000, 18446744073709551615} // max uint64

	buf := new(bytes.Buffer)
	for _, val := range original {
		binary.Write(buf, binary.LittleEndian, val)
	}
	compressed := createCompressed(t, buf.Bytes())

	result, err := DecompressUint64(compressed)
	if err != nil {
		t.Fatalf("DecompressUint64 failed: %v", err)
	}

	if len(result) != len(original) {
		t.Fatalf("length mismatch: want %d, got %d", len(original), len(result))
	}

	for i, val := range result {
		if val != original[i] {
			t.Errorf("value mismatch at index %d: want %d, got %d", i, original[i], val)
		}
	}
}

// Tests for DecompressInt8

func TestDecompressInt8_Basic(t *testing.T) {
	original := []int8{-128, -1, 0, 1, 127} // Min and max int8

	buf := make([]byte, len(original))
	for i, val := range original {
		buf[i] = byte(val)
	}
	compressed := createCompressed(t, buf)

	result, err := DecompressInt8(compressed)
	if err != nil {
		t.Fatalf("DecompressInt8 failed: %v", err)
	}

	if len(result) != len(original) {
		t.Fatalf("length mismatch: want %d, got %d", len(original), len(result))
	}

	for i, val := range result {
		if val != original[i] {
			t.Errorf("value mismatch at index %d: want %d, got %d", i, original[i], val)
		}
	}
}

// Tests for DecompressInt16

func TestDecompressInt16_Basic(t *testing.T) {
	original := []int16{-32768, -100, 0, 100, 32767} // Min and max int16

	buf := new(bytes.Buffer)
	for _, val := range original {
		binary.Write(buf, binary.LittleEndian, val)
	}
	compressed := createCompressed(t, buf.Bytes())

	result, err := DecompressInt16(compressed)
	if err != nil {
		t.Fatalf("DecompressInt16 failed: %v", err)
	}

	if len(result) != len(original) {
		t.Fatalf("length mismatch: want %d, got %d", len(original), len(result))
	}

	for i, val := range result {
		if val != original[i] {
			t.Errorf("value mismatch at index %d: want %d, got %d", i, original[i], val)
		}
	}
}

// Tests for DecompressUint8

func TestDecompressUint8_Basic(t *testing.T) {
	original := []uint8{0, 1, 128, 255} // Min and max uint8

	compressed := createCompressed(t, original)

	result, err := DecompressUint8(compressed)
	if err != nil {
		t.Fatalf("DecompressUint8 failed: %v", err)
	}

	if len(result) != len(original) {
		t.Fatalf("length mismatch: want %d, got %d", len(original), len(result))
	}

	for i, val := range result {
		if val != original[i] {
			t.Errorf("value mismatch at index %d: want %d, got %d", i, original[i], val)
		}
	}
}

// Tests for DecompressUint16

func TestDecompressUint16_Basic(t *testing.T) {
	original := []uint16{0, 100, 32768, 65535} // Min and max uint16

	buf := new(bytes.Buffer)
	for _, val := range original {
		binary.Write(buf, binary.LittleEndian, val)
	}
	compressed := createCompressed(t, buf.Bytes())

	result, err := DecompressUint16(compressed)
	if err != nil {
		t.Fatalf("DecompressUint16 failed: %v", err)
	}

	if len(result) != len(original) {
		t.Fatalf("length mismatch: want %d, got %d", len(original), len(result))
	}

	for i, val := range result {
		if val != original[i] {
			t.Errorf("value mismatch at index %d: want %d, got %d", i, original[i], val)
		}
	}
}

// Tests for DecompressUint32

func TestDecompressUint32_Basic(t *testing.T) {
	original := []uint32{0, 1000, 2147483648, 4294967295} // Max uint32

	buf := new(bytes.Buffer)
	for _, val := range original {
		binary.Write(buf, binary.LittleEndian, val)
	}
	compressed := createCompressed(t, buf.Bytes())

	result, err := DecompressUint32(compressed)
	if err != nil {
		t.Fatalf("DecompressUint32 failed: %v", err)
	}

	if len(result) != len(original) {
		t.Fatalf("length mismatch: want %d, got %d", len(original), len(result))
	}

	for i, val := range result {
		if val != original[i] {
			t.Errorf("value mismatch at index %d: want %d, got %d", i, original[i], val)
		}
	}
}

// Tests for DecompressFloat32

func TestDecompressFloat32_Basic(t *testing.T) {
	original := []float32{1.1, 2.2, -3.3, 0.0, -0.0}

	buf := new(bytes.Buffer)
	for _, val := range original {
		binary.Write(buf, binary.LittleEndian, val)
	}
	compressed := createCompressed(t, buf.Bytes())

	result, err := DecompressFloat32(compressed)
	if err != nil {
		t.Fatalf("DecompressFloat32 failed: %v", err)
	}

	if len(result) != len(original) {
		t.Fatalf("length mismatch: want %d, got %d", len(original), len(result))
	}

	for i, val := range result {
		if val != original[i] {
			t.Errorf("value mismatch at index %d: want %f, got %f", i, original[i], val)
		}
	}
}

// Benchmarks

func BenchmarkDecompressInt64(b *testing.B) {
	// Create test data
	original := make([]int64, 10000)
	for i := range original {
		original[i] = int64(i)
	}

	compressed := createCompressedInt64(&testing.T{}, original)

	b.ResetTimer()
	b.SetBytes(int64(len(original) * 8)) // 8 bytes per int64

	for i := 0; i < b.N; i++ {
		_, err := DecompressInt64(compressed)
		if err != nil {
			b.Fatalf("DecompressInt64 failed: %v", err)
		}
	}
}

func BenchmarkDecompressFloat64(b *testing.B) {
	original := make([]float64, 10000)
	for i := range original {
		original[i] = float64(i) * 1.1
	}

	compressed := createCompressedFloat64(&testing.T{}, original)

	b.ResetTimer()
	b.SetBytes(int64(len(original) * 8))

	for i := 0; i < b.N; i++ {
		_, err := DecompressFloat64(compressed)
		if err != nil {
			b.Fatalf("DecompressFloat64 failed: %v", err)
		}
	}
}
