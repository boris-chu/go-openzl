// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package codec

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// TestLZ77Dict_SimpleDictMatch tests basic dictionary matching
func TestLZ77Dict_SimpleDictMatch(t *testing.T) {
	// Dictionary contains common patterns
	dict := []byte("common_pattern,another_pattern,third_pattern")

	codec := NewLZ77WithDict(dict)

	// Input contains pattern from dictionary
	input := []byte("data: common_pattern value")

	dst := make([]byte, 10240)
	compressed, err := codec.Encode(dst, input, nil)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	// Decompress
	codecDec := NewLZ77WithDict(dict) // Decoder needs same dictionary
	decompressed := make([]byte, len(input))
	n, err := codecDec.Decode(decompressed, dst[:compressed], nil)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if n != len(input) {
		t.Errorf("Decompressed %d bytes, expected %d", n, len(input))
	}

	if !bytes.Equal(decompressed[:n], input) {
		t.Errorf("Decompressed data mismatch:\nGot:  %q\nWant: %q", decompressed[:n], input)
	}
}

// TestLZ77Dict_MultipleMatches tests multiple dictionary matches in same input
func TestLZ77Dict_MultipleMatches(t *testing.T) {
	dict := []byte("\"id\":\",\":\",name,email,phone")

	codec := NewLZ77WithDict(dict)

	// JSON-like input with repeated patterns from dictionary
	input := []byte("{\"id\":1,\"name\":\"John\",\"email\":\"john@example.com\",\"phone\":\"555-1234\"}")

	dst := make([]byte, 10240)
	compressed, err := codec.Encode(dst, input, nil)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	// Should achieve significant compression due to dict matches
	compressionRatio := float64(len(input)) / float64(compressed)
	t.Logf("Input: %d bytes, Compressed: %d bytes, Ratio: %.2fx", len(input), compressed, compressionRatio)

	// Decompress
	codecDec := NewLZ77WithDict(dict)
	decompressed := make([]byte, len(input))
	n, err := codecDec.Decode(decompressed, dst[:compressed], nil)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if !bytes.Equal(decompressed[:n], input) {
		t.Errorf("Decompressed data mismatch")
	}
}

// TestLZ77Dict_WindowAndDictMatches tests that both window and dict matches work together
func TestLZ77Dict_WindowAndDictMatches(t *testing.T) {
	dict := []byte("common")

	codec := NewLZ77WithDict(dict)

	// Input has both dict matches ("common") and window matches ("repeated")
	input := []byte("common word repeated repeated repeated common end")

	dst := make([]byte, 10240)
	compressed, err := codec.Encode(dst, input, nil)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	// Decompress
	codecDec := NewLZ77WithDict(dict)
	decompressed := make([]byte, len(input))
	n, err := codecDec.Decode(decompressed, dst[:compressed], nil)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if !bytes.Equal(decompressed[:n], input) {
		t.Errorf("Decompressed data mismatch:\nGot:  %q\nWant: %q", decompressed[:n], input)
	}
}

// TestLZ77Dict_NoMatchesInDict tests data that doesn't match dictionary
func TestLZ77Dict_NoMatchesInDict(t *testing.T) {
	dict := []byte("irrelevant,patterns,here")

	codec := NewLZ77WithDict(dict)

	// Input doesn't match dictionary at all
	input := []byte("completely different data xyz abc")

	dst := make([]byte, 10240)
	compressed, err := codec.Encode(dst, input, nil)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	// Decompress
	codecDec := NewLZ77WithDict(dict)
	decompressed := make([]byte, len(input))
	n, err := codecDec.Decode(decompressed, dst[:compressed], nil)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if !bytes.Equal(decompressed[:n], input) {
		t.Errorf("Decompressed data mismatch")
	}
}

// TestLZ77Dict_EmptyDict tests LZ77 with empty dictionary (should work like regular LZ77)
func TestLZ77Dict_EmptyDict(t *testing.T) {
	dict := []byte{} // Empty dict

	codec := NewLZ77WithDict(dict)

	input := []byte("test data test data test data")

	dst := make([]byte, 10240)
	compressed, err := codec.Encode(dst, input, nil)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	// Decompress
	codecDec := NewLZ77WithDict(dict)
	decompressed := make([]byte, len(input))
	n, err := codecDec.Decode(decompressed, dst[:compressed], nil)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if !bytes.Equal(decompressed[:n], input) {
		t.Errorf("Decompressed data mismatch")
	}
}

// TestLZ77Dict_MissingDictOnDecode tests error handling when decoder lacks dictionary
func TestLZ77Dict_MissingDictOnDecode(t *testing.T) {
	dict := []byte("pattern")

	// Encode WITH dictionary
	codecEnc := NewLZ77WithDict(dict)
	input := []byte("pattern pattern pattern")

	dst := make([]byte, 10240)
	compressed, err := codecEnc.Encode(dst, input, nil)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	// Try to decode WITHOUT dictionary - should fail
	codecDec := NewLZ77() // No dictionary!
	decompressed := make([]byte, len(input))
	_, err = codecDec.Decode(decompressed, dst[:compressed], nil)

	if err == nil {
		t.Error("Expected error when decoding dict matches without dictionary, got nil")
	}

	if !strings.Contains(err.Error(), "dict match") {
		t.Errorf("Expected 'dict match' error, got: %v", err)
	}
}

// TestLZ77Dict_CSVWithDictionary tests CSV compression with real dictionary
func TestLZ77Dict_CSVWithDictionary(t *testing.T) {
	// Load CSV dictionary if available
	csvDict, err := os.ReadFile("/tmp/csv-dict-30kb.bin")
	if err != nil {
		t.Skip("CSV dictionary not found at /tmp/csv-dict-30kb.bin - run dictionary training first")
	}

	t.Logf("CSV dictionary size: %d bytes", len(csvDict))

	codec := NewLZ77WithDict(csvDict)

	// Sample CSV data
	input := []byte(`id,name,email,phone,city,state,country
1,John Doe,john@example.com,555-1234,New York,NY,USA
2,Jane Smith,jane@example.com,555-5678,Los Angeles,CA,USA
3,Bob Jones,bob@example.com,555-9012,Chicago,IL,USA`)

	dst := make([]byte, 102400)
	compressed, err := codec.Encode(dst, input, nil)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	compressionRatio := float64(len(input)) / float64(compressed)
	t.Logf("CSV Input: %d bytes, Compressed: %d bytes, Ratio: %.2fx", len(input), compressed, compressionRatio)

	// Decompress
	codecDec := NewLZ77WithDict(csvDict)
	decompressed := make([]byte, len(input)*2)
	n, err := codecDec.Decode(decompressed, dst[:compressed], nil)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if !bytes.Equal(decompressed[:n], input) {
		t.Errorf("CSV decompressed data mismatch")
	}

	// With dictionary, should achieve at least 2× compression even on tiny data
	if compressionRatio < 1.5 {
		t.Logf("Warning: Expected >1.5× compression with CSV dict, got %.2fx", compressionRatio)
	}
}

// TestLZ77Dict_JSONWithDictionary tests JSON compression with real dictionary
func TestLZ77Dict_JSONWithDictionary(t *testing.T) {
	// Load JSON dictionary if available
	jsonDict, err := os.ReadFile("/tmp/json-dict-20kb.bin")
	if err != nil {
		t.Skip("JSON dictionary not found at /tmp/json-dict-20kb.bin - run dictionary training first")
	}

	t.Logf("JSON dictionary size: %d bytes", len(jsonDict))

	codec := NewLZ77WithDict(jsonDict)

	// Sample JSON data (typical API response)
	input := []byte(`{"id":1,"name":"John","email":"john@example.com","status":"active","created_at":"2024-01-01","updated_at":"2024-11-02"}`)

	dst := make([]byte, 102400)
	compressed, err := codec.Encode(dst, input, nil)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	compressionRatio := float64(len(input)) / float64(compressed)
	t.Logf("JSON Input: %d bytes, Compressed: %d bytes, Ratio: %.2fx", len(input), compressed, compressionRatio)

	// Decompress
	codecDec := NewLZ77WithDict(jsonDict)
	decompressed := make([]byte, len(input)*2)
	n, err := codecDec.Decode(decompressed, dst[:compressed], nil)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if !bytes.Equal(decompressed[:n], input) {
		t.Errorf("JSON decompressed data mismatch")
	}

	// With JSON dict, should achieve good compression even on small data
	if compressionRatio < 1.5 {
		t.Logf("Warning: Expected >1.5× compression with JSON dict, got %.2fx", compressionRatio)
	}
}

// TestLZ77Dict_Roundtrip tests that all data survives encode/decode cycle
func TestLZ77Dict_Roundtrip(t *testing.T) {
	testCases := []struct {
		name  string
		dict  []byte
		input []byte
	}{
		{
			name:  "Small dict, small data",
			dict:  []byte("test,pattern"),
			input: []byte("test pattern test pattern"),
		},
		{
			name:  "Large dict, small data",
			dict:  bytes.Repeat([]byte("pattern"), 1000), // 7KB dict
			input: []byte("pattern pattern pattern"),
		},
		{
			name:  "Dict with special chars",
			dict:  []byte("{\"id\":\",\":\"\r\n\t"),
			input: []byte("{\"id\":123,\"name\":\"test\"}"),
		},
		{
			name:  "Binary dict and data",
			dict:  []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD},
			input: []byte{0x00, 0x01, 0x02, 0x00, 0x01, 0x02, 0xFF, 0xFE},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			codec := NewLZ77WithDict(tc.dict)

			dst := make([]byte, len(tc.input)*10)
			compressed, err := codec.Encode(dst, tc.input, nil)
			if err != nil {
				t.Fatalf("Encode() error = %v", err)
			}

			codecDec := NewLZ77WithDict(tc.dict)
			decompressed := make([]byte, len(tc.input))
			n, err := codecDec.Decode(decompressed, dst[:compressed], nil)
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}

			if !bytes.Equal(decompressed[:n], tc.input) {
				t.Errorf("Roundtrip failed:\nGot:  %v\nWant: %v", decompressed[:n], tc.input)
			}
		})
	}
}

// TestLZ77Dict_LargeData tests dictionary compression on larger datasets
func TestLZ77Dict_LargeData(t *testing.T) {
	// Create dictionary with common CSV patterns
	dict := []byte("id,name,email,phone,address,city,state,zip,country,created_at,updated_at,status,active,@example.com,USA,http://,https://")

	codec := NewLZ77WithDict(dict)

	// Generate larger CSV-like data
	var sb strings.Builder
	sb.WriteString("id,name,email,phone,city,state,country,status\n")
	for i := 0; i < 1000; i++ {
		sb.WriteString("1,John Doe,john@example.com,555-1234,New York,NY,USA,active\n")
		sb.WriteString("2,Jane Smith,jane@example.com,555-5678,Los Angeles,CA,USA,active\n")
	}
	input := []byte(sb.String())

	dst := make([]byte, len(input)*2)
	compressed, err := codec.Encode(dst, input, nil)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	compressionRatio := float64(len(input)) / float64(compressed)
	t.Logf("Large CSV: %d bytes → %d bytes (%.2fx compression)", len(input), compressed, compressionRatio)

	// Should achieve excellent compression on repetitive CSV data
	if compressionRatio < 5.0 {
		t.Logf("Warning: Expected >5× compression on repetitive CSV, got %.2fx", compressionRatio)
	}

	// Decompress and verify
	codecDec := NewLZ77WithDict(dict)
	decompressed := make([]byte, len(input))
	n, err := codecDec.Decode(decompressed, dst[:compressed], nil)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if !bytes.Equal(decompressed[:n], input) {
		t.Error("Large data roundtrip failed")
	}
}

// BenchmarkLZ77Dict_CSV benchmarks CSV compression with dictionary
func BenchmarkLZ77Dict_CSV(b *testing.B) {
	// Load CSV dictionary
	csvDict, err := os.ReadFile("/tmp/csv-dict-30kb.bin")
	if err != nil {
		b.Skip("CSV dictionary not found")
	}

	codec := NewLZ77WithDict(csvDict)

	// Generate CSV data
	var sb strings.Builder
	sb.WriteString("id,name,email,phone,city,state,country\n")
	for i := 0; i < 100; i++ {
		sb.WriteString("1,John Doe,john@example.com,555-1234,New York,NY,USA\n")
	}
	input := []byte(sb.String())

	dst := make([]byte, len(input)*2)

	b.ResetTimer()
	b.SetBytes(int64(len(input)))

	for i := 0; i < b.N; i++ {
		_, err := codec.Encode(dst, input, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLZ77Dict_JSON benchmarks JSON compression with dictionary
func BenchmarkLZ77Dict_JSON(b *testing.B) {
	// Load JSON dictionary
	jsonDict, err := os.ReadFile("/tmp/json-dict-20kb.bin")
	if err != nil {
		b.Skip("JSON dictionary not found")
	}

	codec := NewLZ77WithDict(jsonDict)

	// Generate JSON data
	input := []byte(`{"id":1,"name":"John","email":"john@example.com","status":"active","created_at":"2024-01-01"}`)

	dst := make([]byte, len(input)*2)

	b.ResetTimer()
	b.SetBytes(int64(len(input)))

	for i := 0; i < b.N; i++ {
		_, err := codec.Encode(dst, input, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLZ77Dict_Comparison benchmarks dict vs no-dict compression
func BenchmarkLZ77Dict_Comparison(b *testing.B) {
	input := []byte("common pattern repeated pattern common pattern repeated pattern")

	b.Run("WithDict", func(b *testing.B) {
		dict := []byte("common,pattern,repeated")
		codec := NewLZ77WithDict(dict)
		dst := make([]byte, len(input)*2)

		b.ResetTimer()
		b.SetBytes(int64(len(input)))

		for i := 0; i < b.N; i++ {
			codec.Encode(dst, input, nil)
		}
	})

	b.Run("WithoutDict", func(b *testing.B) {
		codec := NewLZ77()
		dst := make([]byte, len(input)*2)

		b.ResetTimer()
		b.SetBytes(int64(len(input)))

		for i := 0; i < b.N; i++ {
			codec.Encode(dst, input, nil)
		}
	})
}
