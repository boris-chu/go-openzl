package frame

import (
	"fmt"
)

// Decompress decompresses the payload in a frame
//
// This is a basic decompressor that executes the compression graph
// embedded in the frame to produce the original data.
//
// For now, this is a placeholder that will be implemented as we
// add codec support.
func (f *Frame) Decompress() ([]byte, error) {
	if f == nil || f.Header == nil {
		return nil, fmt.Errorf("invalid frame")
	}

	if len(f.Outputs) == 0 {
		return nil, fmt.Errorf("no outputs in frame")
	}

	// For now, return an error indicating decompression not yet implemented
	return nil, fmt.Errorf("decompression not yet implemented: need to parse graph from payload")
}

// DecompressTo decompresses the payload into the provided buffer
//
// The buffer must be large enough to hold the decompressed data.
// Use frame.Outputs[0].DecompressedSize to determine required size.
func (f *Frame) DecompressTo(dst []byte) (int, error) {
	data, err := f.Decompress()
	if err != nil {
		return 0, err
	}

	if len(dst) < len(data) {
		return 0, ErrBufferTooSmall
	}

	return copy(dst, data), nil
}
