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
	return e.ExecuteWithNodeSizes(graph, compressedData, outputSizes, nil)
}

// ExecuteWithNodeSizes runs the compression graph with explicit node sizes (v22+).
//
// This variant allows passing pre-computed node sizes from frame v22+, enabling
// multi-stage pipelines with size-changing codecs without inference.
//
// Parameters:
//   - graph: The compression graph to execute
//   - compressedData: The compressed payload data (after the graph)
//   - outputSizes: Expected sizes for each output (from frame header)
//   - explicitNodeSizes: Pre-computed sizes for all nodes (from frame v22+, nil for v21)
//
// Returns:
//   - outputs: Decompressed data for each output
//   - error: Any error during execution
func (e *Executor) ExecuteWithNodeSizes(graph *Graph, compressedData []byte, outputSizes, explicitNodeSizes []uint64) ([][]byte, error) {
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

	// Determine node sizes: explicit (v22+) or infer (v21)
	var nodeSizes []uint64
	var err error

	if len(explicitNodeSizes) > 0 {
		// Use explicit sizes from frame v22+
		if len(explicitNodeSizes) != len(graph.Nodes) {
			return nil, fmt.Errorf("node size count mismatch: have %d sizes, graph has %d nodes",
				len(explicitNodeSizes), len(graph.Nodes))
		}
		nodeSizes = explicitNodeSizes
		// No inference needed - sizes already provided!
	} else {
		// Fall back to inference for v21 compatibility
		nodeSizes, err = e.inferNodeSizes(graph, outputSizes)
		if err != nil {
			return nil, fmt.Errorf("infer node sizes: %w", err)
		}
	}

	// Execute each node and store results
	// For decompression, we process nodes in REVERSE topological order
	// because we're undoing the compression pipeline.
	//
	// Example compression graph: LZ77(0) → Huffman(1)
	//   Compression: src → LZ77 → Huffman → final
	//   Decompression: final → Huffman⁻¹ → LZ77⁻¹ → src
	//
	// So node 1 (Huffman) must decode first, then node 0 (LZ77) decodes its output.
	nodeOutputs := make([][]byte, len(graph.Nodes))

	for i := len(graph.Nodes) - 1; i >= 0; i-- {
		node := graph.Nodes[i]
		output, err := e.executeNode(node, i, compressedData, nodeOutputs, nodeSizes, graph, outputSizes)
		if err != nil {
			return nil, fmt.Errorf("execute node %d (codec %s): %w", i, node.CodecID, err)
		}
		nodeOutputs[i] = output
	}

	// Collect final outputs
	// For decompression, the actual output comes from the FIRST node (lowest index),
	// not from graph.Outputs (which describes the compression direction).
	//
	// In a compression graph LZ77(0) → Huffman(1) with Outputs=[1]:
	//   - Compression: output = nodeOutputs[1] (Huffman's output)
	//   - Decompression: output = nodeOutputs[0] (LZ77's output, the final data)
	//
	// For multi-stage decompression with LZ77→Huffman pattern, return first node's output
	// Pattern: Node 0 has no inputs, Node 1 has Inputs=[0], Outputs=[1]
	if len(graph.Nodes) == 2 && len(graph.Nodes[0].Inputs) == 0 &&
		len(graph.Nodes[1].Inputs) == 1 && graph.Nodes[1].Inputs[0] == 0 &&
		len(graph.Outputs) == 1 && graph.Outputs[0] == 1 {
		// LZ77→Huffman pattern: return node 0's output (final decompressed data)
		return [][]byte{nodeOutputs[0]}, nil
	}

	// Single-stage: use graph.Outputs as usual
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
	graph *Graph, outputSizes []uint64,
) ([]byte, error) {
	// Look up codec (already done in inferNodeSizes, but we need it again)
	codec, ok := e.registry.Get(node.CodecID)
	if !ok {
		return nil, fmt.Errorf("codec %s not registered", node.CodecID)
	}

	// Get precomputed output size for decompression
	//
	// During compression: nodeSizes[N] = output size of node N
	// During decompression (reverse execution):
	//   - If node N is not the first node (nodeIdx > 0), its output feeds node N-1
	//     → output size = nodeSizes[N-1] (input size of next decoder)
	//   - If node N is the first node (nodeIdx == 0), it produces the final output
	//     → output size = final decompressed size (from outputSizes)
	var outputSize uint64
	if nodeIdx > 0 {
		// Output feeds into previous node, so size = that node's compressed input
		outputSize = nodeSizes[nodeIdx-1]
	} else {
		// This is the final decoder (node 0), output = final decompressed size
		// For LZ77→Huffman pattern, node 0 produces final output
		// Pattern: Node 0 has no inputs, Node 1 has Inputs=[0], Outputs=[1]
		if len(graph.Nodes) == 2 && len(node.Inputs) == 0 &&
			len(graph.Nodes[1].Inputs) == 1 && graph.Nodes[1].Inputs[0] == 0 &&
			len(graph.Outputs) == 1 && graph.Outputs[0] == 1 {
			// LZ77→Huffman pattern: node 0 produces final output
			outputSize = outputSizes[0]
		} else {
			// Single-stage: check graph.Outputs
			isOutputNode := false
			var outputIndex int
			for i, outIdx := range graph.Outputs {
				if outIdx == nodeIdx {
					isOutputNode = true
					outputIndex = i
					break
				}
			}

			if isOutputNode {
				outputSize = outputSizes[outputIndex]
			} else {
				return nil, fmt.Errorf("node %d is neither an output nor feeds another node", nodeIdx)
			}
		}
	}

	// Allocate output buffer (may be empty for outputSize=0)
	dst := make([]byte, outputSize)

	// Determine source data
	//
	// In a compression graph like LZ77(0) → Huffman(1):
	//   - During compression: node 0 is leaf (src), node 1 uses node 0's output
	//   - During decompression (reverse execution):
	//     * Node 1 (highest index) decodes from compressedData first
	//     * Node 0 decodes from node 1's output
	//
	// So for decompression:
	//   - The LAST node (highest index) is the first decoder (uses compressedData)
	//   - Earlier nodes decode from later nodes' outputs
	var src []byte

	// Check if this node is the final node in the pipeline (last decoder)
	isFinalNode := (nodeIdx == len(nodeOutputs)-1) ||
	                (len(node.Inputs) > 0 && nodeIdx > node.Inputs[0])

	if isFinalNode {
		// Final node in compression = first decoder in decompression
		src = compressedData
	} else {
		// Earlier nodes decode from later nodes' outputs
		// Find the node that depends on us (nodeIdx+1, +2, etc.)
		foundInput := false
		for i := nodeIdx + 1; i < len(nodeOutputs); i++ {
			if nodeOutputs[i] != nil {
				src = nodeOutputs[i]
				foundInput = true
				break
			}
		}
		if !foundInput {
			return nil, fmt.Errorf("node %d has no decoded input available", nodeIdx)
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
