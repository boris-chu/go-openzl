package frame

import (
	"bytes"
	"testing"
)

// TestWriteFrameV21Roundtrip tests v21 frame encoding/decoding
func TestWriteFrameV21Roundtrip(t *testing.T) {
	// Create a simple v21 frame
	original := &Frame{
		Header: &Header{
			Version: 21,
			Flags:   0,
		},
		Outputs: []*Output{
			{
				Type:             TypeSerial,
				DecompressedSize: 1000,
				NumElements:      1000,
			},
		},
		NodeSizes: nil,                            // v21 has no node sizes
		Payload:   []byte{0x01, 0x02, 0x03, 0x04}, // dummy payload
	}

	// Encode
	encoded, err := EncodeFrame(original)
	if err != nil {
		t.Fatalf("EncodeFrame failed: %v", err)
	}

	t.Logf("Encoded v21 frame: %d bytes", len(encoded))

	// Decode
	decoded, err := Parse(encoded)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify header
	if decoded.Header.Version != original.Header.Version {
		t.Errorf("Version: got %d, want %d", decoded.Header.Version, original.Header.Version)
	}

	// Verify outputs
	if len(decoded.Outputs) != len(original.Outputs) {
		t.Errorf("Output count: got %d, want %d", len(decoded.Outputs), len(original.Outputs))
	}

	if decoded.Outputs[0].DecompressedSize != original.Outputs[0].DecompressedSize {
		t.Errorf("Output size: got %d, want %d",
			decoded.Outputs[0].DecompressedSize, original.Outputs[0].DecompressedSize)
	}

	// Verify payload
	if !bytes.Equal(decoded.Payload, original.Payload) {
		t.Errorf("Payload mismatch")
	}

	// v21 should have no node sizes
	if decoded.NodeSizes != nil {
		t.Errorf("v21 frame should have nil NodeSizes, got %v", decoded.NodeSizes)
	}

	t.Logf("✅ v21 roundtrip successful")
}

// TestWriteFrameV22Roundtrip tests v22 frame encoding/decoding
func TestWriteFrameV22Roundtrip(t *testing.T) {
	// Create a v22 frame with node sizes
	original := &Frame{
		Header: &Header{
			Version: 22,
			Flags:   0,
		},
		Outputs: []*Output{
			{
				Type:             TypeSerial,
				DecompressedSize: 1000,
				NumElements:      1000,
			},
		},
		NodeSizes: []uint64{5000, 2500, 1000}, // 3 nodes in pipeline
		Payload:   []byte{0x01, 0x02, 0x03, 0x04},
	}

	// Encode
	encoded, err := EncodeFrame(original)
	if err != nil {
		t.Fatalf("EncodeFrame failed: %v", err)
	}

	t.Logf("Encoded v22 frame: %d bytes (with %d node sizes)",
		len(encoded), len(original.NodeSizes))

	// Decode
	decoded, err := Parse(encoded)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify header
	if decoded.Header.Version != original.Header.Version {
		t.Errorf("Version: got %d, want %d", decoded.Header.Version, original.Header.Version)
	}

	// Verify outputs
	if len(decoded.Outputs) != len(original.Outputs) {
		t.Errorf("Output count: got %d, want %d", len(decoded.Outputs), len(original.Outputs))
	}

	if decoded.Outputs[0].DecompressedSize != original.Outputs[0].DecompressedSize {
		t.Errorf("Output size: got %d, want %d",
			decoded.Outputs[0].DecompressedSize, original.Outputs[0].DecompressedSize)
	}

	// Verify node sizes
	if len(decoded.NodeSizes) != len(original.NodeSizes) {
		t.Errorf("Node size count: got %d, want %d",
			len(decoded.NodeSizes), len(original.NodeSizes))
	}

	for i, size := range original.NodeSizes {
		if decoded.NodeSizes[i] != size {
			t.Errorf("NodeSizes[%d]: got %d, want %d", i, decoded.NodeSizes[i], size)
		}
	}

	// Verify payload
	if !bytes.Equal(decoded.Payload, original.Payload) {
		t.Errorf("Payload mismatch")
	}

	t.Logf("✅ v22 roundtrip successful with node sizes: %v", decoded.NodeSizes)
}

// TestWriteFrameV22WithoutNodeSizes tests that v22 requires node sizes
func TestWriteFrameV22WithoutNodeSizes(t *testing.T) {
	frame := &Frame{
		Header: &Header{
			Version: 22,
			Flags:   0,
		},
		Outputs: []*Output{
			{Type: TypeSerial, DecompressedSize: 1000},
		},
		NodeSizes: nil, // Missing node sizes!
		Payload:   []byte{0x01},
	}

	var buf bytes.Buffer
	_, err := WriteFrameV22(&buf, frame)
	if err == nil {
		t.Error("Expected error for v22 frame without NodeSizes, got nil")
	}

	t.Logf("✅ Correctly rejected v22 frame without NodeSizes: %v", err)
}

// TestWriteFrameMultiOutput tests frames with 2 outputs
func TestWriteFrameMultiOutput(t *testing.T) {
	frame := &Frame{
		Header: &Header{
			Version: 22,
			Flags:   0,
		},
		Outputs: []*Output{
			{Type: TypeSerial, DecompressedSize: 1000, NumElements: 1000},
			{Type: TypeNumeric, DecompressedSize: 2000, NumElements: 2000},
		},
		NodeSizes: []uint64{5000, 3000, 2000, 1000},
		Payload:   []byte{0xAA, 0xBB},
	}

	// Encode
	encoded, err := EncodeFrame(frame)
	if err != nil {
		t.Fatalf("EncodeFrame failed: %v", err)
	}

	// Decode
	decoded, err := Parse(encoded)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify both outputs
	if len(decoded.Outputs) != 2 {
		t.Fatalf("Output count: got %d, want 2", len(decoded.Outputs))
	}

	if decoded.Outputs[0].DecompressedSize != 1000 {
		t.Errorf("Output 0 size: got %d, want 1000", decoded.Outputs[0].DecompressedSize)
	}

	if decoded.Outputs[1].DecompressedSize != 2000 {
		t.Errorf("Output 1 size: got %d, want 2000", decoded.Outputs[1].DecompressedSize)
	}

	// Verify node sizes
	if len(decoded.NodeSizes) != 4 {
		t.Errorf("Node size count: got %d, want 4", len(decoded.NodeSizes))
	}

	t.Logf("✅ Multi-output frame roundtrip successful")
}

// TestFrameSizeEstimate tests frame size estimation
func TestFrameSizeEstimate(t *testing.T) {
	frame := &Frame{
		Header: &Header{
			Version: 22,
			Flags:   0,
		},
		Outputs: []*Output{
			{Type: TypeSerial, DecompressedSize: 1000},
		},
		NodeSizes: []uint64{5000, 2500, 1000},
		Payload:   make([]byte, 100),
	}

	// Estimate size
	estimated := FrameSize(frame)

	// Actual size
	encoded, err := EncodeFrame(frame)
	if err != nil {
		t.Fatalf("EncodeFrame failed: %v", err)
	}
	actual := len(encoded)

	t.Logf("Estimated size: %d bytes", estimated)
	t.Logf("Actual size: %d bytes", actual)

	// Estimate should be close (within 20 bytes due to varint variations)
	diff := estimated - actual
	if diff < 0 {
		diff = -diff
	}

	if diff > 20 {
		t.Errorf("Size estimate off by %d bytes (estimated %d, actual %d)",
			diff, estimated, actual)
	}

	t.Logf("✅ Size estimate within %d bytes", diff)
}

// TestWriteFrameV21ForcesVersion tests that WriteFrameV21 forces version 21
func TestWriteFrameV21ForcesVersion(t *testing.T) {
	frame := &Frame{
		Header: &Header{
			Version: 22, // Try to write v22
			Flags:   0,
		},
		Outputs: []*Output{
			{Type: TypeSerial, DecompressedSize: 1000},
		},
		NodeSizes: []uint64{5000, 1000}, // This should be ignored
		Payload:   []byte{0x01},
	}

	// Write as v21
	var buf bytes.Buffer
	_, err := WriteFrameV21(&buf, frame)
	if err != nil {
		t.Fatalf("WriteFrameV21 failed: %v", err)
	}

	// Decode
	decoded, err := Parse(buf.Bytes())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should be v21, not v22
	if decoded.Header.Version != 21 {
		t.Errorf("Version: got %d, want 21", decoded.Header.Version)
	}

	// Should not have node sizes
	if decoded.NodeSizes != nil {
		t.Errorf("v21 frame should have nil NodeSizes")
	}

	t.Logf("✅ WriteFrameV21 correctly forces version 21")
}

// TestWriteFrameEmptyPayload tests frames with empty payload
func TestWriteFrameEmptyPayload(t *testing.T) {
	frame := &Frame{
		Header: &Header{
			Version: 22,
			Flags:   0,
		},
		Outputs: []*Output{
			{Type: TypeSerial, DecompressedSize: 0},
		},
		NodeSizes: []uint64{0},
		Payload:   []byte{}, // Empty payload
	}

	// Encode
	encoded, err := EncodeFrame(frame)
	if err != nil {
		t.Fatalf("EncodeFrame failed: %v", err)
	}

	// Decode
	decoded, err := Parse(encoded)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify
	if len(decoded.Payload) != 0 {
		t.Errorf("Payload length: got %d, want 0", len(decoded.Payload))
	}

	t.Logf("✅ Empty payload frame roundtrip successful")
}

// TestWriteFrameLargeNodeSizes tests frames with many nodes
func TestWriteFrameLargeNodeSizes(t *testing.T) {
	// Create frame with 10 nodes
	nodeSizes := make([]uint64, 10)
	for i := range nodeSizes {
		nodeSizes[i] = uint64((10 - i) * 1000) // Decreasing sizes
	}

	frame := &Frame{
		Header: &Header{
			Version: 22,
			Flags:   0,
		},
		Outputs: []*Output{
			{Type: TypeSerial, DecompressedSize: 1000},
		},
		NodeSizes: nodeSizes,
		Payload:   []byte{0xFF},
	}

	// Encode
	encoded, err := EncodeFrame(frame)
	if err != nil {
		t.Fatalf("EncodeFrame failed: %v", err)
	}

	t.Logf("Encoded frame with 10 nodes: %d bytes", len(encoded))

	// Decode
	decoded, err := Parse(encoded)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify node sizes
	if len(decoded.NodeSizes) != 10 {
		t.Fatalf("Node count: got %d, want 10", len(decoded.NodeSizes))
	}

	for i, size := range nodeSizes {
		if decoded.NodeSizes[i] != size {
			t.Errorf("NodeSizes[%d]: got %d, want %d", i, decoded.NodeSizes[i], size)
		}
	}

	t.Logf("✅ Large node sizes frame roundtrip successful")
}
