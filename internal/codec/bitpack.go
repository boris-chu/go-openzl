package codec

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/bits"
)

// Bitpack codec compresses integers by packing them into the minimum number of bits required.
//
// Instead of storing each integer in its full width (e.g., 32 bits), bitpacking finds
// the maximum value and packs all integers using only enough bits to represent that maximum.
//
// Example:
//
//	Values: [5, 2, 7, 1, 4]
//	Max: 7 (needs 3 bits)
//	Packed: 5 values × 3 bits = 15 bits instead of 160 bits (32-bit each)
//	Savings: 90.6% compression!
//
// Perfect pairing with Delta and ZigZag:
//
//	Raw timestamps → Delta (big→small) → ZigZag (signed→unsigned) → Bitpack (minimal bits)
//
// Supported element sizes: 1, 2, 4, 8 bytes (uint8, uint16, uint32, uint64)
//
// Format:
//
//	Header: [element_size(1), bits_needed(1), count(4)]
//	Data: Bit-packed values
type Bitpack struct {
	elementSize int // Element size in bytes (1, 2, 4, or 8)
}

// NewBitpack creates a Bitpack codec with the specified element size
func NewBitpack(elementSize int) *Bitpack {
	return &Bitpack{
		elementSize: elementSize,
	}
}

// ID returns the codec identifier
func (b *Bitpack) ID() ID {
	return IDBitpack
}

// Name returns the human-readable name
func (b *Bitpack) Name() string {
	return "Bitpack"
}

// Decode decompresses bit-packed data
//
// Format:
//
//	Header (6 bytes):
//	  - element_size (1 byte)
//	  - bits_needed (1 byte)
//	  - count (4 bytes, little-endian)
//	Data: Bit-packed values
func (b *Bitpack) Decode(dst, src, params []byte) (int, error) {
	if len(src) < 6 {
		return 0, errors.New("bitpack: source too small for header")
	}

	// Read header
	elementSize := int(src[0])
	bitsNeeded := int(src[1])
	count := int(binary.LittleEndian.Uint32(src[2:6]))

	// Validate
	if elementSize != 1 && elementSize != 2 && elementSize != 4 && elementSize != 8 {
		return 0, fmt.Errorf("bitpack: invalid element size %d", elementSize)
	}
	if bitsNeeded > elementSize*8 {
		return 0, fmt.Errorf("bitpack: bits_needed %d > max %d", bitsNeeded, elementSize*8)
	}

	outputSize := count * elementSize
	if len(dst) < outputSize {
		return 0, ErrBufferTooSmall
	}

	// Handle special case: all zeros
	if bitsNeeded == 0 {
		for i := 0; i < outputSize; i++ {
			dst[i] = 0
		}
		return outputSize, nil
	}

	// Unpack values
	data := src[6:]
	bitPos := 0
	bytePos := 0

	for i := 0; i < count; i++ {
		value := uint64(0)
		remainingBits := bitsNeeded
		shift := 0

		// Read bits needed for this value
		for remainingBits > 0 && bytePos < len(data) {
			bitsInByte := 8 - bitPos
			bitsToRead := remainingBits
			if bitsToRead > bitsInByte {
				bitsToRead = bitsInByte
			}

			// Extract bits from current byte
			mask := byte((1 << bitsToRead) - 1)
			bitsValue := (data[bytePos] >> bitPos) & mask

			// Add to value
			value |= uint64(bitsValue) << shift

			// Update positions
			bitPos += bitsToRead
			shift += bitsToRead
			remainingBits -= bitsToRead

			if bitPos >= 8 {
				bytePos++
				bitPos = 0
			}
		}

		// Write value to output based on element size
		offset := i * elementSize
		switch elementSize {
		case 1:
			dst[offset] = byte(value)
		case 2:
			binary.LittleEndian.PutUint16(dst[offset:], uint16(value))
		case 4:
			binary.LittleEndian.PutUint32(dst[offset:], uint32(value))
		case 8:
			binary.LittleEndian.PutUint64(dst[offset:], value)
		}
	}

	return outputSize, nil
}

// Encode compresses data using bitpacking
//
// Algorithm:
//  1. Find maximum value
//  2. Calculate bits needed: ⌈log₂(max + 1)⌉
//  3. Write header (element_size, bits_needed, count)
//  4. Pack each value using bits_needed bits
func (b *Bitpack) Encode(dst, src, params []byte) (int, error) {
	// Determine element size (from params or default)
	elementSize := b.elementSize
	if len(params) > 0 {
		elementSize = int(params[0])
	}

	// Validate
	if elementSize != 1 && elementSize != 2 && elementSize != 4 && elementSize != 8 {
		return 0, fmt.Errorf("bitpack: invalid element size %d", elementSize)
	}
	if len(src)%elementSize != 0 {
		return 0, fmt.Errorf("bitpack: source size %d not aligned to element size %d", len(src), elementSize)
	}

	count := len(src) / elementSize
	if count == 0 {
		return 0, errors.New("bitpack: empty source")
	}

	// Find maximum value
	maxValue := uint64(0)
	for i := 0; i < count; i++ {
		offset := i * elementSize
		var value uint64
		switch elementSize {
		case 1:
			value = uint64(src[offset])
		case 2:
			value = uint64(binary.LittleEndian.Uint16(src[offset:]))
		case 4:
			value = uint64(binary.LittleEndian.Uint32(src[offset:]))
		case 8:
			value = binary.LittleEndian.Uint64(src[offset:])
		}

		if value > maxValue {
			maxValue = value
		}
	}

	// Calculate bits needed
	bitsNeeded := bits.Len64(maxValue)
	if bitsNeeded == 0 {
		bitsNeeded = 1 // Need at least 1 bit, even for all zeros
	}

	// Special case: all zeros
	if maxValue == 0 {
		bitsNeeded = 0 // Encode as 0 bits (no data)
	}

	// Calculate output size
	// Header: 6 bytes (element_size + bits_needed + count)
	// Data: (count * bitsNeeded + 7) / 8 bytes
	dataBits := count * bitsNeeded
	dataBytes := (dataBits + 7) / 8
	totalSize := 6 + dataBytes

	if len(dst) < totalSize {
		return 0, ErrBufferTooSmall
	}

	// Write header
	dst[0] = byte(elementSize)
	dst[1] = byte(bitsNeeded)
	binary.LittleEndian.PutUint32(dst[2:6], uint32(count))

	// Handle special case: all zeros
	if bitsNeeded == 0 {
		return 6, nil
	}

	// Pack values into bits
	bitPos := 0
	bytePos := 6
	currentByte := byte(0)

	for i := 0; i < count; i++ {
		// Read value
		offset := i * elementSize
		var value uint64
		switch elementSize {
		case 1:
			value = uint64(src[offset])
		case 2:
			value = uint64(binary.LittleEndian.Uint16(src[offset:]))
		case 4:
			value = uint64(binary.LittleEndian.Uint32(src[offset:]))
		case 8:
			value = binary.LittleEndian.Uint64(src[offset:])
		}

		// Pack value into bits
		remainingBits := bitsNeeded
		for remainingBits > 0 {
			bitsInByte := 8 - bitPos
			bitsToWrite := remainingBits
			if bitsToWrite > bitsInByte {
				bitsToWrite = bitsInByte
			}

			// Extract low bits from value
			mask := uint64((1 << bitsToWrite) - 1)
			bitsValue := byte(value & mask)

			// Add to current byte
			currentByte |= bitsValue << bitPos

			// Update state
			bitPos += bitsToWrite
			value >>= bitsToWrite
			remainingBits -= bitsToWrite

			// Flush byte if full
			if bitPos >= 8 {
				dst[bytePos] = currentByte
				bytePos++
				currentByte = 0
				bitPos = 0
			}
		}
	}

	// Flush remaining bits
	if bitPos > 0 {
		dst[bytePos] = currentByte
		bytePos++
	}

	return bytePos, nil
}

// PreservesSize returns false because Bitpack changes size.
//
// Bitpack compresses integers by using fewer bits per value, significantly
// reducing the output size. For example, packing 1000 values that fit in
// 3 bits produces ~375 bytes instead of 4000 bytes (for 32-bit integers).
//
// This is a size-changing codec that requires explicit size metadata.
func (b *Bitpack) PreservesSize() bool {
	return false
}
