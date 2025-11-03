// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package purgo

import (
	"bytes"
	"strings"
	"testing"
)

// TestCompressSmart_JSON tests intelligent compression on JSON data
func TestCompressSmart_JSON(t *testing.T) {
	// Create realistic JSON with repeated field names (like BitLocker database)
	jsonTemplate := `{"password_id":"A1B2C3D4-E5F6-7890-ABCD-EF12345670%02d","recovery_password":"123456-789012-345678-901234-567890-123456-789012-000%03d","date_stored":"2025-11-01T12:00:00Z","distinguished_name":"CN=COMPUTER%03d,CN=Computers,DC=ladpss,DC=org"}`

	var jsonBuilder strings.Builder
	jsonBuilder.WriteString(`{"computers":{`)
	for i := 0; i < 50; i++ {
		if i > 0 {
			jsonBuilder.WriteString(",")
		}
		jsonBuilder.WriteString(`"COMPUTER`)
		jsonBuilder.WriteString(strings.Repeat("0", 3-len(strings.Trim(strings.TrimLeft(strings.Trim(strings.TrimSpace(strings.Repeat(" ", 100)), " "), "0"), " "))))
		entry := jsonTemplate
		// Simple replacement for testing
		jsonBuilder.WriteString(entry)
	}
	jsonBuilder.WriteString(`}}`)

	jsonData := []byte(jsonBuilder.String())

	// Test CompressSmart
	compressed, err := CompressSmart(jsonData)
	if err != nil {
		t.Fatalf("CompressSmart failed: %v", err)
	}

	// Should achieve significantly better compression than Huffman-only
	// Expected: 5-15× compression ratio (much better than 1.51× from old Compress())
	ratio := float64(len(jsonData)) / float64(len(compressed))
	if ratio < 2.0 {
		t.Errorf("CompressSmart achieved only %.2f× compression (expected >2×)", ratio)
	}

	t.Logf("CompressSmart: %d bytes → %d bytes (%.2f× compression)", len(jsonData), len(compressed), ratio)

	// Verify decompression works
	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("Decompression failed: %v", err)
	}

	if !bytes.Equal(decompressed, jsonData) {
		t.Errorf("Roundtrip mismatch:\nOriginal:     %d bytes\nDecompressed: %d bytes", len(jsonData), len(decompressed))
	}
}

// TestCompressSmart_RepeatedStrings tests compression on highly repetitive text
func TestCompressSmart_RepeatedStrings(t *testing.T) {
	// Create data with many repeated strings (ideal for LZ77→Huffman)
	data := bytes.Repeat([]byte("password_id recovery_password distinguished_name "), 100)

	compressed, err := CompressSmart(data)
	if err != nil {
		t.Fatalf("CompressSmart failed: %v", err)
	}

	ratio := float64(len(data)) / float64(len(compressed))
	if ratio < 10.0 {
		t.Errorf("CompressSmart achieved only %.2f× compression on repeated strings (expected >10×)", ratio)
	}

	t.Logf("Repeated strings: %d bytes → %d bytes (%.2f× compression)", len(data), len(compressed), ratio)

	// Verify roundtrip
	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("Decompression failed: %v", err)
	}

	if !bytes.Equal(decompressed, data) {
		t.Errorf("Roundtrip mismatch")
	}
}

// TestCompressSmart_SparseData tests compression on sparse/repetitive data
func TestCompressSmart_SparseData(t *testing.T) {
	// Create sparse array (many zeros) - ideal for RLE→Huffman
	data := make([]byte, 1000)
	// Add a few non-zero values
	for i := 0; i < 50; i++ {
		data[i*20] = 1
	}

	compressed, err := CompressSmart(data)
	if err != nil {
		t.Fatalf("CompressSmart failed: %v", err)
	}

	ratio := float64(len(data)) / float64(len(compressed))
	if ratio < 5.0 {
		t.Errorf("CompressSmart achieved only %.2f× compression on sparse data (expected >5×)", ratio)
	}

	t.Logf("Sparse data: %d bytes → %d bytes (%.2f× compression)", len(data), len(compressed), ratio)

	// Verify roundtrip
	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("Decompression failed: %v", err)
	}

	if !bytes.Equal(decompressed, data) {
		t.Errorf("Roundtrip mismatch")
	}
}

// TestCompressSmart_RandomData tests fallback behavior on incompressible data
func TestCompressSmart_RandomData(t *testing.T) {
	// Random data should use Identity codec (no compression)
	data := []byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0, 0x11, 0x22}

	compressed, err := CompressSmart(data)
	if err != nil {
		t.Fatalf("CompressSmart failed: %v", err)
	}

	// Should not expand too much (Identity codec fallback)
	// Small random data (10 bytes) will have frame overhead, so allow up to 3× expansion
	expansionRatio := float64(len(compressed)) / float64(len(data))
	if expansionRatio > 3.0 {
		t.Errorf("CompressSmart expanded data by %.2f× (expected <3×)", expansionRatio)
	}

	t.Logf("Random data: %d bytes → %d bytes (%.2f× size)", len(data), len(compressed), expansionRatio)

	// Verify roundtrip
	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("Decompression failed: %v", err)
	}

	if !bytes.Equal(decompressed, data) {
		t.Errorf("Roundtrip mismatch")
	}
}

// TestCompressSmart_EmptyData tests error handling for empty input
func TestCompressSmart_EmptyData(t *testing.T) {
	_, err := CompressSmart([]byte{})
	if err == nil {
		t.Errorf("CompressSmart should fail on empty data")
	}
}

// TestCompressSmart_VsCompress compares CompressSmart to old Compress
func TestCompressSmart_VsCompress(t *testing.T) {
	// Create data with repeated patterns (JSON-like)
	data := bytes.Repeat([]byte(`{"field":"value","field":"value"}`), 50)

	// Test old Compress (Huffman-only)
	compressedOld, err := Compress(data)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	// Test new CompressSmart (LZ77→Huffman)
	compressedSmart, err := CompressSmart(data)
	if err != nil {
		t.Fatalf("CompressSmart failed: %v", err)
	}

	ratioOld := float64(len(data)) / float64(len(compressedOld))
	ratioSmart := float64(len(data)) / float64(len(compressedSmart))

	t.Logf("Compress (Huffman):       %d bytes (%.2f× compression)", len(compressedOld), ratioOld)
	t.Logf("CompressSmart (LZ77+Huf): %d bytes (%.2f× compression)", len(compressedSmart), ratioSmart)

	// CompressSmart should achieve significantly better compression
	if ratioSmart <= ratioOld {
		t.Errorf("CompressSmart (%.2f×) should beat Compress (%.2f×)", ratioSmart, ratioOld)
	}

	improvement := (ratioSmart / ratioOld) - 1.0
	t.Logf("CompressSmart improvement: %.1f%% better than Compress", improvement*100)
}
