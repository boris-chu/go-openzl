package codec

import (
	"bytes"
	"fmt"
	"testing"
)

// TestIdentity_BasicDecode tests basic decode functionality
func TestIdentity_BasicDecode(t *testing.T) {
	codec := NewIdentity()

	tests := []struct {
		name  string
		input []byte
	}{
		{"empty", []byte{}},
		{"single byte", []byte{0x42}},
		{"small data", []byte{0x01, 0x02, 0x03, 0x04}},
		{"text", []byte("Hello, OpenZL!")},
		{"binary", []byte{0x00, 0xFF, 0x7F, 0x80, 0x01}},
		{"large", bytes.Repeat([]byte("test"), 1000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst := make([]byte, len(tt.input))
			n, err := codec.Decode(dst, tt.input, nil)

			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}

			if n != len(tt.input) {
				t.Errorf("Decode() wrote %d bytes, want %d", n, len(tt.input))
			}

			if !bytes.Equal(dst[:n], tt.input) {
				t.Errorf("Decode() output mismatch:\ngot  %v\nwant %v",
					dst[:n], tt.input)
			}
		})
	}
}

// TestIdentity_Encode tests basic encode functionality
func TestIdentity_Encode(t *testing.T) {
	codec := NewIdentity()

	input := []byte("compress me")
	dst := make([]byte, len(input))

	n, err := codec.Encode(dst, input, nil)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	if n != len(input) {
		t.Errorf("Encode() wrote %d bytes, want %d", n, len(input))
	}

	if !bytes.Equal(dst[:n], input) {
		t.Errorf("Encode() output mismatch")
	}
}

// TestIdentity_Roundtrip tests encode then decode
func TestIdentity_Roundtrip(t *testing.T) {
	codec := NewIdentity()

	original := []byte("roundtrip test data")

	// Encode
	encoded := make([]byte, len(original))
	n1, err := codec.Encode(encoded, original, nil)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	// Decode
	decoded := make([]byte, len(original))
	n2, err := codec.Decode(decoded, encoded[:n1], nil)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	// Verify
	if !bytes.Equal(decoded[:n2], original) {
		t.Errorf("Roundtrip mismatch:\ngot  %v\nwant %v",
			decoded[:n2], original)
	}
}

// TestIdentity_BufferTooSmall tests error handling
func TestIdentity_BufferTooSmall(t *testing.T) {
	codec := NewIdentity()

	src := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	dst := make([]byte, 3) // Too small!

	_, err := codec.Decode(dst, src, nil)
	if err != ErrBufferTooSmall {
		t.Errorf("Decode() error = %v, want ErrBufferTooSmall", err)
	}

	_, err = codec.Encode(dst, src, nil)
	if err != ErrBufferTooSmall {
		t.Errorf("Encode() error = %v, want ErrBufferTooSmall", err)
	}
}

// TestIdentity_ExactSize tests exact buffer size
func TestIdentity_ExactSize(t *testing.T) {
	codec := NewIdentity()

	src := []byte{0x01, 0x02, 0x03}
	dst := make([]byte, 3) // Exact size

	n, err := codec.Decode(dst, src, nil)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if n != 3 {
		t.Errorf("Decode() wrote %d bytes, want 3", n)
	}
}

// TestIdentity_OversizedBuffer tests larger than needed buffer
func TestIdentity_OversizedBuffer(t *testing.T) {
	codec := NewIdentity()

	src := []byte{0x01, 0x02, 0x03}
	dst := make([]byte, 100) // Much larger than needed

	n, err := codec.Decode(dst, src, nil)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if n != 3 {
		t.Errorf("Decode() wrote %d bytes, want 3", n)
	}

	if !bytes.Equal(dst[:n], src) {
		t.Errorf("Decode() output mismatch")
	}
}

// TestIdentity_Parameters tests that parameters are ignored
func TestIdentity_Parameters(t *testing.T) {
	codec := NewIdentity()

	src := []byte{0x01, 0x02, 0x03}
	dst := make([]byte, len(src))

	// Various parameter values should all be ignored
	params := [][]byte{
		nil,
		{},
		{0x00},
		{0x01, 0x02, 0x03, 0x04},
	}

	for _, p := range params {
		n, err := codec.Decode(dst, src, p)
		if err != nil {
			t.Errorf("Decode() with params %v error = %v", p, err)
		}
		if n != len(src) {
			t.Errorf("Decode() with params %v wrote %d bytes, want %d",
				p, n, len(src))
		}
		if !bytes.Equal(dst[:n], src) {
			t.Errorf("Decode() with params %v output mismatch", p)
		}
	}
}

// TestIdentity_Metadata tests codec metadata
func TestIdentity_Metadata(t *testing.T) {
	codec := NewIdentity()

	if codec.ID() != IDIdentity {
		t.Errorf("ID() = %d, want %d", codec.ID(), IDIdentity)
	}

	if codec.Name() != "Identity" {
		t.Errorf("Name() = %q, want %q", codec.Name(), "Identity")
	}
}

// BenchmarkIdentity_Decode benchmarks decode performance
func BenchmarkIdentity_Decode(b *testing.B) {
	codec := NewIdentity()

	sizes := []int{16, 256, 4096, 65536}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			src := make([]byte, size)
			dst := make([]byte, size)

			b.SetBytes(int64(size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := codec.Decode(dst, src, nil)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkIdentity_Encode benchmarks encode performance
func BenchmarkIdentity_Encode(b *testing.B) {
	codec := NewIdentity()

	sizes := []int{16, 256, 4096, 65536}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			src := make([]byte, size)
			dst := make([]byte, size)

			b.SetBytes(int64(size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := codec.Encode(dst, src, nil)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
