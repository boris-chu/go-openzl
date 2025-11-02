// Package graph provides the compression graph execution engine.
//
// OpenZL uses a directed acyclic graph (DAG) of codec transformations to
// compress and decompress data. Each node in the graph represents a codec
// operation, and edges represent data flow between codecs.
//
// Graph Structure:
//
//	Input Data
//	    ↓
//	[Node 0: Delta]  ← params
//	    ↓
//	[Node 1: FSE]
//	    ↓
//	Output Data
//
// During decompression, the graph is executed in reverse order - entropy
// decoding first, then delta decoding, etc.
package graph

import (
	"fmt"

	"github.com/boris-chu/go-openzl/internal/codec"
)

const (
	nilGraphString = "<nil graph>"
)

// Node represents a single codec transformation in the graph.
//
// Each node has:
// - A codec ID identifying which codec to use
// - Parameters specific to that codec
// - Input indices specifying which nodes provide input
type Node struct {
	// CodecID is the ID of the codec to execute
	CodecID codec.ID

	// Params are codec-specific parameters (can be empty)
	Params []byte

	// Inputs are indices of nodes whose outputs feed into this node.
	// For most codecs this is a single input. Some codecs (like merge)
	// may have multiple inputs.
	Inputs []int
}

// Graph represents a complete compression graph.
//
// The graph describes the full pipeline of transformations applied during
// compression. During decompression, we execute the graph to reverse these
// transformations.
type Graph struct {
	// Nodes are the codec transformations in the graph.
	// Nodes are stored in topological order (dependencies before dependents).
	Nodes []*Node

	// Outputs are the indices of nodes that produce the final outputs.
	// Most frames have a single output, but some have multiple (e.g., split data).
	Outputs []int
}

// Validate checks if the graph is well-formed.
//
// A valid graph must:
// - Have at least one node
// - Have at least one output
// - Have all output indices within bounds
// - Have all input indices within bounds and < node index (DAG property)
// - Have no cycles (implied by input < node index requirement)
func (g *Graph) Validate() error {
	if g == nil {
		return fmt.Errorf("nil graph")
	}

	if len(g.Nodes) == 0 {
		return fmt.Errorf("graph has no nodes")
	}

	if len(g.Outputs) == 0 {
		return fmt.Errorf("graph has no outputs")
	}

	// Check output indices
	for i, outIdx := range g.Outputs {
		if outIdx < 0 || outIdx >= len(g.Nodes) {
			return fmt.Errorf("output %d index %d out of bounds (have %d nodes)",
				i, outIdx, len(g.Nodes))
		}
	}

	// Check node input indices
	for i, node := range g.Nodes {
		if node == nil {
			return fmt.Errorf("node %d is nil", i)
		}

		for j, inputIdx := range node.Inputs {
			if inputIdx < 0 || inputIdx >= len(g.Nodes) {
				return fmt.Errorf("node %d input %d index %d out of bounds (have %d nodes)",
					i, j, inputIdx, len(g.Nodes))
			}

			// DAG property: inputs must come from earlier nodes
			// (This prevents cycles)
			if inputIdx >= i {
				return fmt.Errorf("node %d input %d index %d >= node index (not a DAG)",
					i, j, inputIdx)
			}
		}
	}

	return nil
}

// NodeCount returns the number of nodes in the graph
func (g *Graph) NodeCount() int {
	if g == nil {
		return 0
	}
	return len(g.Nodes)
}

// OutputCount returns the number of outputs in the graph
func (g *Graph) OutputCount() int {
	if g == nil {
		return 0
	}
	return len(g.Outputs)
}

// IsLeaf returns true if the node has no inputs (i.e., it's a leaf node
// that operates on the compressed payload directly)
func (n *Node) IsLeaf() bool {
	return len(n.Inputs) == 0
}

// InputCount returns the number of inputs to this node
func (n *Node) InputCount() int {
	return len(n.Inputs)
}

// String returns a human-readable representation of the graph
func (g *Graph) String() string {
	if g == nil {
		return nilGraphString
	}

	result := fmt.Sprintf("Graph{Nodes: %d, Outputs: %v}\n", len(g.Nodes), g.Outputs)
	for i, node := range g.Nodes {
		result += fmt.Sprintf("  Node %d: Codec=%s, Params=%d bytes, Inputs=%v\n",
			i, node.CodecID, len(node.Params), node.Inputs)
	}
	return result
}
