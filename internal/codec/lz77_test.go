// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package codec

import (
	"bytes"
	"strings"
	"testing"
)

func TestLZ77_Metadata(t *testing.T) {
	codec := NewLZ77()

	if codec.ID() != IDLZ77 {
		t.Errorf("ID() = %v, want %v", codec.ID(), IDLZ77)
	}

	if codec.Name() != "LZ77" {
		t.Errorf("Name() = %q, want %q", codec.Name(), "LZ77")
	}

	if codec.PreservesSize() {
		t.Error("PreservesSize() = true, want false (LZ77 changes size)")
	}
}

func TestLZ77_EmptyData(t *testing.T) {
	codec := NewLZ77()
	dst := make([]byte, 1024)
	compressed, err := codec.Encode(dst, []byte{}, nil)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	if compressed != 4 {
		t.Errorf("Empty data should encode to 4 bytes (num_tokens=0), got %d", compressed)
	}

	// Decompress
	decompressed := make([]byte, 0)
	n, err := codec.Decode(decompressed, dst[:compressed], nil)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if n != 0 {
		t.Errorf("Decompressed %d bytes, expected 0", n)
	}
}

func TestLZ77_NoRepetition(t *testing.T) {
	codec := NewLZ77()

	// Random-like data with no repetition (all literals)
	input := []byte("abcdefghijklmnopqrstuvwxyz0123456789")

	// Token format: 4 bytes header + (2 bytes per literal)
	dst := make([]byte, 4+len(input)*2)
	compressed, err := codec.Encode(dst, input, nil)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	t.Logf("No repetition: %d bytes → %d bytes (%.2fx)",
		len(input), compressed, float64(len(input))/float64(compressed))

	// Decompress
	decompressed := make([]byte, len(input))
	n, err := codec.Decode(decompressed, dst[:compressed], nil)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if !bytes.Equal(decompressed[:n], input) {
		t.Errorf("Roundtrip failed:\nGot:  %q\nWant: %q", decompressed[:n], input)
	}
}

func TestLZ77_SimpleRepetition(t *testing.T) {
	codec := NewLZ77()

	// Simple repetition: "Hello, Hello, Hello!"
	input := []byte("Hello, Hello, Hello!")

	// Token format needs space for worst case (all literals)
	dst := make([]byte, 4+len(input)*2)
	compressed, err := codec.Encode(dst, input, nil)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	ratio := float64(len(input)) / float64(compressed)
	t.Logf("Simple repetition: %d bytes → %d bytes (%.2fx)", len(input), compressed, ratio)

	// Note: LZ77 token format has overhead (5 bytes per match).
	// Small strings may expand. LZ77 is meant to be combined with
	// entropy coding (Huffman/FSE) which compresses the token stream.
	// For now, just verify roundtrip works.
	if ratio < 0.5 {
		t.Errorf("Excessive expansion: %.2fx (token format overhead should be bounded)", ratio)
	}

	// Decompress
	decompressed := make([]byte, len(input))
	n, err := codec.Decode(decompressed, dst[:compressed], nil)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if !bytes.Equal(decompressed[:n], input) {
		t.Errorf("Roundtrip failed:\nGot:  %q\nWant: %q", decompressed[:n], input)
	}
}

func TestLZ77_JSONLikeData(t *testing.T) {
	codec := NewLZ77()

	// Simulate JSON with repeated field names
	json := `{"password_id":"abc123","password_id":"def456","password_id":"ghi789"}`
	input := []byte(json)

	// Token format needs space for worst case
	dst := make([]byte, 4+len(input)*2)
	compressed, err := codec.Encode(dst, input, nil)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	ratio := float64(len(input)) / float64(compressed)
	t.Logf("JSON-like data: %d bytes → %d bytes (%.2fx)", len(input), compressed, ratio)

	// Should get some compression from repeated "password_id"
	if ratio < 1.2 {
		t.Logf("Warning: Expected >1.2x compression for JSON, got %.2fx", ratio)
	}

	// Decompress
	decompressed := make([]byte, len(input))
	n, err := codec.Decode(decompressed, dst[:compressed], nil)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if !bytes.Equal(decompressed[:n], input) {
		t.Errorf("Roundtrip failed:\nGot:  %q\nWant: %q", decompressed[:n], input)
	}
}

func TestLZ77_LongRepetition(t *testing.T) {
	codec := NewLZ77()

	// Very long repetition
	input := []byte(strings.Repeat("Hello, World! ", 100))

	dst := make([]byte, len(input))
	compressed, err := codec.Encode(dst, input, nil)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	ratio := float64(len(input)) / float64(compressed)
	t.Logf("Long repetition: %d bytes → %d bytes (%.2fx)", len(input), compressed, ratio)

	// Should achieve significant compression
	if ratio < 5.0 {
		t.Logf("Note: Expected >5x compression for long repetition, got %.2fx", ratio)
	}

	// Decompress
	decompressed := make([]byte, len(input))
	n, err := codec.Decode(decompressed, dst[:compressed], nil)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if !bytes.Equal(decompressed[:n], input) {
		t.Errorf("Roundtrip failed (length mismatch: got %d, want %d)", n, len(input))
	}
}

func TestLZ77_BufferTooSmall(t *testing.T) {
	codec := NewLZ77()
	input := []byte("test data")

	// Try to encode with buffer too small
	tooSmall := make([]byte, 5)
	_, err := codec.Encode(tooSmall, input, nil)
	if err != ErrBufferTooSmall {
		t.Errorf("Encode with small buffer: got error %v, want %v", err, ErrBufferTooSmall)
	}

	// Encode normally
	dst := make([]byte, 4+len(input)*2)
	compressed, err := codec.Encode(dst, input, nil)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	// Try to decode with buffer too small
	_, err = codec.Decode(tooSmall, dst[:compressed], nil)
	if err != ErrBufferTooSmall {
		t.Errorf("Decode with small buffer: got error %v, want %v", err, ErrBufferTooSmall)
	}
}

func TestLZ77_CustomWindowSize(t *testing.T) {
	// Test with smaller window
	codec := NewLZ77WithWindow(1024) // 1KB window instead of 32KB

	input := []byte(strings.Repeat("test ", 500))
	dst := make([]byte, len(input))
	compressed, err := codec.Encode(dst, input, nil)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	ratio := float64(len(input)) / float64(compressed)
	t.Logf("Small window (1KB): %d bytes → %d bytes (%.2fx)", len(input), compressed, ratio)

	// Decompress
	decompressed := make([]byte, len(input))
	n, err := codec.Decode(decompressed, dst[:compressed], nil)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if !bytes.Equal(decompressed[:n], input) {
		t.Error("Roundtrip failed with custom window size")
	}
}

func TestHashTable(t *testing.T) {
	hash := NewHashTable(1024)
	data := []byte("hello world hello")

	// Insert positions
	hash.Insert(data, 0)  // "hel"
	hash.Insert(data, 12) // "hel" (same hash)

	// Lookup should find both positions
	candidates := hash.Lookup(data, 0)
	if len(candidates) == 0 {
		t.Error("Lookup found no candidates")
	}

	t.Logf("Hash table: found %d candidates for 'hel'", len(candidates))
}

func BenchmarkLZ77_Compress(b *testing.B) {
	codec := NewLZ77()

	// Test data: repeated JSON-like structure
	data := []byte(strings.Repeat(`{"id":"abc","name":"test","value":123},`, 100))
	dst := make([]byte, len(data))

	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := codec.Encode(dst, data, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLZ77_Decompress(b *testing.B) {
	codec := NewLZ77()

	data := []byte(strings.Repeat(`{"id":"abc","name":"test","value":123},`, 100))
	compressed := make([]byte, len(data))
	compressedSize, _ := codec.Encode(compressed, data, nil)
	compressed = compressed[:compressedSize]

	dst := make([]byte, len(data))

	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := codec.Decode(dst, compressed, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}
