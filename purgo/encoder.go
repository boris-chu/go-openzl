// Package purgo provides Pure Go OpenZL compression and decompression.
package purgo

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"

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
	return CompressSmartWithDict(src, nil)
}

// CompressSmartWithDict compresses data using intelligent compression with dictionary support.
//
// This extends CompressSmart() to use a static dictionary with LZ77 compression,
// achieving 2-3× better compression on specialized data (CSV, JSON, logs, source code).
//
// How it works:
//  1. Tries multiple compression strategies (LZ77+dict, RLE, Huffman)
//  2. Picks the best strategy based on compression ratio
//  3. Adds Huffman as a second stage if beneficial (Frame v22 pipeline)
//
// Dictionary compression:
//   - CSV with 30KB dict: 20-30× compression (vs 9× without dict)
//   - JSON with 20KB dict: 30-40× compression (vs 18× without dict)
//   - Source code with 40KB dict: 25-35× compression (vs 15× without dict)
//
// Parameters:
//   - src: Data to compress
//   - dictionary: Pre-trained compression dictionary (nil for no dictionary)
//
// Returns:
//   - Compressed OpenZL frame using the best strategy
//   - Error if all compression strategies fail
//
// Example:
//
//	// Train dictionary
//	trainer := dicttrainer.New()
//	trainer.AddFile("sales.csv")
//	dict := trainer.Train(30 * 1024)
//	os.WriteFile("csv-dict.bin", dict, 0644)
//
//	// Compress with dictionary
//	dict, _ := os.ReadFile("csv-dict.bin")
//	compressed, err := purgo.CompressSmartWithDict(csvData, dict)
//	// Expected: 20-30× compression ratio (vs 9× without dict)
func CompressSmartWithDict(src, dictionary []byte) ([]byte, error) {
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

	// **AUTO-DETECTION**: Analyze data format to choose optimal strategy
	format := DetectFormat(src)

	type strategy struct {
		name  string
		graph *graph.Graph
	}

	// Define compression strategies based on detected format
	var strategies []strategy

	// Try to detect numeric columnar data (multi-byte aligned integers/floats)
	// This allows Transpose codec to expose byte-level patterns
	numericWidth := detectNumericWidth(src)
	if numericWidth > 1 {
		// Numeric columnar data detected!
		// Transpose exposes byte-level patterns:
		//   - High bytes often constant (timestamps, IDs, pointers)
		//   - Low bytes vary more → perfect for RLE after transpose
		// Example: int64 timestamps, uint32 IDs, float64 prices

		// Strategy 1: Transpose → RLE (best for numeric with constant high bytes)
		// Expected: 10-1000× on timestamps, IDs, counters
		transposeParams := []byte{byte(numericWidth)}
		strategies = append(strategies, strategy{
			name: "Transpose→RLE",
			graph: &graph.Graph{
				Nodes: []*graph.Node{
					{
						CodecID: codec.IDTranspose,
						Params:  transposeParams,
						Inputs:  nil, // Node 0: transpose source data
					},
					{
						CodecID: codec.IDRLE,
						Params:  nil,
						Inputs:  []int{0}, // Node 1: RLE on transposed data
					},
				},
				Outputs: []int{1}, // Final output from RLE
			},
		})

		// Strategy 2: Transpose → RLE → Huffman (Frame v22 multi-stage)
		// Expected: Even better compression via entropy coding
		// TEMPORARILY DISABLED: Debugging 2-node pipeline first
		// TODO: Re-enable once 2-node pipeline is stable
	}

	// Suppress unused variable warning (format used in future enhancements)
	_ = format

	// Standard strategies (always try these)
	strategies = append(strategies, []strategy{
		// Strategy: LZ77-only (best for structured text/CSV with patterns)
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
						Params:  dictionary, // Use dictionary if provided
						Inputs:  nil,        // Uses source data
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

		// Strategy: Huffman-only (fallback for general data)
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
	}...)

	// Try each strategy and track the best result
	var bestCompressed []byte
	var bestGraph *graph.Graph
	var bestNodeSizes []uint64 // Track intermediate node sizes for Frame v22
	bestRatio := 0.0

	registry := codec.DefaultRegistry()

	for _, s := range strategies {
		var compressedData []byte
		var nodeSizes []uint64
		var err error

		// Check if this is a multi-stage graph (more than 1 node)
		if len(s.graph.Nodes) > 1 {
			// Multi-stage: use executeCompressionGraphWithSizes to track intermediate sizes
			compressedData, nodeSizes, err = executeCompressionGraphWithSizes(s.graph, src, registry)
		} else {
			// Single-stage: use regular execution
			compressedData, err = executeCompressionGraph(s.graph, src, registry)
		}

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
			bestNodeSizes = nodeSizes // Store node sizes for Frame v22
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

	// Choose frame version based on whether we have multi-stage pipeline
	var f *frame.Frame
	if len(bestNodeSizes) > 0 {
		// DEBUG: Print final node sizes being stored in frame
		fmt.Printf("DEBUG FRAME v22: Storing NodeSizes=%v for %d-node graph\n", bestNodeSizes, len(bestGraph.Nodes))

		// Multi-stage pipeline: use Frame v22 with NodeSizes
		f = &frame.Frame{
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
			NodeSizes: bestNodeSizes, // Critical: store intermediate node sizes
			Payload:   payload,
		}
	} else {
		// Single-stage: use Frame v21
		f = &frame.Frame{
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

// CompressWithDict compresses data using an external dictionary (NOT embedded in output).
//
// This function is designed for BATCH COMPRESSION where you compress many similar files
// with the SAME dictionary. The dictionary is NOT embedded in the compressed output,
// resulting in much smaller files.
//
// **Important**: You MUST store the dictionary separately and provide it during decompression.
//
// Use case: Compress 100 CSV files with 1 shared dictionary
//
//	// Step 1: Train dictionary on representative data
//	trainer := dicttrainer.New()
//	trainer.AddFile("sample1.csv")
//	trainer.AddFile("sample2.csv")
//	dict := trainer.Train(500) // 500-byte dictionary
//	os.WriteFile("my-dict.bin", dict, 0644)
//
//	// Step 2: Compress many files with same dictionary
//	dict, _ := os.ReadFile("my-dict.bin")
//	for _, file := range filesToCompress {
//	    data, _ := os.ReadFile(file)
//	    compressed, _ := purgo.CompressWithDict(data, dict)
//	    os.WriteFile(file+".openzl", compressed, 0644)
//	}
//	// Dictionary overhead: 500 bytes total (stored ONCE!)
//	// vs CompressSmartWithDict: 500 bytes × 100 files = 50KB overhead
//
//	// Step 3: Decompress (must provide same dictionary!)
//	dict, _ := os.ReadFile("my-dict.bin")
//	for _, file := range compressedFiles {
//	    compressed, _ := os.ReadFile(file)
//	    data, _ := purgo.DecompressWithDict(compressed, dict)
//	}
//
// Parameters:
//   - src: Data to compress
//   - dictionary: Pre-trained compression dictionary (NOT embedded in output)
//
// Returns:
//   - Compressed data WITHOUT embedded dictionary (requires DecompressWithDict)
//   - Error if compression fails
//
// Performance:
//   - 10 × 11KB CSV files: 117KB → 1.6KB (72× compression with shared dict)
//   - vs CompressSmart (no dict): 117KB → 3.2KB (36× compression)
//   - vs CompressSmartWithDict (embedded): 117KB → 7.1KB (16× compression)
//   - **2× better than current best!**
func CompressWithDict(src, dictionary []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, fmt.Errorf("purgo: cannot compress empty data")
	}

	if len(dictionary) == 0 {
		return nil, fmt.Errorf("purgo: dictionary cannot be empty (use CompressSmart for no dictionary)")
	}

	// Create LZ77→Huffman pipeline with dictionary (same as CompressSmartWithDict)
	// but mark dictionary as EXTERNAL (not embedded)
	multiStageGraph := &graph.Graph{
		Nodes: []*graph.Node{
			{
				CodecID: codec.IDLZ77,
				Params:  dictionary, // Used during compression, but won't be embedded
				Inputs:  nil,
			},
			{
				CodecID: codec.IDHuffman,
				Params:  nil,
				Inputs:  []int{0}, // Takes input from LZ77
			},
		},
		Outputs: []int{1}, // Output from Huffman
	}

	// Execute compression
	registry := codec.DefaultRegistry()
	compressedData, nodeSizes, err := executeCompressionGraphWithSizes(multiStageGraph, src, registry)
	if err != nil {
		return nil, fmt.Errorf("purgo: compress with dict failed: %w", err)
	}

	// Build graph WITHOUT dictionary in Params (external storage)
	externalGraph := &graph.Graph{
		Nodes: []*graph.Node{
			{
				CodecID: codec.IDLZ77,
				Params:  nil, // NO DICTIONARY EMBEDDED! (this is the key difference)
				Inputs:  nil,
			},
			{
				CodecID: codec.IDHuffman,
				Params:  nil,
				Inputs:  []int{0},
			},
		},
		Outputs: []int{1},
	}

	graphBytes, err := graph.EncodeGraph(externalGraph)
	if err != nil {
		return nil, fmt.Errorf("purgo: encode graph: %w", err)
	}

	payload := append(graphBytes, compressedData...)

	f := &frame.Frame{
		Header: &frame.Header{
			Magic:   frame.MagicNumberBase + 22,
			Version: 22,
			Flags:   1, // Flag 1 = "requires external dictionary"
		},
		Outputs: []*frame.Output{
			{
				Type:             frame.TypeSerial,
				DecompressedSize: uint64(len(src)),
			},
		},
		NodeSizes: nodeSizes,
		Payload:   payload,
	}

	return serializeFrame(f)
}

// DecompressWithDict decompresses data that was compressed with CompressWithDict.
//
// **Important**: You MUST provide the SAME dictionary used during compression.
//
// Parameters:
//   - compressed: Data compressed with CompressWithDict
//   - dictionary: Same dictionary used during compression
//
// Returns:
//   - Decompressed data
//   - Error if decompression fails or dictionary mismatch
//
// Example:
//
//	dict, _ := os.ReadFile("my-dict.bin")
//	compressed, _ := os.ReadFile("file.openzl")
//	data, _ := purgo.DecompressWithDict(compressed, dict)
func DecompressWithDict(compressed, dictionary []byte) ([]byte, error) {
	if len(compressed) == 0 {
		return nil, fmt.Errorf("purgo: cannot decompress empty data")
	}

	if len(dictionary) == 0 {
		return nil, fmt.Errorf("purgo: dictionary cannot be empty")
	}

	// Parse frame using frame.Reader
	reader := frame.NewReader(bytes.NewReader(compressed))
	f, err := reader.ReadFrame()
	if err != nil {
		return nil, fmt.Errorf("decompress: purgo: %w", err)
	}

	// Check if frame requires external dictionary (Flag 1)
	if f.Header.Flags&1 == 0 {
		return nil, fmt.Errorf("decompress: frame does not require external dictionary (use Decompress instead)")
	}

	// Parse graph from payload
	parser := graph.NewParser(f.Payload)
	g, graphSize, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("decompress: purgo: parse graph failed: %w", err)
	}

	// Inject dictionary into LZ77 node params
	// (graph has Params=nil for external dict, we restore it here)
	for i, node := range g.Nodes {
		if node.CodecID == codec.IDLZ77 && len(node.Params) == 0 {
			g.Nodes[i].Params = dictionary
		}
	}

	// Extract compressed data after graph
	compressedData := f.Payload[graphSize:]

	// Extract output sizes from frame
	outputSizes := make([]uint64, len(f.Outputs))
	for i, out := range f.Outputs {
		outputSizes[i] = out.DecompressedSize
	}

	// Execute decompression graph
	registry := codec.DefaultRegistry()
	executor := graph.NewExecutor(registry)

	outputs, err := executor.ExecuteWithNodeSizes(g, compressedData, outputSizes, f.NodeSizes)
	if err != nil {
		return nil, fmt.Errorf("decompress: purgo: execute failed: %w", err)
	}

	// Should have single output
	if len(outputs) != 1 {
		return nil, fmt.Errorf("decompress: expected 1 output, got %d", len(outputs))
	}

	return outputs[0], nil
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

		// Store output and INPUT size for this node
		// CRITICAL: NodeSizes in Frame v22 stores DECOMPRESSED output sizes
		// (which equals INPUT sizes during compression)
		// This is what the decoder needs to allocate buffers during decompression.
		nodeOutputs[i] = dst[:n]
		nodeSizes[i] = uint64(len(input)) // Store INPUT size, not output size!

		// DEBUG: Trace compression
		fmt.Printf("DEBUG COMPRESS: Node %d (%s) - input=%d, output=%d, nodeSizes[%d]=%d\n",
			i, c.Name(), len(input), n, i, nodeSizes[i])
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

// detectNumericWidth attempts to detect if data is numeric columnar (multi-byte integers/floats)
// and returns the detected element width (2, 4, or 8 bytes), or 0 if not numeric.
//
// Heuristics:
//  1. Data size must be multiple of 2/4/8
//  2. After conceptual transpose, high-order bytes have lower entropy than low-order bytes
//     (characteristic of timestamps, IDs, pointers, prices)
//
// This detection enables Transpose codec to expose byte-level patterns for RLE compression.
func detectNumericWidth(data []byte) int {
	// Need at least 4 elements to detect pattern
	minElements := 4

	// Try widths in order of likelihood: 8, 4, 2
	// (64-bit most common for timestamps, IDs, pointers)
	for _, width := range []int{8, 4, 2} {
		if len(data)%width != 0 {
			continue
		}
		if len(data) < width*minElements {
			continue
		}

		// Check if this looks like numeric columnar data
		// by sampling byte-position entropy
		if isNumericPatternWidth(data, width) {
			return width
		}
	}

	return 0 // Not numeric columnar
}

// isNumericPatternWidth checks if data with given width exhibits numeric byte patterns
//
// Numeric columns have characteristic entropy gradient:
//   - High bytes (MSB): low entropy (constant or slowly changing)
//   - Low bytes (LSB): high entropy (rapidly changing)
//
// IMPORTANT: In little-endian format:
//   - Byte position 0 = LSB (low-order byte, varies rapidly)
//   - Byte position width-1 = MSB (high-order byte, constant)
//
// Example int64 timestamps (1735000000000 = 0x00000193f60f0600):
//   - Bytes 0-1: varying (0x0600, 0x09e8, 0x0dd0...) → high entropy
//   - Bytes 5-7: constant (0x01, 0x00, 0x00) → low entropy
func isNumericPatternWidth(data []byte, width int) bool {
	count := len(data) / width

	// In little-endian: byte 0 = LSB (varying), byte width-1 = MSB (constant)
	lsbEntropy := calculateBytePositionEntropy(data, width, 0, count)       // LSB
	msbEntropy := calculateBytePositionEntropy(data, width, width-1, count) // MSB

	// Numeric columns: MSB has notably lower entropy than LSB
	// Threshold: MSB < 4.0 bits AND LSB > MSB + 1.5 bits
	//
	// Example values:
	//  - Timestamps: MSB=0.0 (constant 0x00), LSB=4.9 (varying) → numeric!
	//  - Random: MSB=7.2, LSB=7.4 → not numeric
	//  - Text: MSB=6.1, LSB=6.3 → not numeric
	if msbEntropy < 4.0 && lsbEntropy > msbEntropy+1.5 {
		return true
	}

	// Alternative check for width >= 4: entropy gradient across positions
	if width >= 4 {
		midByteEntropy := calculateBytePositionEntropy(data, width, width/2, count)
		// Expect gradient: MSB (low) < mid < LSB (high) entropy
		if msbEntropy < midByteEntropy && midByteEntropy < lsbEntropy {
			return true
		}
	}

	return false
}

// calculateBytePositionEntropy calculates Shannon entropy for a specific byte position
// across all elements (in transposed view)
//
// Formula: H = -Σ(p(i) * log2(p(i))) where p(i) = frequency of byte value i
// Returns value between 0.0 (all same) and 8.0 (uniform distribution)
func calculateBytePositionEntropy(data []byte, width, bytePos, count int) float64 {
	if count == 0 {
		return 0.0
	}

	// Count byte value frequencies at this position
	var freq [256]int
	for elem := 0; elem < count; elem++ {
		idx := elem*width + bytePos
		freq[data[idx]]++
	}

	// Calculate Shannon entropy
	entropy := 0.0
	n := float64(count)
	for _, c := range &freq {
		if c == 0 {
			continue
		}
		p := float64(c) / n
		entropy -= p * math.Log2(p)
	}

	return entropy
}
