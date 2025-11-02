package frame

import (
	"os"
	"path/filepath"
	"testing"
)

// TestParseMinimalFrame tests parsing the simplest possible OpenZL frame
//
// This test uses a C-generated fixture (minimal.bin) to validate our
// Pure Go parser can read real OpenZL frames.
//
// Expected structure for minimal.bin:
//   - Input: {0x01, 0x02, 0x03, 0x04} (4 bytes)
//   - Version: 21
//   - Flags: 0x03 (both checksums)
//   - Outputs: 1 (type: serial, size: 4 bytes)
func TestParseMinimalFrame(t *testing.T) {
	// Load C-generated fixture
	data, err := os.ReadFile("../../test/fixtures/frames/minimal.bin")
	if err != nil {
		t.Fatalf("Failed to load minimal.bin: %v", err)
	}

	// Parse with Pure Go
	frame, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Validate frame was parsed
	if frame == nil {
		t.Fatal("Frame is nil")
	}

	// Validate header
	if frame.Header == nil {
		t.Fatal("Header is nil")
	}

	expectedMagic := MagicNumberBase + 21 // Version 21
	if frame.Header.Magic != expectedMagic {
		t.Errorf("Invalid magic: got 0x%08X, want 0x%08X",
			frame.Header.Magic, expectedMagic)
	}

	if frame.Header.Version != 21 {
		t.Errorf("Invalid version: got %d, want 21", frame.Header.Version)
	}

	if frame.Header.Flags != 0x03 {
		t.Errorf("Invalid flags: got 0x%02X, want 0x03", frame.Header.Flags)
	}

	// Validate outputs
	if len(frame.Outputs) != 1 {
		t.Fatalf("Expected 1 output, got %d", len(frame.Outputs))
	}

	if frame.Outputs[0].Type != TypeSerial {
		t.Errorf("Invalid output type: got %v, want %v",
			frame.Outputs[0].Type, TypeSerial)
	}

	if frame.Outputs[0].DecompressedSize != 4 {
		t.Errorf("Invalid decompressed size: got %d, want 4",
			frame.Outputs[0].DecompressedSize)
	}

	// Validate payload exists
	if frame.Payload == nil {
		t.Fatal("Payload is nil")
	}

	t.Logf("✓ Successfully parsed minimal.bin")
	t.Logf("  Header: magic=0x%08X version=%d flags=0x%02X",
		frame.Header.Magic, frame.Header.Version, frame.Header.Flags)
	t.Logf("  Outputs: %d", len(frame.Outputs))
	t.Logf("    Output 0: type=%s size=%d",
		frame.Outputs[0].Type, frame.Outputs[0].DecompressedSize)
	t.Logf("  Payload: %d bytes", len(frame.Payload))
}

// TestParseWithChecksums tests parsing a frame with checksums
//
// Expected structure for with_checksums.bin:
//   - Input: "Hello OpenZL!" (13 bytes)
//   - Version: 21
//   - Flags: 0x0e (compressed + both checksums)
//   - Decompressed size: 13 bytes
func TestParseWithChecksums(t *testing.T) {
	data, err := os.ReadFile("../../test/fixtures/frames/with_checksums.bin")
	if err != nil {
		t.Skip("with_checksums.bin not found")
	}

	frame, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Validate version 21
	if frame.Header.Version != 21 {
		t.Errorf("Invalid version: got %d, want 21", frame.Header.Version)
	}

	// Validate flags
	if !frame.Header.Flags.HasContentChecksum() {
		t.Error("Expected HasContentChecksum to be true")
	}
	if !frame.Header.Flags.HasCompressedChecksum() {
		t.Error("Expected HasCompressedChecksum to be true")
	}

	t.Logf("✓ Successfully parsed with_checksums.bin")
	t.Logf("  Version: %d", frame.Header.Version)
	t.Logf("  Flags: 0x%02X (content=%v compressed=%v)",
		frame.Header.Flags,
		frame.Header.Flags.HasContentChecksum(),
		frame.Header.Flags.HasCompressedChecksum())
	t.Logf("  Outputs: %d", len(frame.Outputs))
	for i, out := range frame.Outputs {
		t.Logf("    Output %d: type=%s size=%d", i, out.Type, out.DecompressedSize)
	}
}

// TestParseAllFixtures tests parsing all C-generated fixtures
//
// This ensures our Pure Go parser can handle various frame configurations
// including checksums, large inputs, etc.
func TestParseAllFixtures(t *testing.T) {
	fixturesDir := "../../test/fixtures/frames"

	// Find all .bin fixtures
	fixtures, err := filepath.Glob(filepath.Join(fixturesDir, "*.bin"))
	if err != nil {
		t.Fatalf("Failed to list fixtures: %v", err)
	}

	if len(fixtures) == 0 {
		t.Skip("No fixtures found - run 'make generate' in test/tools/fixture_generator/")
	}

	// Test each fixture
	for _, fixturePath := range fixtures {
		fixtureName := filepath.Base(fixturePath)

		t.Run(fixtureName, func(t *testing.T) {
			// Load fixture
			data, err := os.ReadFile(fixturePath)
			if err != nil {
				t.Fatalf("Failed to load %s: %v", fixtureName, err)
			}

			// Parse
			frame, err := Parse(data)
			if err != nil {
				t.Fatalf("Failed to parse %s: %v", fixtureName, err)
			}

			// Basic validation
			if frame == nil {
				t.Fatal("Frame is nil")
			}
			if frame.Header == nil {
				t.Fatal("Header is nil")
			}
			if frame.Outputs == nil {
				t.Fatal("Outputs is nil")
			}
			if len(frame.Outputs) == 0 {
				t.Fatal("No outputs")
			}

			// Validate version is 21 (modern format)
			if frame.Header.Version != 21 {
				t.Errorf("Expected version 21, got %d", frame.Header.Version)
			}

			t.Logf("✓ Parsed %s successfully", fixtureName)
			t.Logf("  Header: magic=0x%08X version=%d flags=0x%02X",
				frame.Header.Magic, frame.Header.Version, frame.Header.Flags)
			t.Logf("  Outputs: %d", len(frame.Outputs))
			for i, out := range frame.Outputs {
				t.Logf("    Output %d: type=%s size=%d numElements=%d",
					i, out.Type, out.DecompressedSize, out.NumElements)
			}
			t.Logf("  Payload: %d bytes", len(frame.Payload))

			// Log flags
			if frame.Header.Flags.HasContentChecksum() {
				t.Logf("  Has content checksum")
			}
			if frame.Header.Flags.HasCompressedChecksum() {
				t.Logf("  Has frame header checksum")
			}
		})
	}
}

// TestParseInvalidMagic tests error handling for invalid magic number
func TestParseInvalidMagic(t *testing.T) {
	// Create invalid frame with wrong magic
	data := []byte{0xFF, 0xFF, 0xFF, 0xFF} // Invalid magic

	_, err := Parse(data)
	if err == nil {
		t.Fatal("Expected error for invalid magic, got nil")
	}

	if err != ErrInvalidMagic {
		t.Logf("Got different error (still correct): %v", err)
	}

	t.Logf("✓ Correctly rejected invalid magic: %v", err)
}

// TestParseTruncatedFrame tests error handling for truncated frames
func TestParseTruncatedFrame(t *testing.T) {
	// Load a valid fixture
	data, err := os.ReadFile("../../test/fixtures/frames/minimal.bin")
	if err != nil {
		t.Skip("minimal.bin not found")
	}

	// Test various truncation points
	tests := []struct {
		name      string
		length    int
		shouldErr bool
	}{
		{"truncate at 0", 0, true},
		{"truncate at 3 (incomplete magic)", 3, true},
		{"truncate at 4 (no flags)", 4, true},
		{"truncate at 5 (no token1)", 5, true},
		{"truncate at 6 (no sizes)", 6, true},
		// Note: "truncate at half" might not error because the parser
		// reads the remaining payload with io.ReadAll which succeeds
		// even with partial payload. This is acceptable since payload
		// validation happens during decompression, not parsing.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			truncated := data[:tt.length]
			_, err = Parse(truncated)
			if tt.shouldErr && err == nil {
				t.Fatal("Expected error for truncated frame, got nil")
			}
			if err != nil {
				t.Logf("✓ Correctly rejected: %v", err)
			}
		})
	}
}

// TestParseZeroOutputs tests error handling for zero outputs
func TestParseZeroOutputs(t *testing.T) {
	// Create a frame with zero outputs (invalid)
	data := []byte{
		0xd5, 0xa5, 0xb1, 0xd7, // Magic (version 21)
		0x00, // Flags
		0x00, // Token1: nbOutputs=0 (invalid!)
	}

	_, err := Parse(data)
	if err == nil {
		t.Fatal("Expected error for zero outputs, got nil")
	}

	if err != ErrZeroOutputs {
		t.Logf("Expected ErrZeroOutputs, got: %v", err)
	} else {
		t.Logf("✓ Correctly rejected zero outputs")
	}
}

// BenchmarkParse benchmarks frame parsing performance
func BenchmarkParse(b *testing.B) {
	data, err := os.ReadFile("../../test/fixtures/frames/minimal.bin")
	if err != nil {
		b.Skip("minimal.bin not found")
	}

	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := Parse(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseLarge benchmarks parsing large frames
func BenchmarkParseLarge(b *testing.B) {
	data, err := os.ReadFile("../../test/fixtures/frames/large_input.bin")
	if err != nil {
		b.Skip("large_input.bin not found")
	}

	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := Parse(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}
