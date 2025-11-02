package graph

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/borischu/go-openzl/internal/codec"
	"github.com/borischu/go-openzl/internal/frame"
)

// TestEndToEnd_IdentityCodec tests complete decompression flow with Identity codec
func TestEndToEnd_IdentityCodec(t *testing.T) {
	// Original data
	original := []byte("Hello, OpenZL!")

	// 1. Create compression graph (Identity codec - no transformation)
	graph := &Graph{
		Nodes: []*Node{
			{
				CodecID: codec.IDIdentity,
				Params:  nil,
				Inputs:  nil, // Leaf node
			},
		},
		Outputs: []int{0},
	}

	// 2. Encode graph to bytes
	graphBytes, err := EncodeGraph(graph)
	if err != nil {
		t.Fatalf("EncodeGraph() error = %v", err)
	}

	// 3. Create frame payload (graph + compressed data)
	// For Identity codec, "compressed" data is same as original
	payload := append(graphBytes, original...)

	// 4. Create frame (simulate frame parsing)
	testFrame := &frame.Frame{
		Header: &frame.Header{
			Magic:   frame.MagicNumberBase + 21,
			Version: 21,
			Flags:   0,
		},
		Outputs: []*frame.Output{
			{
				Type:             frame.TypeSerial,
				DecompressedSize: uint64(len(original)),
			},
		},
		Payload: payload,
	}

	// 5. Parse graph from payload
	parser := NewParser(testFrame.Payload)
	parsedGraph, graphSize, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	t.Logf("Graph size: %d bytes", graphSize)
	t.Logf("Graph: %s", parsedGraph)

	// 6. Extract compressed data (after graph)
	compressedData := testFrame.Payload[graphSize:]
	t.Logf("Compressed data: %d bytes", len(compressedData))

	// 7. Execute graph to decompress
	executor := DefaultExecutor()
	outputSizes := []uint64{testFrame.Outputs[0].DecompressedSize}
	outputs, err := executor.Execute(parsedGraph, compressedData, outputSizes)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// 8. Verify decompressed data matches original
	if len(outputs) != 1 {
		t.Fatalf("Execute() returned %d outputs, want 1", len(outputs))
	}

	decompressed := outputs[0]
	if !bytes.Equal(decompressed, original) {
		t.Errorf("Decompressed data mismatch:\nGot:  %q\nWant: %q", decompressed, original)
	}

	t.Logf("✅ End-to-end decompression successful!")
	t.Logf("   Original: %q (%d bytes)", original, len(original))
	t.Logf("   Decompressed: %q (%d bytes)", decompressed, len(decompressed))
}

// TestEndToEnd_EmptyData tests decompressing empty data
func TestEndToEnd_EmptyData(t *testing.T) {
	original := []byte{}

	graph := &Graph{
		Nodes:   []*Node{{CodecID: codec.IDIdentity, Inputs: nil}},
		Outputs: []int{0},
	}

	graphBytes, err := EncodeGraph(graph)
	if err != nil {
		t.Fatalf("EncodeGraph() error = %v", err)
	}

	payload := append(graphBytes, original...)

	testFrame := &frame.Frame{
		Header: &frame.Header{
			Magic:   frame.MagicNumberBase + 21,
			Version: 21,
		},
		Outputs: []*frame.Output{
			{Type: frame.TypeSerial, DecompressedSize: 0},
		},
		Payload: payload,
	}

	parsedGraph, graphSize, err := NewParser(testFrame.Payload).Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	compressedData := testFrame.Payload[graphSize:]
	executor := DefaultExecutor()
	outputs, err := executor.Execute(parsedGraph, compressedData, []uint64{0})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(outputs[0]) != 0 {
		t.Errorf("Decompressed empty data has %d bytes, want 0", len(outputs[0]))
	}
}

// TestEndToEnd_LargeData tests decompressing larger data
func TestEndToEnd_LargeData(t *testing.T) {
	// 1MB data
	original := bytes.Repeat([]byte("OpenZL compression test data. "), 35000)

	graph := &Graph{
		Nodes:   []*Node{{CodecID: codec.IDIdentity, Inputs: nil}},
		Outputs: []int{0},
	}

	graphBytes, err := EncodeGraph(graph)
	if err != nil {
		t.Fatalf("EncodeGraph() error = %v", err)
	}

	payload := append(graphBytes, original...)

	testFrame := &frame.Frame{
		Header: &frame.Header{
			Magic:   frame.MagicNumberBase + 21,
			Version: 21,
		},
		Outputs: []*frame.Output{
			{Type: frame.TypeSerial, DecompressedSize: uint64(len(original))},
		},
		Payload: payload,
	}

	parsedGraph, graphSize, err := NewParser(testFrame.Payload).Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	compressedData := testFrame.Payload[graphSize:]
	executor := DefaultExecutor()
	outputs, err := executor.Execute(parsedGraph, compressedData, []uint64{uint64(len(original))})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !bytes.Equal(outputs[0], original) {
		t.Error("Large data decompression mismatch")
	}

	t.Logf("✅ Large data test: %d bytes decompressed successfully", len(original))
}

// TestEndToEnd_TwoNodePipeline tests a two-node decompression pipeline
func TestEndToEnd_TwoNodePipeline(t *testing.T) {
	original := []byte("Pipeline test data")

	// Graph: Identity → Identity (two-stage pipeline)
	graph := &Graph{
		Nodes: []*Node{
			{CodecID: codec.IDIdentity, Inputs: nil},       // Stage 1
			{CodecID: codec.IDIdentity, Inputs: []int{0}},  // Stage 2 (uses output of stage 1)
		},
		Outputs: []int{1}, // Final output is from stage 2
	}

	graphBytes, err := EncodeGraph(graph)
	if err != nil {
		t.Fatalf("EncodeGraph() error = %v", err)
	}

	payload := append(graphBytes, original...)

	testFrame := &frame.Frame{
		Header: &frame.Header{
			Magic:   frame.MagicNumberBase + 21,
			Version: 21,
		},
		Outputs: []*frame.Output{
			{Type: frame.TypeSerial, DecompressedSize: uint64(len(original))},
		},
		Payload: payload,
	}

	parsedGraph, graphSize, err := NewParser(testFrame.Payload).Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	compressedData := testFrame.Payload[graphSize:]
	executor := DefaultExecutor()
	outputs, err := executor.Execute(parsedGraph, compressedData, []uint64{uint64(len(original))})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !bytes.Equal(outputs[0], original) {
		t.Errorf("Pipeline decompression mismatch:\nGot:  %q\nWant: %q", outputs[0], original)
	}

	t.Logf("✅ Two-node pipeline: %q decompressed successfully", original)
}

// TestEndToEnd_MultipleOutputs tests graphs with multiple outputs
func TestEndToEnd_MultipleOutputs(t *testing.T) {
	data1 := []byte("Output 1")

	// Graph with two independent outputs
	graph := &Graph{
		Nodes: []*Node{
			{CodecID: codec.IDIdentity, Inputs: nil},
			{CodecID: codec.IDIdentity, Inputs: nil},
		},
		Outputs: []int{0, 1},
	}

	graphBytes, err := EncodeGraph(graph)
	if err != nil {
		t.Fatalf("EncodeGraph() error = %v", err)
	}

	// For multi-output, both nodes read from same payload
	// (This is simplified; real multi-output would split the payload)
	payload := append(graphBytes, data1...)

	testFrame := &frame.Frame{
		Header: &frame.Header{
			Magic:   frame.MagicNumberBase + 21,
			Version: 21,
		},
		Outputs: []*frame.Output{
			{Type: frame.TypeSerial, DecompressedSize: uint64(len(data1))},
			{Type: frame.TypeSerial, DecompressedSize: uint64(len(data1))},
		},
		Payload: payload,
	}

	parsedGraph, graphSize, err := NewParser(testFrame.Payload).Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	compressedData := testFrame.Payload[graphSize:]
	executor := DefaultExecutor()
	outputs, err := executor.Execute(parsedGraph, compressedData, []uint64{uint64(len(data1)), uint64(len(data1))})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(outputs) != 2 {
		t.Fatalf("Execute() returned %d outputs, want 2", len(outputs))
	}

	// Both outputs should be same (both read same data with Identity)
	if !bytes.Equal(outputs[0], data1) {
		t.Errorf("Output 0 mismatch")
	}

	if !bytes.Equal(outputs[1], data1) {
		t.Errorf("Output 1 mismatch")
	}

	t.Logf("✅ Multiple outputs test successful")
}

// TestEndToEnd_DeltaCodec tests complete decompression with Delta codec
func TestEndToEnd_DeltaCodec(t *testing.T) {
	// Create time series data (timestamps increasing by ~1000ms)
	timestamps := []uint64{
		1700000000000,
		1700000001000,
		1700000002000,
		1700000003000,
		1700000004000,
	}

	// Encode timestamps as bytes (little-endian uint64)
	original := make([]byte, len(timestamps)*8)
	for i, ts := range timestamps {
		binary.LittleEndian.PutUint64(original[i*8:], ts)
	}

	// Create graph: Delta codec (8-byte elements)
	graph := &Graph{
		Nodes: []*Node{
			{
				CodecID: codec.IDDelta,
				Params:  []byte{8}, // 8-byte (uint64) elements
				Inputs:  nil,        // Leaf node
			},
		},
		Outputs: []int{0},
	}

	// Encode graph
	graphBytes, err := EncodeGraph(graph)
	if err != nil {
		t.Fatalf("EncodeGraph() error = %v", err)
	}

	// Delta-encode the timestamps manually for testing
	deltaEncoded := make([]byte, len(original))
	var prev uint64
	for i := 0; i < len(timestamps); i++ {
		value := timestamps[i]
		delta := value - prev
		binary.LittleEndian.PutUint64(deltaEncoded[i*8:], delta)
		prev = value
	}

	// Create frame payload (graph + delta-encoded data)
	payload := append(graphBytes, deltaEncoded...)

	testFrame := &frame.Frame{
		Header: &frame.Header{
			Magic:   frame.MagicNumberBase + 21,
			Version: 21,
		},
		Outputs: []*frame.Output{
			{
				Type:             frame.TypeNumeric,
				DecompressedSize: uint64(len(original)),
			},
		},
		Payload: payload,
	}

	// Parse graph
	parsedGraph, graphSize, err := NewParser(testFrame.Payload).Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	t.Logf("Graph size: %d bytes", graphSize)
	t.Logf("Graph: %s", parsedGraph)

	// Execute graph to decompress
	compressedData := testFrame.Payload[graphSize:]
	executor := DefaultExecutor()
	outputs, err := executor.Execute(parsedGraph, compressedData, []uint64{uint64(len(original))})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Verify decompressed data matches original
	decompressed := outputs[0]
	if !bytes.Equal(decompressed, original) {
		t.Errorf("Delta decompression mismatch")

		// Debug: show what we got
		for i := 0; i < len(timestamps); i++ {
			got := binary.LittleEndian.Uint64(decompressed[i*8:])
			want := timestamps[i]
			if got != want {
				t.Errorf("  Timestamp[%d]: got %d, want %d", i, got, want)
			}
		}
	}

	t.Logf("✅ Delta codec end-to-end decompression successful!")
	t.Logf("   Compressed delta values, decompressed to %d timestamps", len(timestamps))

	// Show compression benefit
	t.Logf("   Original timestamps: %v", timestamps)
	deltas := make([]uint64, len(timestamps))
	var p uint64
	for i, ts := range timestamps {
		deltas[i] = ts - p
		p = ts
	}
	t.Logf("   Delta values: %v (smaller numbers = better compression!)", deltas)
}

// BenchmarkEndToEnd benchmarks the complete decompression flow
func BenchmarkEndToEnd(b *testing.B) {
	original := []byte("Benchmark test data for end-to-end decompression")

	graph := &Graph{
		Nodes:   []*Node{{CodecID: codec.IDIdentity, Inputs: nil}},
		Outputs: []int{0},
	}

	graphBytes, _ := EncodeGraph(graph)
	payload := append(graphBytes, original...)

	testFrame := &frame.Frame{
		Header: &frame.Header{
			Magic:   frame.MagicNumberBase + 21,
			Version: 21,
		},
		Outputs: []*frame.Output{
			{Type: frame.TypeSerial, DecompressedSize: uint64(len(original))},
		},
		Payload: payload,
	}

	b.SetBytes(int64(len(original)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		parsedGraph, graphSize, _ := NewParser(testFrame.Payload).Parse()
		compressedData := testFrame.Payload[graphSize:]
		executor := DefaultExecutor()
		_, err := executor.Execute(parsedGraph, compressedData, []uint64{uint64(len(original))})
		if err != nil {
			b.Fatal(err)
		}
	}
}
