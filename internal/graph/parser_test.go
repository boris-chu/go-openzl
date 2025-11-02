package graph

import (
	"bytes"
	"testing"

	"github.com/borischu/go-openzl/internal/codec"
)

// TestEncodeGraph_Simple tests encoding a simple graph
func TestEncodeGraph_Simple(t *testing.T) {
	graph := &Graph{
		Nodes: []*Node{
			{
				CodecID: codec.IDIdentity,
				Params:  nil,
				Inputs:  nil,
			},
		},
		Outputs: []int{0},
	}

	data, err := EncodeGraph(graph)
	if err != nil {
		t.Fatalf("EncodeGraph() error = %v", err)
	}

	if len(data) == 0 {
		t.Error("EncodeGraph() returned empty data")
	}

	// Should contain: nbNodes=1, codecID=0, nbParams=0, nbInputs=0, nbOutputs=1, output=0
	// Minimum size: 1 + 1 + 1 + 1 + 1 + 1 = 6 bytes (all varints of 1 byte)
	if len(data) < 6 {
		t.Errorf("EncodeGraph() returned %d bytes, expected at least 6", len(data))
	}
}

// TestEncodeGraph_WithParams tests encoding a graph with parameters
func TestEncodeGraph_WithParams(t *testing.T) {
	graph := &Graph{
		Nodes: []*Node{
			{
				CodecID: codec.IDDelta,
				Params:  []byte{0x08}, // 8-byte integers
				Inputs:  nil,
			},
		},
		Outputs: []int{0},
	}

	data, err := EncodeGraph(graph)
	if err != nil {
		t.Fatalf("EncodeGraph() error = %v", err)
	}

	// Should include 1 byte of params
	if len(data) < 7 { // 6 + 1 param byte
		t.Errorf("EncodeGraph() returned %d bytes, expected at least 7", len(data))
	}
}

// TestEncodeGraph_TwoNodes tests encoding a two-node graph
func TestEncodeGraph_TwoNodes(t *testing.T) {
	graph := &Graph{
		Nodes: []*Node{
			{
				CodecID: codec.IDIdentity,
				Params:  nil,
				Inputs:  nil,
			},
			{
				CodecID: codec.IDDelta,
				Params:  nil,
				Inputs:  []int{0},
			},
		},
		Outputs: []int{1},
	}

	data, err := EncodeGraph(graph)
	if err != nil {
		t.Fatalf("EncodeGraph() error = %v", err)
	}

	// Node 0: codecID=0, nbParams=0, nbInputs=0 (3 bytes)
	// Node 1: codecID=2, nbParams=0, nbInputs=1, input=0 (4 bytes)
	// nbNodes=2, nbOutputs=1, output=1 (3 bytes)
	// Total: 10 bytes minimum
	if len(data) < 10 {
		t.Errorf("EncodeGraph() returned %d bytes, expected at least 10", len(data))
	}
}

// TestEncodeGraph_InvalidGraph tests error handling for invalid graphs
func TestEncodeGraph_InvalidGraph(t *testing.T) {
	tests := []struct {
		name  string
		graph *Graph
	}{
		{
			name: "nil graph",
			graph: nil,
		},
		{
			name: "no nodes",
			graph: &Graph{
				Nodes:   []*Node{},
				Outputs: []int{0},
			},
		},
		{
			name: "no outputs",
			graph: &Graph{
				Nodes: []*Node{
					{CodecID: codec.IDIdentity},
				},
				Outputs: []int{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EncodeGraph(tt.graph)
			if err == nil {
				t.Error("EncodeGraph() should error for invalid graph")
			}
		})
	}
}

// TestParseGraph_Simple tests parsing a simple graph
func TestParseGraph_Simple(t *testing.T) {
	// Create a simple graph
	original := &Graph{
		Nodes: []*Node{
			{
				CodecID: codec.IDIdentity,
				Params:  nil,
				Inputs:  nil,
			},
		},
		Outputs: []int{0},
	}

	// Encode it
	data, err := EncodeGraph(original)
	if err != nil {
		t.Fatalf("EncodeGraph() error = %v", err)
	}

	// Parse it back
	parser := NewParser(data)
	parsed, graphSize, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Verify graph structure
	if len(parsed.Nodes) != 1 {
		t.Errorf("Parse() got %d nodes, want 1", len(parsed.Nodes))
	}

	if len(parsed.Outputs) != 1 {
		t.Errorf("Parse() got %d outputs, want 1", len(parsed.Outputs))
	}

	if parsed.Outputs[0] != 0 {
		t.Errorf("Parse() output[0] = %d, want 0", parsed.Outputs[0])
	}

	if parsed.Nodes[0].CodecID != codec.IDIdentity {
		t.Errorf("Parse() node codec = %v, want %v", parsed.Nodes[0].CodecID, codec.IDIdentity)
	}

	if graphSize != len(data) {
		t.Errorf("Parse() graphSize = %d, want %d", graphSize, len(data))
	}
}

// TestParseGraph_WithParams tests parsing a graph with parameters
func TestParseGraph_WithParams(t *testing.T) {
	original := &Graph{
		Nodes: []*Node{
			{
				CodecID: codec.IDDelta,
				Params:  []byte{0x08, 0x00, 0x00},
				Inputs:  nil,
			},
		},
		Outputs: []int{0},
	}

	data, err := EncodeGraph(original)
	if err != nil {
		t.Fatalf("EncodeGraph() error = %v", err)
	}

	parser := NewParser(data)
	parsed, _, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if !bytes.Equal(parsed.Nodes[0].Params, original.Nodes[0].Params) {
		t.Errorf("Parse() params = %v, want %v", parsed.Nodes[0].Params, original.Nodes[0].Params)
	}
}

// TestParseGraph_TwoNodes tests parsing a two-node pipeline
func TestParseGraph_TwoNodes(t *testing.T) {
	original := &Graph{
		Nodes: []*Node{
			{
				CodecID: codec.IDIdentity,
				Params:  nil,
				Inputs:  nil,
			},
			{
				CodecID: codec.IDDelta,
				Params:  []byte{0x04},
				Inputs:  []int{0},
			},
		},
		Outputs: []int{1},
	}

	data, err := EncodeGraph(original)
	if err != nil {
		t.Fatalf("EncodeGraph() error = %v", err)
	}

	parser := NewParser(data)
	parsed, _, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(parsed.Nodes) != 2 {
		t.Fatalf("Parse() got %d nodes, want 2", len(parsed.Nodes))
	}

	// Check node 0
	if parsed.Nodes[0].CodecID != codec.IDIdentity {
		t.Errorf("Parse() node[0] codec = %v, want %v", parsed.Nodes[0].CodecID, codec.IDIdentity)
	}

	// Check node 1
	if parsed.Nodes[1].CodecID != codec.IDDelta {
		t.Errorf("Parse() node[1] codec = %v, want %v", parsed.Nodes[1].CodecID, codec.IDDelta)
	}

	if len(parsed.Nodes[1].Inputs) != 1 || parsed.Nodes[1].Inputs[0] != 0 {
		t.Errorf("Parse() node[1] inputs = %v, want [0]", parsed.Nodes[1].Inputs)
	}
}

// TestParseGraph_MultipleOutputs tests parsing graphs with multiple outputs
func TestParseGraph_MultipleOutputs(t *testing.T) {
	original := &Graph{
		Nodes: []*Node{
			{CodecID: codec.IDIdentity, Inputs: nil},
			{CodecID: codec.IDIdentity, Inputs: nil},
		},
		Outputs: []int{0, 1},
	}

	data, err := EncodeGraph(original)
	if err != nil {
		t.Fatalf("EncodeGraph() error = %v", err)
	}

	parser := NewParser(data)
	parsed, _, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(parsed.Outputs) != 2 {
		t.Fatalf("Parse() got %d outputs, want 2", len(parsed.Outputs))
	}

	if parsed.Outputs[0] != 0 || parsed.Outputs[1] != 1 {
		t.Errorf("Parse() outputs = %v, want [0 1]", parsed.Outputs)
	}
}

// TestParseGraph_Roundtrip tests encode→parse roundtrip
func TestParseGraph_Roundtrip(t *testing.T) {
	tests := []struct {
		name  string
		graph *Graph
	}{
		{
			name: "single node",
			graph: &Graph{
				Nodes:   []*Node{{CodecID: codec.IDIdentity, Inputs: nil}},
				Outputs: []int{0},
			},
		},
		{
			name: "two node pipeline",
			graph: &Graph{
				Nodes: []*Node{
					{CodecID: codec.IDIdentity, Inputs: nil},
					{CodecID: codec.IDDelta, Inputs: []int{0}},
				},
				Outputs: []int{1},
			},
		},
		{
			name: "with params",
			graph: &Graph{
				Nodes: []*Node{
					{CodecID: codec.IDDelta, Params: []byte{0x08}, Inputs: nil},
				},
				Outputs: []int{0},
			},
		},
		{
			name: "multiple outputs",
			graph: &Graph{
				Nodes: []*Node{
					{CodecID: codec.IDIdentity, Inputs: nil},
					{CodecID: codec.IDIdentity, Inputs: nil},
				},
				Outputs: []int{0, 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			data, err := EncodeGraph(tt.graph)
			if err != nil {
				t.Fatalf("EncodeGraph() error = %v", err)
			}

			// Parse
			parsed, _, err := NewParser(data).Parse()
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			// Verify structure matches
			if len(parsed.Nodes) != len(tt.graph.Nodes) {
				t.Errorf("node count mismatch: got %d, want %d", len(parsed.Nodes), len(tt.graph.Nodes))
			}

			if len(parsed.Outputs) != len(tt.graph.Outputs) {
				t.Errorf("output count mismatch: got %d, want %d", len(parsed.Outputs), len(tt.graph.Outputs))
			}
		})
	}
}

// TestParseGraph_Errors tests error handling in parser
func TestParseGraph_Errors(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
	}{
		{
			name:    "empty payload",
			payload: []byte{},
		},
		{
			name:    "truncated (no nodes)",
			payload: []byte{0x01}, // nbNodes=1 but no node data
		},
		{
			name:    "invalid varint",
			payload: []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, // Overflow
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.payload)
			_, _, err := parser.Parse()
			if err == nil {
				t.Error("Parse() should error for invalid payload")
			}
		})
	}
}

// TestParseSimple tests the simplified parser
func TestParseSimple(t *testing.T) {
	// Minimal payload
	payload := []byte{0x01, 0x02, 0x03, 0x04, 0x05}

	graph, graphSize, err := ParseSimple(payload)
	if err != nil {
		t.Fatalf("ParseSimple() error = %v", err)
	}

	if graph.NodeCount() != 1 {
		t.Errorf("ParseSimple() got %d nodes, want 1", graph.NodeCount())
	}

	if graph.Nodes[0].CodecID != codec.IDIdentity {
		t.Errorf("ParseSimple() codec = %v, want %v", graph.Nodes[0].CodecID, codec.IDIdentity)
	}

	if graphSize != 3 {
		t.Errorf("ParseSimple() graphSize = %d, want 3", graphSize)
	}
}

// TestReadVarint tests varint reading
func TestReadVarint(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    uint64
		wantErr bool
	}{
		{
			name: "zero",
			data: []byte{0x00},
			want: 0,
		},
		{
			name: "one byte max (127)",
			data: []byte{0x7F},
			want: 127,
		},
		{
			name: "two bytes (128)",
			data: []byte{0x80, 0x01},
			want: 128,
		},
		{
			name: "two bytes (300)",
			data: []byte{0xAC, 0x02},
			want: 300,
		},
		{
			name:    "truncated",
			data:    []byte{0x80}, // More bytes expected
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.data)
			got, err := readVarint(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("readVarint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("readVarint() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestWriteVarint tests varint writing
func TestWriteVarint(t *testing.T) {
	tests := []struct {
		name  string
		value uint64
	}{
		{name: "zero", value: 0},
		{name: "one", value: 1},
		{name: "127", value: 127},
		{name: "128", value: 128},
		{name: "300", value: 300},
		{name: "16384", value: 16384},
		{name: "max uint32", value: 0xFFFFFFFF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			writeVarint(&buf, tt.value)

			// Read it back
			got, err := readVarint(&buf)
			if err != nil {
				t.Fatalf("readVarint() error = %v", err)
			}

			if got != tt.value {
				t.Errorf("roundtrip failed: wrote %d, read %d", tt.value, got)
			}
		})
	}
}
