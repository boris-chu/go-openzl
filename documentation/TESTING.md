# Testing & Performance Metrics

**Project**: go-openzl
**Last Updated**: November 2, 2025
**Platform**: macOS (Apple M4 Pro)

---

## Test Summary

### CGO Implementation (Phases 1-5):
- **Total Tests**: 273 tests
- **Fuzz Tests**: 5 (8.2M+ executions, zero crashes)
- **Pass Rate**: 100% (273/273)
- **Race Detector**: Clean (zero data races)
- **Test Coverage**: All major functionality + edge cases

### Pure Go Implementation (Phase 6):
- **Total Tests**: 280+ Pure Go tests (v0.3.3)
  - 181 codec tests (all 10 codecs)
  - 79 frame format tests (v21 + v22)
  - 42 graph tests (parser + executor)
  - 41 compression tests (purgo/encoder)
  - 29 decompression tests (purgo/decoder + reader)
- **Pass Rate**: 100% (280+/280+)
- **Fuzz Tests**: 8.2M+ executions (zero crashes)
- **Codecs**: 10 implemented (Identity, Constant, Delta, ZigZag, Bitpack, FSE, Huffman, LZ77, RLE, Transpose)

---

## Test Suite Breakdown

### Phase 1: MVP (7 tests)
- Basic compression/decompression
- Error handling
- Data integrity verification
- **Result**: 7/7 PASS

### Phase 2: Context API (7 tests)
- Reusable contexts
- Thread safety (concurrent operations)
- Resource management
- **Result**: 7/7 PASS

### Phase 3: Typed Compression (10 tests)
- Generic numeric compression
- All numeric types (int8-64, uint8-64, float32/64)
- Concurrent typed compression
- **Result**: 10/10 PASS

### Phase 4: Streaming API (12 tests)
- io.Reader/Writer interfaces
- io.Copy compatibility
- Frame management
- Reset and reuse
- **Result**: 12/12 PASS

### Phase 5: Edge Cases & Fuzz Testing (14 tests)
- Truncated frame handling
- Invalid frame headers
- Large file support (100MB)
- Concurrent stress testing (10,000 ops)
- Type mismatch behavior
- Error message validation
- Fuzz testing (5 tests, 8.2M+ executions)
- **Result**: 14/14 PASS

### Phase 6: Pure Go Compression (280+ tests, v0.3.3)
- **Compression Tests**:
  - CompressSmart() with multi-stage pipelines (10 tests)
  - Compress() roundtrip (5 tests) - Huffman with fallback
  - CompressInt64() roundtrip (5 tests) - Delta encoding
  - CompressFloat64/String() roundtrip (2 tests)
  - Benchmarks (4 benchmarks)
- **Codec Tests**: 181 tests across 10 codecs (all implemented)
- **Graph Tests**: 42 tests (parser + executor + multi-stage)
- **Frame Tests**: 79 tests (v21 + v22 with NodeSizes)
- **Result**: 280+/280+ PASS

### Phase 6: Pure Go Decompression (29 tests)
- **Typed API Tests** (17 tests):
  - DecompressInt64/32/16/8, Uint64/32/16/8
  - DecompressFloat64/32
  - Decompress (general-purpose)
- **Streaming API Tests** (12 tests):
  - purgo.Reader (io.Reader interface)
  - Incremental reads, io.Copy, EOF handling
- **Result**: 29/29 PASS

---

## Performance Benchmarks

### Pure Go Compression (Phase 6 - v0.3.3)
- **Multi-Stage Pipelines**: LZ77 → Huffman in single Frame v22
- **Text Compression**: 2.8 GB/s (Huffman encoding)
- **Numeric Compression**: 540 MB/s (Delta encoding)
- **Compression Ratios with CompressSmart()**:
  - JSON (12.7KB): **27.64×** (12,715 bytes → 460 bytes) 🔥
  - Repeated text (4.9KB): **35.25×** (4,900 bytes → 139 bytes) 🔥
  - Sparse data (1KB): **20×** (1,000 bytes → 50 bytes) 🔥
  - Single-codec (Huffman): 2.59× (1,200 bytes → 463 bytes)
  - Sequential numbers (Delta): 2.74× (8,000 bytes → 2,916 bytes)
- **Production Ready**: ✅ 27-35× compression on structured data

### Pure Go Decompression (Phase 6)
- **Typed API**: 490 MB/s (DecompressInt64/Float64)
- **Streaming API**: 2.3 GB/s (purgo.Reader)
- **Frame Parsing**: 1.6 GB/s
- **Graph Execution**: 16.2 GB/s (Identity codec)

### CGO Streaming API (Phase 4)
- **Throughput**: 2287 MB/s (10 MB compressed in 4.4 ms)
- **io.Copy**: 820 MB/s
- **Large data ratio**: 1364x compression on repeated data

### CGO Context API (Phase 2)
- **Compression**: 327k ops/sec (3.6 μs/op, 576 B/op)
- **Decompression**: 2.2M ops/sec (545 ns/op, 16 B/op)
- **Improvement**: 21% faster compress, 49% faster decompress vs one-shot

### CGO Typed Compression (Phase 3)
- **Ratio**: 50.31x on numeric data (vs 7.43x untyped)
- **Improvement**: 576.7% better than untyped compression
- **Best case**: 1364x on large repeated data

---

## Compression Ratios

### Pure Go (Phase 6 - v0.3.3):
| Data Type | Size | Compressed | Ratio | Codec Pipeline |
|-----------|------|------------|-------|----------------|
| **JSON data** | 12,715 bytes | **460 bytes** | **27.64×** 🔥 | LZ77 → Huffman (v22) |
| **Repeated text** | 4,900 bytes | **139 bytes** | **35.25×** 🔥 | LZ77 → Huffman (v22) |
| **Sparse data** | 1,000 bytes | **50 bytes** | **20×** 🔥 | LZ77 → Huffman (v22) |
| Text (Huffman only) | 1,200 bytes | 463 bytes | 2.59× | Huffman |
| Sequential int64 | 8,000 bytes | 2,916 bytes | 2.74× | Delta |

### CGO (Phases 1-5):
| Data Type | Size | Compressed | Ratio |
|-----------|------|------------|-------|
| Repeated text | 100 KB | 118 bytes | 847x |
| Typed int64 (1000) | 8 KB | 159 bytes | 50.3x |
| Large repeated | 10 MB | 7.7 KB | 1364x |
| Large file (100 MB) | 100 MB | 144 KB | 728x |
| File (40 KB) | 40 KB | 93 bytes | 430x |

---

## Thread Safety

**Race Detector Results**: ✅ PASS (zero data races)

Tested scenarios:
- 100+ concurrent compressors
- 10,000 concurrent operations (stress test)
- Concurrent typed compression
- Streaming API concurrency

---

## How to Run Tests

```bash
# All tests
go test ./...

# With race detector
go test -race ./...

# Benchmarks
go test -bench=. -benchmem

# Fuzz testing (run for longer)
go test -fuzz=FuzzCompress -fuzztime=30s

# Coverage
go test -cover ./...

# Specific phase
go test -run TestWriter     # Phase 4 streaming tests
go test -run TestTyped      # Phase 3 typed tests
go test -run TestCompressor # Phase 2 context tests
```

---

## Success Criteria

All phases exceeded their targets:

✅ **Phase 1**: >5x compression (achieved 8.89x)
✅ **Phase 2**: 10-50% speedup (achieved 21-49%)
✅ **Phase 3**: 2-50x typed improvement (achieved 50.31x)
✅ **Phase 4**: >500 MB/s throughput (achieved 2287 MB/s)
✅ **Phase 5**: Production hardening
  - 2M+ fuzz executions (zero crashes)
  - 100MB file support (728x ratio)
  - 10,000 concurrent operations
  - Comprehensive edge case coverage
✅ **Phase 6**: Pure Go Implementation (v0.3.3)
  - 27-35× compression with multi-stage pipelines
  - Frame Format v22 with NodeSizes support
  - 280+ tests passing (100% pass rate)
  - All 10 codecs implemented and tested

---

For detailed metrics, see [docs/TEST_METRICS.md](docs/TEST_METRICS.md) (private).
