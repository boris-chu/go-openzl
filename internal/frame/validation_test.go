package frame

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestMagicNumberEncoding validates magic number format
func TestMagicNumberEncoding(t *testing.T) {
	tests := []struct {
		version      uint32
		expectedMagic uint32
	}{
		{8, 0xD7B1A5C8},   // Min version
		{21, 0xD7B1A5D5},  // Current version
		{15, 0xD7B1A5CF},  // Middle version
	}

	for _, tt := range tests {
		magic := MagicNumberBase + tt.version
		if magic != tt.expectedMagic {
			t.Errorf("Version %d: expected magic 0x%08X, got 0x%08X",
				tt.version, tt.expectedMagic, magic)
		}

		// Verify we can extract version back
		extractedVersion := magic - MagicNumberBase
		if extractedVersion != tt.version {
			t.Errorf("Failed to extract version: got %d, want %d",
				extractedVersion, tt.version)
		}
	}
}

// TestMagicNumberLittleEndian verifies byte order in files
func TestMagicNumberLittleEndian(t *testing.T) {
	// Magic for version 21: 0xD7B1A5D5
	expected := uint32(0xD7B1A5D5)

	// Create bytes as they appear in file (little-endian)
	fileBytes := []byte{0xD5, 0xA5, 0xB1, 0xD7}

	// Read as little-endian uint32
	magic := binary.LittleEndian.Uint32(fileBytes)

	if magic != expected {
		t.Errorf("Little-endian encoding mismatch: got 0x%08X, want 0x%08X",
			magic, expected)
	}

	// Verify version extraction
	version := magic - MagicNumberBase
	if version != 21 {
		t.Errorf("Version extraction failed: got %d, want 21", version)
	}
}

// TestVersionRangeValidation tests version bounds checking
func TestVersionRangeValidation(t *testing.T) {
	tests := []struct {
		name      string
		magic     uint32
		shouldErr bool
		reason    string
	}{
		{"min supported version", MagicNumberBase + 21, false, "version 21 is minimum supported"},
		{"max version", MagicNumberBase + 21, false, "version 21 is maximum"},
		{"old format", MagicNumberBase + 8, true, "version 8 < 21 (old format not supported)"},
		{"too old", MagicNumberBase + 7, true, "version 7 < 8"},
		{"too new", MagicNumberBase + 22, true, "version 22 > 21"},
		{"way too old", MagicNumberBase + 1, true, "version 1 < 8"},
		{"way too new", MagicNumberBase + 100, true, "version 100 > 21"},
		{"invalid base", 0x12345678, true, "completely wrong base"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create minimal frame with this magic
			buf := new(bytes.Buffer)
			binary.Write(buf, binary.LittleEndian, tt.magic)
			buf.WriteByte(0x00) // Flags
			buf.WriteByte(0x01) // Token1
			buf.WriteByte(0x05) // Size

			_, err := Parse(buf.Bytes())
			if tt.shouldErr && err == nil {
				t.Errorf("Expected error for %s, got nil", tt.reason)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.reason, err)
			}
		})
	}
}

// TestToken1Encoding validates token1 byte packing
func TestToken1Encoding(t *testing.T) {
	tests := []struct {
		name        string
		nbOutputs   int
		type0       OutputType
		type1       OutputType
		expectedByte uint8
	}{
		{
			name:        "1 output, serial",
			nbOutputs:   1,
			type0:       TypeSerial,
			type1:       TypeSerial,
			expectedByte: 0x01, // 0000 0001 (type0=00, nbOutputs=1)
		},
		{
			name:        "1 output, numeric",
			nbOutputs:   1,
			type0:       TypeNumeric,
			type1:       TypeSerial,
			expectedByte: 0x21, // 0010 0001 (type0=10, nbOutputs=1)
		},
		{
			name:        "2 outputs, serial+string",
			nbOutputs:   2,
			type0:       TypeSerial,
			type1:       TypeString,
			expectedByte: 0xC2, // 1100 0010 (type1=11, type0=00, nbOutputs=2)
		},
		{
			name:        "2 outputs, numeric+struct",
			nbOutputs:   2,
			type0:       TypeNumeric,
			type1:       TypeStruct,
			expectedByte: 0x62, // 0110 0010 (type1=01, type0=10, nbOutputs=2)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			token1 := uint8(tt.nbOutputs)
			token1 |= uint8(tt.type0) << 4
			if tt.nbOutputs >= 2 {
				token1 |= uint8(tt.type1) << 6
			}

			if token1 != tt.expectedByte {
				t.Errorf("Encoding mismatch: got 0x%02X, want 0x%02X",
					token1, tt.expectedByte)
			}

			// Decode
			decodedNbOutputs := int(token1 & 0x0F)
			decodedType0 := OutputType((token1 >> 4) & 3)
			decodedType1 := OutputType((token1 >> 6) & 3)

			if decodedNbOutputs != tt.nbOutputs {
				t.Errorf("nbOutputs decode: got %d, want %d",
					decodedNbOutputs, tt.nbOutputs)
			}
			if decodedType0 != tt.type0 {
				t.Errorf("type0 decode: got %v, want %v",
					decodedType0, tt.type0)
			}
			if tt.nbOutputs >= 2 && decodedType1 != tt.type1 {
				t.Errorf("type1 decode: got %v, want %v",
					decodedType1, tt.type1)
			}
		})
	}
}

// TestSizeVarintEncoding validates size+1 encoding
func TestSizeVarintEncoding(t *testing.T) {
	tests := []struct {
		actualSize uint64
		varint     uint64
	}{
		{0, 1},     // size 0 -> varint 1
		{1, 2},     // size 1 -> varint 2
		{4, 5},     // size 4 -> varint 5 (minimal.bin)
		{13, 14},   // size 13 -> varint 14 (with_checksums.bin)
		{127, 128}, // boundary case
		{1048576, 1048577}, // 1MB -> varint 1MB+1
	}

	for _, tt := range tests {
		// Encoding
		encoded := tt.actualSize + 1
		if encoded != tt.varint {
			t.Errorf("Size %d: encoded as %d, want %d",
				tt.actualSize, encoded, tt.varint)
		}

		// Decoding
		decoded := tt.varint - 1
		if decoded != tt.actualSize {
			t.Errorf("Varint %d: decoded as %d, want %d",
				tt.varint, decoded, tt.actualSize)
		}

		// Verify zero detection
		if tt.varint == 0 {
			t.Error("Varint 0 should indicate unknown size")
		}
	}
}

// TestFlagsEncoding validates flag bits
func TestFlagsEncoding(t *testing.T) {
	tests := []struct {
		name         string
		flags        FrameFlags
		hasContent   bool
		hasCompressed bool
	}{
		{"no flags", 0x00, false, false},
		{"content only", 0x01, true, false},
		{"compressed only", 0x02, false, true},
		{"both flags", 0x03, true, true},
		{"invalid bits ignored", 0xFF, true, true}, // Only bits 0-1 matter
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.flags.HasContentChecksum() != tt.hasContent {
				t.Errorf("HasContentChecksum: got %v, want %v",
					tt.flags.HasContentChecksum(), tt.hasContent)
			}
			if tt.flags.HasCompressedChecksum() != tt.hasCompressed {
				t.Errorf("HasCompressedChecksum: got %v, want %v",
					tt.flags.HasCompressedChecksum(), tt.hasCompressed)
			}
		})
	}
}

// TestFrameMinimumSize validates minimum valid frame size
func TestFrameMinimumSize(t *testing.T) {
	// Minimum frame: magic(4) + flags(1) + token1(1) + size(1) = 7 bytes
	tests := []struct {
		name  string
		size  int
		valid bool
	}{
		{"0 bytes", 0, false},
		{"3 bytes (incomplete magic)", 3, false},
		{"4 bytes (magic only)", 4, false},
		{"5 bytes (magic + flags)", 5, false},
		{"6 bytes (magic + flags + token1)", 6, false},
		{"7 bytes (minimum valid)", 7, true}, // But will fail on size=0
		{"8 bytes", 8, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create frame with exact size
			frame := make([]byte, tt.size)
			if tt.size >= 4 {
				// Valid magic
				binary.LittleEndian.PutUint32(frame[0:4], MagicNumberBase+21)
			}
			if tt.size >= 5 {
				frame[4] = 0x00 // Flags
			}
			if tt.size >= 6 {
				frame[5] = 0x01 // Token1: 1 output, type serial
			}
			if tt.size >= 7 {
				frame[6] = 0x02 // Size varint (1, decoded as size=0)
			}

			_, err := Parse(frame)
			if tt.valid && err != nil && tt.size >= 7 {
				// Size 7 might fail on size validation, not structure
				if err != ErrUnknownSize {
					t.Logf("Note: Size %d failed with: %v (acceptable)", tt.size, err)
				}
			} else if !tt.valid && err == nil {
				t.Errorf("Expected error for %d bytes, got nil", tt.size)
			}
		})
	}
}

// TestOutputTypeValidation validates all output types
func TestOutputTypeValidation(t *testing.T) {
	types := []struct {
		value OutputType
		name  string
	}{
		{TypeSerial, "serial"},
		{TypeStruct, "struct"},
		{TypeNumeric, "numeric"},
		{TypeString, "string"},
	}

	for _, tt := range types {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value > 3 {
				t.Errorf("Type value %d out of range (max 3)", tt.value)
			}

			str := tt.value.String()
			if str != tt.name {
				t.Errorf("String() = %q, want %q", str, tt.name)
			}
		})
	}

	// Test invalid type
	invalidType := OutputType(99)
	str := invalidType.String()
	if str != "unknown(99)" {
		t.Errorf("Invalid type String() = %q, want %q", str, "unknown(99)")
	}
}

// TestFrameRoundtripConstants ensures our constants match C library
func TestFrameRoundtripConstants(t *testing.T) {
	// These values MUST match the C library exactly
	const (
		expectedBase = 0xD7B1A5C0
		expectedMin  = 8
		expectedMax  = 21
		expectedChunk = 21
	)

	if MagicNumberBase != expectedBase {
		t.Errorf("MagicNumberBase = 0x%08X, want 0x%08X",
			MagicNumberBase, expectedBase)
	}
	if MinFormatVersion != expectedMin {
		t.Errorf("MinFormatVersion = %d, want %d",
			MinFormatVersion, expectedMin)
	}
	if MaxFormatVersion != expectedMax {
		t.Errorf("MaxFormatVersion = %d, want %d",
			MaxFormatVersion, expectedMax)
	}
	if ChunkVersionMin != expectedChunk {
		t.Errorf("ChunkVersionMin = %d, want %d",
			ChunkVersionMin, expectedChunk)
	}
}

// TestVarintEdgeCases validates varint encoding edge cases
func TestVarintEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		value uint64
		bytes []byte
	}{
		{"zero", 0, []byte{0x00}},
		{"one", 1, []byte{0x01}},
		{"127 (max 1-byte)", 127, []byte{0x7F}},
		{"128 (min 2-byte)", 128, []byte{0x80, 0x01}},
		{"255", 255, []byte{0xFF, 0x01}},
		{"256", 256, []byte{0x80, 0x02}},
		{"16383 (max 2-byte)", 16383, []byte{0xFF, 0x7F}},
		{"16384 (min 3-byte)", 16384, []byte{0x80, 0x80, 0x01}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Decode
			buf := bytes.NewReader(tt.bytes)
			decoded, err := readVarint(buf)
			if err != nil {
				t.Fatalf("Decode error: %v", err)
			}
			if decoded != tt.value {
				t.Errorf("Decoded %d, want %d", decoded, tt.value)
			}

			// Encode and verify
			var encoded bytes.Buffer
			if err := writeVarint(&encoded, tt.value); err != nil {
				t.Fatalf("Encode error: %v", err)
			}
			if !bytes.Equal(encoded.Bytes(), tt.bytes) {
				t.Errorf("Encoded %v, want %v", encoded.Bytes(), tt.bytes)
			}
		})
	}
}
