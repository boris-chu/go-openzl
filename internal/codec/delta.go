package codec

import (
	"encoding/binary"
	"fmt"
)

// Delta codec stores differences between consecutive values.
//
// Format:
//
//	First value: stored as-is (raw)
//	Subsequent values: delta from previous value
//
// Example:
//
//	Input:  [100, 102, 105, 103, 108]
//	Deltas: [100, 2, 3, -2, 5]
//
// Supports element widths: 1, 2, 4, 8 bytes (uint8, uint16, uint32, uint64)
// Negative deltas are stored using two's complement.
//
// Use cases:
//   - Time series with monotonically increasing values
//   - Sorted sequences
//   - Incremental counters
//   - Any data where consecutive values change slowly
type Delta struct {
	elementSize int // Element size in bytes (1, 2, 4, or 8)
}

// NewDelta creates a Delta codec with the specified element size.
//
// elementSize must be 1, 2, 4, or 8 bytes.
func NewDelta(elementSize int) *Delta {
	return &Delta{
		elementSize: elementSize,
	}
}

// ID returns the codec identifier
func (c *Delta) ID() ID {
	return IDDelta
}

// Name returns the human-readable name
func (c *Delta) Name() string {
	return "Delta"
}

// Decode decodes delta-encoded data back to original values.
//
// The parameter encodes the element size (1, 2, 4, or 8 bytes).
// If params is empty, element size defaults to 8 bytes.
//
// Decoding algorithm:
//
//	output[0] = input[0]  (first value is raw)
//	output[i] = output[i-1] + input[i]  (subsequent values are deltas)
func (c *Delta) Decode(dst, src, params []byte) (int, error) {
	// Parse element size from params
	elementSize := c.elementSize
	if len(params) > 0 {
		elementSize = int(params[0])
	}

	// Validate element size
	if elementSize != 1 && elementSize != 2 && elementSize != 4 && elementSize != 8 {
		return 0, fmt.Errorf("invalid element size: %d (must be 1, 2, 4, or 8)", elementSize)
	}

	// Check input alignment
	if len(src)%elementSize != 0 {
		return 0, fmt.Errorf("source length %d not aligned to element size %d", len(src), elementSize)
	}

	numElements := len(src) / elementSize

	// Check output buffer size
	requiredSize := numElements * elementSize
	if len(dst) < requiredSize {
		return 0, ErrBufferTooSmall
	}

	// Handle empty input
	if numElements == 0 {
		return 0, nil
	}

	// Decode based on element size
	// Note: Scalar versions are faster than SIMD for Delta due to data dependencies.
	// The Go compiler optimizes the simple scalar loops very well (15 GB/s vs 9 GB/s for SIMD).
	// SIMD implementations are available in delta_simd_amd64.go for reference/future assembly work.
	switch elementSize {
	case 1:
		return c.decode8(dst, src, numElements)
	case 2:
		return c.decode16(dst, src, numElements)
	case 4:
		return c.decode32(dst, src, numElements)
	case 8:
		return c.decode64(dst, src, numElements)
	default:
		return 0, fmt.Errorf("unsupported element size: %d", elementSize)
	}
}

// decode8 decodes 1-byte elements
func (c *Delta) decode8(dst, src []byte, numElements int) (int, error) {
	var prev uint8

	for i := 0; i < numElements; i++ {
		delta := src[i]
		value := prev + delta
		dst[i] = value
		prev = value
	}

	return numElements, nil
}

// decode16 decodes 2-byte elements
func (c *Delta) decode16(dst, src []byte, numElements int) (int, error) {
	var prev uint16

	for i := 0; i < numElements; i++ {
		offset := i * 2
		delta := binary.LittleEndian.Uint16(src[offset:])
		value := prev + delta
		binary.LittleEndian.PutUint16(dst[offset:], value)
		prev = value
	}

	return numElements * 2, nil
}

// decode32 decodes 4-byte elements
func (c *Delta) decode32(dst, src []byte, numElements int) (int, error) {
	var prev uint32

	for i := 0; i < numElements; i++ {
		offset := i * 4
		delta := binary.LittleEndian.Uint32(src[offset:])
		value := prev + delta
		binary.LittleEndian.PutUint32(dst[offset:], value)
		prev = value
	}

	return numElements * 4, nil
}

// decode64 decodes 8-byte elements
func (c *Delta) decode64(dst, src []byte, numElements int) (int, error) {
	var prev uint64

	for i := 0; i < numElements; i++ {
		offset := i * 8
		delta := binary.LittleEndian.Uint64(src[offset:])
		value := prev + delta
		binary.LittleEndian.PutUint64(dst[offset:], value)
		prev = value
	}

	return numElements * 8, nil
}

// Encode encodes data using delta encoding.
//
// Encoding algorithm:
//
//	output[0] = input[0]  (first value is raw)
//	output[i] = input[i] - input[i-1]  (subsequent values are deltas)
func (c *Delta) Encode(dst, src, params []byte) (int, error) {
	// Parse element size from params
	elementSize := c.elementSize
	if len(params) > 0 {
		elementSize = int(params[0])
	}

	// Validate element size
	if elementSize != 1 && elementSize != 2 && elementSize != 4 && elementSize != 8 {
		return 0, fmt.Errorf("invalid element size: %d (must be 1, 2, 4, or 8)", elementSize)
	}

	// Check input alignment
	if len(src)%elementSize != 0 {
		return 0, fmt.Errorf("source length %d not aligned to element size %d", len(src), elementSize)
	}

	numElements := len(src) / elementSize

	// Check output buffer size
	requiredSize := numElements * elementSize
	if len(dst) < requiredSize {
		return 0, ErrBufferTooSmall
	}

	// Handle empty input
	if numElements == 0 {
		return 0, nil
	}

	// Encode based on element size
	// Note: Scalar versions are faster than SIMD for Delta due to data dependencies.
	// The Go compiler optimizes the simple scalar loops very well (15 GB/s vs 9 GB/s for SIMD).
	// SIMD implementations are available in delta_simd_amd64.go for reference/future assembly work.
	switch elementSize {
	case 1:
		return c.encode8(dst, src, numElements)
	case 2:
		return c.encode16(dst, src, numElements)
	case 4:
		return c.encode32(dst, src, numElements)
	case 8:
		return c.encode64(dst, src, numElements)
	default:
		return 0, fmt.Errorf("unsupported element size: %d", elementSize)
	}
}

// encode8 encodes 1-byte elements
func (c *Delta) encode8(dst, src []byte, numElements int) (int, error) {
	var prev uint8

	for i := 0; i < numElements; i++ {
		value := src[i]
		delta := value - prev
		dst[i] = delta
		prev = value
	}

	return numElements, nil
}

// encode16 encodes 2-byte elements
func (c *Delta) encode16(dst, src []byte, numElements int) (int, error) {
	var prev uint16

	for i := 0; i < numElements; i++ {
		offset := i * 2
		value := binary.LittleEndian.Uint16(src[offset:])
		delta := value - prev
		binary.LittleEndian.PutUint16(dst[offset:], delta)
		prev = value
	}

	return numElements * 2, nil
}

// encode32 encodes 4-byte elements
func (c *Delta) encode32(dst, src []byte, numElements int) (int, error) {
	var prev uint32

	for i := 0; i < numElements; i++ {
		offset := i * 4
		value := binary.LittleEndian.Uint32(src[offset:])
		delta := value - prev
		binary.LittleEndian.PutUint32(dst[offset:], delta)
		prev = value
	}

	return numElements * 4, nil
}

// encode64 encodes 8-byte elements
func (c *Delta) encode64(dst, src []byte, numElements int) (int, error) {
	var prev uint64

	for i := 0; i < numElements; i++ {
		offset := i * 8
		value := binary.LittleEndian.Uint64(src[offset:])
		delta := value - prev
		binary.LittleEndian.PutUint64(dst[offset:], delta)
		prev = value
	}

	return numElements * 8, nil
}

// PreservesSize returns true because Delta always produces output
// of the same size as its input.
//
// Delta encodes differences but maintains the same number of elements
// and element size. For example, 100 uint64s → 100 uint64 deltas.
func (c *Delta) PreservesSize() bool {
	return true
}
