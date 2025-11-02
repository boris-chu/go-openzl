package graph

import (
	"testing"

	"github.com/borischu/go-openzl/internal/codec"
)

// TestGraph_Validate tests graph validation
func TestGraph_Validate(t *testing.T) {
	tests := []struct {
		name    string
		graph   *Graph
		wantErr bool
	}{
		{
			name:    "nil graph",
			graph:   nil,
			wantErr: true,
		},
		{
			name: "empty graph",
			graph: &Graph{
				Nodes:   []*Node{},
				Outputs: []int{},
			},
			wantErr: true,
		},
		{
			name: "no outputs",
			graph: &Graph{
				Nodes: []*Node{
					{CodecID: codec.IDIdentity},
				},
				Outputs: []int{},
			},
			wantErr: true,
		},
		{
			name: "valid single node",
			graph: &Graph{
				Nodes: []*Node{
					{CodecID: codec.IDIdentity},
				},
				Outputs: []int{0},
			},
			wantErr: false,
		},
		{
			name: "output index out of bounds",
			graph: &Graph{
				Nodes: []*Node{
					{CodecID: codec.IDIdentity},
				},
				Outputs: []int{1}, // Only node 0 exists
			},
			wantErr: true,
		},
		{
			name: "negative output index",
			graph: &Graph{
				Nodes: []*Node{
					{CodecID: codec.IDIdentity},
				},
				Outputs: []int{-1},
			},
			wantErr: true,
		},
		{
			name: "nil node",
			graph: &Graph{
				Nodes: []*Node{
					nil,
				},
				Outputs: []int{0},
			},
			wantErr: true,
		},
		{
			name: "valid two-node pipeline",
			graph: &Graph{
				Nodes: []*Node{
					{CodecID: codec.IDIdentity, Inputs: nil},      // Leaf node
					{CodecID: codec.IDIdentity, Inputs: []int{0}}, // Uses output of node 0
				},
				Outputs: []int{1},
			},
			wantErr: false,
		},
		{
			name: "input index >= node index (cycle)",
			graph: &Graph{
				Nodes: []*Node{
					{CodecID: codec.IDIdentity, Inputs: []int{1}}, // References future node
					{CodecID: codec.IDIdentity, Inputs: nil},
				},
				Outputs: []int{1},
			},
			wantErr: true,
		},
		{
			name: "input index out of bounds",
			graph: &Graph{
				Nodes: []*Node{
					{CodecID: codec.IDIdentity, Inputs: nil},
					{CodecID: codec.IDIdentity, Inputs: []int{5}}, // Node 5 doesn't exist
				},
				Outputs: []int{1},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.graph.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestGraph_NodeCount tests node count
func TestGraph_NodeCount(t *testing.T) {
	tests := []struct {
		name  string
		graph *Graph
		want  int
	}{
		{
			name:  "nil graph",
			graph: nil,
			want:  0,
		},
		{
			name: "empty graph",
			graph: &Graph{
				Nodes: []*Node{},
			},
			want: 0,
		},
		{
			name: "single node",
			graph: &Graph{
				Nodes: []*Node{
					{CodecID: codec.IDIdentity},
				},
			},
			want: 1,
		},
		{
			name: "three nodes",
			graph: &Graph{
				Nodes: []*Node{
					{CodecID: codec.IDIdentity},
					{CodecID: codec.IDDelta},
					{CodecID: codec.IDFSE},
				},
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.graph.NodeCount(); got != tt.want {
				t.Errorf("NodeCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGraph_OutputCount tests output count
func TestGraph_OutputCount(t *testing.T) {
	tests := []struct {
		name  string
		graph *Graph
		want  int
	}{
		{
			name:  "nil graph",
			graph: nil,
			want:  0,
		},
		{
			name: "no outputs",
			graph: &Graph{
				Outputs: []int{},
			},
			want: 0,
		},
		{
			name: "single output",
			graph: &Graph{
				Outputs: []int{0},
			},
			want: 1,
		},
		{
			name: "two outputs",
			graph: &Graph{
				Outputs: []int{0, 1},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.graph.OutputCount(); got != tt.want {
				t.Errorf("OutputCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestNode_IsLeaf tests leaf node detection
func TestNode_IsLeaf(t *testing.T) {
	tests := []struct {
		name string
		node *Node
		want bool
	}{
		{
			name: "leaf node (no inputs)",
			node: &Node{
				CodecID: codec.IDIdentity,
				Inputs:  nil,
			},
			want: true,
		},
		{
			name: "leaf node (empty inputs)",
			node: &Node{
				CodecID: codec.IDIdentity,
				Inputs:  []int{},
			},
			want: true,
		},
		{
			name: "non-leaf node (one input)",
			node: &Node{
				CodecID: codec.IDIdentity,
				Inputs:  []int{0},
			},
			want: false,
		},
		{
			name: "non-leaf node (multiple inputs)",
			node: &Node{
				CodecID: codec.IDIdentity,
				Inputs:  []int{0, 1},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.node.IsLeaf(); got != tt.want {
				t.Errorf("IsLeaf() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestNode_InputCount tests input count
func TestNode_InputCount(t *testing.T) {
	tests := []struct {
		name string
		node *Node
		want int
	}{
		{
			name: "no inputs",
			node: &Node{
				CodecID: codec.IDIdentity,
				Inputs:  nil,
			},
			want: 0,
		},
		{
			name: "one input",
			node: &Node{
				CodecID: codec.IDIdentity,
				Inputs:  []int{0},
			},
			want: 1,
		},
		{
			name: "three inputs",
			node: &Node{
				CodecID: codec.IDIdentity,
				Inputs:  []int{0, 1, 2},
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.node.InputCount(); got != tt.want {
				t.Errorf("InputCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGraph_String tests string representation
func TestGraph_String(t *testing.T) {
	graph := &Graph{
		Nodes: []*Node{
			{
				CodecID: codec.IDIdentity,
				Params:  []byte{1, 2, 3},
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

	str := graph.String()
	if str == "" {
		t.Error("String() returned empty string")
	}

	// Check it contains key information
	if str == "<nil graph>" {
		t.Error("String() returned nil graph message for valid graph")
	}

	// Nil graph
	var nilGraph *Graph
	if nilGraph.String() != "<nil graph>" {
		t.Error("String() should return '<nil graph>' for nil graph")
	}
}
