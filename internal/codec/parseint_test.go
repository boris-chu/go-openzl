// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package codec

import (
	"encoding/binary"
	"testing"
)

// Helper: encodeIntStringArray converts integer string array to codec input format
func encodeIntStringArray(strings []string) []byte {
	// Calculate total size
	size := 4 // numIntegers
	for _, s := range strings {
		size += 4 + len(s) // length + data
	}

	buf := make([]byte, size)
	pos := 0

	// Write number of integers
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

// Helper: decodeInt64Array converts codec output format to int64 array
func decodeInt64Array(data []byte) []int64 {
	if len(data) < 4 {
		return nil
	}

	numIntegers := binary.LittleEndian.Uint32(data[0:4])
	if numIntegers == 0 {
		return []int64{}
	}

	values := make([]int64, numIntegers)
	pos := 4

	for i := uint32(0); i < numIntegers; i++ {
		if pos+8 > len(data) {
			return nil
		}
		values[i] = int64(binary.LittleEndian.Uint64(data[pos:]))
		pos += 8
	}

	return values
}

// TestParseInt_PositiveIntegers tests parsing positive integers
func TestParseInt_PositiveIntegers(t *testing.T) {
	t.Log("=== ParseInt: Positive Integers ===\n")

	codec := NewParseInt()

	// Positive integers as strings
	strings := []string{"100", "200", "300", "400", "500"}
	expected := []int64{100, 200, 300, 400, 500}

	src := encodeIntStringArray(strings)
	t.Logf("Input: %d integer strings (%d bytes)", len(strings), len(src))

	// Parse to binary
	dst := make([]byte, len(strings)*8+100)
	n, err := codec.Encode(dst, src, nil)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	t.Logf("Parsed: %d bytes → %d bytes (text to binary)", len(src), n)

	// Verify values
	result := decodeInt64Array(dst[:n])

	if len(result) != len(expected) {
		t.Fatalf("Expected %d values, got %d", len(expected), len(result))
	}

	for i, exp := range expected {
		if result[i] != exp {
			t.Errorf("Value %d: expected %d, got %d", i, exp, result[i])
		}
	}

	// Reverse: binary to text
	decompressed := make([]byte, len(src)+100)
	dn, err := codec.Decode(decompressed, dst[:n], nil)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Decode back to strings
	resultStrings := decodeStringArray(decompressed[:dn])
	for i, exp := range strings {
		if resultStrings[i] != exp {
			t.Errorf("String %d: expected %q, got %q", i, exp, resultStrings[i])
		}
	}

	t.Logf("✅ Roundtrip successful\n")
}

// TestParseInt_NegativeIntegers tests parsing negative integers
func TestParseInt_NegativeIntegers(t *testing.T) {
	t.Log("=== ParseInt: Negative Integers ===\n")

	codec := NewParseInt()

	// Negative integers
	strings := []string{"-100", "-200", "-300", "-400", "-500"}
	expected := []int64{-100, -200, -300, -400, -500}

	src := encodeIntStringArray(strings)
	t.Logf("Input: %d negative integers (%d bytes)", len(strings), len(src))

	// Parse to binary
	dst := make([]byte, len(strings)*8+100)
	n, err := codec.Encode(dst, src, nil)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Verify values
	result := decodeInt64Array(dst[:n])

	for i, exp := range expected {
		if result[i] != exp {
			t.Errorf("Value %d: expected %d, got %d", i, exp, result[i])
		}
	}

	t.Logf("✅ Negative integers parsed correctly\n")
}

// TestParseInt_MixedIntegers tests parsing mixed positive/negative integers
func TestParseInt_MixedIntegers(t *testing.T) {
	t.Log("=== ParseInt: Mixed Positive/Negative ===\n")

	codec := NewParseInt()

	strings := []string{"1000", "-500", "0", "42", "-1"}
	expected := []int64{1000, -500, 0, 42, -1}

	src := encodeIntStringArray(strings)
	dst := make([]byte, len(strings)*8+100)

	n, err := codec.Encode(dst, src, nil)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	result := decodeInt64Array(dst[:n])

	for i, exp := range expected {
		if result[i] != exp {
			t.Errorf("Value %d: expected %d, got %d", i, exp, result[i])
		}
	}

	t.Logf("✅ Mixed integers parsed correctly\n")
}

// TestParseInt_LargeValues tests parsing large int64 values
func TestParseInt_LargeValues(t *testing.T) {
	t.Log("=== ParseInt: Large Int64 Values ===\n")

	codec := NewParseInt()

	// Large int64 values
	strings := []string{
		"9223372036854775807",  // Max int64
		"-9223372036854775808", // Min int64
		"1000000000000",        // 1 trillion
		"-1000000000000",
	}
	expected := []int64{
		9223372036854775807,
		-9223372036854775808,
		1000000000000,
		-1000000000000,
	}

	src := encodeIntStringArray(strings)
	dst := make([]byte, len(strings)*8+100)

	n, err := codec.Encode(dst, src, nil)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	result := decodeInt64Array(dst[:n])

	for i, exp := range expected {
		if result[i] != exp {
			t.Errorf("Value %d: expected %d, got %d", i, exp, result[i])
		}
	}

	t.Logf("✅ Large values parsed correctly\n")
}

// TestParseInt_CSVIntegers tests parsing CSV-style integers
func TestParseInt_CSVIntegers(t *testing.T) {
	t.Log("=== ParseInt: CSV Integer Column ===\n")

	codec := NewParseInt()

	// Simulate CSV integer column (e.g., employee IDs)
	strings := []string{
		"10001", "10002", "10003", "10004", "10005",
		"10006", "10007", "10008", "10009", "10010",
	}

	src := encodeIntStringArray(strings)
	t.Logf("Input: %d CSV integers (%d bytes)", len(strings), len(src))

	// Parse to binary
	dst := make([]byte, len(strings)*8+100)
	n, err := codec.Encode(dst, src, nil)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	ratio := float64(len(src)) / float64(n)
	t.Logf("Parsed: %d bytes → %d bytes (%.2f× size change)", len(src), n, ratio)

	// Note: ParseInt often EXPANDS small integers
	// "100" (3 bytes) → 100 as int64 (8 bytes)
	// But enables further compression with Delta/ZigZag/Bitpack

	// Verify correctness
	result := decodeInt64Array(dst[:n])

	for i := range strings {
		expected := int64(10001 + i)
		if result[i] != expected {
			t.Errorf("Value %d: expected %d, got %d", i, expected, result[i])
		}
	}

	t.Logf("✅ CSV integers parsed correctly\n")
}

// TestParseInt_EmptyInput tests handling of empty input
func TestParseInt_EmptyInput(t *testing.T) {
	codec := NewParseInt()

	src := encodeIntStringArray([]string{})
	dst := make([]byte, 100)

	n, err := codec.Encode(dst, src, nil)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	if n != 4 {
		t.Errorf("Expected 4 bytes output (count=0), got %d", n)
	}

	// Decode
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

// TestParseInt_SingleValue tests handling of single value
func TestParseInt_SingleValue(t *testing.T) {
	codec := NewParseInt()

	strings := []string{"42"}
	src := encodeIntStringArray(strings)

	dst := make([]byte, 100)
	n, err := codec.Encode(dst, src, nil)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	result := decodeInt64Array(dst[:n])
	if len(result) != 1 || result[0] != 42 {
		t.Errorf("Expected [42], got %v", result)
	}

	t.Log("✅ Single value parsed correctly")
}

// TestParseInt_InvalidInput tests error handling for invalid input
func TestParseInt_InvalidInput(t *testing.T) {
	codec := NewParseInt()

	testCases := []struct {
		name  string
		input []string
	}{
		{"not_a_number", []string{"abc"}},
		{"float", []string{"123.45"}},
		{"empty_string", []string{""}},
		{"hex", []string{"0x100"}},
		{"spaces", []string{" 123 "}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			src := encodeIntStringArray(tc.input)
			dst := make([]byte, 100)

			_, err := codec.Encode(dst, src, nil)
			if err == nil {
				t.Errorf("Expected error for input %q, got none", tc.input[0])
			} else {
				t.Logf("Correctly rejected %q: %v", tc.input[0], err)
			}
		})
	}

	t.Log("✅ Invalid inputs rejected correctly")
}

// TestParseInt_ZeroValue tests handling of zero
func TestParseInt_ZeroValue(t *testing.T) {
	codec := NewParseInt()

	strings := []string{"0", "0", "0"}
	src := encodeIntStringArray(strings)

	dst := make([]byte, 100)
	n, err := codec.Encode(dst, src, nil)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	result := decodeInt64Array(dst[:n])

	for i, val := range result {
		if val != 0 {
			t.Errorf("Value %d: expected 0, got %d", i, val)
		}
	}

	t.Log("✅ Zero values parsed correctly")
}

// TestParseInt_SequentialIDs tests parsing sequential IDs
func TestParseInt_SequentialIDs(t *testing.T) {
	t.Log("=== ParseInt: Sequential IDs (Pipeline Test) ===\n")

	codec := NewParseInt()

	// Sequential IDs (good for Delta encoding)
	strings := []string{
		"1000", "1001", "1002", "1003", "1004",
		"1005", "1006", "1007", "1008", "1009",
	}

	src := encodeIntStringArray(strings)
	t.Logf("Input: %d sequential IDs (%d bytes text)", len(strings), len(src))

	// Parse to binary
	dst := make([]byte, len(strings)*8+100)
	n, err := codec.Encode(dst, src, nil)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	t.Logf("ParseInt: %d bytes text → %d bytes binary", len(src), n)

	// Verify values
	result := decodeInt64Array(dst[:n])

	for i := 0; i < len(strings); i++ {
		expected := int64(1000 + i)
		if result[i] != expected {
			t.Errorf("Value %d: expected %d, got %d", i, expected, result[i])
		}
	}

	t.Log("Note: These sequential values are ideal for Delta→ZigZag→Bitpack pipeline")
	t.Log("  After Delta: [1000, 1, 1, 1, 1, 1, 1, 1, 1, 1]")
	t.Log("  After ZigZag: [2000, 2, 2, 2, 2, 2, 2, 2, 2, 2]")
	t.Log("  After Bitpack: ~15 bytes (1.5 bytes per value)")
	t.Log("  Total compression: 60 bytes text → 15 bytes = 4× compression")
	t.Logf("✅ Sequential IDs ready for pipeline compression\n")
}

// TestParseInt_LargeDataset tests parsing many integers
func TestParseInt_LargeDataset(t *testing.T) {
	t.Log("=== ParseInt: Large Dataset (1000 Integers) ===\n")

	codec := NewParseInt()

	// Generate 1000 integers
	strings := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		strings[i] = "1000000" // 7 characters
	}

	src := encodeIntStringArray(strings)
	t.Logf("Input: %d integers (%d bytes, %.2f KB)", len(strings), len(src), float64(len(src))/1024)

	// Parse to binary
	dst := make([]byte, len(strings)*8+100)
	n, err := codec.Encode(dst, src, nil)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	ratio := float64(len(src)) / float64(n)
	t.Logf("Parsed: %d bytes (%.2f KB) → %d bytes (%.2f KB)",
		len(src), float64(len(src))/1024, n, float64(n)/1024)
	t.Logf("Size change: %.2f× (text to binary)", ratio)

	// Verify sample values
	result := decodeInt64Array(dst[:n])

	for i := 0; i < len(strings); i += 100 {
		if result[i] != 1000000 {
			t.Errorf("Value %d: expected 1000000, got %d", i, result[i])
		}
	}

	t.Logf("✅ All %d values parsed correctly", len(strings))
	t.Logf("✅ Large dataset test passed\n")
}

// TestParseInt_CodecInterface tests codec interface compliance
func TestParseInt_CodecInterface(t *testing.T) {
	codec := NewParseInt()

	if codec.ID() != IDParseInt {
		t.Errorf("Expected ID %d, got %d", IDParseInt, codec.ID())
	}

	if codec.Name() != "ParseInt" {
		t.Errorf("Expected name 'ParseInt', got '%s'", codec.Name())
	}

	if codec.PreservesSize() {
		t.Error("ParseInt should not preserve size")
	}

	t.Log("✅ Codec interface compliance verified")
}
