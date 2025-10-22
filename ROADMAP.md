# go-openzl Roadmap

## Project Vision

Create idiomatic, high-performance Go bindings for Meta's OpenZL format-aware compression library, making OpenZL accessible to the Go ecosystem with excellent performance and developer experience.

## Current Status

- ✅ **Phase 0**: Foundation complete
- ✅ **Phase 1**: MVP complete - Working compression/decompression
- ⏳ **Phase 2**: Context API (next)
- ⏳ **Phase 3**: Typed compression
- ⏳ **Phase 4**: Streaming API
- ⏳ **Phase 5**: Production hardening

---

## Phase 1: Minimum Viable Product ✅ COMPLETE

**Timeline**: Completed Q4 2025
**Goal**: Working prototype with basic compression/decompression

### Delivered Features
- ✅ Simple `Compress()` and `Decompress()` functions
- ✅ Internal CGO bindings to OpenZL C API
- ✅ Comprehensive error handling
- ✅ Unit tests (100% passing)
- ✅ Benchmarks (187k compress ops/sec, 773k decompress ops/sec)
- ✅ Example program demonstrating usage
- ✅ 8.89:1 compression ratio on repeated data

### Test Coverage
```
✓ Text compression
✓ Binary data handling
✓ Repeated data (8.89:1 ratio)
✓ Empty input error handling
✓ Corrupted data detection
```

**See**: [Phase 1 Complete Report](docs/PHASE_1_COMPLETE.md)

---

## Phase 2: Context-Based API

**Timeline**: Q1 2026
**Status**: Planned
**Goal**: Reusable contexts for better performance and control

### Planned Features

#### Reusable Compression Contexts
```go
compressor, err := openzl.NewCompressor()
defer compressor.Close()

// Reuse context for multiple operations (10x+ faster)
for _, data := range inputs {
    compressed, err := compressor.Compress(data)
    // ...
}
```

#### Thread Safety
- Concurrent-safe `Compressor` and `Decompressor` types
- Internal mutex protection
- Safe for use across multiple goroutines

#### Options Pattern
```go
compressor, err := openzl.NewCompressor(
    openzl.WithCompressionLevel(9),
    openzl.WithChecksum(true),
)
```

### Success Criteria
- [ ] 10-50% performance improvement vs one-shot API
- [ ] Thread-safe verified with race detector
- [ ] Configurable compression parameters
- [ ] Zero memory leaks under repeated use

---

## Phase 3: Typed Compression

**Timeline**: Q1-Q2 2026
**Status**: Planned
**Goal**: Format-aware compression for structured data

### Planned Features

#### Numeric Array Compression
```go
// 2-5x better compression for sorted/structured data
numbers := []int64{1, 2, 3, 4, 5, 100, 101, 102}
compressed, err := compressor.CompressNumeric(numbers)
```

Supported types:
- `[]int32`, `[]int64`
- `[]uint32`, `[]uint64`
- `[]float32`, `[]float64`

#### Struct Compression
```go
type LogEntry struct {
    Timestamp int64
    Level     uint8
    Message   string
}

logs := []LogEntry{...}
compressed, err := compressor.CompressStruct(logs)
```

#### Type Safety
- Compile-time type checking
- Reflection-based type discovery
- Clear error messages for type mismatches

### Success Criteria
- [ ] 2-5x better compression on sorted integers
- [ ] Works with all Go numeric types
- [ ] Type-safe API using Go generics
- [ ] Benchmark comparison vs untyped compression

---

## Phase 4: Streaming API

**Timeline**: Q2 2026
**Status**: Planned
**Goal**: Standard library integration with io.Reader/Writer

### Planned Features

#### Writer Interface
```go
file, _ := os.Create("output.zl")
writer := openzl.NewWriter(file)
defer writer.Close()

// Compress data as it's written
io.Copy(writer, sourceReader)
```

#### Reader Interface
```go
file, _ := os.Open("input.zl")
reader := openzl.NewReader(file)

// Decompress data as it's read
io.Copy(destWriter, reader)
```

#### Buffering
- Automatic buffer management
- Configurable buffer sizes
- Efficient streaming for large files

### Success Criteria
- [ ] Works seamlessly with `io.Copy`
- [ ] Proper EOF handling
- [ ] Performance comparable to stdlib compression
- [ ] Can compress/decompress files >100MB

---

## Phase 5: Production Ready

**Timeline**: Q2-Q3 2026
**Status**: Planned
**Goal**: Harden for production use, v1.0.0 release

### Testing & Quality
- [ ] Fuzz testing (1M+ inputs without crashes)
- [ ] Memory leak detection (valgrind/sanitizers)
- [ ] Edge case testing (nil, empty, huge inputs)
- [ ] Error path coverage
- [ ] Thread safety stress tests

### Performance
- [ ] Comprehensive benchmarks vs other libraries
- [ ] Memory profiling and optimization
- [ ] CPU profiling and optimization
- [ ] Performance regression tests

### Documentation
- [ ] Complete godoc for all exports (100% coverage)
- [ ] Migration guides from other compression libraries
- [ ] Cookbook with common patterns
- [ ] Performance tuning guide
- [ ] API stability guarantees

### Platform Support
- [ ] Linux (amd64, arm64)
- [ ] macOS (amd64, arm64)
- [ ] Windows (amd64)
- [ ] CI/CD for all platforms

### Release
- [ ] Semantic versioning policy
- [ ] Changelog
- [ ] GitHub releases with binaries
- [ ] v1.0.0 release candidate
- [ ] Community feedback period
- [ ] v1.0.0 stable release

---

## Future Possibilities (Post v1.0)

### Advanced Features
- Custom compression graphs
- Selector API for adaptive compression
- Profile-guided optimization
- Hardware acceleration (SIMD, GPU)

### Language Bindings
- Build examples for other languages using CGO artifacts
- Python bindings using cffi/ctypes
- Node.js bindings using node-gyp

### Ecosystem Integration
- gRPC integration
- HTTP middleware for compressed responses
- Database driver integration
- Cloud storage SDK integration

---

## How to Contribute

We welcome contributors at any phase! Here's how you can help:

### Current Needs (Phase 2)
- [ ] Context-based API implementation
- [ ] Thread safety testing
- [ ] Options pattern design
- [ ] Performance benchmarking

### General Contributions
- **Testing**: Add more test cases, fuzz tests
- **Documentation**: Examples, guides, tutorials
- **Performance**: Profiling, optimization
- **Platform support**: Test on different OS/architectures
- **Bug reports**: Issues, edge cases

### Getting Started
1. Read [CONTRIBUTING.md](CONTRIBUTING.md)
2. Check [GitHub Issues](https://github.com/borischu/go-openzl/issues) for tasks
3. Join discussions in Issues or Discussions
4. Submit PRs with improvements

---

## Performance Goals

### Phase 1 (Current)
- ✅ Compress: ~187k ops/sec (5.7μs/op)
- ✅ Decompress: ~773k ops/sec (1.6μs/op)

### Phase 2 (Target)
- 🎯 Compress: ~300k ops/sec (context reuse)
- 🎯 Decompress: ~1M ops/sec (context reuse)

### Phase 3 (Target)
- 🎯 Typed compression: 2-5x better ratio
- 🎯 Maintain same speed or better

### Phase 4 (Target)
- 🎯 Streaming: >500 MB/sec throughput
- 🎯 Low memory overhead (<10MB for large files)

---

## Timeline Summary

| Phase | Duration | Target | Key Deliverable |
|-------|----------|--------|----------------|
| Phase 0 | 1 week | Q4 2025 | Project setup ✅ |
| Phase 1 | 2 weeks | Q4 2025 | Working MVP ✅ |
| Phase 2 | 2 weeks | Q1 2026 | Context API |
| Phase 3 | 2 weeks | Q1-Q2 2026 | Typed compression |
| Phase 4 | 2 weeks | Q2 2026 | Streaming API |
| Phase 5 | 3 weeks | Q2-Q3 2026 | Production ready |
| **v1.0.0** | **12 weeks** | **Q3 2026** | **Stable Release** |

---

## Success Metrics

### Technical Metrics
- Test coverage: >90%
- Benchmark performance: within 10% of C library
- Zero known memory leaks
- Zero data races

### Community Metrics
- GitHub stars: >100
- Contributors: >5
- Production users: >3 companies
- Issues resolved: >90%

---

## Questions or Ideas?

- **Discussions**: [GitHub Discussions](https://github.com/borischu/go-openzl/discussions)
- **Issues**: [GitHub Issues](https://github.com/borischu/go-openzl/issues)
- **Contributing**: See [CONTRIBUTING.md](CONTRIBUTING.md)

**Star the project** to show your interest and help us attract more contributors!

---

Last updated: October 2025 (Phase 1 Complete)
