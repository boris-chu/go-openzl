package graph

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/boris-chu/go-openzl/internal/codec"
)

// Parser reads compression graphs from OpenZL frame payloads.
//
// The graph is encoded at the beginning of the frame payload using a compact
// binary format. After the graph comes the actual compressed data.
type Parser struct {
	r *bytes.Reader
}

// NewParser creates a new graph parser for the given payload
func NewParser(payload []byte) *Parser {
	return &Parser{
		r: bytes.NewReader(payload),
	}
}

// Parse reads a graph from the payload.
//
// Returns the graph and the offset where compressed data begins (after the graph).
//
// Graph Format (working hypothesis based on C library analysis):
//
//	Byte 0: nbNodes (varint) - number of nodes in graph
//	For each node:
//	  - codecID (varint)
//	  - nbParams (varint)
//	  - params (nbParams bytes)
//	  - nbInputs (varint)
//	  - inputs (nbInputs × varint, each is a node index)
//	Byte X: nbOutputs (varint) - number of output nodes
//	Output node indices (nbOutputs × varint)
//
// This format may need adjustment as we analyze more real frames.
func (p *Parser) Parse() (*Graph, int, error) {
	if p == nil || p.r == nil {
		return nil, 0, fmt.Errorf("nil parser")
	}

	startPos := p.r.Size() - int64(p.r.Len())

	// For now, we'll implement a simple parser for Identity-only graphs.
	// This will be expanded as we understand the real format better.

	// Read number of nodes
	nbNodes, err := readVarint(p.r)
	if err != nil {
		return nil, 0, fmt.Errorf("read node count: %w", err)
	}

	if nbNodes == 0 {
		return nil, 0, fmt.Errorf("graph has zero nodes")
	}

	if nbNodes > 1000 {
		return nil, 0, fmt.Errorf("graph has too many nodes: %d (max 1000)", nbNodes)
	}

	nodes := make([]*Node, nbNodes)

	// Read each node
	for i := uint64(0); i < nbNodes; i++ {
		node, err := p.parseNode()
		if err != nil {
			return nil, 0, fmt.Errorf("parse node %d: %w", i, err)
		}
		nodes[i] = node
	}

	// Read number of outputs
	nbOutputs, err := readVarint(p.r)
	if err != nil {
		return nil, 0, fmt.Errorf("read output count: %w", err)
	}

	if nbOutputs == 0 {
		return nil, 0, fmt.Errorf("graph has zero outputs")
	}

	outputs := make([]int, nbOutputs)
	for i := uint64(0); i < nbOutputs; i++ {
		outIdx, err := readVarint(p.r)
		if err != nil {
			return nil, 0, fmt.Errorf("read output %d: %w", i, err)
		}
		outputs[i] = int(outIdx)
	}

	graph := &Graph{
		Nodes:   nodes,
		Outputs: outputs,
	}

	// Validate graph structure
	if err := graph.Validate(); err != nil {
		return nil, 0, fmt.Errorf("invalid graph: %w", err)
	}

	// Calculate offset where compressed data begins
	currentPos := p.r.Size() - int64(p.r.Len())
	graphSize := int(currentPos - startPos)

	return graph, graphSize, nil
}

// parseNode reads a single node from the payload
func (p *Parser) parseNode() (*Node, error) {
	// Read codec ID
	codecID, err := readVarint(p.r)
	if err != nil {
		return nil, fmt.Errorf("read codec ID: %w", err)
	}

	// Read parameter count
	nbParams, err := readVarint(p.r)
	if err != nil {
		return nil, fmt.Errorf("read param count: %w", err)
	}

	// Read parameters
	var params []byte
	if nbParams > 0 {
		if nbParams > 1024*1024 { // 1MB max for params
			return nil, fmt.Errorf("parameter size too large: %d bytes", nbParams)
		}
		params = make([]byte, nbParams)
		if _, err := io.ReadFull(p.r, params); err != nil {
			return nil, fmt.Errorf("read params: %w", err)
		}
	}

	// Read input count
	nbInputs, err := readVarint(p.r)
	if err != nil {
		return nil, fmt.Errorf("read input count: %w", err)
	}

	// Read input indices
	var inputs []int
	if nbInputs > 0 {
		inputs = make([]int, nbInputs)
		for i := uint64(0); i < nbInputs; i++ {
			inputIdx, err := readVarint(p.r)
			if err != nil {
				return nil, fmt.Errorf("read input %d: %w", i, err)
			}
			inputs[i] = int(inputIdx)
		}
	}

	return &Node{
		CodecID: codec.ID(codecID),
		Params:  params,
		Inputs:  inputs,
	}, nil
}

// readVarint reads a LEB128 varint from the reader
func readVarint(r io.ByteReader) (uint64, error) {
	var result uint64
	var shift uint
	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}

		result |= uint64(b&0x7F) << shift
		if b&0x80 == 0 {
			break
		}

		shift += 7
		if shift >= 64 {
			return 0, fmt.Errorf("varint overflow")
		}
	}
	return result, nil
}

// ParseSimple is a simplified parser for testing.
//
// It assumes a simple single-node Identity codec graph, which is common
// for basic compression scenarios.
func ParseSimple(payload []byte) (*Graph, int, error) {
	// Simplest possible graph: 1 node (Identity codec), 1 output
	// This matches what we'd expect for incompressible data

	if len(payload) < 3 {
		return nil, 0, fmt.Errorf("payload too small for graph")
	}

	// For testing, we'll construct a simple Identity graph manually
	// Real implementation will parse from payload
	graph := &Graph{
		Nodes: []*Node{
			{
				CodecID: codec.IDIdentity,
				Params:  nil,
				Inputs:  nil, // Leaf node (no inputs = reads from payload)
			},
		},
		Outputs: []int{0}, // Node 0 is the output
	}

	// For now, assume graph is 3 bytes (this is a placeholder)
	// Real graph size will be determined by parsing
	graphSize := 3

	return graph, graphSize, nil
}

// MustParse is like Parse but panics on error (useful for tests)
func MustParse(payload []byte) (*Graph, int) {
	graph, size, err := NewParser(payload).Parse()
	if err != nil {
		panic(fmt.Sprintf("parse graph: %v", err))
	}
	return graph, size
}

// EncodeGraph encodes a graph to bytes (for creating test fixtures)
//
// This is the inverse of Parse - it writes a graph in the wire format.
func EncodeGraph(g *Graph) ([]byte, error) {
	if err := g.Validate(); err != nil {
		return nil, fmt.Errorf("invalid graph: %w", err)
	}

	var buf bytes.Buffer

	// Write number of nodes
	_ = writeVarint(&buf, uint64(len(g.Nodes))) // bytes.Buffer never fails

	// Write each node
	for _, node := range g.Nodes {
		// Codec ID
		_ = writeVarint(&buf, uint64(node.CodecID))

		// Params
		_ = writeVarint(&buf, uint64(len(node.Params)))
		buf.Write(node.Params)

		// Inputs
		_ = writeVarint(&buf, uint64(len(node.Inputs)))
		for _, inputIdx := range node.Inputs {
			_ = writeVarint(&buf, uint64(inputIdx))
		}
	}

	// Write outputs
	_ = writeVarint(&buf, uint64(len(g.Outputs)))
	for _, outIdx := range g.Outputs {
		_ = writeVarint(&buf, uint64(outIdx))
	}

	return buf.Bytes(), nil
}

// writeVarint writes a LEB128 varint to the buffer
func writeVarint(w io.ByteWriter, value uint64) error {
	for {
		b := byte(value & 0x7F)
		value >>= 7
		if value != 0 {
			b |= 0x80
		}
		if err := w.WriteByte(b); err != nil {
			return err
		}
		if value == 0 {
			break
		}
	}
	return nil
}

// DecodeU16LE decodes a little-endian uint16
func DecodeU16LE(b []byte) uint16 {
	return binary.LittleEndian.Uint16(b)
}

// DecodeU32LE decodes a little-endian uint32
func DecodeU32LE(b []byte) uint32 {
	return binary.LittleEndian.Uint32(b)
}
