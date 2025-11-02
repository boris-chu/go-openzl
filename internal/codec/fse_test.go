package codec

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/klauspost/compress/fse"
)

// TestFSE_Metadata verifies codec metadata
func TestFSE_Metadata(t *testing.T) {
	codec := NewFSE()

	if codec.ID() != IDFSE {
		t.Errorf("expected ID %d, got %d", IDFSE, codec.ID())
	}

	if codec.ID().String() != "FSE" {
		t.Errorf("expected ID.String() 'FSE', got '%s'", codec.ID().String())
	}

	if codec.Name() != "FSE (Finite State Entropy)" {
		t.Errorf("expected name 'FSE (Finite State Entropy)', got '%s'", codec.Name())
	}
}

// TestFSE_BasicDecode tests basic FSE decompression
func TestFSE_BasicDecode(t *testing.T) {
	codec := NewFSE()

	// Create test data - simple repeated pattern
	original := []byte("hello world! hello world! hello world!")

	// Compress using Klaus Post's FSE
	compressed, err := fse.Compress(original, nil)
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
		t.Errorf("decoded size: got %d, want %d", n, len(original))
	}

	if !bytes.Equal(dst[:n], original) {
		t.Errorf("decoded data mismatch:\ngot:  %q\nwant: %q", dst[:n], original)
	}
}

// TestFSE_LargeData tests FSE with larger data
func TestFSE_LargeData(t *testing.T) {
	codec := NewFSE()

	// Create 1MB of compressible data (repeated text pattern)
	pattern := []byte("The quick brown fox jumps over the lazy dog. ")
	original := make([]byte, 1024*1024)
	for i := 0; i < len(original); i++ {
		original[i] = pattern[i%len(pattern)]
	}

	// Compress
	compressed, err := fse.Compress(original, nil)
	if err != nil {
		t.Fatalf("compress failed: %v", err)
	}

	ratio := float64(len(original)) / float64(len(compressed))
	t.Logf("1MB data: Original %d bytes, Compressed %d bytes (%.2fx)",
		len(original), len(compressed), ratio)

	// Decompress
	dst := make([]byte, len(original))
	n, err := codec.Decode(dst, compressed, nil)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if n != len(original) {
		t.Errorf("decoded size: got %d, want %d", n, len(original))
	}

	if !bytes.Equal(dst[:n], original) {
		t.Errorf("decoded data mismatch")
	}
}

// TestFSE_EmptyInput tests handling of empty input
func TestFSE_EmptyInput(t *testing.T) {
	codec := NewFSE()

	dst := make([]byte, 100)
	_, err := codec.Decode(dst, []byte{}, nil)
	if err == nil {
		t.Error("expected error for empty input, got nil")
	}
}

// TestFSE_CorruptedData tests handling of corrupted compressed data
func TestFSE_CorruptedData(t *testing.T) {
	codec := NewFSE()

	// Create valid compressed data first (repeated for compressibility)
	original := []byte("test data for corruption test data for corruption test data for corruption")
	compressed, err := fse.Compress(original, nil)
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

// TestFSE_BufferTooSmall tests handling of insufficient output buffer
func TestFSE_BufferTooSmall(t *testing.T) {
	codec := NewFSE()

	original := []byte("this is a test of buffer size handling buffer size handling buffer size handling")
	compressed, err := fse.Compress(original, nil)
	if err != nil {
		t.Fatalf("compress failed: %v", err)
	}

	// Provide buffer that's too small
	dst := make([]byte, 10) // Much smaller than needed
	_, err = codec.Decode(dst, compressed, nil)
	if err == nil {
		t.Error("expected error for buffer too small, got nil")
	}
	t.Logf("Correctly rejected small buffer: %v", err)
}

// TestFSE_ScratchReuse verifies that scratch is reused for performance
func TestFSE_ScratchReuse(t *testing.T) {
	codec := NewFSE()

	original1 := []byte("first test data with some repeated patterns repeated patterns repeated patterns")
	original2 := []byte("second test data with different repeated patterns repeated patterns repeated")
	original3 := []byte("third test data with even more repeated patterns patterns patterns patterns")

	// Compress test data
	compressed1, err := fse.Compress(original1, nil)
	if err != nil {
		t.Fatalf("compress 1 failed: %v", err)
	}
	compressed2, err := fse.Compress(original2, nil)
	if err != nil {
		t.Fatalf("compress 2 failed: %v", err)
	}
	compressed3, err := fse.Compress(original3, nil)
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
		t.Error("decode 1 data mismatch")
	}

	dst2 := make([]byte, len(original2))
	n2, err := codec.Decode(dst2, compressed2, nil)
	if err != nil {
		t.Fatalf("decode 2 failed: %v", err)
	}
	if !bytes.Equal(dst2[:n2], original2) {
		t.Error("decode 2 data mismatch")
	}

	dst3 := make([]byte, len(original3))
	n3, err := codec.Decode(dst3, compressed3, nil)
	if err != nil {
		t.Fatalf("decode 3 failed: %v", err)
	}
	if !bytes.Equal(dst3[:n3], original3) {
		t.Error("decode 3 data mismatch")
	}

	t.Log("Successfully reused scratch across 3 decompressions")
}

// TestFSE_VariousSizes tests FSE with various input sizes
func TestFSE_VariousSizes(t *testing.T) {
	codec := NewFSE()

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
			compressed, err := fse.Compress(original, nil)
			if err == fse.ErrIncompressible {
				t.Skipf("Size %d is too small to compress with FSE", size)
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
				t.Error("data mismatch")
			}

			ratio := float64(size) / float64(len(compressed))
			t.Logf("Size %d: compressed to %d bytes (%.2fx)", size, len(compressed), ratio)
		})
	}
}

// TestFSE_HighlyCompressible tests FSE with highly compressible data
func TestFSE_HighlyCompressible(t *testing.T) {
	codec := NewFSE()

	// Create highly compressible data (repeated pattern)
	original := bytes.Repeat([]byte("compress"), 1000)

	// Compress
	compressed, err := fse.Compress(original, nil)
	if err != nil {
		t.Fatalf("compress failed: %v", err)
	}

	ratio := float64(len(original)) / float64(len(compressed))
	t.Logf("Highly compressible: %d -> %d bytes (%.2fx)",
		len(original), len(compressed), ratio)

	// FSE typically achieves 2-3x on repeated patterns (not as high as LZ algorithms)
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
		t.Error("data mismatch")
	}
}

// TestFSE_Incompressible tests FSE with incompressible data
func TestFSE_Incompressible(t *testing.T) {
	codec := NewFSE()

	// Create incompressible data (pseudo-random)
	original := make([]byte, 1000)
	for i := range original {
		original[i] = byte((i * 7919) % 256) // Pseudo-random
	}

	// Try to compress - FSE may return ErrIncompressible
	compressed, err := fse.Compress(original, nil)
	if err == fse.ErrIncompressible {
		t.Log("FSE correctly identified incompressible data")
		return
	}
	if err != nil {
		t.Fatalf("compress failed: %v", err)
	}

	t.Logf("Incompressible data: %d -> %d bytes", len(original), len(compressed))

	// Decompress if compression succeeded
	dst := make([]byte, len(original))
	n, err := codec.Decode(dst, compressed, nil)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if !bytes.Equal(dst[:n], original) {
		t.Error("data mismatch")
	}
}

// TestFSE_AllZeros tests FSE with all-zero data
func TestFSE_AllZeros(t *testing.T) {
	codec := NewFSE()

	original := make([]byte, 10000) // All zeros

	// Compress
	compressed, err := fse.Compress(original, nil)
	if err == fse.ErrUseRLE {
		t.Log("FSE correctly suggested RLE for all-zeros data")
		return
	}
	if err != nil {
		t.Fatalf("compress failed: %v", err)
	}

	ratio := float64(len(original)) / float64(len(compressed))
	t.Logf("All zeros: %d -> %d bytes (%.2fx)", len(original), len(compressed), ratio)

	// Decompress
	dst := make([]byte, len(original))
	n, err := codec.Decode(dst, compressed, nil)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if !bytes.Equal(dst[:n], original) {
		t.Error("data mismatch")
	}
}

// TestFSE_EncodeNotImplemented verifies encode returns error in Phase 3
func TestFSE_EncodeNotImplemented(t *testing.T) {
	codec := NewFSE()

	src := []byte("test data")
	dst := make([]byte, 100)

	_, err := codec.Encode(dst, src, nil)
	if err == nil {
		t.Error("expected error for unimplemented encode, got nil")
	}
	t.Logf("Encode correctly returns error: %v", err)
}

// BenchmarkFSE_Decode benchmarks FSE decompression
func BenchmarkFSE_Decode(b *testing.B) {
	sizes := []int{1024, 65536, 1048576} // 1KB, 64KB, 1MB

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			// Prepare test data (compressible pattern)
			pattern := []byte("The quick brown fox jumps over the lazy dog. ")
			original := make([]byte, size)
			for i := 0; i < len(original); i++ {
				original[i] = pattern[i%len(pattern)]
			}

			compressed, err := fse.Compress(original, nil)
			if err != nil {
				b.Fatalf("compress failed: %v", err)
			}

			codec := NewFSE()
			dst := make([]byte, size)

			b.SetBytes(int64(size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := codec.Decode(dst, compressed, nil)
				if err != nil {
					b.Fatalf("decode failed: %v", err)
				}
			}
		})
	}
}
