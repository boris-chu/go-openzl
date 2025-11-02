package graph

import (
	"fmt"

	"github.com/borischu/go-openzl/internal/codec"
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

	// Execute each node and store results
	nodeOutputs := make([][]byte, len(graph.Nodes))

	for i, node := range graph.Nodes {
		output, err := e.executeNode(node, i, compressedData, nodeOutputs, outputSizes, graph.Outputs)
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

// executeNode executes a single node in the graph
func (e *Executor) executeNode(node *Node, nodeIdx int, compressedData []byte, nodeOutputs [][]byte, outputSizes []uint64, graphOutputs []int) ([]byte, error) {
	// Look up codec
	codec, ok := e.registry.Get(node.CodecID)
	if !ok {
		return nil, fmt.Errorf("codec %s not registered", node.CodecID)
	}

	// Determine output size
	// If this node is a graph output, use the size from frame header
	// Otherwise, we need to infer it (for now, use a heuristic)
	var outputSize uint64
	for i, outIdx := range graphOutputs {
		if outIdx == nodeIdx {
			outputSize = outputSizes[i]
			break
		}
	}

	// If not a final output, we need to determine size another way
	// For now, use a simple heuristic based on codec type
	if outputSize == 0 {
		// For intermediate nodes, estimate output size
		// This is codec-dependent; for Identity it's the same as input
		// For now, use compressed data size as a reasonable default
		outputSize = uint64(len(compressedData))
	}

	// Allocate output buffer
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
