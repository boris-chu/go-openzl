package frame

import (
	"bytes"
	"fmt"
	"io"
)

// Parse parses a complete OpenZL frame from the given data
//
// This is a convenience function that wraps NewReader and ReadFrame.
// For streaming usage, use NewReader directly.
//
// Example:
//
//	data, _ := os.ReadFile("compressed.bin")
//	frame, err := frame.Parse(data)
//	if err != nil {
//	    log.Fatal(err)
//	}
func Parse(data []byte) (*Frame, error) {
	r := NewReader(bytes.NewReader(data))
	return r.ReadFrame()
}

// ReadFrame reads a complete frame from the reader
//
// This method reads the frame header and output specifications.
// For version >= 21, it follows the modern chunk-based format.
func (r *Reader) ReadFrame() (*Frame, error) {
	// Read frame header (magic + flags)
	header, err := r.readHeader()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	// For version >= 21, read outputs using modern format
	if header.Version < ChunkVersionMin {
		return nil, fmt.Errorf("version %d < %d: old format not yet implemented", header.Version, ChunkVersionMin)
	}

	// Read token1 (nbOutputs + types)
	token1, err := readU8(r.r)
	if err != nil {
		return nil, fmt.Errorf("read token1: %w", err)
	}

	// Extract nbOutputs from lower 4 bits
	nbOutputs := int(token1 & 0x0F)
	if nbOutputs == 0 {
		return nil, ErrZeroOutputs
	}

	// Decode output types
	outputs := make([]*Output, nbOutputs)
	if nbOutputs >= 1 {
		// First output type from bits 4-5
		outputs[0] = &Output{
			Type: OutputType((token1 >> 4) & 3),
		}
	}
	if nbOutputs >= 2 {
		// Second output type from bits 6-7
		outputs[1] = &Output{
			Type: OutputType((token1 >> 6) & 3),
		}
	}
	// TODO: Handle > 2 outputs (requires reading additional type bytes)
	if nbOutputs > 2 {
		return nil, fmt.Errorf("nbOutputs > 2 not yet implemented")
	}

	// Read first byte of sizes section - must be non-zero
	firstByte, err := readU8(r.r)
	if err != nil {
		return nil, fmt.Errorf("read sizes first byte: %w", err)
	}
	if firstByte == 0 {
		return nil, ErrUnknownSize
	}

	// Put first byte back into buffer for varint reading
	// We need to read it as part of the first varint
	buf := []byte{firstByte}
	combined := io.MultiReader(bytes.NewReader(buf), r.r)

	// Read decompressed sizes for each output
	for i := 0; i < nbOutputs; i++ {
		v, err := readVarint(combined)
		if err != nil {
			return nil, fmt.Errorf("read size for output %d: %w", i, err)
		}
		if v == 0 {
			return nil, ErrUnknownSize
		}
		// Actual size = varint - 1
		outputs[i].DecompressedSize = v - 1

		// For string types, read number of elements
		if outputs[i].Type == TypeString {
			numElts, err := readVarint(combined)
			if err != nil {
				return nil, fmt.Errorf("read num elements for output %d: %w", i, err)
			}
			outputs[i].NumElements = numElts
		} else {
			// For serial/numeric/struct, numElements = size
			outputs[i].NumElements = outputs[i].DecompressedSize
		}
	}

	// Read frame header checksum if present
	if header.Flags.HasCompressedChecksum() {
		// TODO: Implement frame header checksum validation
		// For now, just skip the checksum byte
		_, err := readU8(r.r)
		if err != nil {
			return nil, fmt.Errorf("read frame header checksum: %w", err)
		}
	}

	// NEW (v22+): Read intermediate node sizes
	var nodeSizes []uint64
	if header.Version >= NodeSizesVersionMin {
		// Read number of nodes
		nbNodes, err := readVarint(combined)
		if err != nil {
			return nil, fmt.Errorf("read node count (v22): %w", err)
		}

		if nbNodes > 0 {
			nodeSizes = make([]uint64, nbNodes)
			for i := uint64(0); i < nbNodes; i++ {
				size, err := readVarint(combined)
				if err != nil {
					return nil, fmt.Errorf("read node %d size (v22): %w", i, err)
				}
				nodeSizes[i] = size
			}
		}
	}

	// Read remaining payload
	payload, err := io.ReadAll(r.r)
	if err != nil {
		return nil, fmt.Errorf("read payload: %w", err)
	}

	return &Frame{
		Header:    header,
		Outputs:   outputs,
		NodeSizes: nodeSizes, // nil for v21, populated for v22+
		Payload:   payload,
	}, nil
}

// readHeader reads the frame header (magic + flags)
func (r *Reader) readHeader() (*Header, error) {
	// Read magic number (4 bytes LE)
	magic, err := readU32LE(r.r)
	if err != nil {
		return nil, err
	}

	// Extract version from magic
	if magic < MagicNumberBase || magic > MagicNumberBase+MaxFormatVersion+16 {
		return nil, ErrInvalidMagic
	}

	version := magic - MagicNumberBase
	if version < MinFormatVersion || version > MaxFormatVersion {
		return nil, fmt.Errorf("%w: version %d not in range [%d, %d]",
			ErrUnsupportedVersion, version, MinFormatVersion, MaxFormatVersion)
	}

	header := &Header{
		Magic:   magic,
		Version: version,
	}

	// For version >= 21, read flags
	if version >= ChunkVersionMin {
		flags, err := readU8(r.r)
		if err != nil {
			return nil, fmt.Errorf("read flags: %w", err)
		}
		header.Flags = FrameFlags(flags)
	}

	return header, nil
}
