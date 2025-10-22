# Go Code Style Guide for go-openzl

## Comment Standards

### General Principles

1. **All exported identifiers MUST have doc comments**
2. **Comments should explain WHY, not WHAT**
3. **Use complete sentences with proper punctuation**
4. **Start comments with the name of the thing being documented**
5. **Be concise but comprehensive**

### Package Comments

Every package should have a package-level doc comment in a `doc.go` file:

```go
// Package openzl provides Go bindings for Meta's OpenZL format-aware compression library.
//
// OpenZL delivers high compression ratios while preserving high speed, a level of performance
// that is out of reach for generic compressors.
//
// # Quick Start
//
// For simple compression:
//
//	compressed, err := openzl.Compress([]byte("hello world"))
//	if err != nil {
//		log.Fatal(err)
//	}
//
// # Performance
//
// This library provides excellent performance through direct C bindings.
// See benchmarks for detailed metrics.
package openzl
```

### Function/Method Comments

**Required elements:**
1. What the function does (first sentence)
2. Parameter descriptions (if non-obvious)
3. Return value descriptions (if non-obvious)
4. Error conditions
5. Usage example (for complex APIs)
6. Important notes, warnings, or gotchas

**Good Example:**

```go
// Compress compresses the input data using OpenZL with default settings.
// It returns the compressed data or an error.
//
// The compression uses OpenZL's format-aware compression with automatic
// format detection. For better performance with repeated operations,
// use the Compressor type instead.
//
// Returns ErrEmptyInput if src is empty.
//
// Example:
//
//	data := []byte("hello world")
//	compressed, err := openzl.Compress(data)
//	if err != nil {
//		log.Fatal(err)
//	}
func Compress(src []byte) ([]byte, error) {
	// Implementation
}
```

**Bad Example:**

```go
// Compress compresses data
func Compress(src []byte) ([]byte, error) {
	// Implementation
}
```

### Type Comments

Exported types must document:
1. What the type represents
2. Zero value behavior
3. Thread safety guarantees
4. Typical usage pattern

**Good Example:**

```go
// Compressor provides a reusable compression context for OpenZL compression.
//
// A Compressor maintains internal state and can be reused across multiple
// compression operations, which is significantly faster than creating a new
// context for each operation.
//
// Compressor is safe for concurrent use by multiple goroutines. Each compression
// operation is serialized using an internal mutex.
//
// Example:
//
//	c, err := openzl.NewCompressor()
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer c.Close()
//
//	// Compress multiple inputs efficiently
//	for _, data := range inputs {
//		compressed, err := c.Compress(data)
//		// ...
//	}
type Compressor struct {
	ctx    *cgo.CCtx
	mu     sync.Mutex
	closed bool
}
```

### Field Comments

Unexported fields should be commented if their purpose isn't obvious:

```go
type Compressor struct {
	ctx *cgo.CCtx // Underlying C compression context

	mu sync.Mutex // Protects ctx from concurrent access

	closed bool // Tracks whether Close() has been called
}
```

### Constant/Variable Comments

Document constants and variables, especially:
- Error values
- Configuration defaults
- Magic numbers

```go
var (
	// ErrEmptyInput indicates that the input buffer is empty.
	// This is returned by compression and decompression functions
	// when passed a zero-length byte slice.
	ErrEmptyInput = errors.New("openzl: empty input")

	// ErrBufferTooSmall indicates that the destination buffer is too small
	// to hold the compressed or decompressed data. Use CompressBound() to
	// determine the required buffer size.
	ErrBufferTooSmall = errors.New("openzl: buffer too small")
)

const (
	// DefaultCompressionLevel is the default compression level used
	// when not explicitly specified. Higher values provide better
	// compression at the cost of speed. Valid range: 1-9.
	DefaultCompressionLevel = 6
)
```

### Internal Package Comments

Internal packages should still have good comments, but can be less formal:

```go
// Package cgo provides low-level CGO bindings to the OpenZL C library.
//
// This package is internal and should not be used directly. Use the openzl
// package instead, which provides a safe, idiomatic Go interface.
//
// The bindings in this package are thin wrappers around the OpenZL C API,
// handling memory management, error translation, and type conversions.
package cgo
```

### Inline Comments

Use inline comments sparingly, only when the code isn't self-explanatory:

**Good - explains non-obvious behavior:**

```go
// Set default format version (required by OpenZL)
result := C.ZL_CCtx_setParameter(ctx, C.ZL_CParam_formatVersion, C.ZL_MAX_FORMAT_VERSION)
```

**Bad - states the obvious:**

```go
// Create compression context
ctx := C.ZL_CCtx_create()
```

### TODO Comments

Use TODO comments for planned improvements:

```go
// TODO(username): Add support for custom compression parameters
// TODO: Optimize memory allocation for small inputs (< 1KB)
```

### Deprecation Comments

Mark deprecated functions clearly:

```go
// Deprecated: Use NewCompressor instead. This function will be removed in v2.0.0.
//
// CompressWithLevel compresses data with a specific compression level.
// The new Compressor API provides better performance and more flexibility.
func CompressWithLevel(src []byte, level int) ([]byte, error) {
	// Implementation
}
```

### Error Comments

When returning errors, explain what failed and why:

```go
if len(src) == 0 {
	return nil, ErrEmptyInput
}

if C.ZL_isError(result) != 0 {
	return 0, fmt.Errorf("compress failed: %w", c.getError(result))
}
```

### Performance Notes

Document performance characteristics when relevant:

```go
// CompressBound returns the maximum compressed size for input of the given size.
// This provides an upper bound for buffer allocation, though actual compressed
// size is typically much smaller.
//
// The returned size is conservative and suitable for worst-case scenarios.
// For typical data, actual compressed size is 50-90% smaller.
func CompressBound(srcSize int) int {
	return cgo.CompressBound(srcSize)
}
```

### CGO-Specific Comments

For CGO code, explain the C interaction:

```go
// Compress compresses src into dst using the OpenZL C API.
//
// This function calls ZL_CCtx_compress from the OpenZL C library,
// which requires proper memory alignment and buffer sizing.
// The caller must ensure dst is large enough (use CompressBound).
//
// Returns the number of bytes written to dst or an error.
func (c *CCtx) Compress(dst, src []byte) (int, error) {
	// Convert Go slices to C pointers
	result := C.ZL_CCtx_compress(
		c.ctx,
		unsafe.Pointer(&dst[0]),
		C.size_t(len(dst)),
		unsafe.Pointer(&src[0]),
		C.size_t(len(src)),
	)

	// Check for errors using OpenZL's Result type
	if C.ZL_isError(result) != 0 {
		return 0, c.getError(result)
	}

	// Extract the actual compressed size
	return int(C.ZL_validResult(result)), nil
}
```

## Documentation Examples

### Example Code in Comments

Use the `Example` pattern for runnable examples:

```go
// Example usage:
//
//	c, err := openzl.NewCompressor()
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer c.Close()
//
//	data := []byte("hello world")
//	compressed, err := c.Compress(data)
//	if err != nil {
//		log.Fatal(err)
//	}
```

Or create proper Example functions:

```go
func ExampleCompress() {
	data := []byte("hello world")
	compressed, err := openzl.Compress(data)
	if err != nil {
		log.Fatal(err)
	}

	decompressed, err := openzl.Decompress(compressed)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s\n", decompressed)
	// Output: hello world
}
```

## Comment Quality Checklist

Before committing, verify:

- [ ] All exported types, functions, methods have doc comments
- [ ] Doc comments start with the name of the thing being documented
- [ ] Comments use complete sentences with proper punctuation
- [ ] Complex logic has inline comments explaining WHY
- [ ] Error conditions are documented
- [ ] Thread safety is documented for concurrent types
- [ ] Performance characteristics are noted when relevant
- [ ] Examples are provided for non-trivial APIs
- [ ] No obvious comments that just restate the code
- [ ] CGO code explains C interaction

## Tools

Run these to check documentation:

```bash
# Check for missing documentation
go doc -all | grep "UNDOCUMENTED"

# Generate documentation locally
go doc -all

# View documentation in browser
godoc -http=:6060
# Then visit: http://localhost:6060/pkg/github.com/borischu/go-openzl/
```

## Reference

Follow these standards:
- [Effective Go - Commentary](https://go.dev/doc/effective_go#commentary)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments#doc-comments)
- [Godoc: documenting Go code](https://go.dev/blog/godoc)
