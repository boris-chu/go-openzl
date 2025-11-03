// Package frame provides Pure Go implementation for reading and writing OpenZL frame format.
//
// OpenZL uses a self-describing frame format where the compression graph travels
// with the compressed data. This allows a universal decoder to decompress any
// OpenZL frame by executing the embedded compression graph.
//
// Frame Structure (Version >= 21):
//
//	┌──────────────────────────────────────────────┐
//	│ Fixed Header (5 bytes minimum):              │
//	│   00-03: Magic Number (uint32 LE)            │
//	│          = 0xD7B1A5C0 + version               │
//	│   04:    Flags (uint8)                        │
//	├──────────────────────────────────────────────┤
//	│ Variable Section:                             │
//	│   05:    Token1 (uint8)                       │
//	│          Lower 4 bits: nbOutputs              │
//	│          Upper 4 bits: First 2 output types   │
//	│   06+:   Output Sizes (varints)               │
//	│   XX+:   Compressed Payload                   │
//	├──────────────────────────────────────────────┤
//	│ Optional Footer:                              │
//	│   - Frame header checksum                     │
//	│   - Content checksum                          │
//	└──────────────────────────────────────────────┘
package frame

import (
	"fmt"
	"io"
)

// Magic number constants
const (
	// MagicNumberBase is the base magic number for OpenZL frames
	// Version is embedded: Magic = MagicNumberBase + version
	MagicNumberBase uint32 = 0xD7B1A5C0

	// MinFormatVersion is the minimum supported format version
	MinFormatVersion uint32 = 8

	// MaxFormatVersion is the maximum supported format version
	MaxFormatVersion uint32 = 22

	// ChunkVersionMin is the version where modern chunk-based format starts
	ChunkVersionMin uint32 = 21

	// NodeSizesVersionMin is the version where intermediate node sizes are stored (v22+)
	// This enables multi-stage pipelines with size-changing codecs in a single frame
	NodeSizesVersionMin uint32 = 22
)

// Header size constants
const (
	MagicSize     = 4
	FlagsSize     = 1
	Token1Size    = 1
	MinHeaderSize = MagicSize + FlagsSize + Token1Size // 6 bytes
)

// Frame flags (version >= 21)
type FrameFlags uint8

const (
	// FlagContentChecksum indicates the content has a checksum
	FlagContentChecksum FrameFlags = 1 << 0

	// FlagCompressedChecksum indicates the frame header has a checksum
	FlagCompressedChecksum FrameFlags = 1 << 1
)

// HasContentChecksum returns true if content checksum flag is set
func (f FrameFlags) HasContentChecksum() bool {
	return (f & FlagContentChecksum) != 0
}

// HasCompressedChecksum returns true if compressed checksum flag is set
func (f FrameFlags) HasCompressedChecksum() bool {
	return (f & FlagCompressedChecksum) != 0
}

// OutputType represents the type of an output stream
type OutputType uint8

const (
	// TypeSerial represents raw bytes
	TypeSerial OutputType = 0
	// TypeStruct represents structured data
	TypeStruct OutputType = 1
	// TypeNumeric represents numbers
	TypeNumeric OutputType = 2
	// TypeString represents strings
	TypeString OutputType = 3
)

// String returns the string representation of the output type
func (t OutputType) String() string {
	switch t {
	case TypeSerial:
		return "serial"
	case TypeStruct:
		return "struct"
	case TypeNumeric:
		return "numeric"
	case TypeString:
		return "string"
	default:
		return fmt.Sprintf("unknown(%d)", t)
	}
}

// Output represents a single output stream in the frame
type Output struct {
	Type             OutputType // Type of output
	DecompressedSize uint64     // Size of decompressed data
	NumElements      uint64     // Number of elements (for strings)
}

// Header represents the frame header
type Header struct {
	Magic   uint32     // Magic number with embedded version
	Version uint32     // Format version extracted from magic
	Flags   FrameFlags // Frame flags bitfield
}

// Frame represents a complete OpenZL frame
type Frame struct {
	Header    *Header   // Frame header
	Outputs   []*Output // Output streams
	NodeSizes []uint64  // Intermediate node sizes (v22+, nil for v21)
	Payload   []byte    // Compressed payload data
}

// Common errors
var (
	// ErrInvalidMagic indicates the frame has an invalid magic number
	ErrInvalidMagic = fmt.Errorf("invalid frame magic number")

	// ErrUnsupportedVersion indicates the frame version is not supported
	ErrUnsupportedVersion = fmt.Errorf("unsupported frame format version")

	// ErrCorruptedFrame indicates the frame data is corrupted
	ErrCorruptedFrame = fmt.Errorf("frame corruption detected")

	// ErrBufferTooSmall indicates the destination buffer is too small
	ErrBufferTooSmall = fmt.Errorf("buffer too small")

	// ErrUnexpectedEOF indicates unexpected end of data
	ErrUnexpectedEOF = fmt.Errorf("unexpected end of frame data")

	// ErrInvalidVarint indicates invalid varint encoding
	ErrInvalidVarint = fmt.Errorf("invalid varint encoding")

	// ErrZeroOutputs indicates the frame has zero outputs (not supported)
	ErrZeroOutputs = fmt.Errorf("frame with zero outputs not supported")

	// ErrUnknownSize indicates the frame has unknown size (not supported)
	ErrUnknownSize = fmt.Errorf("frames with unknown size not supported")

	// ErrChecksumMismatch indicates checksum validation failed
	ErrChecksumMismatch = fmt.Errorf("checksum mismatch")
)

// Reader provides methods for reading frames
type Reader struct {
	r io.Reader
}

// NewReader creates a new frame reader
func NewReader(r io.Reader) *Reader {
	return &Reader{r: r}
}

// Writer provides methods for writing frames
type Writer struct {
	w io.Writer
}

// NewWriter creates a new frame writer
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}
