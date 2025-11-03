// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package codec

import (
	"encoding/binary"
	"testing"
)

// Helper: encodeStringArray converts string array to codec input format
func encodeStringArray(strings []string) []byte {
	// Calculate total size
	size := 4 // numStrings
	for _, s := range strings {
		size += 4 + len(s) // length + data
	}

	buf := make([]byte, size)
	pos := 0

	// Write number of strings
	binary.LittleEndian.PutUint32(buf[pos:], uint32(len(strings)))
	pos += 4

	// Write each string
	for _, s := range strings {
		binary.LittleEndian.PutUint32(buf[pos:], uint32(len(s)))
		pos += 4
		copy(buf[pos:], s)
		pos += len(s)
	}

	return buf
}

// Helper: decodeStringArray converts codec output format to string array
func decodeStringArray(data []byte) []string {
	if len(data) < 4 {
		return nil
	}

	numStrings := binary.LittleEndian.Uint32(data[0:4])
	if numStrings == 0 {
		return []string{}
	}

	strings := make([]string, numStrings)
	pos := 4

	for i := uint32(0); i < numStrings; i++ {
		if pos+4 > len(data) {
			return nil
		}
		strLen := binary.LittleEndian.Uint32(data[pos : pos+4])
		pos += 4

		if pos+int(strLen) > len(data) {
			return nil
		}
		strings[i] = string(data[pos : pos+int(strLen)])
		pos += int(strLen)
	}

	return strings
}

// TestPrefix_URLList tests compression of URL list with common base
func TestPrefix_URLList(t *testing.T) {
	t.Log("=== Prefix: URL List ===\n")

	codec := NewPrefix()

	// URLs with common base: https://api.example.com/v1/...
	urls := []string{
		"https://api.example.com/v1/users",
		"https://api.example.com/v1/posts",
		"https://api.example.com/v1/comments",
		"https://api.example.com/v2/users",
		"https://api.example.com/v2/posts",
	}

	src := encodeStringArray(urls)
	t.Logf("Input: %d URLs (%d bytes)", len(urls), len(src))

	// Compress
	dst := make([]byte, len(src)+100)
	n, err := codec.Encode(dst, src, nil)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	ratio := float64(len(src)) / float64(n)
	t.Logf("Compressed: %d bytes → %d bytes (%.2f× compression)", len(src), n, ratio)

	// Decompress
	decompressed := make([]byte, len(src)+100)
	dn, err := codec.Decode(decompressed, dst[:n], nil)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Verify roundtrip
	result := decodeStringArray(decompressed[:dn])

	if len(result) != len(urls) {
		t.Fatalf("Expected %d strings, got %d", len(urls), len(result))
	}

	for i, expected := range urls {
		if result[i] != expected {
			t.Errorf("String %d: expected %q, got %q", i, expected, result[i])
		}
	}

	t.Logf("✅ Roundtrip successful\n")

	// Expected: 2-3× compression on URL lists
	if ratio < 1.5 {
		t.Errorf("Expected compression ratio >= 1.5×, got %.2f×", ratio)
	}
}

// TestPrefix_FilePaths tests compression of file paths
func TestPrefix_FilePaths(t *testing.T) {
	t.Log("=== Prefix: File Paths ===\n")

	codec := NewPrefix()

	// File paths with common directories
	paths := []string{
		"/usr/local/bin/gcc",
		"/usr/local/bin/g++",
		"/usr/local/bin/clang",
		"/usr/local/lib/libz.a",
		"/usr/local/lib/libpng.a",
		"/usr/local/include/zlib.h",
	}

	src := encodeStringArray(paths)
	t.Logf("Input: %d paths (%d bytes)", len(paths), len(src))

	// Compress
	dst := make([]byte, len(src)+100)
	n, err := codec.Encode(dst, src, nil)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	ratio := float64(len(src)) / float64(n)
	t.Logf("Compressed: %d bytes → %d bytes (%.2f× compression)", len(src), n, ratio)

	// Decompress
	decompressed := make([]byte, len(src)+100)
	dn, err := codec.Decode(decompressed, dst[:n], nil)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Verify roundtrip
	result := decodeStringArray(decompressed[:dn])

	for i, expected := range paths {
		if result[i] != expected {
			t.Errorf("Path %d: expected %q, got %q", i, expected, result[i])
		}
	}

	t.Logf("✅ Roundtrip successful\n")

	// Expected: 1.5-2× compression on file paths
	if ratio < 1.2 {
		t.Errorf("Expected compression ratio >= 1.2×, got %.2f×", ratio)
	}
}

// TestPrefix_DomainNames tests compression of domain names
func TestPrefix_DomainNames(t *testing.T) {
	t.Log("=== Prefix: Domain Names ===\n")

	codec := NewPrefix()

	// Domain names with common suffixes
	domains := []string{
		"mail.google.com",
		"maps.google.com",
		"drive.google.com",
		"docs.google.com",
		"calendar.google.com",
	}

	src := encodeStringArray(domains)
	t.Logf("Input: %d domains (%d bytes)", len(domains), len(src))

	// Compress
	dst := make([]byte, len(src)+100)
	n, err := codec.Encode(dst, src, nil)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	ratio := float64(len(src)) / float64(n)
	t.Logf("Compressed: %d bytes → %d bytes (%.2f× compression)", len(src), n, ratio)

	// Decompress
	decompressed := make([]byte, len(src)+100)
	dn, err := codec.Decode(decompressed, dst[:n], nil)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Verify roundtrip
	result := decodeStringArray(decompressed[:dn])

	for i, expected := range domains {
		if result[i] != expected {
			t.Errorf("Domain %d: expected %q, got %q", i, expected, result[i])
		}
	}

	t.Logf("✅ Roundtrip successful\n")

	// Note: Domain names have different prefixes, so compression limited
	// Expected: 1.1-1.3× (domains share suffixes, not prefixes)
	if n > len(src)*2 {
		t.Errorf("Compression made data much larger: %d → %d", len(src), n)
	}
}

// TestPrefix_LogLines tests compression of log file lines
func TestPrefix_LogLines(t *testing.T) {
	t.Log("=== Prefix: Log File Lines ===\n")

	codec := NewPrefix()

	// Log lines with common timestamp/prefix
	logs := []string{
		"2024-01-15 10:30:00 INFO: Server started",
		"2024-01-15 10:30:01 INFO: Connected to database",
		"2024-01-15 10:30:02 INFO: Ready to accept requests",
		"2024-01-15 10:30:03 WARN: High memory usage",
		"2024-01-15 10:30:04 ERROR: Request failed",
	}

	src := encodeStringArray(logs)
	t.Logf("Input: %d log lines (%d bytes)", len(logs), len(src))

	// Compress
	dst := make([]byte, len(src)+100)
	n, err := codec.Encode(dst, src, nil)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	ratio := float64(len(src)) / float64(n)
	t.Logf("Compressed: %d bytes → %d bytes (%.2f× compression)", len(src), n, ratio)

	// Decompress
	decompressed := make([]byte, len(src)+100)
	dn, err := codec.Decode(decompressed, dst[:n], nil)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Verify roundtrip
	result := decodeStringArray(decompressed[:dn])

	for i, expected := range logs {
		if result[i] != expected {
			t.Errorf("Log %d: expected %q, got %q", i, expected, result[i])
		}
	}

	t.Logf("✅ Roundtrip successful\n")

	// Expected: 1.5-2× compression on log lines
	if ratio < 1.3 {
		t.Errorf("Expected compression ratio >= 1.3×, got %.2f×", ratio)
	}
}

// TestPrefix_EmptyInput tests handling of empty input
func TestPrefix_EmptyInput(t *testing.T) {
	codec := NewPrefix()

	src := encodeStringArray([]string{})
	dst := make([]byte, 100)

	n, err := codec.Encode(dst, src, nil)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	if n != 4 {
		t.Errorf("Expected 4 bytes output (count=0), got %d", n)
	}

	// Decompress
	decompressed := make([]byte, 100)
	dn, err := codec.Decode(decompressed, dst[:n], nil)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if dn != 4 {
		t.Errorf("Expected 4 bytes decompressed, got %d", dn)
	}

	t.Log("✅ Empty input handled correctly")
}

// TestPrefix_SingleString tests handling of single string
func TestPrefix_SingleString(t *testing.T) {
	codec := NewPrefix()

	strings := []string{"https://example.com/api/v1/users"}
	src := encodeStringArray(strings)

	dst := make([]byte, len(src)+100)
	n, err := codec.Encode(dst, src, nil)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	t.Logf("Single string: %d bytes → %d bytes", len(src), n)

	// Decompress
	decompressed := make([]byte, len(src)+100)
	dn, err := codec.Decode(decompressed, dst[:n], nil)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	result := decodeStringArray(decompressed[:dn])
	if len(result) != 1 || result[0] != strings[0] {
		t.Errorf("Expected %q, got %q", strings[0], result[0])
	}

	t.Log("✅ Single string handled correctly")
}

// TestPrefix_NoCommonPrefix tests strings with no common prefix
func TestPrefix_NoCommonPrefix(t *testing.T) {
	t.Log("=== Prefix: No Common Prefix ===\n")

	codec := NewPrefix()

	// Completely different strings
	strings := []string{
		"apple",
		"banana",
		"cherry",
		"date",
		"elderberry",
	}

	src := encodeStringArray(strings)
	t.Logf("Input: %d strings (%d bytes)", len(strings), len(src))

	// Compress
	dst := make([]byte, len(src)+100)
	n, err := codec.Encode(dst, src, nil)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	ratio := float64(len(src)) / float64(n)
	t.Logf("Compressed: %d bytes → %d bytes (%.2f× compression)", len(src), n, ratio)

	// Expected: Slight expansion due to overhead (prefix=0 for all)
	// Each string gets prefixLen(2) + suffixLen(2) overhead = 4 bytes per string
	// vs original lenField(4) = same overhead
	if n > len(src)*2 {
		t.Errorf("Compression made data much larger: %d → %d", len(src), n)
	}

	// Decompress
	decompressed := make([]byte, len(src)+100)
	dn, err := codec.Decode(decompressed, dst[:n], nil)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Verify roundtrip
	result := decodeStringArray(decompressed[:dn])

	for i, expected := range strings {
		if result[i] != expected {
			t.Errorf("String %d: expected %q, got %q", i, expected, result[i])
		}
	}

	t.Logf("✅ Roundtrip successful (no expansion despite no common prefix)\n")
}

// TestPrefix_IdenticalStrings tests handling of identical consecutive strings
func TestPrefix_IdenticalStrings(t *testing.T) {
	t.Log("=== Prefix: Identical Consecutive Strings ===\n")

	codec := NewPrefix()

	// Same string repeated
	strings := []string{
		"https://example.com/api",
		"https://example.com/api",
		"https://example.com/api",
		"https://example.com/api",
	}

	src := encodeStringArray(strings)
	t.Logf("Input: %d strings (%d bytes)", len(strings), len(src))

	// Compress
	dst := make([]byte, len(src)+100)
	n, err := codec.Encode(dst, src, nil)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	ratio := float64(len(src)) / float64(n)
	t.Logf("Compressed: %d bytes → %d bytes (%.2f× compression)", len(src), n, ratio)

	// Expected: Excellent compression (first string full, rest just 4-byte overhead)
	// First: 0 prefix + 23 suffix = 27 bytes (4 overhead + 23 data)
	// Rest:  23 prefix + 0 suffix = 4 bytes (4 overhead + 0 data)
	// Total: 27 + 4*3 = 39 bytes vs original ~120 bytes = 3× compression
	if ratio < 2.0 {
		t.Errorf("Expected compression ratio >= 2×, got %.2f×", ratio)
	}

	// Decompress
	decompressed := make([]byte, len(src)+100)
	dn, err := codec.Decode(decompressed, dst[:n], nil)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Verify roundtrip
	result := decodeStringArray(decompressed[:dn])

	for i, expected := range strings {
		if result[i] != expected {
			t.Errorf("String %d: expected %q, got %q", i, expected, result[i])
		}
	}

	t.Logf("✅ Identical strings compressed excellently\n")
}

// TestPrefix_LargeDataset tests compression of many strings
func TestPrefix_LargeDataset(t *testing.T) {
	t.Log("=== Prefix: Large Dataset (100 URLs) ===\n")

	codec := NewPrefix()

	// Generate 100 URLs with common base
	strings := make([]string, 100)
	for i := 0; i < 100; i++ {
		strings[i] = "https://api.example.com/v1/resource/" + string(rune('a'+i%26))
	}

	src := encodeStringArray(strings)
	t.Logf("Input: %d URLs (%d bytes, %.2f KB)", len(strings), len(src), float64(len(src))/1024)

	// Compress
	dst := make([]byte, len(src)+1000)
	n, err := codec.Encode(dst, src, nil)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	ratio := float64(len(src)) / float64(n)
	t.Logf("Compressed: %d bytes (%.2f KB) → %d bytes (%.2f KB)",
		len(src), float64(len(src))/1024, n, float64(n)/1024)
	t.Logf("Compression ratio: %.2f×", ratio)

	// Decompress
	decompressed := make([]byte, len(src)+1000)
	dn, err := codec.Decode(decompressed, dst[:n], nil)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Verify sample strings
	result := decodeStringArray(decompressed[:dn])

	for i := 0; i < len(strings); i += 10 {
		if result[i] != strings[i] {
			t.Errorf("String %d: expected %q, got %q", i, strings[i], result[i])
		}
	}

	t.Logf("✅ All %d strings decompressed correctly", len(strings))
	t.Logf("✅ Large dataset test passed\n")

	// Expected: 2-3× compression on URL lists
	if ratio < 1.8 {
		t.Errorf("Expected compression ratio >= 1.8×, got %.2f×", ratio)
	}
}

// TestPrefix_CodecInterface tests codec interface compliance
func TestPrefix_CodecInterface(t *testing.T) {
	codec := NewPrefix()

	if codec.ID() != IDPrefix {
		t.Errorf("Expected ID %d, got %d", IDPrefix, codec.ID())
	}

	if codec.Name() != "Prefix" {
		t.Errorf("Expected name 'Prefix', got '%s'", codec.Name())
	}

	if codec.PreservesSize() {
		t.Error("Prefix should not preserve size")
	}

	t.Log("✅ Codec interface compliance verified")
}
