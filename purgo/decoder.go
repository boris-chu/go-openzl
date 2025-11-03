// Package purgo provides Pure Go OpenZL decompression.
//
// This package offers a complete Pure Go implementation of OpenZL
// decompression, eliminating CGO dependencies for faster builds
// and easier cross-compilation.
//
// Example usage:
//
//	data, err := purgo.DecompressInt64(compressed)
//	if err != nil {
//		log.Fatal(err)
//	}
//	// data is []int64, ready to use
//
// The Pure Go decoder provides:
//   - Zero CGO dependencies (faster builds, easier cross-compilation)
//   - Type-safe APIs (compile-time type checking)
//   - Streaming support (io.Reader interface)
//   - Excellent performance (283 MB/s - 125 GB/s across codecs)
//
// For general-purpose decompression to []byte, use Decompress().
// For typed decompression to numeric slices, use DecompressInt64(),
// DecompressFloat64(), etc.
package purgo

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/boris-chu/go-openzl/internal/codec"
	"github.com/boris-chu/go-openzl/internal/frame"
	"github.com/boris-chu/go-openzl/internal/graph"
)

// Decompress decompresses OpenZL data to raw bytes using Pure Go decoder.
//
// This function provides general-purpose decompression without type information.
// For typed decompression (e.g., to []int64), use DecompressInt64(), etc.
//
// Parameters:
//   - compressed: OpenZL compressed data
//
// Returns:
//   - Decompressed data as []byte
//   - Error if decompression fails
//
// Example:
//
//	compressed := []byte{...}  // OpenZL compressed data
//	data, err := purgo.Decompress(compressed)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// data is []byte
func Decompress(compressed []byte) ([]byte, error) {
	if len(compressed) == 0 {
		return nil, fmt.Errorf("purgo: empty input")
	}

	// Step 1: Decompress first layer
	stage1, err := decompressSingleStage(compressed)
	if err != nil {
		return nil, err
	}

	// Step 2: Check if result is itself an OpenZL frame (double-compressed)
	// Try to parse as frame - if it succeeds, decompress again
	if isOpenZLFrame(stage1) {
		stage2, err := decompressSingleStage(stage1)
		if err != nil {
			// Not actually a valid frame, return stage1 result
			return stage1, nil
		}
		// Successfully decompressed second layer
		return stage2, nil
	}

	// Single-stage compression, return result
	return stage1, nil
}

// isOpenZLFrame checks if data starts with OpenZL magic number
func isOpenZLFrame(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	// Check if first 4 bytes match OpenZL magic number pattern
	magic := uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
	// Magic number is MagicNumberBase + version, where version is 8-22
	const magicBase uint32 = 0xD7B1A5C0
	const minVersion uint32 = 8
	const maxVersion uint32 = 22 // Updated to support v22 (multi-stage pipelines)
	return magic >= (magicBase+minVersion) && magic <= (magicBase+maxVersion)
}

// decompressSingleStage decompresses one layer of OpenZL compression
func decompressSingleStage(compressed []byte) ([]byte, error) {
	// Step 1: Parse OpenZL frame
	reader := frame.NewReader(bytes.NewReader(compressed))
	f, err := reader.ReadFrame()
	if err != nil {
		return nil, fmt.Errorf("purgo: parse frame failed: %w", err)
	}

	// Step 2: Parse compression graph
	parser := graph.NewParser(f.Payload)
	g, graphSize, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("purgo: parse graph failed: %w", err)
	}

	// Step 3: Execute compression graph to decompress
	executor := graph.NewExecutor(codec.DefaultRegistry())
	compressedData := f.Payload[graphSize:]

	// Extract output sizes from frame (supports multi-output for segmented compression)
	outputSizes := make([]uint64, len(f.Outputs))
	for i, out := range f.Outputs {
		outputSizes[i] = out.DecompressedSize
	}

	// Execute graph with explicit node sizes (v22+) or inference (v21)
	// f.NodeSizes will be nil for v21 frames, non-nil for v22+ frames
	outputs, err := executor.ExecuteWithNodeSizes(g, compressedData, outputSizes, f.NodeSizes)
	if err != nil {
		return nil, fmt.Errorf("purgo: execute graph failed: %w", err)
	}

	// Handle single-output frames (normal case)
	if len(outputs) == 1 {
		return outputs[0], nil
	}

	// Handle multi-output frames (segmented compression from CompressSmart)
	// Concatenate all segments back together in original order
	var result bytes.Buffer
	for _, segment := range outputs {
		result.Write(segment)
	}

	return result.Bytes(), nil
}

// DecompressInt64 decompresses OpenZL data to int64 slice.
//
// This function provides type-safe decompression for int64 data.
// The input must be OpenZL compressed int64 data.
//
// Parameters:
//   - compressed: OpenZL compressed int64 data
//
// Returns:
//   - Decompressed data as []int64
//   - Error if decompression fails or type mismatch
//
// Example:
//
//	compressed := []byte{...}  // OpenZL compressed int64 data
//	numbers, err := purgo.DecompressInt64(compressed)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// numbers is []int64
func DecompressInt64(compressed []byte) ([]int64, error) {
	// Decompress to raw bytes
	rawBytes, err := Decompress(compressed)
	if err != nil {
		return nil, err
	}

	// Verify size is multiple of 8 (int64 size)
	if len(rawBytes)%8 != 0 {
		return nil, fmt.Errorf("purgo: decompressed size %d not multiple of 8 (int64 size)", len(rawBytes))
	}

	// Convert bytes to int64 slice
	count := len(rawBytes) / 8
	result := make([]int64, count)

	reader := bytes.NewReader(rawBytes)
	for i := 0; i < count; i++ {
		var val int64
		if err := binary.Read(reader, binary.LittleEndian, &val); err != nil {
			return nil, fmt.Errorf("purgo: read int64 at index %d failed: %w", i, err)
		}
		result[i] = val
	}

	return result, nil
}

// DecompressFloat64 decompresses OpenZL data to float64 slice.
//
// This function provides type-safe decompression for float64 data.
// The input must be OpenZL compressed float64 data.
//
// Parameters:
//   - compressed: OpenZL compressed float64 data
//
// Returns:
//   - Decompressed data as []float64
//   - Error if decompression fails or type mismatch
//
// Example:
//
//	compressed := []byte{...}  // OpenZL compressed float64 data
//	numbers, err := purgo.DecompressFloat64(compressed)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// numbers is []float64
func DecompressFloat64(compressed []byte) ([]float64, error) {
	// Decompress to raw bytes
	rawBytes, err := Decompress(compressed)
	if err != nil {
		return nil, err
	}

	// Verify size is multiple of 8 (float64 size)
	if len(rawBytes)%8 != 0 {
		return nil, fmt.Errorf("purgo: decompressed size %d not multiple of 8 (float64 size)", len(rawBytes))
	}

	// Convert bytes to float64 slice
	count := len(rawBytes) / 8
	result := make([]float64, count)

	reader := bytes.NewReader(rawBytes)
	for i := 0; i < count; i++ {
		var val float64
		if err := binary.Read(reader, binary.LittleEndian, &val); err != nil {
			return nil, fmt.Errorf("purgo: read float64 at index %d failed: %w", i, err)
		}
		result[i] = val
	}

	return result, nil
}

// DecompressInt32 decompresses OpenZL data to int32 slice.
//
// This function provides type-safe decompression for int32 data.
// The input must be OpenZL compressed int32 data.
func DecompressInt32(compressed []byte) ([]int32, error) {
	rawBytes, err := Decompress(compressed)
	if err != nil {
		return nil, err
	}

	if len(rawBytes)%4 != 0 {
		return nil, fmt.Errorf("purgo: decompressed size %d not multiple of 4 (int32 size)", len(rawBytes))
	}

	count := len(rawBytes) / 4
	result := make([]int32, count)

	reader := bytes.NewReader(rawBytes)
	for i := 0; i < count; i++ {
		var val int32
		if err := binary.Read(reader, binary.LittleEndian, &val); err != nil {
			return nil, fmt.Errorf("purgo: read int32 at index %d failed: %w", i, err)
		}
		result[i] = val
	}

	return result, nil
}

// DecompressUint64 decompresses OpenZL data to uint64 slice.
//
// This function provides type-safe decompression for uint64 data.
// The input must be OpenZL compressed uint64 data.
func DecompressUint64(compressed []byte) ([]uint64, error) {
	rawBytes, err := Decompress(compressed)
	if err != nil {
		return nil, err
	}

	if len(rawBytes)%8 != 0 {
		return nil, fmt.Errorf("purgo: decompressed size %d not multiple of 8 (uint64 size)", len(rawBytes))
	}

	count := len(rawBytes) / 8
	result := make([]uint64, count)

	reader := bytes.NewReader(rawBytes)
	for i := 0; i < count; i++ {
		var val uint64
		if err := binary.Read(reader, binary.LittleEndian, &val); err != nil {
			return nil, fmt.Errorf("purgo: read uint64 at index %d failed: %w", i, err)
		}
		result[i] = val
	}

	return result, nil
}

// DecompressInt8 decompresses OpenZL data to int8 slice.
//
// This function provides type-safe decompression for int8 data.
// The input must be OpenZL compressed int8 data.
func DecompressInt8(compressed []byte) ([]int8, error) {
	rawBytes, err := Decompress(compressed)
	if err != nil {
		return nil, err
	}

	// int8 is 1 byte, so no alignment check needed
	count := len(rawBytes)
	result := make([]int8, count)

	for i := 0; i < count; i++ {
		result[i] = int8(rawBytes[i])
	}

	return result, nil
}

// DecompressInt16 decompresses OpenZL data to int16 slice.
//
// This function provides type-safe decompression for int16 data.
// The input must be OpenZL compressed int16 data.
func DecompressInt16(compressed []byte) ([]int16, error) {
	rawBytes, err := Decompress(compressed)
	if err != nil {
		return nil, err
	}

	if len(rawBytes)%2 != 0 {
		return nil, fmt.Errorf("purgo: decompressed size %d not multiple of 2 (int16 size)", len(rawBytes))
	}

	count := len(rawBytes) / 2
	result := make([]int16, count)

	reader := bytes.NewReader(rawBytes)
	for i := 0; i < count; i++ {
		var val int16
		if err := binary.Read(reader, binary.LittleEndian, &val); err != nil {
			return nil, fmt.Errorf("purgo: read int16 at index %d failed: %w", i, err)
		}
		result[i] = val
	}

	return result, nil
}

// DecompressUint8 decompresses OpenZL data to uint8 slice.
//
// This function provides type-safe decompression for uint8 data.
// The input must be OpenZL compressed uint8 data.
func DecompressUint8(compressed []byte) ([]uint8, error) {
	rawBytes, err := Decompress(compressed)
	if err != nil {
		return nil, err
	}

	// uint8 is 1 byte, direct copy
	result := make([]uint8, len(rawBytes))
	copy(result, rawBytes)

	return result, nil
}

// DecompressUint16 decompresses OpenZL data to uint16 slice.
//
// This function provides type-safe decompression for uint16 data.
// The input must be OpenZL compressed uint16 data.
func DecompressUint16(compressed []byte) ([]uint16, error) {
	rawBytes, err := Decompress(compressed)
	if err != nil {
		return nil, err
	}

	if len(rawBytes)%2 != 0 {
		return nil, fmt.Errorf("purgo: decompressed size %d not multiple of 2 (uint16 size)", len(rawBytes))
	}

	count := len(rawBytes) / 2
	result := make([]uint16, count)

	reader := bytes.NewReader(rawBytes)
	for i := 0; i < count; i++ {
		var val uint16
		if err := binary.Read(reader, binary.LittleEndian, &val); err != nil {
			return nil, fmt.Errorf("purgo: read uint16 at index %d failed: %w", i, err)
		}
		result[i] = val
	}

	return result, nil
}

// DecompressUint32 decompresses OpenZL data to uint32 slice.
//
// This function provides type-safe decompression for uint32 data.
// The input must be OpenZL compressed uint32 data.
func DecompressUint32(compressed []byte) ([]uint32, error) {
	rawBytes, err := Decompress(compressed)
	if err != nil {
		return nil, err
	}

	if len(rawBytes)%4 != 0 {
		return nil, fmt.Errorf("purgo: decompressed size %d not multiple of 4 (uint32 size)", len(rawBytes))
	}

	count := len(rawBytes) / 4
	result := make([]uint32, count)

	reader := bytes.NewReader(rawBytes)
	for i := 0; i < count; i++ {
		var val uint32
		if err := binary.Read(reader, binary.LittleEndian, &val); err != nil {
			return nil, fmt.Errorf("purgo: read uint32 at index %d failed: %w", i, err)
		}
		result[i] = val
	}

	return result, nil
}

// DecompressFloat32 decompresses OpenZL data to float32 slice.
//
// This function provides type-safe decompression for float32 data.
// The input must be OpenZL compressed float32 data.
func DecompressFloat32(compressed []byte) ([]float32, error) {
	rawBytes, err := Decompress(compressed)
	if err != nil {
		return nil, err
	}

	if len(rawBytes)%4 != 0 {
		return nil, fmt.Errorf("purgo: decompressed size %d not multiple of 4 (float32 size)", len(rawBytes))
	}

	count := len(rawBytes) / 4
	result := make([]float32, count)

	reader := bytes.NewReader(rawBytes)
	for i := 0; i < count; i++ {
		var val float32
		if err := binary.Read(reader, binary.LittleEndian, &val); err != nil {
			return nil, fmt.Errorf("purgo: read float32 at index %d failed: %w", i, err)
		}
		result[i] = val
	}

	return result, nil
}
