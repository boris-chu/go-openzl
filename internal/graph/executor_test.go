package graph

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/boris-chu/go-openzl/internal/codec"
)

// TestExecutor_SimpleIdentity tests executing a simple Identity codec graph
func TestExecutor_SimpleIdentity(t *testing.T) {
	// Create simple graph: Identity codec, single output
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

	// Test data
	input := []byte("test data")

	// Execute
	executor := DefaultExecutor()
	output, err := executor.ExecuteSimple(graph, input, uint64(len(input)))
	if err != nil {
		t.Fatalf("ExecuteSimple() error = %v", err)
	}

	// Verify output
	if !bytes.Equal(output, input) {
		t.Errorf("ExecuteSimple() = %v, want %v", output, input)
	}
}

// TestExecutor_TwoNodePipeline tests a two-node Identity pipeline
func TestExecutor_TwoNodePipeline(t *testing.T) {
	// Graph: Identity → Identity
	graph := &Graph{
		Nodes: []*Node{
			{
				CodecID: codec.IDIdentity,
				Params:  nil,
				Inputs:  nil, // Leaf node
			},
			{
				CodecID: codec.IDIdentity,
				Params:  nil,
				Inputs:  []int{0}, // Uses output from node 0
			},
		},
		Outputs: []int{1}, // Final output is node 1
	}

	input := []byte("pipeline test")

	executor := DefaultExecutor()
	output, err := executor.ExecuteSimple(graph, input, uint64(len(input)))
	if err != nil {
		t.Fatalf("ExecuteSimple() error = %v", err)
	}

	if !bytes.Equal(output, input) {
		t.Errorf("ExecuteSimple() = %v, want %v", output, input)
	}
}

// TestExecutor_EmptyData tests decompressing empty data
func TestExecutor_EmptyData(t *testing.T) {
	graph := &Graph{
		Nodes: []*Node{
			{CodecID: codec.IDIdentity, Inputs: nil},
		},
		Outputs: []int{0},
	}

	executor := DefaultExecutor()
	output, err := executor.ExecuteSimple(graph, []byte{}, 0)
	if err != nil {
		t.Fatalf("ExecuteSimple() error = %v", err)
	}

	if len(output) != 0 {
		t.Errorf("ExecuteSimple() returned %d bytes, want 0", len(output))
	}
}

// TestExecutor_LargeData tests decompressing larger data
func TestExecutor_LargeData(t *testing.T) {
	graph := &Graph{
		Nodes: []*Node{
			{CodecID: codec.IDIdentity, Inputs: nil},
		},
		Outputs: []int{0},
	}

	// 1MB test data
	input := bytes.Repeat([]byte("test"), 256*1024)

	executor := DefaultExecutor()
	output, err := executor.ExecuteSimple(graph, input, uint64(len(input)))
	if err != nil {
		t.Fatalf("ExecuteSimple() error = %v", err)
	}

	if !bytes.Equal(output, input) {
		t.Error("ExecuteSimple() output doesn't match input")
	}
}

// TestExecutor_MissingCodec tests error handling for missing codec
func TestExecutor_MissingCodec(t *testing.T) {
	graph := &Graph{
		Nodes: []*Node{
			{
				CodecID: codec.ID(999), // Non-existent codec
				Inputs:  nil,
			},
		},
		Outputs: []int{0},
	}

	executor := DefaultExecutor()
	_, err := executor.ExecuteSimple(graph, []byte("test"), 4)
	if err == nil {
		t.Error("ExecuteSimple() should error for missing codec")
	}
}

// TestExecutor_InvalidGraph tests error handling for invalid graphs
func TestExecutor_InvalidGraph(t *testing.T) {
	tests := []struct {
		name  string
		graph *Graph
	}{
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
		{
			name: "output index out of bounds",
			graph: &Graph{
				Nodes: []*Node{
					{CodecID: codec.IDIdentity},
				},
				Outputs: []int{5},
			},
		},
	}

	executor := DefaultExecutor()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executor.ExecuteSimple(tt.graph, []byte("test"), 4)
			if err == nil {
				t.Error("ExecuteSimple() should error for invalid graph")
			}
		})
	}
}

// TestExecutor_MultipleOutputs tests graphs with multiple outputs
func TestExecutor_MultipleOutputs(t *testing.T) {
	// Graph with two independent outputs
	graph := &Graph{
		Nodes: []*Node{
			{CodecID: codec.IDIdentity, Inputs: nil},
			{CodecID: codec.IDIdentity, Inputs: nil},
		},
		Outputs: []int{0, 1},
	}

	input := []byte("test")
	executor := DefaultExecutor()

	outputs, err := executor.Execute(graph, input, []uint64{4, 4})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(outputs) != 2 {
		t.Errorf("Execute() returned %d outputs, want 2", len(outputs))
	}

	for i, output := range outputs {
		if !bytes.Equal(output, input) {
			t.Errorf("Execute() output[%d] = %v, want %v", i, output, input)
		}
	}
}

// TestExecutor_OutputSizeMismatch tests error handling for size mismatch
func TestExecutor_OutputSizeMismatch(t *testing.T) {
	graph := &Graph{
		Nodes: []*Node{
			{CodecID: codec.IDIdentity, Inputs: nil},
		},
		Outputs: []int{0},
	}

	executor := DefaultExecutor()

	// Graph has 1 output but we provide 2 sizes
	_, err := executor.Execute(graph, []byte("test"), []uint64{4, 4})
	if err == nil {
		t.Error("Execute() should error when output size count doesn't match graph outputs")
	}
}

// TestExecutor_NilRegistry tests error handling for nil registry
func TestExecutor_NilRegistry(t *testing.T) {
	executor := &Executor{registry: nil}

	graph := &Graph{
		Nodes: []*Node{
			{CodecID: codec.IDIdentity, Inputs: nil},
		},
		Outputs: []int{0},
	}

	_, err := executor.ExecuteSimple(graph, []byte("test"), 4)
	if err == nil {
		t.Error("ExecuteSimple() should error for nil registry")
	}
}

// TestExecutor_CustomRegistry tests using a custom codec registry
func TestExecutor_CustomRegistry(t *testing.T) {
	// Create custom registry with only Identity
	registry := codec.NewRegistry()
	registry.Register(codec.NewIdentity())

	executor := NewExecutor(registry)

	graph := &Graph{
		Nodes: []*Node{
			{CodecID: codec.IDIdentity, Inputs: nil},
		},
		Outputs: []int{0},
	}

	input := []byte("custom registry test")
	output, err := executor.ExecuteSimple(graph, input, uint64(len(input)))
	if err != nil {
		t.Fatalf("ExecuteSimple() error = %v", err)
	}

	if !bytes.Equal(output, input) {
		t.Errorf("ExecuteSimple() = %v, want %v", output, input)
	}
}

// BenchmarkExecutor_SimpleIdentity benchmarks simple Identity execution
func BenchmarkExecutor_SimpleIdentity(b *testing.B) {
	graph := &Graph{
		Nodes: []*Node{
			{CodecID: codec.IDIdentity, Inputs: nil},
		},
		Outputs: []int{0},
	}

	sizes := []int{16, 256, 4096, 65536}
	executor := DefaultExecutor()

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			input := bytes.Repeat([]byte("a"), size)
			b.SetBytes(int64(size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := executor.ExecuteSimple(graph, input, uint64(size))
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkExecutor_TwoNodePipeline benchmarks two-node pipeline
func BenchmarkExecutor_TwoNodePipeline(b *testing.B) {
	graph := &Graph{
		Nodes: []*Node{
			{CodecID: codec.IDIdentity, Inputs: nil},
			{CodecID: codec.IDIdentity, Inputs: []int{0}},
		},
		Outputs: []int{1},
	}

	input := bytes.Repeat([]byte("a"), 4096)
	executor := DefaultExecutor()
	b.SetBytes(4096)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := executor.ExecuteSimple(graph, input, 4096)
		if err != nil {
			b.Fatal(err)
		}
	}
}
