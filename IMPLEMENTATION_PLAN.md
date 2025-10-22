# go-openzl Implementation Plan

## Overview

This document outlines the phased approach to implementing Go bindings for Meta's OpenZL compression library. The plan is designed to deliver a working prototype quickly while maintaining a clear path to a production-ready library.

---

## Phase 0: Foundation (Week 1)

**Goal**: Set up project infrastructure and build system

### Tasks

#### 0.1 Project Structure
- [x] Initialize Git repository
- [x] Add LICENSE and CONTRIBUTING.md
- [x] Create README.md
- [ ] Create Go module (`go mod init`)
- [ ] Set up package structure
- [ ] Add .gitignore

#### 0.2 OpenZL C Library Integration
- [ ] Add OpenZL as git submodule or vendored source
- [ ] Create Makefile for building libopenzl
- [ ] Test C library builds on local platform
- [ ] Document build requirements

#### 0.3 Development Environment
- [ ] Set up pre-commit hooks (gofmt, go vet)
- [ ] Configure golangci-lint
- [ ] Create initial test infrastructure
- [ ] Document development setup

### Deliverables
- Working Go module structure
- Buildable OpenZL C library
- Basic CI/CD pipeline (GitHub Actions)

### Success Criteria
- `go mod tidy` runs successfully
- `make build-openzl` produces libopenzl.a
- Tests can import the package

---

## Phase 1: Minimal Viable Prototype (Weeks 2-3)

**Goal**: Create the simplest possible working compression/decompression

### Architecture

```
┌─────────────────────────────────────┐
│  Phase 1: Simple API                │
│                                     │
│  func Compress(src []byte) []byte   │
│  func Decompress(src []byte) []byte │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│  internal/cgo (Low-level bindings)  │
│  - Context lifecycle                │
│  - Basic compress/decompress        │
│  - Error handling                   │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│  OpenZL C Library                   │
│  - ZL_CCtx_create/free              │
│  - ZL_CCtx_compress                 │
│  - ZL_decompress                    │
└─────────────────────────────────────┘
```

### Implementation Details

#### 1.1 Internal CGO Package (`internal/cgo`)

**File**: `internal/cgo/openzl.go`

Minimal C bindings:
```go
package cgo

/*
#cgo CFLAGS: -I${SRCDIR}/../../vendor/openzl/include
#cgo LDFLAGS: -L${SRCDIR}/../../vendor/openzl/lib -lopenzl -lm
#include <stdlib.h>
#include <openzl/openzl.h>
*/
import "C"
import "unsafe"

// CCtx wraps ZL_CCtx
type CCtx struct {
    ctx *C.ZL_CCtx
}

// NewCCtx creates a compression context
func NewCCtx() (*CCtx, error) {
    ctx := C.ZL_CCtx_create()
    if ctx == nil {
        return nil, errors.New("failed to create compression context")
    }
    return &CCtx{ctx: ctx}, nil
}

// Free releases the compression context
func (c *CCtx) Free() {
    if c.ctx != nil {
        C.ZL_CCtx_free(c.ctx)
        c.ctx = nil
    }
}

// Compress compresses src into dst
func (c *CCtx) Compress(dst, src []byte) (int, error) {
    if len(src) == 0 {
        return 0, errors.New("empty input")
    }

    result := C.ZL_CCtx_compress(
        c.ctx,
        unsafe.Pointer(&dst[0]),
        C.size_t(len(dst)),
        unsafe.Pointer(&src[0]),
        C.size_t(len(src)),
    )

    if C.ZL_isError(result) {
        return 0, c.getError(result)
    }

    return int(result), nil
}

// DCtx wraps ZL_DCtx
type DCtx struct {
    ctx *C.ZL_DCtx
}

// NewDCtx creates a decompression context
func NewDCtx() (*DCtx, error) {
    ctx := C.ZL_DCtx_create()
    if ctx == nil {
        return nil, errors.New("failed to create decompression context")
    }
    return &DCtx{ctx: ctx}, nil
}

// Free releases the decompression context
func (d *DCtx) Free() {
    if d.ctx != nil {
        C.ZL_DCtx_free(d.ctx)
        d.ctx = nil
    }
}

// Decompress decompresses src into dst
func (d *DCtx) Decompress(dst, src []byte) (int, error) {
    result := C.ZL_DCtx_decompress(
        d.ctx,
        unsafe.Pointer(&dst[0]),
        C.size_t(len(dst)),
        unsafe.Pointer(&src[0]),
        C.size_t(len(src)),
    )

    if C.ZL_isError(result) {
        return 0, d.getError(result)
    }

    return int(result), nil
}

// GetDecompressedSize returns the size needed for decompression
func GetDecompressedSize(src []byte) (int, error) {
    result := C.ZL_getDecompressedSize(
        unsafe.Pointer(&src[0]),
        C.size_t(len(src)),
    )

    if C.ZL_isError(result) {
        return 0, errorFromCode(result)
    }

    return int(result), nil
}

// CompressBound returns upper bound for compressed size
func CompressBound(srcSize int) int {
    return int(C.ZL_compressBound(C.size_t(srcSize)))
}
```

#### 1.2 Simple Public API (`openzl/simple.go`)

**File**: `simple.go`

```go
package openzl

import (
    "fmt"
    "github.com/yourusername/go-openzl/internal/cgo"
)

// Compress compresses the input data using OpenZL with default settings.
// It returns the compressed data or an error.
//
// This is a simple one-shot compression function. For more control,
// use the Compressor type.
func Compress(src []byte) ([]byte, error) {
    if len(src) == 0 {
        return nil, ErrEmptyInput
    }

    // Create compression context
    ctx, err := cgo.NewCCtx()
    if err != nil {
        return nil, fmt.Errorf("create context: %w", err)
    }
    defer ctx.Free()

    // Allocate destination buffer
    dstSize := cgo.CompressBound(len(src))
    dst := make([]byte, dstSize)

    // Compress
    n, err := ctx.Compress(dst, src)
    if err != nil {
        return nil, fmt.Errorf("compress: %w", err)
    }

    return dst[:n], nil
}

// Decompress decompresses OpenZL-compressed data.
// It returns the decompressed data or an error.
//
// This is a simple one-shot decompression function. For more control,
// use the Decompressor type.
func Decompress(src []byte) ([]byte, error) {
    if len(src) == 0 {
        return nil, ErrEmptyInput
    }

    // Get decompressed size
    dstSize, err := cgo.GetDecompressedSize(src)
    if err != nil {
        return nil, fmt.Errorf("get decompressed size: %w", err)
    }

    // Allocate destination buffer
    dst := make([]byte, dstSize)

    // Create decompression context
    ctx, err := cgo.NewDCtx()
    if err != nil {
        return nil, fmt.Errorf("create context: %w", err)
    }
    defer ctx.Free()

    // Decompress
    n, err := ctx.Decompress(dst, src)
    if err != nil {
        return nil, fmt.Errorf("decompress: %w", err)
    }

    return dst[:n], nil
}
```

#### 1.3 Error Handling (`errors.go`)

**File**: `errors.go`

```go
package openzl

import "errors"

var (
    // ErrEmptyInput indicates that the input buffer is empty
    ErrEmptyInput = errors.New("openzl: empty input")

    // ErrBufferTooSmall indicates that the destination buffer is too small
    ErrBufferTooSmall = errors.New("openzl: buffer too small")

    // ErrCorruptedData indicates that the compressed data is corrupted
    ErrCorruptedData = errors.New("openzl: corrupted data")

    // ErrInvalidParameter indicates an invalid parameter was passed
    ErrInvalidParameter = errors.New("openzl: invalid parameter")
)
```

#### 1.4 Basic Tests (`simple_test.go`)

**File**: `simple_test.go`

```go
package openzl_test

import (
    "bytes"
    "testing"

    "github.com/yourusername/go-openzl"
)

func TestCompressDecompress(t *testing.T) {
    tests := []struct {
        name  string
        input []byte
    }{
        {
            name:  "simple text",
            input: []byte("hello world"),
        },
        {
            name:  "repeated data",
            input: bytes.Repeat([]byte("test"), 100),
        },
        {
            name:  "binary data",
            input: []byte{0x00, 0x01, 0x02, 0xFF, 0xFE},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Compress
            compressed, err := openzl.Compress(tt.input)
            if err != nil {
                t.Fatalf("Compress() error = %v", err)
            }

            t.Logf("Original: %d bytes, Compressed: %d bytes, Ratio: %.2f",
                len(tt.input), len(compressed),
                float64(len(tt.input))/float64(len(compressed)))

            // Decompress
            decompressed, err := openzl.Decompress(compressed)
            if err != nil {
                t.Fatalf("Decompress() error = %v", err)
            }

            // Verify
            if !bytes.Equal(tt.input, decompressed) {
                t.Errorf("Decompressed data doesn't match original")
            }
        })
    }
}

func TestCompressEmpty(t *testing.T) {
    _, err := openzl.Compress([]byte{})
    if err == nil {
        t.Error("Expected error for empty input")
    }
}

func TestDecompressCorrupted(t *testing.T) {
    corrupted := []byte{0x00, 0x01, 0x02, 0x03}
    _, err := openzl.Decompress(corrupted)
    if err == nil {
        t.Error("Expected error for corrupted data")
    }
}
```

### Tasks Breakdown

#### Week 2
- [ ] Create `internal/cgo` package with basic bindings
- [ ] Implement `NewCCtx`, `Free`, `Compress`
- [ ] Implement `NewDCtx`, `Free`, `Decompress`
- [ ] Implement error handling helpers
- [ ] Create simple public API functions
- [ ] Write basic unit tests

#### Week 3
- [ ] Add comprehensive error handling
- [ ] Test with various data types (text, binary, repeated)
- [ ] Add benchmarks
- [ ] Document the simple API
- [ ] Create usage examples
- [ ] Test on Linux, macOS, Windows (if possible)

### Deliverables
- Working `Compress()` and `Decompress()` functions
- Passing unit tests
- Basic benchmarks
- Simple example program

### Success Criteria
- Can compress and decompress "Hello, World!"
- Round-trip test passes for various inputs
- Memory doesn't leak (use `go test -race`)
- Documentation is clear

---

## Phase 2: Context-Based API (Weeks 4-5)

**Goal**: Add reusable contexts for better performance and control

### Architecture

```go
// Compressor holds a compression context that can be reused
type Compressor struct {
    ctx *cgo.CCtx
}

func NewCompressor() (*Compressor, error)
func (c *Compressor) Compress(dst, src []byte) (int, error)
func (c *Compressor) CompressBound(srcSize int) int
func (c *Compressor) Close() error

// Decompressor holds a decompression context that can be reused
type Decompressor struct {
    ctx *cgo.DCtx
}

func NewDecompressor() (*Decompressor, error)
func (d *Decompressor) Decompress(dst, src []byte) (int, error)
func (d *Decompressor) GetDecompressedSize(src []byte) (int, error)
func (d *Decompressor) Close() error
```

### Implementation Details

#### 2.1 Compressor Type (`compressor.go`)

```go
package openzl

import (
    "fmt"
    "sync"

    "github.com/yourusername/go-openzl/internal/cgo"
)

// Compressor provides a reusable compression context.
// It is safe for concurrent use by multiple goroutines.
type Compressor struct {
    ctx   *cgo.CCtx
    mu    sync.Mutex
    closed bool
}

// NewCompressor creates a new Compressor.
func NewCompressor(opts ...Option) (*Compressor, error) {
    ctx, err := cgo.NewCCtx()
    if err != nil {
        return nil, err
    }

    c := &Compressor{ctx: ctx}

    // Apply options
    for _, opt := range opts {
        if err := opt.applyCompress(c); err != nil {
            ctx.Free()
            return nil, fmt.Errorf("apply option: %w", err)
        }
    }

    return c, nil
}

// Compress compresses src into dst.
// dst must be large enough to hold the compressed data.
// Use CompressBound to determine the required size.
func (c *Compressor) Compress(dst, src []byte) (int, error) {
    c.mu.Lock()
    defer c.mu.Unlock()

    if c.closed {
        return 0, ErrContextClosed
    }

    if len(src) == 0 {
        return 0, ErrEmptyInput
    }

    if len(dst) < c.CompressBound(len(src)) {
        return 0, ErrBufferTooSmall
    }

    return c.ctx.Compress(dst, src)
}

// CompressBound returns the maximum compressed size for input of given size.
func (c *Compressor) CompressBound(srcSize int) int {
    return cgo.CompressBound(srcSize)
}

// Close releases the compression context.
// After calling Close, the Compressor cannot be used.
func (c *Compressor) Close() error {
    c.mu.Lock()
    defer c.mu.Unlock()

    if c.closed {
        return nil
    }

    c.ctx.Free()
    c.closed = true
    return nil
}
```

#### 2.2 Options Pattern (`options.go`)

```go
package openzl

// Option configures a Compressor or Decompressor
type Option interface {
    applyCompress(*Compressor) error
    applyDecompress(*Decompressor) error
}

type option struct {
    compressFn   func(*Compressor) error
    decompressFn func(*Decompressor) error
}

func (o option) applyCompress(c *Compressor) error {
    if o.compressFn != nil {
        return o.compressFn(c)
    }
    return nil
}

func (o option) applyDecompress(d *Decompressor) error {
    if o.decompressFn != nil {
        return o.decompressFn(d)
    }
    return nil
}

// WithCompressionLevel sets the compression level (1-9)
func WithCompressionLevel(level int) Option {
    return option{
        compressFn: func(c *Compressor) error {
            return c.ctx.SetParameter(cgo.ParamCompressionLevel, level)
        },
    }
}

// WithChecksum enables content checksum verification
func WithChecksum(enable bool) Option {
    val := 0
    if enable {
        val = 1
    }
    return option{
        compressFn: func(c *Compressor) error {
            return c.ctx.SetParameter(cgo.ParamContentChecksum, val)
        },
        decompressFn: func(d *Decompressor) error {
            return d.ctx.SetParameter(cgo.ParamCheckContentChecksum, val)
        },
    }
}
```

### Tasks Breakdown

#### Week 4
- [ ] Implement `Compressor` type with mutex for thread-safety
- [ ] Implement `Decompressor` type
- [ ] Add options pattern for configuration
- [ ] Update internal CGO to support parameters
- [ ] Add context lifecycle tests

#### Week 5
- [ ] Add concurrent usage tests
- [ ] Add benchmarks comparing one-shot vs context reuse
- [ ] Document context-based API
- [ ] Create examples showing reuse benefits
- [ ] Test edge cases (close twice, use after close, etc.)

### Deliverables
- `Compressor` and `Decompressor` types
- Options for configuration
- Thread-safety guarantees
- Performance benchmarks

### Success Criteria
- Context can be reused multiple times
- Thread-safe operation verified
- 10-50% performance improvement over one-shot API for repeated operations
- No memory leaks with repeated use

---

## Phase 3: Typed Compression (Weeks 6-7)

**Goal**: Support OpenZL's typed compression for better ratios

### API Design

```go
// CompressNumeric compresses a numeric array with type awareness
func (c *Compressor) CompressNumeric(dst []byte, src interface{}) (int, error)

// CompressStruct compresses fixed-width structures
func (c *Compressor) CompressStruct(dst []byte, structWidth int, src []byte) (int, error)

// DecompressNumeric decompresses into a numeric array
func (d *Decompressor) DecompressNumeric(dst interface{}, src []byte) error

// DecompressStruct decompresses fixed-width structures
func (d *Decompressor) DecompressStruct(dst []byte, src []byte) (int, error)
```

### Implementation Priorities

1. **Numeric arrays** (int32, int64, float32, float64)
2. **Struct arrays** (fixed-width records)
3. **String arrays** (variable-length with lengths)

### Tasks Breakdown

#### Week 6
- [ ] Add TypedRef creation in CGO layer
- [ ] Implement `CompressNumeric` for basic types
- [ ] Add reflection-based type checking
- [ ] Implement `DecompressNumeric`

#### Week 7
- [ ] Implement struct compression
- [ ] Add tests with various numeric types
- [ ] Benchmark typed vs untyped compression
- [ ] Document typed API

### Deliverables
- Typed compression for numeric arrays
- Struct compression support
- Benchmarks showing improved ratios

### Success Criteria
- 2-5x better compression for sorted integers
- Type safety via Go type system
- Clear error messages for type mismatches

---

## Phase 4: Streaming API (Weeks 8-9)

**Goal**: io.Reader/Writer interface support

### API Design

```go
type Writer struct {
    w   io.Writer
    ctx *Compressor
    buf []byte
}

func NewWriter(w io.Writer, opts ...Option) (*Writer, error)
func (w *Writer) Write(p []byte) (int, error)
func (w *Writer) Close() error

type Reader struct {
    r   io.Reader
    ctx *Decompressor
    buf []byte
}

func NewReader(r io.Reader, opts ...Option) (*Reader, error)
func (r *Reader) Read(p []byte) (int, error)
```

### Tasks Breakdown

#### Week 8
- [ ] Implement `Writer` with buffering
- [ ] Implement `Reader` with buffering
- [ ] Add tests with various buffer sizes
- [ ] Test with standard library (io.Copy, bufio, etc.)

#### Week 9
- [ ] Optimize buffer management
- [ ] Add streaming benchmarks
- [ ] Document streaming API
- [ ] Create examples (compress file, network stream, etc.)

### Deliverables
- `io.Writer` implementation
- `io.Reader` implementation
- Integration with stdlib

### Success Criteria
- Works with `io.Copy`
- Proper EOF handling
- Reasonable buffer sizes (4KB-64KB)

---

## Phase 5: Production Ready (Weeks 10-12)

**Goal**: Harden for production use

### Tasks

#### Week 10: Testing & Quality
- [ ] Fuzz testing for all APIs
- [ ] Memory leak detection with valgrind/sanitizers
- [ ] Edge case testing (nil, empty, huge inputs)
- [ ] Error path testing
- [ ] Thread safety stress tests

#### Week 11: Performance
- [ ] Comprehensive benchmarks
- [ ] Memory profiling and optimization
- [ ] CPU profiling and optimization
- [ ] Comparison benchmarks vs stdlib compression

#### Week 12: Documentation & Polish
- [ ] Complete godoc for all exports
- [ ] Write migration guide from other libraries
- [ ] Create cookbook with common patterns
- [ ] Performance tuning guide
- [ ] Finalize examples

### Deliverables
- Fuzz tests
- Performance benchmarks
- Complete documentation
- v1.0.0 release candidate

### Success Criteria
- No memory leaks detected
- No data races detected
- Documentation coverage 100%
- Ready for production use

---

## Build System

### Makefile

```makefile
.PHONY: all build test bench clean build-openzl

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build targets
all: test build

build:
	$(GOBUILD) -v ./...

test:
	$(GOTEST) -v -race ./...

bench:
	$(GOTEST) -bench=. -benchmem ./...

coverage:
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out

# OpenZL C library
build-openzl:
	cd vendor/openzl && make lib BUILD_TYPE=OPT
	mkdir -p vendor/openzl/lib
	cp vendor/openzl/libopenzl.a vendor/openzl/lib/

clean:
	$(GOCMD) clean
	rm -f coverage.out
	cd vendor/openzl && make clean

# Development
fmt:
	gofmt -s -w .

lint:
	golangci-lint run

# CI targets
ci: fmt lint test bench

.DEFAULT_GOAL := all
```

---

## Success Metrics

### Phase 1 (MVP)
- [ ] Compress/decompress "Hello World" successfully
- [ ] 100% test coverage for simple API
- [ ] No memory leaks in basic tests

### Phase 2 (Context API)
- [ ] 10x faster for repeated compression (vs creating new context each time)
- [ ] Thread-safe verified with race detector
- [ ] Used in at least 2 example programs

### Phase 3 (Typed)
- [ ] 2x better ratio for sorted int arrays
- [ ] Works with all numeric Go types
- [ ] Type safety enforced at compile time

### Phase 4 (Streaming)
- [ ] Works seamlessly with io.Copy
- [ ] Comparable performance to stdlib compression
- [ ] Can compress/decompress files >100MB

### Phase 5 (Production)
- [ ] Zero known bugs
- [ ] Fuzzing finds no crashes (1M+ inputs)
- [ ] Documentation complete
- [ ] At least 5 stars on GitHub 😊

---

## Risk Management

### Technical Risks

**Risk**: CGO complexity and platform-specific issues
- **Mitigation**: Start with Linux/macOS, add Windows later
- **Mitigation**: Comprehensive CI matrix

**Risk**: Memory management bugs (leaks, use-after-free)
- **Mitigation**: Use ASAN/UBSAN in tests
- **Mitigation**: valgrind in CI
- **Mitigation**: Careful finalizer use

**Risk**: Performance not competitive
- **Mitigation**: Early benchmarking
- **Mitigation**: Profile-guided optimization
- **Mitigation**: Compare with C benchmarks

### Project Risks

**Risk**: Scope creep
- **Mitigation**: Stick to phase boundaries
- **Mitigation**: Phase 1 must work before Phase 2

**Risk**: OpenZL API changes
- **Mitigation**: Vendor specific OpenZL version
- **Mitigation**: Track upstream releases

---

## Timeline Summary

| Phase | Duration | Key Deliverable |
|-------|----------|----------------|
| Phase 0 | 1 week | Project setup, buildable C library |
| Phase 1 | 2 weeks | Working MVP (Compress/Decompress) |
| Phase 2 | 2 weeks | Context-based API |
| Phase 3 | 2 weeks | Typed compression |
| Phase 4 | 2 weeks | Streaming API |
| Phase 5 | 3 weeks | Production hardening |
| **Total** | **12 weeks** | **v1.0.0 Release** |

---

## Next Steps

1. **Review this plan** - Get feedback, adjust timeline
2. **Start Phase 0** - Set up project structure
3. **Weekly check-ins** - Track progress, adjust as needed
4. **Community engagement** - Share progress, get early feedback

Let's build something great! 🚀
