package frame

import (
	"encoding/binary"
	"io"
)

// Varint encoding constants (LEB128 format)
const (
	varintContinueBit = 0x80 // Continuation bit (MSB)
	varintDataBits    = 0x7F // Data bits (7 bits)
	varintMaxShift    = 64   // Maximum bit shift
	varintShiftIncr   = 7    // Shift increment per byte
	varintMaxBytes    = 10   // Maximum bytes for uint64 (ceil(64/7))
)

// readByte reads a single byte from the reader
func readByte(r io.Reader) (byte, error) {
	var b [1]byte
	_, err := io.ReadFull(r, b[:])
	return b[0], err
}

// hasContinueBit checks if the byte has the continuation bit set
func hasContinueBit(b byte) bool {
	return b&varintContinueBit != 0
}

// readVarint reads a variable-length unsigned integer (LEB128 encoding)
//
// LEB128 (Little Endian Base 128) encoding:
//   - Each byte contains 7 bits of data
//   - MSB (bit 7) is the continuation bit
//   - If MSB = 1, more bytes follow
//   - If MSB = 0, this is the last byte
//
// Example: 0x96 0x01 = 150
//   - 0x96 = 10010110 -> continue=1, data=0010110 (22)
//   - 0x01 = 00000001 -> continue=0, data=0000001 (1)
//   - Result: (1 << 7) | 22 = 128 + 22 = 150
func readVarint(r io.Reader) (uint64, error) {
	var result uint64
	var shift uint

	for i := 0; i < varintMaxBytes; i++ {
		b, err := readByte(r)
		if err != nil {
			if err == io.EOF {
				return 0, ErrUnexpectedEOF
			}
			return 0, err
		}

		// Extract data bits and add to result
		result |= uint64(b&varintDataBits) << shift

		// Check if this is the last byte
		if !hasContinueBit(b) {
			return result, nil
		}

		shift += varintShiftIncr
		if shift >= varintMaxShift {
			return 0, ErrInvalidVarint
		}
	}

	// Too many bytes (> 10)
	return 0, ErrInvalidVarint
}

// writeVarint writes a variable-length unsigned integer (LEB128 encoding)
func writeVarint(w io.Writer, value uint64) error {
	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(buf, value)
	_, err := w.Write(buf[:n])
	return err
}

// readU8 reads an unsigned 8-bit integer
func readU8(r io.Reader) (uint8, error) {
	return readByte(r)
}

// readU32LE reads an unsigned 32-bit integer (little-endian)
func readU32LE(r io.Reader) (uint32, error) {
	var buf [4]byte
	_, err := io.ReadFull(r, buf[:])
	if err != nil {
		if err == io.EOF {
			return 0, ErrUnexpectedEOF
		}
		return 0, err
	}
	return binary.LittleEndian.Uint32(buf[:]), nil
}
