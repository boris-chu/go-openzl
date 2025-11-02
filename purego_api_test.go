//go:build !cgo
// +build !cgo

// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package openzl

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/boris-chu/go-openzl/internal/codec"
	"github.com/boris-chu/go-openzl/internal/frame"
	"github.com/boris-chu/go-openzl/internal/graph"
)

// TestPureGoDecompress verifies that Decompress works in Pure Go mode.
func TestPureGoDecompress(t *testing.T) {
	// Use test data compressed with CGO (from fixtures or pre-generated)
	// For this test, we'll use purgo to create compressed data
	original := []byte("hello world from Pure Go")

	// Create compressed data using purgo helper
	compressed := createCompressed(t, original)

	// Test Decompress (should use Pure Go implementation)
	result, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}

	if string(result) != string(original) {
		t.Errorf("Decompress mismatch:\nwant: %q\ngot:  %q", original, result)
	}
}

// TestPureGoDecompressNumeric verifies that DecompressNumeric works in Pure Go mode.
func TestPureGoDecompressNumeric(t *testing.T) {
	original := []int64{1, 2, 3, 4, 5, 100, 101, 102}

	// Create compressed data
	compressed := createCompressedInt64(t, original)

	// Test DecompressNumeric (should use Pure Go implementation)
	result, err := DecompressNumeric[int64](compressed)
	if err != nil {
		t.Fatalf("DecompressNumeric failed: %v", err)
	}

	if len(result) != len(original) {
		t.Fatalf("length mismatch: got %d, want %d", len(result), len(original))
	}

	for i := range original {
		if result[i] != original[i] {
			t.Errorf("mismatch at index %d: got %d, want %d", i, result[i], original[i])
		}
	}
}

// TestPureGoCompressionWorks verifies that compression works in Pure Go mode.
func TestPureGoCompressionWorks(t *testing.T) {
	// Test Compress - should work now!
	data := []byte("test data for compression")
	compressed, err := Compress(data)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}
	if len(compressed) == 0 {
		t.Error("Compress returned empty data")
	}

	// Verify roundtrip
	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}
	if string(decompressed) != string(data) {
		t.Error("Roundtrip mismatch")
	}

	// Test CompressNumeric - should work now!
	numbers := []int64{1, 2, 3, 4, 5}
	compressedNums, err := CompressNumeric(numbers)
	if err != nil {
		t.Fatalf("CompressNumeric failed: %v", err)
	}
	if len(compressedNums) == 0 {
		t.Error("CompressNumeric returned empty data")
	}

	// Verify roundtrip
	decompressedNums, err := DecompressNumeric[int64](compressedNums)
	if err != nil {
		t.Fatalf("DecompressNumeric failed: %v", err)
	}
	if len(decompressedNums) != len(numbers) {
		t.Error("Numeric roundtrip length mismatch")
	}

	// Test NewCompressor - should still return error (context API not available)
	_, err = NewCompressor()
	if err == nil {
		t.Error("NewCompressor should return error in Pure Go mode")
	}

	// Test NewWriter - should still return error (streaming writer not available)
	_, err = NewWriter(nil)
	if err == nil {
		t.Error("NewWriter should return error in Pure Go mode")
	}
}

// Helper functions (same as purgo tests)

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
	if len(f.Outputs) > 0 {
		writeVarintTest(frameBuf, f.Outputs[0].DecompressedSize+1)
	}

	// Write payload
	frameBuf.Write(f.Payload)

	return frameBuf.Bytes()
}

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

func createCompressedInt64(t *testing.T, data []int64) []byte {
	t.Helper()
	buf := new(bytes.Buffer)
	for _, val := range data {
		binary.Write(buf, binary.LittleEndian, val)
	}
	return createCompressed(t, buf.Bytes())
}
