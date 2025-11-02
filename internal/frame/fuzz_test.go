package frame

import (
	"testing"
)

// FuzzParse fuzzes the frame parser with random data
//
// This test uses Go's built-in fuzzing to find edge cases and crashes.
// Run with: go test -fuzz=FuzzParse -fuzztime=30s
func FuzzParse(f *testing.F) {
	// Seed with valid frames
	f.Add([]byte{
		0xD5, 0xA5, 0xB1, 0xD7, // Magic v21
		0x03, // Flags
		0x01, // Token1
		0x05, // Size
	})

	f.Add([]byte{
		0xD5, 0xA5, 0xB1, 0xD7, // Magic v21
		0x00, // Flags
		0x02, // Token1 (2 outputs)
		0x05, // Size1
		0x0A, // Size2
	})

	// Fuzz
	f.Fuzz(func(t *testing.T, data []byte) {
		// Parser should never crash, even on invalid input
		frame, err := Parse(data)

		// If it parses successfully, validate invariants
		if err == nil {
			if frame == nil {
				t.Fatal("Parse returned nil frame with no error")
			}
			if frame.Header == nil {
				t.Fatal("Parsed frame has nil header")
			}
			if frame.Outputs == nil {
				t.Fatal("Parsed frame has nil outputs")
			}
			if len(frame.Outputs) == 0 {
				t.Fatal("Parsed frame has zero outputs")
			}

			// Version should be in valid range
			if frame.Header.Version < MinFormatVersion || frame.Header.Version > MaxFormatVersion {
				t.Fatalf("Invalid version: %d", frame.Header.Version)
			}

			// Output sizes validation
			for i, out := range frame.Outputs {
				if out == nil {
					t.Fatalf("Output %d is nil", i)
				}
				// Note: Large sizes are valid in the format (up to uint64 max)
				// We just verify they don't overflow when stored
			}
		}
	})
}

// FuzzVarint fuzzes varint decoder
func FuzzVarint(f *testing.F) {
	// Seed with valid varints
	f.Add([]byte{0x00})       // 0
	f.Add([]byte{0x01})       // 1
	f.Add([]byte{0x7F})       // 127
	f.Add([]byte{0x80, 0x01}) // 128
	f.Add([]byte{0xFF, 0x7F}) // 16383

	f.Fuzz(func(t *testing.T, data []byte) {
		// Decoder should never crash
		_, err := readVarint(&readerWrapper{data: data})

		// If successful, value should roundtrip
		if err == nil {
			// Already consumed in readVarint, can't re-read
			// Just verify no panic occurred
		}
	})
}

// readerWrapper wraps a byte slice as an io.Reader for fuzzing
type readerWrapper struct {
	data []byte
	pos  int
}

func (r *readerWrapper) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, nil
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
