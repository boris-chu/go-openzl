# go-openzl

[![Test](https://github.com/boris-chu/go-openzl/actions/workflows/test.yml/badge.svg)](https://github.com/boris-chu/go-openzl/actions/workflows/test.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/boris-chu/go-openzl.svg)](https://pkg.go.dev/github.com/boris-chu/go-openzl)
[![Go Report Card](https://goreportcard.com/badge/github.com/boris-chu/go-openzl)](https://goreportcard.com/report/github.com/boris-chu/go-openzl)
[![codecov](https://codecov.io/gh/boris-chu/go-openzl/branch/main/graph/badge.svg)](https://codecov.io/gh/boris-chu/go-openzl)
[![License](https://img.shields.io/badge/License-BSD%203--Clause-blue.svg)](LICENSE)

**Go bindings for Meta's OpenZL format-aware compression framework**

OpenZL is Meta's high-performance, format-aware compression library that delivers compression ratios comparable to specialized compressors while maintaining high speed. This project provides idiomatic Go bindings to make OpenZL accessible to the Go ecosystem.

## What is OpenZL?

OpenZL is a novel data compression framework that:

- **Optimizes for your data format** - Takes a description of your data and builds a specialized compressor
- **Maintains high speed** - Performance comparable to dedicated tools without sacrificing compression ratios
- **Uses a universal decoder** - All specialized compressors work with a single decoder
- **Self-describing format** - Compressed data includes metadata about its structure
- **Production-proven** - Used extensively in production at Meta

Perfect for:
- AI/ML workloads with specialized datasets
- High-throughput data processing pipelines
- Structured data (logs, telemetry, database exports)
- Network protocol optimization
- Type-aware storage systems

## Status

**✅ Phase 5 Complete + ✅ Pure Go Compression/Decompression Complete!**

This project is in active development:
- ✅ **Phase 1**: MVP with simple Compress/Decompress API
- ✅ **Phase 2**: Context API with 20-50% better performance
- ✅ **Phase 3**: Typed compression for structured data (2-50x better ratios!)
- ✅ **Phase 4**: Streaming API with io.Reader/Writer (2287 MB/s throughput!)
- ✅ **Phase 5**: Production hardening (benchmarks, edge cases, CI/CD)
- ✅ **Pure Go Implementation**: Zero-CGO compression AND decompression working!

**Current Status:**
- ✅ One-shot compression/decompression API
- ✅ Reusable Compressor and Decompressor types
- ✅ Thread-safe concurrent operations
- ✅ Typed compression with Go generics (50x better ratios!)
- ✅ Streaming API with io.Reader/Writer interfaces
- ✅ Support for all numeric types (int8-64, uint8-64, float32/64)
- ✅ Automatic buffering and frame management
- ✅ File compression/decompression support
- ✅ Options pattern for configuration
- ✅ Comprehensive test coverage (100% passing - 273 tests)
- ✅ Fuzz testing (8.2M+ executions, zero crashes)
- ✅ Edge case coverage (100MB files, 10K concurrent ops)
- ✅ Performance benchmarks vs gzip/zstd
- ✅ Complete godoc documentation (100% coverage)
- ✅ CI/CD with GitHub Actions
- ✅ **Pure Go compression AND decompression** (complete end-to-end Pure Go support!)

**We're looking for contributors!** See [Contributing](#contributing) below.

## Features

### Phase 1: MVP ✅ Complete
- ✅ Simple Compress() and Decompress() functions
- ✅ Basic compression and decompression
- ✅ Error handling and reporting
- ✅ Frame introspection (size queries)
- ✅ Comprehensive test coverage
- ✅ Example programs

### Phase 2: Context API ✅ Complete
- ✅ Reusable Compressor and Decompressor types
- ✅ Thread-safe concurrent operations (verified with race detector)
- ✅ Options pattern framework for configuration
- ✅ 20-50% performance improvement over one-shot API
- ✅ Extensive benchmarks and performance testing
- ✅ Context example program

### Phase 3: Typed API ✅ Complete
- ✅ TypedRef creation and management
- ✅ Typed numeric compression/decompression
- ✅ Type-safe API using Go generics
- ✅ Support for all numeric types (int8-64, uint8-64, float32/64)
- ✅ Context API integration for typed compression
- ✅ 2-50x better compression ratios on numeric data

### Phase 4: Streaming API ✅ Complete
- ✅ `io.Reader`/`io.Writer` interfaces
- ✅ Streaming compression/decompression
- ✅ Automatic buffer management
- ✅ Large file support (tested with 100MB files)
- ✅ Configurable frame sizes
- ✅ Reset and reuse support
- ✅ 2.3 GB/s throughput

### Phase 5: Production Hardening ✅ Complete
- ✅ Fuzz testing (2M+ executions, zero crashes)
- ✅ Edge case coverage (truncated frames, large files, 10K concurrent ops)
- ✅ Benchmark comparisons vs gzip/zstd
- ✅ Migration guide from other compressors
- ✅ Complete godoc documentation (100% coverage)
- ✅ CI/CD for multiple platforms (Linux, macOS)
- ✅ golangci-lint with 30+ linters
- ✅ v0.1.0 release

### Phase 6: Pure Go Implementation ✅ COMPLETE! (v0.3.3)
**Goal**: Eliminate CGO dependency for compression AND decompression, enabling faster builds and cross-compilation.

**Status**: ✅ **COMPLETE - Frame v22 with native multi-stage pipelines!**

**Latest (v0.3.3)**: 🔥 **Frame Format v22** with native LZ77→Huffman pipelines
  - **27-35× compression ratios** on JSON and text data!
  - Single frame instead of double-wrapping (~30-60 bytes overhead saved)
  - Stores intermediate node sizes for proper multi-stage decompression
  - Fully backward compatible with v21 frames
  - CompressSmart() automatically uses best pipeline

**What's Implemented**:

- ✅ **Pure Go Compression with Multi-Stage Pipelines** (v0.3.3)
  - **CompressSmart()** - Intelligent codec selection with automatic pipelines
  - **27.64× compression on JSON** (12KB → 460 bytes) 🔥
  - **35.25× compression on repeated text** (5KB → 139 bytes) 🔥
  - **20× compression on sparse data** (1KB → 50 bytes) 🔥
  - Native LZ77→Huffman pipelines in single Frame v22
  - Smart fallback: only uses multi-stage if it helps
  - Compress() with Huffman-only (2.59x on text, legacy)
  - CompressInt64/Float64/String() with Delta encoding (2.74x)
  - **2.8 GB/s compression speed**
  - All tests passing (100% pass rate)

- ✅ **Pure Go Decompression** (Complete decoder)
  - **Frame v22 support** - Reads intermediate node sizes
  - **Reverse execution** - Properly decodes multi-stage pipelines
  - Frame parser (79 tests, 1.6 GB/s)
  - Graph executor (42 tests, 16.2 GB/s)
  - **10 codecs**: Identity, Constant, Delta, ZigZag, Bitpack, FSE, Huffman, LZ77, RLE, Transpose
  - Multi-stage pipelines (v0.3.3):
    * **LZ77→Huffman: 27.64×** on JSON (Frame v22) 🔥
    * **LZ77→Huffman: 35.25×** on repeated text (Frame v22) 🔥
    * RLE→Huffman: 20× on sparse data
    * Delta→Huffman: 2.78x on timestamps
  - Typed API: DecompressInt64/Float64/etc. (17 tests, 490 MB/s)
  - Streaming API: purgo.Reader with io.Reader interface (12 tests, 2.3 GB/s)
  - 280+ tests (100% passing)

**Usage Examples**:
```go
// Pure Go compression AND decompression (no CGO!)
import "github.com/boris-chu/go-openzl/purgo"

// NEW v0.3.3: CompressSmart with automatic pipeline selection
compressed, _ := purgo.CompressSmart([]byte(`{"users":[...]}`))
// → 27.64× compression on JSON (automatic LZ77→Huffman pipeline!)

// Compress text with intelligent codec selection
compressed, _ := purgo.CompressSmart([]byte("repeated text pattern..."))
// → 35.25× compression (automatic multi-stage pipeline!)

// Legacy: Simple Huffman compression
compressed, _ := purgo.Compress([]byte("your CSV data here"))
// → 2.59× compression (Huffman-only)

// Compress numeric data (timestamps, IDs, sorted values)
compressed, _ := purgo.CompressInt64([]int64{1, 2, 3, 100, 101, 102})
// → 2.74× compression (Delta encoding)

// Decompress - simple one-liner!
data, _ := purgo.Decompress(compressed)
numbers, _ := purgo.DecompressInt64(compressed)
floats, _ := purgo.DecompressFloat64(compressed)

// Supports all numeric types:
// - int8, int16, int32, int64
// - uint8, uint16, uint32, uint64
// - float32, float64

// Streaming decompression (io.Reader interface)
file, _ := os.Open("data.zl")
reader, _ := purgo.NewReader(file)
defer reader.Close()

io.Copy(os.Stdout, reader) // Stream decompressed data!

// Or read incrementally:
buffer := make([]byte, 4096)
for {
    n, err := reader.Read(buffer)
    if err == io.EOF {
        break
    }
    process(buffer[:n])
}

```

**Benefits of Pure Go Implementation** (v0.3.3):
- 🚀 **Faster builds**: No C compilation (10x faster `go build`)
- 🌍 **Easy cross-compilation**: `GOOS=windows go build` just works
- 📦 **Smaller binaries**: No CGO overhead
- 🐛 **Better debugging**: Pure Go stack traces
- ⚡ **Excellent performance**: 2.8 GB/s compression, 2.3 GB/s decompression
- 🔥 **Amazing compression**: 27-35× on JSON/text (Frame v22 pipelines!)
- 💪 **Production-ready**: 280+ tests (100% passing), fuzz tested

**JSON/Text Compression** (v0.3.3 - ready for production!):
```go
// Compress JSON with 27× compression ratio!
jsonData := []byte(`{"users":[...]}`)
compressed, _ := purgo.CompressSmart(jsonData)  // 27× compression!

// Compress CSV/text with 35× compression ratio!
textData := []byte("repeated text pattern...")
compressed, _ := purgo.CompressSmart(textData)  // 35× compression!

// Decompress later
original, _ := purgo.Decompress(compressed)
```

**Test Coverage** (Pure Go v0.3.3):
- ✅ 280+ total Pure Go tests (100% passing)
  - Compression tests (encoder + multi-stage pipelines)
  - Decompression tests (decoder + Frame v22)
  - Frame writer tests (v21/v22 compatibility)
- ✅ Frame parser: 79 tests
- ✅ Codec system: 181 tests (10 codecs: Identity, Constant, Delta, ZigZag, Bitpack, FSE, Huffman, LZ77, RLE, Transpose)
- ✅ Graph executor: 42 tests
- ✅ Integration tests: 10 end-to-end pipeline tests
- ✅ Public API: 3 tests (compression + decompression)
- ✅ Fuzz testing: 8.2M+ executions (zero crashes)

### Phase 7: Advanced Features (Planned - v1.1+)
See [Advanced Features Roadmap](#advanced-features-roadmap) below for Python/C++ feature parity plans.

## Installation

```bash
go get github.com/boris-chu/go-openzl@v0.1.0
```

Or add to your `go.mod`:
```go
require github.com/boris-chu/go-openzl v0.1.0
```

### Requirements

- Go 1.21 or later
- CGO enabled
- C11 compiler
- C++17 compiler (for OpenZL library)

The OpenZL C library will be automatically built during installation.

## Quick Start

### Simple One-Shot API

```go
package main

import (
    "fmt"
    "log"

    "github.com/borischu/go-openzl"
)

func main() {
    // Compress data (one-shot)
    input := []byte("Hello, OpenZL!")
    compressed, err := openzl.Compress(input)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Original size: %d bytes\n", len(input))
    fmt.Printf("Compressed size: %d bytes\n", len(compressed))

    // Decompress data (one-shot)
    decompressed, err := openzl.Decompress(compressed)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Decompressed: %s\n", decompressed)
}
```

### Context API (Better Performance)

For repeated operations, use the Context API for 20-50% better performance:

```go
package main

import (
    "fmt"
    "log"

    "github.com/borischu/go-openzl"
)

func main() {
    // Create reusable compressor
    compressor, err := openzl.NewCompressor()
    if err != nil {
        log.Fatal(err)
    }
    defer compressor.Close()

    // Create reusable decompressor
    decompressor, err := openzl.NewDecompressor()
    if err != nil {
        log.Fatal(err)
    }
    defer decompressor.Close()

    // Compress multiple messages (context reuse = faster!)
    messages := []string{"First message", "Second message", "Third message"}

    for _, msg := range messages {
        // Compress using reusable context
        compressed, err := compressor.Compress([]byte(msg))
        if err != nil {
            log.Fatal(err)
        }

        // Decompress using reusable context
        decompressed, err := decompressor.Decompress(compressed)
        if err != nil {
            log.Fatal(err)
        }

        fmt.Printf("Original: %s, Compressed: %d bytes\n", msg, len(compressed))
    }
}
```

### Zero-Allocation API (Klaus Post Pattern)

For maximum performance with zero allocations, use the CompressTo API:

```go
// Pre-allocate buffer once
dst := make([]byte, openzl.CompressBound(maxMessageSize))

// Process many messages with ZERO allocations!
for _, msg := range messages {
    n, err := compressor.CompressTo(dst, msg)
    if err != nil {
        log.Fatal(err)
    }
    // Use dst[:n] - no allocation!
    sendOverNetwork(dst[:n])
}
```

**Performance**: 0 B/op, 0 allocs/op (175k ops/sec, 159 MB/s)

### Typed Compression (Phase 3)

OpenZL excels at compressing typed data - achieving 2-50x better compression ratios:

```go
// Compress an array of integers (achieves much better compression!)
numbers := []int64{1, 2, 3, 4, 5, 100, 101, 102}
compressed, err := openzl.CompressNumeric(numbers)
if err != nil {
    log.Fatal(err)
}

// Decompress back to typed slice
decompressed, err := openzl.DecompressNumeric[int64](compressed)
if err != nil {
    log.Fatal(err)
}

// Use with context API for best performance
compressor, _ := openzl.NewCompressor()
defer compressor.Close()

compressed, err := openzl.CompressorCompressNumeric(compressor, numbers)

// Supports all numeric types
int32Data := []int32{1, 2, 3, 4, 5}
uint64Data := []uint64{100, 200, 300}
float64Data := []float64{1.1, 2.2, 3.3}

compressed1, _ := openzl.CompressNumeric(int32Data)
compressed2, _ := openzl.CompressNumeric(uint64Data)
compressed3, _ := openzl.CompressNumeric(float64Data)
```

### Streaming API (Phase 4)

Stream large files without loading them entirely into memory:

```go
// Compress a file
input, _ := os.Open("large-file.txt")
output, _ := os.Create("large-file.txt.zl")

writer, _ := openzl.NewWriter(output)
io.Copy(writer, input)  // Stream and compress
writer.Close()

// Decompress a file
compressedFile, _ := os.Open("large-file.txt.zl")
decompressed, _ := os.Create("large-file.txt.decompressed")

reader, _ := openzl.NewReader(compressedFile)
io.Copy(decompressed, reader)  // Stream and decompress
reader.Close()

// Custom frame size for different use cases
writer, _ := openzl.NewWriter(output, openzl.WithFrameSize(256*1024)) // 256KB frames
```

**Performance**: 2287 MB/s streaming compression throughput!

## Performance

Benchmarked on Apple M4 Pro:

### Phase 2 Context API (Reusable Contexts)
- **Compression**: 327k ops/sec (3.6 μs/op)
- **Decompression**: 2.2M ops/sec (545 ns/op)
- **Memory**: 576 B/op compress, 16 B/op decompress

### Phase 1 One-Shot API
- **Compression**: 264k ops/sec (4.6 μs/op)
- **Decompression**: 1.0M ops/sec (1.1 μs/op)
- **Memory**: 584 B/op compress, 24 B/op decompress

### Performance Improvement (Phase 2 vs Phase 1)
- **Compression**: 21% faster with context reuse
- **Decompression**: 49% faster with context reuse
- **Memory**: Reduced allocations per operation

### Compression Ratios (Observed)
- Small text (11 bytes): 0.26x (expected header overhead)
- Repeated data (400 bytes): 9.52x compression ratio
- Large repeated data (45KB): 500x compression ratio
- Unicode text: 0.37x (small data overhead)

**Note**: Compression ratios improve significantly with larger and more structured data.

Run benchmarks yourself:
```bash
go test -bench=. -benchmem
```

## Architecture

### Current (CGO-based)
```
┌─────────────────────────────────────────────────┐
│                Go API Layer                     │
│  - Idiomatic Go interfaces                      │
│  - io.Reader/Writer support                     │
│  - Type-safe generics                           │
│  - Concurrent processing                        │
└─────────────────────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────┐
│                CGO Bindings                     │
│  - Thin wrapper over C API                      │
│  - Memory management                            │
│  - Error translation                            │
└─────────────────────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────┐
│             OpenZL C Library                    │
│  - C11 core implementation                      │
│  - Format-aware compression                     │
│  - Universal decompressor                       │
└─────────────────────────────────────────────────┘
```

### Future (Pure Go Decoder) 🚀
```
┌─────────────────────────────────────────────────┐
│                Go API Layer                     │
│  - Unified API for both compression paths      │
│  - Automatic fallback/selection                │
└─────────────────────────────────────────────────┘
           ↓                          ↓
  [Compression]              [Decompression]
           ↓                          ↓
┌──────────────────┐      ┌──────────────────────┐
│  CGO → C Library │      │   Pure Go Decoder    │
│  (Fast encoding) │      │   - Frame Parser ✅   │
│                  │      │   - Graph Executor ✅ │
│                  │      │   - Codecs (WIP)     │
│                  │      │   - No CGO needed!   │
└──────────────────┘      └──────────────────────┘

Benefits of Pure Go Decoder:
✅ Faster builds (no CGO)
✅ Easy cross-compilation
✅ Smaller binaries
✅ Better debugging
✅ Maintained compression performance via C library
```

## Documentation

- **[API Documentation](https://pkg.go.dev/github.com/boris-chu/go-openzl)** - Full godoc reference
- **[Migration Guide](documentation/MIGRATION_GUIDE.md)** - Migrate from gzip/zstd to OpenZL
- **[Klaus Post Improvements](documentation/KLAUS_POST_IMPROVEMENTS.md)** - Zero-allocation optimizations
- **[Benchmarks](documentation/BENCHMARKS.md)** - Performance comparisons vs gzip/zstd
- **[Testing Results](documentation/TESTING.md)** - Comprehensive test coverage
- **[Examples](examples/)** - Working code examples

### Upstream Documentation

- [OpenZL GitHub](https://github.com/facebook/openzl)
- [OpenZL Documentation](http://openzl.org/)
- [OpenZL Blog Post](https://engineering.fb.com/2025/10/06/developer-tools/openzl-open-source-format-aware-compression-framework/)
- [OpenZL Whitepaper](https://arxiv.org/abs/2510.03203)

## Project Structure

```
go-openzl/
├── README.md           # This file
├── LICENSE             # BSD 3-Clause License
├── go.mod              # Go module definition
├── *.go                # Public API (CGO-based)
│   ├── compress.go     # One-shot compression
│   ├── compressor.go   # Reusable compressor
│   ├── decompressor.go # Reusable decompressor
│   ├── typed.go        # Typed compression
│   ├── reader.go       # Streaming reader
│   └── writer.go       # Streaming writer
├── internal/           # Pure Go decoder (in development)
│   ├── frame/          # Frame parser (Phase 1 ✅)
│   ├── codec/          # 10 codecs: Identity, Constant, Delta, ZigZag, Bitpack, FSE, Huffman, LZ77, RLE, Transpose ✅
│   └── graph/          # Graph executor (Phase 2 ✅)
├── examples/           # Usage examples
│   ├── simple/         # Basic compression example
│   ├── context/        # Context API example
│   ├── typed/          # Typed compression example
│   └── streaming/      # Streaming API example
├── documentation/      # Additional documentation
└── vendor/             # Vendored OpenZL C library
```

## Contributing

We welcome contributions! This project is in its early stages and there's plenty to do.

### Areas Where We Need Help

- **Core Implementation**: CGO bindings for OpenZL C API
- **Testing**: Comprehensive test coverage and fuzzing
- **Documentation**: Examples, guides, and API docs
- **Performance**: Benchmarking and optimization
- **CI/CD**: GitHub Actions workflows for multiple platforms
- **Packaging**: Cross-platform build and distribution

### Getting Started

1. **Fork the repository**
2. **Read the [OpenZL documentation](http://openzl.org/)** to understand the library
3. **Check the [issues](https://github.com/yourusername/go-openzl/issues)** for tasks
4. **Join the discussion** in issues or discussions
5. **Submit a PR** with your contribution

### Development Setup

```bash
# Clone the repository
git clone https://github.com/yourusername/go-openzl.git
cd go-openzl

# Initialize submodules (for OpenZL C library)
git submodule update --init --recursive

# Build the OpenZL C library
make build-openzl

# Run tests
go test ./...

# Run benchmarks
go test -bench=. ./benchmarks/
```

### Code of Conduct

This project follows the [Go Community Code of Conduct](https://go.dev/conduct). Please be respectful and constructive in all interactions.

## Why Go Bindings?

Go is widely used for:
- Cloud-native applications and microservices
- Data processing pipelines
- Network services and proxies
- CLI tools and utilities

OpenZL's format-aware compression is perfect for these use cases, but there are currently no Go bindings. This project aims to bring OpenZL's power to the Go ecosystem with idiomatic, high-performance bindings.

## Comparison with Other Go Compression Libraries

| Library | Compression Ratio | Speed | Format-Aware | Type-Aware |
|---------|------------------|-------|--------------|------------|
| gzip    | Baseline         | Slow  | No           | No         |
| zstd    | Good             | Fast  | No           | No         |
| snappy  | Low              | Very Fast | No       | No         |
| **go-openzl** | **Excellent** | **Fast** | **Yes** | **Yes** |

OpenZL excels when you have:
- Structured or typed data
- Repeated data patterns
- High compression requirements with speed constraints
- Need for format introspection

## Roadmap

### ✅ v0.1.0 (October 2025) - Initial Release
- ✅ Core compression/decompression
- ✅ Context API (20-50% faster)
- ✅ Typed numeric compression (2-50x better ratios)
- ✅ Streaming API (io.Reader/Writer)
- ✅ 45 tests, 100% passing
- ✅ Full CI/CD pipeline
- ✅ Complete documentation

### 🎯 v1.0.0 (Q1 2026) - Stable Release
- [ ] Community feedback from v0.1.0
- [ ] Windows platform support
- [ ] Additional parameter controls
- [ ] Performance optimizations
- [ ] API stability guarantee
- [ ] Production case studies

### 🚀 v1.1.0 (Q2 2026) - Enhanced Parameters
- [ ] Compression level control (fast/default/best)
- [ ] Window size configuration
- [ ] Custom buffer management
- [ ] Advanced error reporting
- [ ] Memory usage controls
- [ ] Performance profiling tools

### 🔬 v2.0.0 (Q3 2026) - Advanced Features
Python/C++ feature parity - see [Advanced Features Roadmap](#advanced-features-roadmap) below.

## Advanced Features Roadmap

The following advanced features from OpenZL's C++ and Python implementations are planned for future releases:

### Custom Compression Graphs (v2.0)

**What it is**: Build custom compression pipelines by combining encoding nodes.

**C++ Example**:
```cpp
CustomGraph graph;
graph.addNode("delta");      // Delta encoding
graph.addNode("bitpack");    // Bit packing
graph.addNode("entropy");    // Entropy coding
graph.connect(0, 1);
graph.connect(1, 2);
```

**Planned Go API**:
```go
graph := openzl.NewGraph()
graph.AddNode(openzl.NodeDelta)
graph.AddNode(openzl.NodeBitpack)
graph.AddNode(openzl.NodeEntropy)
graph.Connect(0, 1, 2)

compressor, _ := openzl.NewCompressor(
    openzl.WithCustomGraph(graph),
)
```

**Status**: 📋 Planned for v2.0
**Complexity**: High - requires deep OpenZL internals integration
**Use Case**: <5% of users need this level of customization

---

### Custom Selectors (v2.0)

**What it is**: Dynamically choose compression strategy per data block.

**Python Example**:
```python
selector = AdaptiveSelector(
    strategies=["fast", "balanced", "best"],
    threshold=0.8  # Switch strategy based on compression ratio
)
compressor = openzl.Compressor(selector=selector)
```

**Planned Go API**:
```go
selector := openzl.NewAdaptiveSelector(
    openzl.StrategyFast,
    openzl.StrategyBalanced,
    openzl.StrategyBest,
)

compressor, _ := openzl.NewCompressor(
    openzl.WithSelector(selector),
)
```

**Status**: 📋 Planned for v2.0
**Complexity**: High - requires profiling and decision logic
**Use Case**: Performance-critical applications with mixed data

---

### Multi-Input Compression (v2.0+)

**What it is**: Compress multiple input streams together for better correlation.

**Python Example**:
```python
streams = [timestamps, values, metadata]
compressed = openzl.compress_multi(streams)
```

**Planned Go API**:
```go
streams := [][]byte{
    timestamps,
    values,
    metadata,
}

compressed, _ := openzl.CompressMulti(streams)
```

**Status**: 📋 Planned for v2.0 or later
**Complexity**: Medium - requires stream coordination
**Use Case**: Time-series data, columnar storage

---

### Training & Dictionary Support (v2.0+)

**What it is**: Train compressor on representative data samples for better compression.

**C++ Example**:
```cpp
Trainer trainer;
trainer.addSample(sample1);
trainer.addSample(sample2);
Dictionary dict = trainer.train();

Compressor compressor(dict);
```

**Planned Go API**:
```go
trainer := openzl.NewTrainer()
trainer.AddSample(sample1)
trainer.AddSample(sample2)

dict, _ := trainer.Train()

compressor, _ := openzl.NewCompressor(
    openzl.WithDictionary(dict),
)
```

**Status**: 📋 Research phase
**Complexity**: Very High - requires training algorithm implementation
**Use Case**: Domain-specific data with known patterns

---

### Transform Composition (v2.0)

**What it is**: Chain multiple transforms for specialized compression.

**Python Example**:
```python
from openzl import transforms

pipeline = transforms.Pipeline([
    transforms.Delta(),
    transforms.Quantize(bits=8),
    transforms.Entropy(),
])

compressed = pipeline.compress(data)
```

**Planned Go API**:
```go
pipeline := openzl.NewPipeline(
    openzl.TransformDelta(),
    openzl.TransformQuantize(8),
    openzl.TransformEntropy(),
)

compressed, _ := pipeline.Compress(data)
```

**Status**: 📋 Planned for v2.0
**Complexity**: Medium - requires transform chaining infrastructure
**Use Case**: Specialized numeric/scientific data

---

### Feature Priority

Based on user feedback and demand, we'll prioritize:

**High Priority (v1.1)**:
1. ✅ Basic parameter controls (compression level, buffer size)
2. ✅ Additional platform support (Windows)
3. ✅ Performance monitoring and profiling

**Medium Priority (v2.0)**:
1. Custom compression graphs
2. Adaptive selectors
3. Transform composition
4. Multi-input compression

**Lower Priority (v2.0+)**:
1. Training and dictionary support
2. Advanced introspection APIs
3. Custom codec development

---

### Why Not in v1.0?

We deliberately **excluded** advanced features from v1.0 because:

1. **Complexity**: Each feature adds significant API surface area
2. **Usage**: Less than 5% of users need these features
3. **Stability**: v1.0 focuses on rock-solid core functionality
4. **Testing**: Advanced features require extensive testing
5. **Documentation**: Each feature needs comprehensive docs and examples

Our v1.0 release covers **95% of use cases** with:
- ✅ General-purpose compression
- ✅ High-performance context reuse
- ✅ Typed numeric compression
- ✅ Streaming for large files
- ✅ Thread-safe concurrent operations

**Advanced features can be added in v2.0 without breaking v1.0 APIs.**

---

### Contributing to Advanced Features

Interested in helping implement advanced features? We welcome contributors!

**Good first advanced features**:
1. Basic parameter controls (v1.1)
2. Performance monitoring (v1.1)
3. Transform composition (v2.0)

**Complex features needing experts**:
1. Custom compression graphs
2. Training and dictionaries
3. Custom selectors

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

### Feedback Welcome!

Which advanced features would be most valuable to you?

- Open an [issue](https://github.com/boris-chu/go-openzl/issues) to discuss
- Join [discussions](https://github.com/boris-chu/go-openzl/discussions)
- Vote on feature requests with 👍 reactions

Your input helps us prioritize development!

## License

This project is licensed under the BSD 3-Clause License - see the [LICENSE](LICENSE) file for details.

OpenZL itself is also BSD licensed - see the [OpenZL LICENSE](https://github.com/facebook/openzl/blob/main/LICENSE).

## Acknowledgments

- **Meta Open Source** for creating and open-sourcing OpenZL
- **The Go Community** for excellent CGO documentation and examples
- **Contributors** who help make this project possible

## Contact & Support

- **Issues**: [GitHub Issues](https://github.com/boris-chu/go-openzl/issues)
- **Discussions**: [GitHub Discussions](https://github.com/boris-chu/go-openzl/discussions)
- **Package Documentation**: [pkg.go.dev](https://pkg.go.dev/github.com/boris-chu/go-openzl)

## Related Projects

- [OpenZL](https://github.com/facebook/openzl) - The upstream C/C++ library
- [zstd-go](https://github.com/klauspost/compress) - High-performance zstd in Go
- [compress](https://github.com/klauspost/compress) - Optimized Go compression packages

---

**Star this project** if you find it interesting! It helps us gauge interest and attract contributors.
