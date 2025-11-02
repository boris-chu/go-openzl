package graph

import (
	"fmt"

	"github.com/boris-chu/go-openzl/internal/codec"
)

// Executor executes a compression graph to decompress data.
//
// The executor:
// 1. Validates the graph structure
// 2. Looks up codecs from the registry
// 3. Executes nodes in topological order
// 4. Returns the final decompressed outputs
type Executor struct {
	registry *codec.Registry
}

// NewExecutor creates a new graph executor with the given codec registry
func NewExecutor(registry *codec.Registry) *Executor {
	return &Executor{
		registry: registry,
	}
}

// Execute runs the compression graph to decompress data.
//
// Parameters:
//   - graph: The compression graph to execute
//   - compressedData: The compressed payload data (after the graph)
//   - outputSizes: Expected sizes for each output (from frame header)
//
// Returns:
//   - outputs: Decompressed data for each output
//   - error: Any error during execution
//
// The graph is executed in topological order:
//  1. Leaf nodes (no inputs) decompress directly from compressedData
//  2. Other nodes decompress using outputs from their input nodes
//  3. Final outputs are collected and returned
func (e *Executor) Execute(graph *Graph, compressedData []byte, outputSizes []uint64) ([][]byte, error) {
	if e == nil {
		return nil, fmt.Errorf("nil executor")
	}

	if e.registry == nil {
		return nil, fmt.Errorf("nil codec registry")
	}

	// Validate graph
	if err := graph.Validate(); err != nil {
		return nil, fmt.Errorf("invalid graph: %w", err)
	}

	// Validate output sizes
	if len(outputSizes) != len(graph.Outputs) {
		return nil, fmt.Errorf("output size mismatch: have %d sizes, graph has %d outputs",
			len(outputSizes), len(graph.Outputs))
	}

	// Compute all node output sizes using smart inference
	nodeSizes, err := e.inferNodeSizes(graph, outputSizes)
	if err != nil {
		return nil, fmt.Errorf("infer node sizes: %w", err)
	}

	// Execute each node and store results
	nodeOutputs := make([][]byte, len(graph.Nodes))

	for i, node := range graph.Nodes {
		output, err := e.executeNode(node, i, compressedData, nodeOutputs, nodeSizes)
		if err != nil {
			return nil, fmt.Errorf("execute node %d (codec %s): %w", i, node.CodecID, err)
		}
		nodeOutputs[i] = output
	}

	// Collect final outputs
	outputs := make([][]byte, len(graph.Outputs))
	for i, outIdx := range graph.Outputs {
		outputs[i] = nodeOutputs[outIdx]
	}

	return outputs, nil
}

// inferNodeSizes computes the output size for each node using smart inference.
//
// Algorithm:
// 1. Start with final output sizes from frame header
// 2. For size-preserving codecs, propagate sizes backward to their inputs
// 3. Size-changing codecs require explicit size information
func (e *Executor) inferNodeSizes(graph *Graph, outputSizes []uint64) ([]uint64, error) {
	nodeSizes := make([]uint64, len(graph.Nodes))
	sizeKnown := make([]bool, len(graph.Nodes))

	// Step 1: Mark final output nodes with their known sizes
	for i, outIdx := range graph.Outputs {
		nodeSizes[outIdx] = outputSizes[i]
		sizeKnown[outIdx] = true
	}

	// Step 2: Backward propagation for size-preserving codecs
	// For a size-preserving codec, its input has the same size as its output
	// Process nodes in reverse topological order (from outputs to inputs)
	for nodeIdx := len(graph.Nodes) - 1; nodeIdx >= 0; nodeIdx-- {
		node := graph.Nodes[nodeIdx]

		// Skip if size not yet known
		if !sizeKnown[nodeIdx] {
			continue
		}

		// This node's size is known
		// Look up codec to see if it's size-preserving
		codec, ok := e.registry.Get(node.CodecID)
		if !ok {
			return nil, fmt.Errorf("codec %s not registered", node.CodecID)
		}

		// If this is a size-preserving codec, propagate size to its inputs
		if codec.PreservesSize() && len(node.Inputs) > 0 {
			// Size-preserving codec: input size = output size
			for _, inputIdx := range node.Inputs {
				if !sizeKnown[inputIdx] {
					nodeSizes[inputIdx] = nodeSizes[nodeIdx]
					sizeKnown[inputIdx] = true
				}
			}
		}
	}

	// Step 3: Validate that all nodes have sizes
	// (Size-changing codecs in intermediate positions without size metadata will fail here)
	for i, known := range sizeKnown {
		if !known {
			return nil, fmt.Errorf("cannot infer output size for node %d (codec %s): "+
				"size-changing codec in intermediate position requires explicit size metadata",
				i, graph.Nodes[i].CodecID)
		}
	}

	return nodeSizes, nil
}

// executeNode executes a single node in the graph
func (e *Executor) executeNode(
	node *Node, nodeIdx int, compressedData []byte,
	nodeOutputs [][]byte, nodeSizes []uint64,
) ([]byte, error) {
	// Look up codec (already done in inferNodeSizes, but we need it again)
	codec, ok := e.registry.Get(node.CodecID)
	if !ok {
		return nil, fmt.Errorf("codec %s not registered", node.CodecID)
	}

	// Get precomputed output size (can be 0 for empty data)
	outputSize := nodeSizes[nodeIdx]

	// Allocate output buffer (may be empty for outputSize=0)
	dst := make([]byte, outputSize)

	// Determine source data
	var src []byte
	if node.IsLeaf() {
		// Leaf node: decompress from compressed payload
		src = compressedData
	} else {
		// Non-leaf: use output from first input node
		// (Multi-input codecs will be handled later)
		if len(node.Inputs) == 0 {
			return nil, fmt.Errorf("non-leaf node has no inputs")
		}

		inputIdx := node.Inputs[0]
		if inputIdx >= nodeIdx {
			return nil, fmt.Errorf("input index %d >= node index %d (not topologically sorted)", inputIdx, nodeIdx)
		}

		src = nodeOutputs[inputIdx]
		if src == nil {
			return nil, fmt.Errorf("input node %d has no output", inputIdx)
		}
	}

	// Execute codec
	n, err := codec.Decode(dst, src, node.Params)
	if err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	// Return exactly the decoded bytes
	return dst[:n], nil
}

// ExecuteSimple executes a simple single-output graph.
//
// This is a convenience wrapper for the common case of a single output.
// Returns the decompressed data or an error.
func (e *Executor) ExecuteSimple(graph *Graph, compressedData []byte, outputSize uint64) ([]byte, error) {
	if len(graph.Outputs) != 1 {
		return nil, fmt.Errorf("graph has %d outputs, expected 1", len(graph.Outputs))
	}

	outputs, err := e.Execute(graph, compressedData, []uint64{outputSize})
	if err != nil {
		return nil, err
	}

	return outputs[0], nil
}

// DefaultExecutor returns an executor with the default codec registry
func DefaultExecutor() *Executor {
	return NewExecutor(codec.DefaultRegistry())
}
