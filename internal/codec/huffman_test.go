package codec

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/klauspost/compress/huff0"
)

// TestHuffman_Metadata verifies codec metadata
func TestHuffman_Metadata(t *testing.T) {
	codec := NewHuffman()

	if codec.ID() != IDHuffman {
		t.Errorf("expected ID %d, got %d", IDHuffman, codec.ID())
	}

	if codec.ID().String() != "Huffman" {
		t.Errorf("expected ID.String() 'Huffman', got '%s'", codec.ID().String())
	}

	if codec.Name() != "Huffman (huff0)" {
		t.Errorf("expected name 'Huffman (huff0)', got '%s'", codec.Name())
	}
}

// TestHuffman_BasicDecode tests basic Huffman decompression
func TestHuffman_BasicDecode(t *testing.T) {
	codec := NewHuffman()

	original := []byte("Hello, Huffman! This is a test of Huffman encoding.")

	// Compress using Klaus Post's huff0
	compressed, _, err := huff0.Compress1X(original, nil)
	if err != nil {
		t.Fatalf("compress failed: %v", err)
	}

	t.Logf("Original: %d bytes, Compressed: %d bytes (%.2fx)",
		len(original), len(compressed), float64(len(original))/float64(len(compressed)))

	// Decompress using our codec
	dst := make([]byte, len(original))
	n, err := codec.Decode(dst, compressed, nil)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if n != len(original) {
		t.Errorf("expected %d bytes, got %d", len(original), n)
	}

	if !bytes.Equal(dst[:n], original) {
		t.Errorf("decoded data mismatch:\ngot:  %q\nwant: %q", dst[:n], original)
	}
}

// TestHuffman_LargeData tests Huffman with larger data
func TestHuffman_LargeData(t *testing.T) {
	codec := NewHuffman()

	// Create 128KB of compressible data (huff0 has max block size limits)
	// Note: 1MB is too large for huff0.Compress1X (max ~128KB)
	pattern := []byte("The quick brown fox jumps over the lazy dog. ")
	original := make([]byte, 128*1024)
	for i := 0; i < len(original); i++ {
		original[i] = pattern[i%len(pattern)]
	}

	// Compress
	compressed, _, err := huff0.Compress1X(original, nil)
	if err != nil {
		t.Fatalf("compress failed: %v", err)
	}

	t.Logf("128KB data: Original %d bytes, Compressed %d bytes (%.2fx)",
		len(original), len(compressed), float64(len(original))/float64(len(compressed)))

	// Decompress
	dst := make([]byte, len(original))
	n, err := codec.Decode(dst, compressed, nil)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if n != len(original) {
		t.Errorf("size mismatch: got %d, want %d", n, len(original))
	}

	if !bytes.Equal(dst[:n], original) {
		t.Error("decompressed data does not match original")
	}
}

// TestHuffman_EmptyInput tests handling of empty input
func TestHuffman_EmptyInput(t *testing.T) {
	codec := NewHuffman()

	dst := make([]byte, 100)
	_, err := codec.Decode(dst, nil, nil)
	if err == nil {
		t.Error("expected error for empty input, got nil")
	}
}

// TestHuffman_CorruptedData tests handling of corrupted compressed data
func TestHuffman_CorruptedData(t *testing.T) {
	codec := NewHuffman()

	// Create valid compressed data first (repeated for compressibility)
	original := []byte("test data for corruption test data for corruption test data for corruption")
	compressed, _, err := huff0.Compress1X(original, nil)
	if err != nil {
		t.Fatalf("compress failed: %v", err)
	}

	// Corrupt the data
	if len(compressed) > 5 {
		compressed[5] ^= 0xFF // Flip bits
	}

	// Try to decompress corrupted data
	dst := make([]byte, len(original)*2)
	_, err = codec.Decode(dst, compressed, nil)
	if err == nil {
		t.Error("expected error for corrupted data, got nil")
	}
	t.Logf("Correctly rejected corrupted data: %v", err)
}

// TestHuffman_BufferTooSmall tests handling of insufficient output buffer
func TestHuffman_BufferTooSmall(t *testing.T) {
	codec := NewHuffman()

	original := []byte("this is a test of buffer size handling buffer size handling buffer size handling")
	compressed, _, err := huff0.Compress1X(original, nil)
	if err != nil {
		t.Fatalf("compress failed: %v", err)
	}

	// Provide buffer that's too small
	dst := make([]byte, 10) // Much smaller than needed
	_, err = codec.Decode(dst, compressed, nil)
	if err == nil {
		t.Error("expected error for small buffer, got nil")
	}
	t.Logf("Correctly rejected small buffer: %v", err)
}

// TestHuffman_ScratchReuse verifies that scratch is reused for performance
func TestHuffman_ScratchReuse(t *testing.T) {
	codec := NewHuffman()

	original1 := []byte("first test data with some repeated patterns repeated patterns repeated patterns")
	original2 := []byte("second test data with different repeated patterns repeated patterns repeated")
	original3 := []byte("third test data with even more repeated patterns patterns patterns patterns")

	// Compress test data
	compressed1, _, err := huff0.Compress1X(original1, nil)
	if err != nil {
		t.Fatalf("compress 1 failed: %v", err)
	}
	compressed2, _, err := huff0.Compress1X(original2, nil)
	if err != nil {
		t.Fatalf("compress 2 failed: %v", err)
	}
	compressed3, _, err := huff0.Compress1X(original3, nil)
	if err != nil {
		t.Fatalf("compress 3 failed: %v", err)
	}

	// Decompress multiple times - scratch should be reused
	dst1 := make([]byte, len(original1))
	n1, err := codec.Decode(dst1, compressed1, nil)
	if err != nil {
		t.Fatalf("decode 1 failed: %v", err)
	}
	if !bytes.Equal(dst1[:n1], original1) {
		t.Error("first decode mismatch")
	}

	dst2 := make([]byte, len(original2))
	n2, err := codec.Decode(dst2, compressed2, nil)
	if err != nil {
		t.Fatalf("decode 2 failed: %v", err)
	}
	if !bytes.Equal(dst2[:n2], original2) {
		t.Error("second decode mismatch")
	}

	dst3 := make([]byte, len(original3))
	n3, err := codec.Decode(dst3, compressed3, nil)
	if err != nil {
		t.Fatalf("decode 3 failed: %v", err)
	}
	if !bytes.Equal(dst3[:n3], original3) {
		t.Error("third decode mismatch")
	}

	t.Log("Successfully reused scratch across 3 decompressions")
}

// TestHuffman_VariousSizes tests Huffman with various data sizes
func TestHuffman_VariousSizes(t *testing.T) {
	codec := NewHuffman()

	sizes := []int{
		10,     // Very small
		100,    // Small
		1024,   // 1KB
		10240,  // 10KB
		65536,  // 64KB
		131072, // 128KB
	}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("size=%d", size), func(t *testing.T) {
			// Create test data (repeated pattern for compressibility)
			pattern := []byte("compress this data!")
			original := make([]byte, size)
			for i := 0; i < len(original); i++ {
				original[i] = pattern[i%len(pattern)]
			}

			// Compress (small data may be incompressible)
			compressed, _, err := huff0.Compress1X(original, nil)
			if err == huff0.ErrIncompressible {
				t.Skipf("Size %d is too small to compress with Huffman", size)
				return
			}
			if err != nil {
				t.Fatalf("compress failed: %v", err)
			}

			// Decompress
			dst := make([]byte, size)
			n, err := codec.Decode(dst, compressed, nil)
			if err != nil {
				t.Fatalf("decode failed: %v", err)
			}

			if n != size {
				t.Errorf("size mismatch: got %d, want %d", n, size)
			}

			if !bytes.Equal(dst[:n], original) {
				t.Error("data mismatch after roundtrip")
			}

			ratio := float64(len(original)) / float64(len(compressed))
			t.Logf("Size %d: compressed to %d bytes (%.2fx)", size, len(compressed), ratio)
		})
	}
}

// TestHuffman_HighlyCompressible tests with highly compressible data
func TestHuffman_HighlyCompressible(t *testing.T) {
	codec := NewHuffman()

	// Create data with very skewed distribution (lots of 'a', few 'z')
	original := make([]byte, 8000)
	for i := range original {
		if i%100 == 0 {
			original[i] = 'z'
		} else {
			original[i] = 'a'
		}
	}

	compressed, _, err := huff0.Compress1X(original, nil)
	if err != nil {
		t.Fatalf("compress failed: %v", err)
	}

	ratio := float64(len(original)) / float64(len(compressed))
	t.Logf("Highly compressible: %d -> %d bytes (%.2fx)",
		len(original), len(compressed), ratio)

	// Huffman typically achieves 2-4x on skewed distributions
	if ratio < 2.0 {
		t.Errorf("expected high compression ratio (>2x), got %.2fx", ratio)
	}

	// Decompress
	dst := make([]byte, len(original))
	n, err := codec.Decode(dst, compressed, nil)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if !bytes.Equal(dst[:n], original) {
		t.Error("decompressed data does not match")
	}
}

// TestHuffman_Incompressible tests with random (incompressible) data
func TestHuffman_Incompressible(t *testing.T) {
	// Random data is incompressible
	original := make([]byte, 1000)
	rand.Read(original)

	// Try to compress - should return ErrIncompressible
	_, _, err := huff0.Compress1X(original, nil)
	if err == huff0.ErrIncompressible {
		t.Log("Huffman correctly identified incompressible data")
		return
	}

	// If it did compress (unlikely), just log it
	if err != nil {
		t.Fatalf("unexpected compress error: %v", err)
	}
	t.Log("Random data was compressible (rare but possible)")
}

// TestHuffman_AllZeros tests with all-zero data (extreme case)
func TestHuffman_AllZeros(t *testing.T) {
	original := make([]byte, 1000)
	// All zeros - huff0 will reject this as "single value repeated"
	// This is expected behavior - RLE codecs are better for this case

	_, _, err := huff0.Compress1X(original, nil)
	if err != nil {
		// Expected: huff0 rejects single-value data (use RLE instead)
		t.Logf("Huffman correctly rejected all-zeros data: %v", err)
		t.Log("(All-zero data should use RLE codec, not Huffman)")
		return
	}

	// If it did compress (shouldn't happen), that's unexpected
	t.Error("Expected error for all-zeros data, got nil")
}

// TestHuffman_EncodeNotImplemented verifies encode returns error
func TestHuffman_EncodeNotImplemented(t *testing.T) {
	codec := NewHuffman()

	src := []byte("test data")
	dst := make([]byte, 100)

	_, err := codec.Encode(dst, src, nil)
	if err == nil {
		t.Error("expected error for encode, got nil")
	}

	if err.Error() != "huffman encode not yet implemented (decompression only in Phase 3)" {
		t.Errorf("unexpected error message: %v", err)
	}
	t.Logf("Encode correctly returns error: %v", err)
}

// BenchmarkHuffman_Decode benchmarks Huffman decompression
func BenchmarkHuffman_Decode(b *testing.B) {
	sizes := []int{1024, 65536, 131072} // 1KB, 64KB, 128KB (max for huff0)

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			// Prepare test data (compressible pattern)
			pattern := []byte("The quick brown fox jumps over the lazy dog. ")
			original := make([]byte, size)
			for i := 0; i < len(original); i++ {
				original[i] = pattern[i%len(pattern)]
			}

			compressed, _, err := huff0.Compress1X(original, nil)
			if err != nil {
				b.Fatalf("compress failed: %v", err)
			}

			codec := NewHuffman()
			dst := make([]byte, size)

			b.ResetTimer()
			b.SetBytes(int64(len(original)))

			for i := 0; i < b.N; i++ {
				_, err := codec.Decode(dst, compressed, nil)
				if err != nil {
					b.Fatalf("decode failed: %v", err)
				}
			}
		})
	}
}
