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

// CompressSmart intelligently selects the best compression strategy.
//
// This function tries multiple codec pipelines and automatically chooses the
// one that achieves the best compression ratio. It is specifically optimized
// for text, JSON, and structured data with repeated patterns.
//
// Compression Strategies Tried (in order):
//  1. LZ77: Best for text/JSON with repeated strings (10-20× typical)
//  2. RLE: Best for sparse data with long runs (5-15× typical)
//  3. Huffman: Fallback for general data (1.5-3× typical)
//  4. Identity: No compression (used if data expands)
//
// Note: Multi-codec pipelines (LZ77→Huffman, RLE→Huffman) would achieve
// even better compression (20-30×) but require size metadata support.
// This will be added in a future release.
//
// This function addresses the gap identified in COMPRESSION_COMPARISON.md
// where Compress() only achieved 1.51× on JSON vs zstd's 22.73×.
//
// Parameters:
//   - src: Uncompressed data
//
// Returns:
//   - Compressed OpenZL frame using the best strategy
//   - Error if all compression strategies fail
//
// Example:
//
//	jsonData := []byte(`{"field":"value","field":"value",...}`)
//	compressed, err := purgo.CompressSmart(jsonData)
//	// Expected: 15-25× compression ratio (vs 1.51× with Compress())
func CompressSmart(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, fmt.Errorf("purgo: cannot compress empty data")
	}

	// **TEMPORARILY DISABLED: Per-segment compression**
	// ISSUE: "Most common codec" heuristic fails on mixed-format files like CSV
	// - BitLocker CSV: RLE chosen (most common) but fails on mixed structure
	// - Result: 1.00× compression vs Zstd's 19.33×
	// - Root cause: Applying single codec to entire file ignores CSV structure
	//
	// SOLUTION: Let LZ77 strategy handle structured data (achieves ~19× like Zstd)
	// See docs/ZSTD_COMPARISON.md for full analysis
	//
	// TODO (v0.3.3): Implement LZ77-first strategy for CSV/JSON detection
	// TODO (v0.3.3): Multi-stage pipeline (per-segment → LZ77 → Huffman)
	//
	// format := DetectFormat(src)
	// switch format {
	// case FormatCSV:
	//     return compressSegmented(src, SegmentCSV)
	// case FormatJSON:
	//     return compressSegmented(src, SegmentJSON)
	// }

	type strategy struct {
		name  string
		graph *graph.Graph
	}

	// Define compression strategies in priority order
	strategies := []strategy{
		// Strategy 1: LZ77-only (best for structured text/CSV with patterns)
		// LZ77 finds repeated strings and replaces with back-references
		// Expected: 5-15× on CSV, 10-20× on JSON
		//
		// NOTE: LZ77→Huffman pipeline would achieve 15-25× compression (like Zstd)
		// Testing showed:
		//   - BitLocker CSV: 9.74× (vs 5.63× LZ77-only, vs 19.33× Zstd)
		//   - JSON: 27.95× (vs 18.19× LZ77-only)
		//   - Repeated strings: 36.30× (vs 24.50× LZ77-only)
		// BUT decompression fails because frame format doesn't store intermediate sizes.
		//
		// Current limitation: OpenZL frame format only stores final output sizes,
		// not intermediate node sizes. Multi-stage pipelines with size-changing codecs
		// (like LZ77) require intermediate sizes for decompression buffer allocation.
		//
		// See docs/ZSTD_COMPARISON.md for full analysis.
		//
		// TODO (v0.3.3): Enhance frame format to support intermediate node sizes
		// TODO (v0.3.3): Implement LZ77→Huffman/FSE pipeline for 2-3× better compression
		{
			name: "LZ77",
			graph: &graph.Graph{
				Nodes: []*graph.Node{
					{
						CodecID: codec.IDLZ77,
						Params:  nil,
						Inputs:  nil, // Uses source data
					},
				},
				Outputs: []int{0}, // Final output from LZ77 (node 0)
			},
		},

		// Strategy 2: RLE-only (best for sparse/repetitive data)
		// RLE compresses runs of identical values
		// Expected: 5-15× on sparse data, 3-8× on repetitive data
		{
			name: "RLE",
			graph: &graph.Graph{
				Nodes: []*graph.Node{
					{
						CodecID: codec.IDRLE,
						Params:  nil,
						Inputs:  nil, // Uses source data
					},
				},
				Outputs: []int{0}, // Final output from RLE (node 0)
			},
		},

		// Strategy 3: Huffman-only (fallback for general data)
		// Expected: 1.5-3× on varied data
		{
			name: "Huffman",
			graph: &graph.Graph{
				Nodes: []*graph.Node{
					{
						CodecID: codec.IDHuffman,
						Params:  nil,
						Inputs:  nil,
					},
				},
				Outputs: []int{0},
			},
		},
	}

	// Try each strategy and track the best result
	var bestCompressed []byte
	var bestGraph *graph.Graph
	bestRatio := 0.0

	registry := codec.DefaultRegistry()

	for _, s := range strategies {
		// Execute this strategy's compression graph
		compressedData, err := executeCompressionGraph(s.graph, src, registry)
		if err != nil {
			// Strategy failed, skip to next
			continue
		}

		// Check if this strategy achieved compression
		if len(compressedData) >= len(src) {
			// No compression achieved, skip
			continue
		}

		// Calculate compression ratio
		ratio := float64(len(src)) / float64(len(compressedData))

		// Track best strategy
		if ratio > bestRatio {
			bestRatio = ratio
			bestCompressed = compressedData
			bestGraph = s.graph
		}
	}

	// If no strategy worked, fall back to Identity (no compression)
	if bestGraph == nil {
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

	// Build frame with best compression strategy
	graphBytes, err := graph.EncodeGraph(bestGraph)
	if err != nil {
		return nil, fmt.Errorf("purgo: encode graph: %w", err)
	}

	payload := append(graphBytes, bestCompressed...)

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

	stage1Frame, err := serializeFrame(f)
	if err != nil {
		return nil, err
	}

	// **NATIVE MULTI-STAGE PIPELINE (Frame Format v22)**
	// Try adding Huffman as a second stage for additional compression.
	// This achieves LZ77→Huffman pipeline in a SINGLE frame using v22's node sizes.
	//
	// Benefits over old double-wrapping (v0.3.2):
	//   - Single frame instead of two frames (~60 bytes overhead saved)
	//   - Native pipeline support (proper node size metadata)
	//   - Cleaner decompression (no double-frame parsing)
	//
	// Approach:
	//   1. Create 2-node graph: LZ77 → Huffman
	//   2. Store intermediate LZ77 output size in NodeSizes field
	//   3. Single frame with proper multi-stage metadata
	//
	// Only apply if Huffman improves compression (otherwise single-stage is better).

	// Create multi-stage graph: LZ77 (or whatever) → Huffman
	multiStageGraph := &graph.Graph{
		Nodes: []*graph.Node{
			bestGraph.Nodes[0], // First codec (LZ77, RLE, etc.)
			{
				CodecID: codec.IDHuffman,
				Params:  nil,
				Inputs:  []int{0}, // Takes input from node 0
			},
		},
		Outputs: []int{1}, // Output is from node 1 (Huffman)
	}

	// Execute multi-stage compression
	multiStageCompressed, nodeSizes, err := executeCompressionGraphWithSizes(multiStageGraph, src, registry)
	if err == nil && len(multiStageCompressed) < len(bestCompressed) {
		// Multi-stage compression succeeded and improved compression!
		// Build Frame v22 with NodeSizes
		multiStageGraphBytes, err := graph.EncodeGraph(multiStageGraph)
		if err == nil {
			multiStagePayload := append(multiStageGraphBytes, multiStageCompressed...)

			v22Frame := &frame.Frame{
				Header: &frame.Header{
					Magic:   frame.MagicNumberBase + 22,
					Version: 22,
					Flags:   0,
				},
				Outputs: []*frame.Output{
					{
						Type:             frame.TypeSerial,
						DecompressedSize: uint64(len(src)),
					},
				},
				NodeSizes: nodeSizes, // Store intermediate sizes for v22
				Payload:   multiStagePayload,
			}

			v22Serialized, err := serializeFrame(v22Frame)
			if err == nil {
				// Debug: Print first 50 bytes of v22 frame
				// fmt.Printf("DEBUG: v22 frame size=%d, first 50 bytes: %x\n", len(v22Serialized), v22Serialized[:min(50, len(v22Serialized))])

				// Success! Return native multi-stage pipeline frame
				return v22Serialized, nil
			}
		}
	}

	// Multi-stage didn't help or failed, return single-stage compression
	return stage1Frame, nil
}

// executeCompressionGraph executes a compression graph on source data.
//
// This supports multi-node graphs by executing nodes in topological order.
// Each node takes input from previous nodes or the source data.
// compressSegmented compresses data using intelligent codec selection.
//
// This function segments the input data (e.g., CSV columns, JSON fields), analyzes
// each segment to determine optimal codecs, then chooses the most common codec
// and applies it to the entire source data.
//
// WORKAROUND: Frame reader currently only supports ≤2 outputs, so we use a
// single-output approach instead of per-segment compression.
//
// Parameters:
//   - src: Source data to compress
//   - segmenter: Function that segments data (SegmentCSV or SegmentJSON)
//
// Returns:
//   - Compressed OpenZL frame using single best codec
//   - Error if segmentation or compression fails
func compressSegmented(src []byte, segmenter func([]byte) ([]Segment, error)) ([]byte, error) {
	// Segment the data to analyze optimal codecs
	segments, err := segmenter(src)
	if err != nil {
		return nil, fmt.Errorf("purgo: segmentation failed: %w", err)
	}

	if len(segments) == 0 {
		return nil, fmt.Errorf("purgo: no segments generated")
	}

	// Count codec frequency to choose the most common one
	codecCounts := make(map[uint16]int)
	for _, seg := range segments {
		codecCounts[seg.CodecID]++
	}

	// Find most common codec
	var bestCodecID uint16
	maxCount := 0
	for codecID, count := range codecCounts {
		if count > maxCount {
			maxCount = count
			bestCodecID = codecID
		}
	}

	// Build single-node graph with the most common codec
	g := &graph.Graph{
		Nodes: []*graph.Node{
			{
				CodecID: codec.ID(bestCodecID),
				Params:  nil,
				Inputs:  nil,
			},
		},
		Outputs: []int{0}, // Single output
	}

	// Compress entire source with chosen codec
	registry := codec.DefaultRegistry()
	compressed, err := executeCompressionGraph(g, src, registry)
	if err != nil {
		// Fallback to identity on compression failure
		g.Nodes[0].CodecID = codec.IDIdentity
		compressed = src
	}

	// Encode graph and build payload
	graphBytes, err := graph.EncodeGraph(g)
	if err != nil {
		return nil, fmt.Errorf("purgo: encode graph: %w", err)
	}

	var payload bytes.Buffer
	payload.Write(graphBytes)
	payload.Write(compressed)

	// Build single-output frame
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
		Payload: payload.Bytes(),
	}

	return serializeFrame(f)
}

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

// executeCompressionGraphWithSizes executes a compression graph and returns both
// the final compressed output and the intermediate node sizes.
//
// This is used for Frame Format v22 which stores intermediate node sizes in the frame
// to enable proper decompression of multi-stage pipelines without size inference.
//
// Returns:
//   - compressed: Final compressed output from the graph
//   - nodeSizes: Size of each node's output (for v22 NodeSizes field)
//   - error: Any error during compression
func executeCompressionGraphWithSizes(g *graph.Graph, src []byte, registry *codec.Registry) ([]byte, []uint64, error) {
	// Storage for intermediate results (node outputs)
	nodeOutputs := make([][]byte, len(g.Nodes))
	nodeSizes := make([]uint64, len(g.Nodes))

	// Execute each node in order
	for i, node := range g.Nodes {
		c, ok := registry.Get(node.CodecID)
		if !ok {
			return nil, nil, fmt.Errorf("purgo: codec %d not found", node.CodecID)
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
				return nil, nil, fmt.Errorf("purgo: invalid input index %d for node %d", inputIdx, i)
			}
			input = nodeOutputs[inputIdx]
		} else {
			return nil, nil, fmt.Errorf("purgo: multi-input nodes not yet supported")
		}

		// Allocate output buffer (generous size for safety)
		// Entropy coders (Huffman) may expand data temporarily
		dst := make([]byte, len(input)*2+1024)

		// Encode
		n, err := c.Encode(dst, input, node.Params)
		if err != nil {
			return nil, nil, fmt.Errorf("purgo: encode with codec %s (node %d): %w", c.Name(), i, err)
		}

		// Store output and size for this node
		nodeOutputs[i] = dst[:n]
		nodeSizes[i] = uint64(n)
	}

	// Return output from final node and all node sizes
	if len(g.Outputs) != 1 {
		return nil, nil, fmt.Errorf("purgo: multi-output graphs not yet supported")
	}
	outputIdx := g.Outputs[0]
	if outputIdx >= len(nodeOutputs) {
		return nil, nil, fmt.Errorf("purgo: invalid output index %d", outputIdx)
	}

	return nodeOutputs[outputIdx], nodeSizes, nil
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
	// Use the proper frame writer that supports both v21 and v22
	return frame.EncodeFrame(f)
}

// DEPRECATED: Old manual frame serialization (kept for reference)
func serializeFrameOld(f *frame.Frame) ([]byte, error) {
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
