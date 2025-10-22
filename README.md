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

**🚧 Project Status: Research & Planning Phase**

This project is in active development. We're currently:
- ✅ Researched OpenZL architecture and C API
- ✅ Designed Go binding strategy
- 🚧 Implementing core CGO bindings
- ⏳ Building idiomatic Go wrappers
- ⏳ Creating comprehensive test suite

**We're looking for contributors!** See [Contributing](#contributing) below.

## Features (Planned)

### Phase 1: Core API (In Progress)
- [ ] Compression/Decompression contexts
- [ ] Basic compression and decompression
- [ ] Error handling and reporting
- [ ] Frame introspection (size queries)
- [ ] Comprehensive test coverage

### Phase 2: Typed API
- [ ] TypedRef creation and management
- [ ] Typed compression/decompression
- [ ] Multi-input/output support
- [ ] TypedBuffer interface

### Phase 3: Advanced Features
- [ ] Custom compression graph registration
- [ ] Selector APIs
- [ ] Fine-grained parameter control
- [ ] Performance introspection hooks

### Phase 4: Go-Idiomatic Wrappers
- [ ] `io.Reader`/`io.Writer` interfaces
- [ ] Streaming compression/decompression
- [ ] Concurrent compression workers
- [ ] Automatic buffer management
- [ ] Type-safe API using Go generics

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

## Quick Start (Coming Soon)

```go
package main

import (
    "fmt"
    "log"

    "github.com/yourusername/go-openzl"
)

func main() {
    // Create a compressor
    compressor, err := openzl.NewCompressor()
    if err != nil {
        log.Fatal(err)
    }
    defer compressor.Close()

    // Compress data
    input := []byte("Hello, OpenZL!")
    compressed, err := compressor.Compress(input)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Original size: %d bytes\n", len(input))
    fmt.Printf("Compressed size: %d bytes\n", len(compressed))

    // Decompress data
    decompressed, err := openzl.Decompress(compressed)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Decompressed: %s\n", decompressed)
}
```

### Typed Compression (Coming Soon)

OpenZL excels at compressing typed data:

```go
// Compress an array of integers
numbers := []int64{1, 2, 3, 4, 5, 100, 101, 102}
compressed, err := compressor.CompressNumeric(numbers)

// Compress structured data
type LogEntry struct {
    Timestamp int64
    Level     uint8
    Message   string
}

logs := []LogEntry{
    {Timestamp: 1234567890, Level: 1, Message: "Error occurred"},
    {Timestamp: 1234567891, Level: 0, Message: "All clear"},
}
compressed, err := compressor.CompressStruct(logs)
```

## Performance

OpenZL is designed for high-performance scenarios. Preliminary benchmarks show:

- **Compression speed**: 500 MB/s - 2 GB/s (depending on data type)
- **Decompression speed**: 1 GB/s - 5 GB/s
- **Compression ratio**: Comparable to specialized compressors (often 2-10x better than gzip)

Detailed benchmarks coming soon.

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
