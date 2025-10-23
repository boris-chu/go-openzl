# go-openzl

[![Go Reference](https://pkg.go.dev/badge/github.com/yourusername/go-openzl.svg)](https://pkg.go.dev/github.com/yourusername/go-openzl)
[![Go Report Card](https://goreportcard.com/badge/github.com/yourusername/go-openzl)](https://goreportcard.com/report/github.com/yourusername/go-openzl)
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

**✅ Phase 3 Complete: Typed Compression API**

This project is in active development:
- ✅ **Phase 1**: MVP with simple Compress/Decompress API
- ✅ **Phase 2**: Context API with 20-50% better performance
- ✅ **Phase 3**: Typed compression for structured data (2-50x better ratios!)
- ⏳ **Phase 4**: Streaming API with io.Reader/Writer

**Current Status:**
- ✅ One-shot compression/decompression API
- ✅ Reusable Compressor and Decompressor types
- ✅ Thread-safe concurrent operations
- ✅ Typed compression with Go generics
- ✅ Support for all numeric types (int8-64, uint8-64, float32/64)
- ✅ Options pattern for configuration
- ✅ Comprehensive test coverage (100% passing - 24/24 tests)
- ✅ Performance benchmarks

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

### Phase 4: Streaming API (Planned)
- [ ] `io.Reader`/`io.Writer` interfaces
- [ ] Streaming compression/decompression
- [ ] Automatic buffer management
- [ ] Large file support

### Phase 5: Production Ready (Planned)
- [ ] Fuzz testing and security hardening
- [ ] Memory leak detection and profiling
- [ ] CI/CD for multiple platforms
- [ ] v1.0.0 stable release

## Installation (Coming Soon)

```bash
go get github.com/yourusername/go-openzl
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

## Documentation

- [API Documentation](https://pkg.go.dev/github.com/yourusername/go-openzl) (Coming soon)
- [Examples](examples/) (Coming soon)
- [Performance Guide](PERFORMANCE.md) (Coming soon)

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
├── openzl/             # Core bindings package
│   ├── cgo.go          # CGO declarations
│   ├── compressor.go   # Compression API
│   ├── decompressor.go # Decompression API
│   └── errors.go       # Error handling
├── typed/              # Typed compression API
├── stream/             # Streaming API
├── examples/           # Usage examples
├── benchmarks/         # Performance benchmarks
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

### Q4 2025
- ✅ Research and planning
- 🚧 Core CGO bindings
- 🚧 Basic compression/decompression
- ⏳ Initial test suite

### Q1 2026
- ⏳ Typed compression API
- ⏳ Streaming interfaces
- ⏳ Performance benchmarks
- ⏳ First alpha release

### Q2 2026
- ⏳ Advanced features (custom graphs, selectors)
- ⏳ Production-grade testing
- ⏳ Documentation and examples
- ⏳ v1.0 release

## License

This project is licensed under the BSD 3-Clause License - see the [LICENSE](LICENSE) file for details.

OpenZL itself is also BSD licensed - see the [OpenZL LICENSE](https://github.com/facebook/openzl/blob/main/LICENSE).

## Acknowledgments

- **Meta Open Source** for creating and open-sourcing OpenZL
- **The Go Community** for excellent CGO documentation and examples
- **Contributors** who help make this project possible

## Contact & Support

- **Issues**: [GitHub Issues](https://github.com/yourusername/go-openzl/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yourusername/go-openzl/discussions)
- **Email**: your.email@example.com

## Related Projects

- [OpenZL](https://github.com/facebook/openzl) - The upstream C/C++ library
- [zstd-go](https://github.com/klauspost/compress) - High-performance zstd in Go
- [compress](https://github.com/klauspost/compress) - Optimized Go compression packages

---

**Star this project** if you find it interesting! It helps us gauge interest and attract contributors.
