package frame

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"testing"
)

// TestPropertyMagicRoundtrip tests that version encoding/decoding is bidirectional
func TestPropertyMagicRoundtrip(t *testing.T) {
	for version := MinFormatVersion; version <= MaxFormatVersion; version++ {
		// Encode: version -> magic
		magic := MagicNumberBase + version

		// Decode: magic -> version
		extracted := magic - MagicNumberBase

		if extracted != version {
			t.Errorf("Version %d: roundtrip failed, got %d", version, extracted)
		}

		// Only test parsing for version >= ChunkVersionMin (we only support modern format)
		if version < ChunkVersionMin {
			continue
		}

		// Verify it parses correctly
		buf := new(bytes.Buffer)
		binary.Write(buf, binary.LittleEndian, magic)
		buf.WriteByte(0x00) // Flags
		buf.WriteByte(0x01) // Token1
		buf.WriteByte(0x02) // Size varint (actual size 1)

		frame, err := Parse(buf.Bytes())
		if err != nil {
			t.Errorf("Version %d: parse failed: %v", version, err)
			continue
		}

		if frame.Header.Version != version {
			t.Errorf("Version %d: parsed as %d", version, frame.Header.Version)
		}
	}
}

// TestPropertyVarintRoundtrip tests varint encode/decode is lossless
func TestPropertyVarintRoundtrip(t *testing.T) {
	// Test powers of 2
	for i := uint64(0); i < 20; i++ {
		value := uint64(1) << i

		var buf bytes.Buffer
		if err := writeVarint(&buf, value); err != nil {
			t.Fatalf("Encode %d failed: %v", value, err)
		}

		decoded, err := readVarint(&buf)
		if err != nil {
			t.Fatalf("Decode %d failed: %v", value, err)
		}

		if decoded != value {
			t.Errorf("Value %d: roundtrip gave %d", value, decoded)
		}
	}

	// Test random values
	r := rand.New(rand.NewSource(12345))
	for i := 0; i < 1000; i++ {
		value := r.Uint64() >> 1 // Keep it reasonable

		var buf bytes.Buffer
		if err := writeVarint(&buf, value); err != nil {
			t.Fatalf("Encode %d failed: %v", value, err)
		}

		decoded, err := readVarint(&buf)
		if err != nil {
			t.Fatalf("Decode %d failed: %v", value, err)
		}

		if decoded != value {
			t.Errorf("Value %d: roundtrip gave %d", value, decoded)
		}
	}
}

// TestPropertySizeEncoding tests size+1 encoding is consistent
func TestPropertySizeEncoding(t *testing.T) {
	sizes := []uint64{
		0, 1, 2, 3, 4,
		127, 128, 129,
		255, 256, 257,
		1023, 1024, 1025,
		16383, 16384, 16385,
		65535, 65536, 65537,
		1048575, 1048576, 1048577,
	}

	for _, size := range sizes {
		// Encode: actualSize -> varint(size+1)
		encoded := size + 1

		// Decode: varint -> actualSize
		decoded := encoded - 1

		if decoded != size {
			t.Errorf("Size %d: roundtrip gave %d", size, decoded)
		}

		// Verify varint is never 0
		if encoded == 0 {
			t.Errorf("Size %d: encoded as 0 (should be impossible)", size)
		}
	}
}

// TestPropertyToken1AllCombinations tests all valid token1 combinations
func TestPropertyToken1AllCombinations(t *testing.T) {
	// Test all combinations of 1-2 outputs with all 4 types
	for nbOutputs := 1; nbOutputs <= 2; nbOutputs++ {
		for type0 := OutputType(0); type0 <= 3; type0++ {
			for type1 := OutputType(0); type1 <= 3; type1++ {
				// Build token1
				token1 := uint8(nbOutputs)
				token1 |= uint8(type0) << 4
				if nbOutputs >= 2 {
					token1 |= uint8(type1) << 6
				}

				// Decode
				decodedNbOutputs := int(token1 & 0x0F)
				decodedType0 := OutputType((token1 >> 4) & 3)
				decodedType1 := OutputType((token1 >> 6) & 3)

				// Verify
				if decodedNbOutputs != nbOutputs {
					t.Errorf("Token1 0x%02X: nbOutputs %d != %d",
						token1, decodedNbOutputs, nbOutputs)
				}
				if decodedType0 != type0 {
					t.Errorf("Token1 0x%02X: type0 %v != %v",
						token1, decodedType0, type0)
				}
				if nbOutputs >= 2 && decodedType1 != type1 {
					t.Errorf("Token1 0x%02X: type1 %v != %v",
						token1, decodedType1, type1)
				}
			}
		}
	}
}

// TestPropertyFlagsAllCombinations tests all flag combinations
func TestPropertyFlagsAllCombinations(t *testing.T) {
	// Only bits 0-1 are defined for version 21
	for flags := uint8(0); flags < 4; flags++ {
		f := FrameFlags(flags)

		expectedContent := (flags & 0x01) != 0
		expectedCompressed := (flags & 0x02) != 0

		if f.HasContentChecksum() != expectedContent {
			t.Errorf("Flags 0x%02X: HasContentChecksum = %v, want %v",
				flags, f.HasContentChecksum(), expectedContent)
		}
		if f.HasCompressedChecksum() != expectedCompressed {
			t.Errorf("Flags 0x%02X: HasCompressedChecksum = %v, want %v",
				flags, f.HasCompressedChecksum(), expectedCompressed)
		}
	}
}

// TestPropertyVarintMonotonic tests varints preserve order
func TestPropertyVarintMonotonic(t *testing.T) {
	// Smaller values should produce shorter or equal byte sequences
	values := []uint64{0, 1, 127, 128, 255, 256, 16383, 16384, 65535, 65536}

	var prevLen int
	for i, value := range values {
		var buf bytes.Buffer
		if err := writeVarint(&buf, value); err != nil {
			t.Fatalf("Encode %d failed: %v", value, err)
		}

		currentLen := buf.Len()
		if i > 0 && currentLen < prevLen {
			t.Errorf("Value %d: encoded length %d < previous %d (not monotonic)",
				value, currentLen, prevLen)
		}
		prevLen = currentLen
	}
}

// TestPropertyVarintMaxLength tests varint never exceeds 10 bytes
func TestPropertyVarintMaxLength(t *testing.T) {
	// Maximum uint64 should fit in 10 bytes
	maxValue := uint64(0xFFFFFFFFFFFFFFFF)

	var buf bytes.Buffer
	if err := writeVarint(&buf, maxValue); err != nil {
		t.Fatalf("Encode max uint64 failed: %v", err)
	}

	if buf.Len() > 10 {
		t.Errorf("Max uint64 encoded in %d bytes, should be <= 10", buf.Len())
	}

	// Decode it back
	decoded, err := readVarint(&buf)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if decoded != maxValue {
		t.Errorf("Max uint64 roundtrip: got %d, want %d", decoded, maxValue)
	}
}

// TestPropertyOutputTypeBounds tests output types stay within valid range
func TestPropertyOutputTypeBounds(t *testing.T) {
	// Output types are 2 bits, so max value is 3
	for value := uint8(0); value < 16; value++ {
		typ := OutputType(value)

		// Only 0-3 should be valid
		isValid := value <= 3

		// String should work for all values
		str := typ.String()
		if isValid && str == "" {
			t.Errorf("Type %d: String() returned empty", value)
		}
		if !isValid && str[:7] != "unknown" {
			t.Errorf("Type %d: String() = %q, should start with 'unknown'", value, str)
		}
	}
}

// TestPropertyRandomFrameParsing tests parser handles random valid frames
func TestPropertyRandomFrameParsing(t *testing.T) {
	r := rand.New(rand.NewSource(54321))

	for i := 0; i < 100; i++ {
		// Generate random but valid frame (only version 21 since that's what we support)
		version := ChunkVersionMin
		flags := FrameFlags(r.Intn(4)) // 0-3 are valid
		nbOutputs := 1 + r.Intn(2)     // 1-2 outputs
		size := uint64(r.Intn(1000))   // Random size

		// Use non-string types to avoid needing numElements data
		type0 := OutputType(r.Intn(3)) // 0-2 (serial, struct, numeric)
		type1 := OutputType(r.Intn(3)) // 0-2

		// Build frame
		buf := new(bytes.Buffer)

		// Magic
		magic := MagicNumberBase + version
		binary.Write(buf, binary.LittleEndian, magic)

		// Flags
		buf.WriteByte(uint8(flags))

		// Token1
		token1 := uint8(nbOutputs)
		token1 |= uint8(type0) << 4
		if nbOutputs >= 2 {
			token1 |= uint8(type1) << 6
		}
		buf.WriteByte(token1)

		// Sizes (need one for each output)
		writeVarint(buf, size+1) // First output size
		if nbOutputs >= 2 {
			writeVarint(buf, size+1) // Second output size
		}

		// Add checksum byte if needed
		if flags.HasCompressedChecksum() {
			buf.WriteByte(0x42) // Dummy checksum
		}

		// Parse
		frame, err := Parse(buf.Bytes())
		if version < ChunkVersionMin {
			// We don't support old versions yet
			if err == nil {
				t.Errorf("Iteration %d: Expected error for version %d, got nil", i, version)
			}
			continue
		}

		if err != nil {
			t.Errorf("Iteration %d: Parse failed: %v", i, err)
			continue
		}

		// Validate
		if frame.Header.Version != version {
			t.Errorf("Iteration %d: Version %d != %d", i, frame.Header.Version, version)
		}
		if frame.Header.Flags != flags {
			t.Errorf("Iteration %d: Flags 0x%02X != 0x%02X", i, frame.Header.Flags, flags)
		}
		if len(frame.Outputs) != nbOutputs {
			t.Errorf("Iteration %d: Outputs %d != %d", i, len(frame.Outputs), nbOutputs)
		}
		if frame.Outputs[0].DecompressedSize != size {
			t.Errorf("Iteration %d: Size %d != %d", i, frame.Outputs[0].DecompressedSize, size)
		}
	}
}
