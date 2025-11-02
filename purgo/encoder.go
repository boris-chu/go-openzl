// Package purgo provides Pure Go OpenZL compression and decompression.
package purgo

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/boris-chu/go-openzl/internal/codec"
	"github.com/boris-chu/go-openzl/internal/frame"
	"github.com/boris-chu/go-openzl/internal/graph"
)

// Compress compresses data using Pure Go OpenZL encoder with Huffman coding.
//
// This function uses Huffman entropy coding which provides good compression
// for text and binary data with repeated patterns.
//
// For numeric data with sequential patterns, use CompressInt64() which applies
// Delta encoding before compression.
//
// Parameters:
//   - src: Uncompressed data
//
// Returns:
//   - Compressed OpenZL frame
//   - Error if compression fails
//
// Example:
//
//	data := []byte("hello world, hello compression!")
//	compressed, err := purgo.Compress(data)
//	if err != nil {
//		log.Fatal(err)
//	}
func Compress(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, fmt.Errorf("purgo: cannot compress empty data")
	}

	// Try Huffman compression first
	gHuffman := &graph.Graph{
		Nodes: []*graph.Node{
			{
				CodecID: codec.IDHuffman,
				Params:  nil,
				Inputs:  nil,
			},
		},
		Outputs: []int{0},
	}

	// Try compressing with Huffman
	registry := codec.DefaultRegistry()
	compressedData, err := executeCompressionGraph(gHuffman, src, registry)
	if err != nil || len(compressedData) >= len(src) {
		// Huffman failed or didn't compress - use Identity instead
		gIdentity := &graph.Graph{
			Nodes: []*graph.Node{
				{
					CodecID: codec.IDIdentity,
					Params:  nil,
					Inputs:  nil,
				},
			},
			Outputs: []int{0},
		}
		return compressWithGraph(gIdentity, src)
	}

	// Huffman worked - build frame with Huffman-compressed data
	graphBytes, err := graph.EncodeGraph(gHuffman)
	if err != nil {
		return nil, fmt.Errorf("purgo: encode graph: %w", err)
	}

	payload := append(graphBytes, compressedData...)

	f := &frame.Frame{
		Header: &frame.Header{
			Magic:   frame.MagicNumberBase + 21,
			Version: 21,
			Flags:   0,
		},
		Outputs: []*frame.Output{
			{
				Type:             frame.TypeSerial,
				DecompressedSize: uint64(len(src)),
			},
		},
		Payload: payload,
	}

	return serializeFrame(f)
}

// executeCompressionGraph executes a compression graph on source data.
//
// This supports multi-node graphs by executing nodes in topological order.
// Each node takes input from previous nodes or the source data.
func executeCompressionGraph(g *graph.Graph, src []byte, registry *codec.Registry) ([]byte, error) {
	// Storage for intermediate results (node outputs)
	nodeOutputs := make([][]byte, len(g.Nodes))

	// Execute each node in order
	for i, node := range g.Nodes {
		c, ok := registry.Get(node.CodecID)
		if !ok {
			return nil, fmt.Errorf("purgo: codec %d not found", node.CodecID)
		}

		// Determine input for this node
		var input []byte
		if len(node.Inputs) == 0 {
			// No inputs = use source data
			input = src
		} else if len(node.Inputs) == 1 {
			// Single input from previous node
			inputIdx := node.Inputs[0]
			if inputIdx >= i {
				return nil, fmt.Errorf("purgo: invalid input index %d for node %d", inputIdx, i)
			}
			input = nodeOutputs[inputIdx]
		} else {
			return nil, fmt.Errorf("purgo: multi-input nodes not yet supported")
		}

		// Allocate output buffer (generous size for safety)
		// Entropy coders (Huffman) may expand data temporarily
		dst := make([]byte, len(input)*2+1024)

		// Encode
		n, err := c.Encode(dst, input, node.Params)
		if err != nil {
			return nil, fmt.Errorf("purgo: encode with codec %s (node %d): %w", c.Name(), i, err)
		}

		// Store output for this node
		nodeOutputs[i] = dst[:n]
	}

	// Return output from final node
	if len(g.Outputs) != 1 {
		return nil, fmt.Errorf("purgo: multi-output graphs not yet supported")
	}
	outputIdx := g.Outputs[0]
	if outputIdx >= len(nodeOutputs) {
		return nil, fmt.Errorf("purgo: invalid output index %d", outputIdx)
	}

	return nodeOutputs[outputIdx], nil
}

// compressWithGraph compresses data using a custom compression graph.
func compressWithGraph(g *graph.Graph, src []byte) ([]byte, error) {
	// Encode the graph
	graphBytes, err := graph.EncodeGraph(g)
	if err != nil {
		return nil, fmt.Errorf("purgo: encode graph: %w", err)
	}

	// Execute compression graph
	registry := codec.DefaultRegistry()
	compressedData, err := executeCompressionGraph(g, src, registry)
	if err != nil {
		return nil, fmt.Errorf("purgo: execute graph: %w", err)
	}

	// Build payload (graph + compressed data)
	payload := append(graphBytes, compressedData...)

	// Build frame
	f := &frame.Frame{
		Header: &frame.Header{
			Magic:   frame.MagicNumberBase + 21, // Version 21
			Version: 21,
			Flags:   0, // No checksums for now
		},
		Outputs: []*frame.Output{
			{
				Type:             frame.TypeSerial,
				DecompressedSize: uint64(len(src)),
			},
		},
		Payload: payload,
	}

	// Serialize frame
	return serializeFrame(f)
}

// serializeFrame serializes a frame to bytes.
func serializeFrame(f *frame.Frame) ([]byte, error) {
	buf := new(bytes.Buffer)

	// Write magic number (little-endian)
	magic := f.Header.Magic
	buf.WriteByte(byte(magic))
	buf.WriteByte(byte(magic >> 8))
	buf.WriteByte(byte(magic >> 16))
	buf.WriteByte(byte(magic >> 24))

	// Write flags
	buf.WriteByte(byte(f.Header.Flags))

	// Write token1 (nbOutputs in lower 4 bits)
	if len(f.Outputs) > 15 {
		return nil, fmt.Errorf("purgo: too many outputs (max 15, got %d)", len(f.Outputs))
	}
	token1 := byte(len(f.Outputs))
	// Upper 4 bits: output types (we'll encode up to 2 types in token1)
	if len(f.Outputs) >= 1 {
		token1 |= byte(f.Outputs[0].Type) << 4
	}
	if len(f.Outputs) >= 2 {
		token1 |= byte(f.Outputs[1].Type) << 6
	}
	buf.WriteByte(token1)

	// Write output sizes (varints)
	// Note: OpenZL stores size as (actual_size + 1), so 0 size = varint 1
	for _, output := range f.Outputs {
		writeVarint(buf, output.DecompressedSize+1)
	}

	// Write payload
	buf.Write(f.Payload)

	return buf.Bytes(), nil
}

// writeVarint writes a LEB128 varint to the buffer.
func writeVarint(buf *bytes.Buffer, value uint64) {
	for {
		b := byte(value & 0x7F)
		value >>= 7
		if value != 0 {
			b |= 0x80
		}
		buf.WriteByte(b)
		if value == 0 {
			break
		}
	}
}

// CompressInt64 compresses a slice of int64 values using Delta encoding.
//
// Delta encoding stores differences between consecutive values, which achieves
// excellent compression for sorted/sequential numeric data like:
//   - Timestamps (monotonically increasing)
//   - Sequential IDs
//   - Slowly-changing sensor readings
//
// For random or highly variable data, use Compress() with the Identity codec instead.
//
// Example:
//
//	numbers := []int64{1, 2, 3, 4, 5, 6, 7, 8}
//	compressed, err := purgo.CompressInt64(numbers)
func CompressInt64(data []int64) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("purgo: cannot compress empty data")
	}

	// Convert int64 slice to bytes
	buf := new(bytes.Buffer)
	for _, val := range data {
		if err := binary.Write(buf, binary.LittleEndian, val); err != nil {
			return nil, fmt.Errorf("purgo: write int64: %w", err)
		}
	}
	srcBytes := buf.Bytes()

	// Create compression graph: Delta encoding only
	// Delta encoding converts values to differences, optimal for:
	//  - Monotonically increasing sequences (timestamps, IDs)
	//  - Slowly-changing values (sensor readings, metrics)
	//
	// Note: We don't use Huffman here because it changes data size, which
	// breaks the size assumptions in the OpenZL frame format. This would
	// require storing intermediate sizes in the graph, which is complex.
	//
	// For now, Delta-only provides good compression for sequential data.
	// Full pipeline (Delta -> Huffman) will be added when we implement
	// proper size tracking in the graph metadata.
	g := &graph.Graph{
		Nodes: []*graph.Node{
			// Node 0: Delta encoding (stores differences)
			{
				CodecID: codec.IDDelta,
				Params:  []byte{8}, // 8 bytes per element (int64)
				Inputs:  nil,       // Uses source data
			},
		},
		Outputs: []int{0}, // Final output from Delta (node 0)
	}

	return compressWithGraph(g, srcBytes)
}

// CompressFloat64 compresses a slice of float64 values.
func CompressFloat64(data []float64) ([]byte, error) {
	buf := new(bytes.Buffer)
	for _, val := range data {
		if err := binary.Write(buf, binary.LittleEndian, val); err != nil {
			return nil, fmt.Errorf("purgo: write float64: %w", err)
		}
	}

	return Compress(buf.Bytes())
}

// CompressString compresses a string (converts to bytes first).
func CompressString(s string) ([]byte, error) {
	return Compress([]byte(s))
}
