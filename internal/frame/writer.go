package frame

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// WriteFrame writes an OpenZL frame to the writer.
//
// The frame format version is determined by the Header.Version field:
//   - v21: No intermediate node sizes (backward compatible)
//   - v22+: Includes intermediate node sizes for multi-stage pipelines
//
// Parameters:
//   - w: Writer to write the frame to
//   - frame: Frame to write
//
// Returns:
//   - Number of bytes written
//   - Error if write fails
func WriteFrame(w io.Writer, frame *Frame) (int, error) {
	if frame == nil || frame.Header == nil {
		return 0, fmt.Errorf("nil frame or header")
	}

	if len(frame.Outputs) == 0 {
		return 0, ErrZeroOutputs
	}

	// Use a buffer to calculate total size
	buf := &bytes.Buffer{}

	// Step 1: Write magic number (version embedded)
	magic := MagicNumberBase + frame.Header.Version
	if err := binary.Write(buf, binary.LittleEndian, magic); err != nil {
		return 0, fmt.Errorf("write magic: %w", err)
	}

	// Step 2: Write flags
	buf.WriteByte(uint8(frame.Header.Flags))

	// Step 3: Encode token1 (nbOutputs + output types)
	if len(frame.Outputs) > 2 {
		return 0, fmt.Errorf("more than 2 outputs not yet supported")
	}

	token1 := uint8(len(frame.Outputs))
	if len(frame.Outputs) >= 1 {
		token1 |= uint8(frame.Outputs[0].Type) << 4
	}
	if len(frame.Outputs) >= 2 {
		token1 |= uint8(frame.Outputs[1].Type) << 6
	}
	buf.WriteByte(token1)

	// Step 4: Write output sizes (size+1 encoding to avoid 0)
	for i, output := range frame.Outputs {
		if err := writeVarint(buf, output.DecompressedSize+1); err != nil {
			return 0, fmt.Errorf("write output %d size: %w", i, err)
		}

		// Write numElements for string types
		if output.Type == TypeString {
			if err := writeVarint(buf, output.NumElements); err != nil {
				return 0, fmt.Errorf("write output %d num elements: %w", i, err)
			}
		}
	}

	// Step 5: Write frame header checksum if flag set
	if frame.Header.Flags.HasCompressedChecksum() {
		// TODO: Implement actual checksum calculation
		// For now, write a dummy byte
		buf.WriteByte(0x00)
	}

	// Step 6 (v22+): Write intermediate node sizes
	if frame.Header.Version >= NodeSizesVersionMin {
		nbNodes := len(frame.NodeSizes)
		if err := writeVarint(buf, uint64(nbNodes)); err != nil {
			return 0, fmt.Errorf("write node count (v22): %w", err)
		}

		for i, size := range frame.NodeSizes {
			if err := writeVarint(buf, size); err != nil {
				return 0, fmt.Errorf("write node %d size (v22): %w", i, err)
			}
		}
	}

	// Step 7: Write payload (graph + compressed data)
	if _, err := buf.Write(frame.Payload); err != nil {
		return 0, fmt.Errorf("write payload: %w", err)
	}

	// Write everything to the output writer
	n, err := w.Write(buf.Bytes())
	if err != nil {
		return n, fmt.Errorf("write frame: %w", err)
	}

	return n, nil
}

// EncodeFrame encodes a frame to bytes.
//
// This is a convenience function that wraps WriteFrame with a buffer.
//
// Example:
//
//	frame := &Frame{
//	    Header: &Header{Version: 22},
//	    Outputs: []*Output{{DecompressedSize: 1000}},
//	    NodeSizes: []uint64{5000, 1000},
//	    Payload: compressedData,
//	}
//	data, err := EncodeFrame(frame)
func EncodeFrame(frame *Frame) ([]byte, error) {
	buf := &bytes.Buffer{}
	_, err := WriteFrame(buf, frame)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// WriteFrameV21 writes a v21 frame (no node sizes).
//
// This is provided for backward compatibility when interoperating
// with systems that only support v21.
func WriteFrameV21(w io.Writer, frame *Frame) (int, error) {
	// Force version to 21 and clear node sizes
	v21Frame := &Frame{
		Header: &Header{
			Magic:   frame.Header.Magic,
			Version: 21, // Force v21
			Flags:   frame.Header.Flags,
		},
		Outputs:   frame.Outputs,
		NodeSizes: nil, // v21 doesn't support node sizes
		Payload:   frame.Payload,
	}
	return WriteFrame(w, v21Frame)
}

// WriteFrameV22 writes a v22 frame (with node sizes).
//
// This is the recommended format for new data, enabling multi-stage
// pipelines with size-changing codecs.
//
// Requirements:
//   - frame.NodeSizes must be non-nil and match the number of nodes in the graph
//   - All node sizes must be correct
func WriteFrameV22(w io.Writer, frame *Frame) (int, error) {
	if frame.Header.Version < NodeSizesVersionMin {
		// Update version to v22 if needed
		frame.Header.Version = NodeSizesVersionMin
	}

	if len(frame.NodeSizes) == 0 {
		return 0, fmt.Errorf("v22 frame requires non-empty NodeSizes")
	}

	return WriteFrame(w, frame)
}

// FrameSize estimates the size of an encoded frame in bytes.
//
// This is useful for pre-allocating buffers. The estimate includes:
//   - Header: ~6 bytes
//   - Output sizes: ~2-5 bytes per output (varint encoded)
//   - Node sizes (v22+): ~2-5 bytes per node (varint encoded)
//   - Payload: len(frame.Payload)
func FrameSize(frame *Frame) int {
	if frame == nil {
		return 0
	}

	// Fixed header
	size := MagicSize + FlagsSize + Token1Size // 6 bytes

	// Output sizes (estimate 3 bytes per varint on average)
	size += len(frame.Outputs) * 3

	// Checksum byte if needed
	if frame.Header != nil && frame.Header.Flags.HasCompressedChecksum() {
		size++
	}

	// Node sizes (v22+)
	if frame.Header != nil && frame.Header.Version >= NodeSizesVersionMin {
		size += 2                        // nbNodes varint (1-2 bytes)
		size += len(frame.NodeSizes) * 3 // node sizes (estimate 3 bytes each)
	}

	// Payload
	size += len(frame.Payload)

	return size
}
