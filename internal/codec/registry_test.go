package codec

import (
	"testing"
)

// TestRegistry_RegisterAndGet tests basic registry operations
func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	identity := NewIdentity()

	reg.Register(identity)

	codec, ok := reg.Get(IDIdentity)
	if !ok {
		t.Fatal("Identity codec not found in registry")
	}

	if codec.ID() != IDIdentity {
		t.Errorf("Got codec ID %d, want %d", codec.ID(), IDIdentity)
	}

	if codec.Name() != "Identity" {
		t.Errorf("Got codec name %q, want %q", codec.Name(), "Identity")
	}
}

// TestRegistry_Has tests the Has method
func TestRegistry_Has(t *testing.T) {
	reg := NewRegistry()

	if reg.Has(IDIdentity) {
		t.Error("Registry should not have Identity codec before registration")
	}

	reg.Register(NewIdentity())

	if !reg.Has(IDIdentity) {
		t.Error("Registry should have Identity codec after registration")
	}

	if reg.Has(IDDelta) {
		t.Error("Registry should not have Delta codec")
	}
}

// TestRegistry_MustGet tests MustGet success and panic
func TestRegistry_MustGet(t *testing.T) {
	reg := NewRegistry()
	reg.Register(NewIdentity())

	// Should succeed
	codec := reg.MustGet(IDIdentity)
	if codec.ID() != IDIdentity {
		t.Errorf("Got codec ID %d, want %d", codec.ID(), IDIdentity)
	}

	// Should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustGet should panic for missing codec")
		}
	}()
	reg.MustGet(IDDelta) // Not registered, should panic
}

// TestRegistry_IDs tests getting all registered IDs
func TestRegistry_IDs(t *testing.T) {
	reg := NewRegistry()

	// Empty registry
	ids := reg.IDs()
	if len(ids) != 0 {
		t.Errorf("Empty registry should have 0 IDs, got %d", len(ids))
	}

	// Register Identity
	reg.Register(NewIdentity())
	ids = reg.IDs()
	if len(ids) != 1 {
		t.Fatalf("Registry should have 1 ID, got %d", len(ids))
	}
	if ids[0] != IDIdentity {
		t.Errorf("Got ID %d, want %d", ids[0], IDIdentity)
	}
}

// TestRegistry_Replace tests replacing a codec
func TestRegistry_Replace(t *testing.T) {
	reg := NewRegistry()

	// Register first time
	reg.Register(NewIdentity())
	codec1, _ := reg.Get(IDIdentity)

	// Register again (should replace)
	reg.Register(NewIdentity())
	codec2, _ := reg.Get(IDIdentity)

	// Both should have same ID but be different instances
	if codec1.ID() != codec2.ID() {
		t.Error("Codec IDs should match")
	}
}

// TestDefaultRegistry tests the default registry
func TestDefaultRegistry(t *testing.T) {
	reg := DefaultRegistry()

	// Should have Identity codec
	if !reg.Has(IDIdentity) {
		t.Error("Default registry should have Identity codec")
	}

	codec, ok := reg.Get(IDIdentity)
	if !ok {
		t.Fatal("Identity codec not found in default registry")
	}

	if codec.ID() != IDIdentity {
		t.Errorf("Got codec ID %d, want %d", codec.ID(), IDIdentity)
	}

	// Verify it works
	input := []byte("test")
	output := make([]byte, len(input))
	n, err := codec.Decode(output, input, nil)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if n != len(input) {
		t.Errorf("Decoded %d bytes, want %d", n, len(input))
	}

	// Should have Delta codec
	if !reg.Has(IDDelta) {
		t.Error("Default registry should have Delta codec")
	}

	deltaCodec, ok := reg.Get(IDDelta)
	if !ok {
		t.Fatal("Delta codec not found in default registry")
	}

	if deltaCodec.ID() != IDDelta {
		t.Errorf("Got codec ID %d, want %d", deltaCodec.ID(), IDDelta)
	}
}

// TestCodecID_String tests codec ID string representation
func TestCodecID_String(t *testing.T) {
	tests := []struct {
		id   ID
		want string
	}{
		{IDIdentity, "Identity"},
		{IDConstant, "Constant"},
		{IDDelta, "Delta"},
		{IDZigZag, "ZigZag"},
		{IDBitpack, "Bitpack"},
		{IDTranspose, "Transpose"},
		{IDQuantize, "Quantize"},
		{IDFSE, "FSE"},
		{IDHuffman, "Huffman"},
		{IDZstd, "Zstd"},
		{ID(999), "Unknown(999)"},
	}

	for _, tt := range tests {
		got := tt.id.String()
		if got != tt.want {
			t.Errorf("ID(%d).String() = %q, want %q", tt.id, got, tt.want)
		}
	}
}
