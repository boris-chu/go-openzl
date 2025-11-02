package codec

// Identity is a passthrough codec that copies input to output unchanged.
//
// This is the simplest codec in OpenZL. It performs no transformation,
// making it useful for:
// - Testing the codec framework
// - Skipping compression for incompressible data
// - Placeholder in compression graphs
//
// Performance: Identity should be extremely fast - just a memory copy.
type Identity struct{}

// NewIdentity creates a new Identity codec.
func NewIdentity() *Identity {
	return &Identity{}
}

// ID returns the codec identifier.
func (c *Identity) ID() ID {
	return IDIdentity
}

// Name returns the codec name.
func (c *Identity) Name() string {
	return "Identity"
}

// Decode copies src to dst unchanged.
//
// Parameters are ignored (Identity has no parameters).
func (c *Identity) Decode(dst, src, params []byte) (int, error) {
	if len(dst) < len(src) {
		return 0, ErrBufferTooSmall
	}
	return copy(dst, src), nil
}

// Encode copies src to dst unchanged.
//
// Parameters are ignored (Identity has no parameters).
func (c *Identity) Encode(dst, src, params []byte) (int, error) {
	if len(dst) < len(src) {
		return 0, ErrBufferTooSmall
	}
	return copy(dst, src), nil
}

// PreservesSize returns true because Identity always produces output
// of the same size as its input (1:1 copy).
func (c *Identity) PreservesSize() bool {
	return true
}
