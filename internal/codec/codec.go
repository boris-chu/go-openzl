// Package codec provides the interface and registry for OpenZL codecs.
//
// Codecs are the building blocks of OpenZL compression. Each codec performs
// a specific transformation (delta encoding, entropy coding, etc.). Multiple
// codecs are chained together in a compression graph to achieve the final
// compressed output.
package codec

import (
	"fmt"
	"io"
)

// Codec name constants
const (
	nameIdentity = "Identity"
	nameConstant = "Constant"
	nameDelta    = "Delta"
	nameZigZag   = "ZigZag"
	nameBitpack  = "Bitpack"
)

// Codec is the interface all OpenZL codecs must implement.
//
// A codec transforms data from one representation to another. During compression,
// it encodes data into a more compact form. During decompression, it decodes
// data back to the original form.
type Codec interface {
	// ID returns the unique identifier for this codec.
	// This ID is stored in the frame to identify which codec to use.
	ID() ID

	// Name returns the human-readable name of this codec.
	Name() string

	// Decode transforms compressed input to decompressed output.
	//
	// Parameters:
	//   dst    - Destination buffer for decompressed data
	//   src    - Source buffer containing compressed data
	//   params - Codec-specific parameters (from frame graph)
	//
	// Returns the number of bytes written to dst, or an error.
	//
	// The decoder must:
	// - Not write more than len(dst) bytes
	// - Return the exact number of bytes written
	// - Return an error if dst is too small
	Decode(dst, src, params []byte) (int, error)

	// Encode transforms decompressed input to compressed output.
	//
	// This is used during compression (Phase 3). For Phase 2, we only
	// implement Decode to enable decompression.
	//
	// Parameters are the same as Decode but reversed.
	Encode(dst, src, params []byte) (int, error)

	// PreservesSize returns true if this codec always produces output
	// of the same size as its input.
	//
	// Size-preserving codecs (Identity, Delta, ZigZag, Constant) allow
	// for size inference in multi-node pipelines. Size-changing codecs
	// (Huffman, FSE, Bitpack) require explicit size metadata.
	PreservesSize() bool
}

// ID uniquely identifies a codec within OpenZL.
//
// These IDs must match the C library's codec IDs for compatibility.
type ID uint16

// Codec IDs
//
// These values are part of the OpenZL wire format and must remain stable.
// IDs < 256 are reserved for standard OpenZL codecs.
// IDs >= 256 are available for custom codecs.
const (
	// IDIdentity is the passthrough codec (no transformation)
	IDIdentity ID = 0

	// IDConstant fills output with a constant value
	IDConstant ID = 1

	// IDDelta performs delta encoding (store differences)
	IDDelta ID = 2

	// IDZigZag performs zigzag encoding (signed to unsigned)
	IDZigZag ID = 3

	// IDBitpack packs integers into minimal bits
	IDBitpack ID = 4

	// IDTranspose transposes 2D data
	IDTranspose ID = 5

	// IDQuantize performs lossy quantization
	IDQuantize ID = 6

	// IDFSE is Finite State Entropy coding
	IDFSE ID = 10

	// IDHuffman is Huffman coding
	IDHuffman ID = 11

	// IDLZ77 is LZ77 dictionary compression
	IDLZ77 ID = 12

	// IDRLE is Run-Length Encoding
	IDRLE ID = 13

	// IDRangePack compresses numeric data by subtracting min and packing to narrowest type
	IDRangePack ID = 14

	// IDPrefix extracts common prefixes from consecutive strings
	IDPrefix ID = 15

	// IDParseInt parses integer strings to int64 binary values
	IDParseInt ID = 16

	// IDZstd is Zstandard compression
	IDZstd ID = 20

	// More codecs will be added as we implement them
)

// String returns the name of the codec ID.
func (id ID) String() string {
	switch id {
	case IDIdentity:
		return nameIdentity
	case IDConstant:
		return nameConstant
	case IDDelta:
		return nameDelta
	case IDZigZag:
		return nameZigZag
	case IDBitpack:
		return nameBitpack
	case IDTranspose:
		return "Transpose"
	case IDQuantize:
		return "Quantize"
	case IDFSE:
		return "FSE"
	case IDHuffman:
		return "Huffman"
	case IDLZ77:
		return "LZ77"
	case IDRLE:
		return "RLE"
	case IDRangePack:
		return "RangePack"
	case IDPrefix:
		return "Prefix"
	case IDParseInt:
		return "ParseInt"
	case IDZstd:
		return "Zstd"
	default:
		return fmt.Sprintf("Unknown(%d)", id)
	}
}

// Registry manages the collection of available codecs.
//
// During decompression, the registry is used to look up codecs by ID
// to execute the compression graph.
type Registry struct {
	codecs map[ID]Codec
}

// NewRegistry creates a new codec registry.
func NewRegistry() *Registry {
	return &Registry{
		codecs: make(map[ID]Codec),
	}
}

// Register adds a codec to the registry.
//
// If a codec with the same ID already exists, it is replaced.
func (r *Registry) Register(codec Codec) {
	r.codecs[codec.ID()] = codec
}

// Get retrieves a codec by ID.
//
// Returns the codec and true if found, nil and false otherwise.
func (r *Registry) Get(id ID) (Codec, bool) {
	codec, ok := r.codecs[id]
	return codec, ok
}

// Has checks if a codec is registered.
func (r *Registry) Has(id ID) bool {
	_, ok := r.codecs[id]
	return ok
}

// MustGet retrieves a codec by ID or panics if not found.
//
// This is useful in tests or when you know the codec must exist.
func (r *Registry) MustGet(id ID) Codec {
	codec, ok := r.codecs[id]
	if !ok {
		panic(fmt.Sprintf("codec %s not registered", id))
	}
	return codec
}

// IDs returns all registered codec IDs.
func (r *Registry) IDs() []ID {
	ids := make([]ID, 0, len(r.codecs))
	for id := range r.codecs {
		ids = append(ids, id)
	}
	return ids
}

// DefaultRegistry returns a registry with all standard codecs.
//
// For Phase 2, this starts with just the Identity codec.
// More codecs will be added as we implement them.
func DefaultRegistry() *Registry {
	reg := NewRegistry()
	// Phase 2: Register codecs (complete)
	reg.Register(NewIdentity())
	reg.Register(NewConstant(4)) // Default to 4-byte elements
	reg.Register(NewDelta(8))    // Default to 8-byte (uint64) elements
	reg.Register(NewZigZag(4))   // Default to 4-byte (int32) elements
	reg.Register(NewBitpack(4))  // Default to 4-byte (uint32) elements

	// Phase 3 (Entropy Coding): FSE, Huffman, ANS, Range Coding
	reg.Register(NewFSE())     // FSE (Finite State Entropy) - Klaus Post library
	reg.Register(NewHuffman()) // Huffman (huff0) - Klaus Post library

	// Phase 5 (Advanced Codecs): LZ77, RLE, Transpose, ROLZ, etc.
	reg.Register(NewLZ77())      // LZ77 dictionary compression - critical for JSON/text
	reg.Register(NewRLE())       // RLE run-length encoding - critical for sparse/repetitive data
	reg.Register(NewTranspose()) // Transpose byte streams - exposes patterns for other codecs
	reg.Register(NewRangePack()) // RangePack numeric compression - critical for timestamps/IDs
	reg.Register(NewPrefix())    // Prefix extraction - critical for URLs/paths
	reg.Register(NewParseInt())  // ParseInt text-to-binary - critical for CSV parsing

	return reg
}

// Common errors
var (
	// ErrBufferTooSmall indicates the destination buffer is too small
	ErrBufferTooSmall = fmt.Errorf("destination buffer too small")

	// ErrInvalidParams indicates codec parameters are invalid
	ErrInvalidParams = fmt.Errorf("invalid codec parameters")

	// ErrCorruptedData indicates the input data is corrupted
	ErrCorruptedData = fmt.Errorf("corrupted data")

	// ErrCodecNotFound indicates a codec ID is not in the registry
	ErrCodecNotFound = fmt.Errorf("codec not found")
)

// Helper: DecodeToWriter decodes data and writes to an io.Writer
//
// This is useful for streaming decompression.
func DecodeToWriter(w io.Writer, codec Codec, src, params []byte, dstSize int) error {
	dst := make([]byte, dstSize)
	n, err := codec.Decode(dst, src, params)
	if err != nil {
		return err
	}
	_, err = w.Write(dst[:n])
	return err
}
