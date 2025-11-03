# Test Organization

This document explains the test structure for go-openzl (v0.3.3).

## Test Levels

### 1. Public API Tests (Root Level)

**Location**: `*_test.go` in project root
**Purpose**: Test the CGO-based public API  
**Implementation**: Uses CGO bindings to OpenZL C library

**Files**:
- `simple_test.go` - Basic compress/decompress
- `compressor_test.go` - Compressor context API
- `typed_test.go` - Typed compression
- `stream_test.go` - Streaming API
- `benchmark_test.go` - Performance baselines
- `benchmark_comparison_test.go` - vs gzip, zstd
- `edge_case_test.go` - Edge cases, large files
- `fuzz_test.go` - Fuzz testing
- `klaus_post_improvements_test.go` - Optimizations

**Status**: ✅ Active - CGO API validation

### 2. Pure Go API Tests (purgo/)

**Location**: `purgo/*_test.go`
**Purpose**: Test Pure Go compression/decompression (v0.3.3)
**Implementation**: Pure Go, zero CGO

**Files**:
- `compress_smart_test.go` - CompressSmart with multi-stage pipelines
- `compress_test.go` - Basic compression
- `decoder_test.go` - Decompression  
- `reader_test.go` - Streaming reader
- `writer_test.go` - Streaming writer
- `analyzer_test.go` - Codec detection

**Key Tests**:
- ✅ CompressSmart: 27.64× on JSON, 35.25× on repeated text
- ✅ Multi-stage pipelines (LZ77→Huffman Frame v22)
- ✅ 280+ tests (100% passing)

**Status**: ✅ Active - Production ready (v0.3.3)

### 3. Frame Format Tests (internal/frame/)

**Files**:
- `reader_test.go` - Frame v21/v22 parsing
- `writer_test.go` - Frame v21/v22 writing (v0.3.3)
- `validation_test.go` - Format validation
- `property_test.go` - Property-based testing
- `fuzz_test.go` - Fuzzing

**Key Features**:
- ✅ Frame v22 with NodeSizes (v0.3.3)
- ✅ 7 writer tests, 79 parser tests
- ✅ 8.2M fuzz executions

**Status**: ✅ Active - Frame v22 complete

### 4. Codec Tests (internal/codec/)

**Files** (all 10 codecs):
- `identity_test.go`, `constant_test.go`, `delta_test.go`
- `zigzag_test.go`, `bitpack_test.go`, `transpose_test.go`  
- `rle_test.go`, `lz77_test.go`
- `huffman_test.go`, `fse_test.go`

**Status**: ✅ 181 tests - All codecs complete

### 5. Graph Executor Tests (internal/graph/)

**Files**:
- `graph_test.go` - Graph structures
- `executor_test.go` - Graph execution  
- `parser_test.go` - Graph parsing
- `integration_test.go` - End-to-end pipelines

**Key Features**:
- ✅ Reverse execution for decompression (v0.3.3)
- ✅ Multi-stage pipeline support
- ✅ 42 executor tests

**Status**: ✅ Active - Multi-stage pipelines working

## Version Timeline

- **v0.1.0**: CGO bindings ✅
- **v0.2.0**: Pure Go decompression ✅
- **v0.3.0-v0.3.2**: Pure Go compression ✅
- **v0.3.3**: Frame v22 & multi-stage pipelines ✅

## Test Execution

```bash
# All tests
go test ./...

# CGO API only
go test -v .

# Pure Go only
go test -v ./purgo/...

# Frame tests
go test -v ./internal/frame/...

# Codec tests
go test -v ./internal/codec/...

# Benchmarks
go test -bench=. -benchtime=3s

# Fuzzing
go test -fuzz=FuzzParse -fuzztime=30s ./internal/frame/...
```

## Test Coverage (v0.3.3)

- CGO API: ✅ 9 test files
- Pure Go API: ✅ 280+ tests
- Frame Parser: ✅ 79 tests + fuzzing
- Codecs: ✅ 181 tests (10 codecs)
- Graph: ✅ 42 tests
- **Overall**: >80% line coverage ✅

## Contributing Tests

1. **CGO API** → `*_test.go` (root)
2. **Pure Go API** → `purgo/*_test.go`
3. **Frame format** → `internal/frame/*_test.go`
4. **Codecs** → `internal/codec/*_test.go`
5. **Graph** → `internal/graph/*_test.go`

---

**Last Updated**: November 2, 2025  
**Version**: v0.3.3 (Frame v22)
**Status**: 280+ tests passing ✅
