# Test Organization

This document explains the test structure for go-openzl.

## Test Levels

### 1. Public API Tests (Root Level)

**Location**: `*.go` in project root
**Purpose**: Test the public API that users interact with
**Implementation**: Currently uses CGO bindings to OpenZL C library

**Files**:
- `simple_test.go` - Basic compress/decompress roundtrip
- `compressor_test.go` - Compressor context API
- `typed_test.go` - Typed compression (CompressNumeric)
- `stream_test.go` - Streaming API (Reader/Writer)
- `benchmark_test.go` - Performance baselines
- `benchmark_comparison_test.go` - vs gzip, zstd, etc.
- `edge_case_test.go` - Edge cases, large files, stress tests
- `fuzz_test.go` - Fuzz testing of public API
- `klaus_post_improvements_test.go` - Optimization patterns

**Why at root level?**
- Tests the user-facing API
- Validates CGO implementation (current)
- Will validate Pure Go implementation (future)
- Integration tests for entire stack

**Status**: ✅ Active - These tests are critical and should be maintained

### 2. Pure Go Implementation Tests (internal/)

**Location**: `internal/frame/*_test.go`
**Purpose**: Test Pure Go implementation internals
**Implementation**: Pure Go, zero CGO

**Files**:
- `internal/frame/reader_test.go` - Frame parsing
- `internal/frame/validation_test.go` - Format validation (22 tests)
- `internal/frame/property_test.go` - Property-based testing (9 tests)
- `internal/frame/fuzz_test.go` - Fuzzing frame parser (8.2M executions)

**Why in internal/?**
- Tests implementation details
- Pure Go only (no CGO)
- Low-level format verification
- Not part of public API

**Status**: ✅ Active - Phase 1 complete

### 3. Future: Codec Tests (internal/codec/)

**Location**: `internal/codec/*_test.go` (Phase 2)
**Purpose**: Test individual codec implementations
**Implementation**: Pure Go

**Planned structure**:
```
internal/codec/
├── identity/
│   ├── identity.go
│   └── identity_test.go
├── delta/
│   ├── delta.go
│   └── delta_test.go
├── zigzag/
│   ├── zigzag.go
│   └── zigzag_test.go
└── entropy/
    ├── fse.go
    ├── fse_test.go
    ├── huffman.go
    └── huffman_test.go
```

**Status**: 📝 Planned - Phase 2

## Test Strategy by Phase

### Phase 1: Frame Parser (Complete ✅)

**Goal**: Parse OpenZL frames without decompression

**Tests**:
- ✅ internal/frame/reader_test.go - Basic parsing
- ✅ internal/frame/validation_test.go - Format validation
- ✅ internal/frame/property_test.go - Property-based
- ✅ internal/frame/fuzz_test.go - Robustness

**Coverage**: 79 tests, 8.2M fuzz executions

### Phase 2: Simple Codecs (Next)

**Goal**: Implement basic codecs, enable decompression

**Tests planned**:
- internal/codec/identity/identity_test.go
- internal/codec/delta/delta_test.go
- Root tests validate end-to-end works

**Integration**: Root tests (simple_test.go, etc.) will pass when Pure Go is complete

### Phase 3: Complex Codecs

**Goal**: Entropy coding (FSE, Huffman)

**Tests planned**:
- internal/codec/entropy/*_test.go
- Property tests for codec correctness
- Benchmark vs klauspost/compress

### Phase 4: Production Hardening

**Goal**: Replace CGO with Pure Go in public API

**Tests**:
- ✅ Root tests ALREADY EXIST
- Just swap backend from CGO to Pure Go
- All existing tests should pass
- Compare benchmarks (CGO vs Pure Go)

## When to Archive Tests?

**Archive if**:
- Test is for removed feature
- Test is superseded by better test
- Test is obsolete (API changed)

**Keep if**:
- Tests public API (even if backend changes)
- Tests critical functionality
- Provides performance baseline
- Edge case coverage

## Current Recommendation: KEEP ALL

**All 9 root test files should be kept** because:

1. ✅ Test public API users depend on
2. ✅ Provide CGO baseline for Pure Go comparison
3. ✅ Will validate Pure Go when ready
4. ✅ Edge case coverage is valuable
5. ✅ Benchmarks track performance over time

**No files need archiving** at this time.

## Test Execution

### Run all tests
```bash
go test ./...
```

### Run public API tests only
```bash
go test -v .
```

### Run Pure Go frame parser tests only
```bash
go test -v ./internal/frame/...
```

### Run benchmarks
```bash
go test -bench=. -benchtime=3s
```

### Run fuzzing
```bash
# Fuzz public API
go test -fuzz=FuzzCompress -fuzztime=30s

# Fuzz frame parser
go test -fuzz=FuzzParse -fuzztime=30s ./internal/frame/...
```

## Test Coverage Goals

### Current Status
- Public API: ✅ Well covered (9 test files)
- Frame Parser: ✅ Excellent (79 tests, fuzzing)
- Codecs: ⏳ Phase 2 (not started)
- Graph System: ⏳ Phase 4 (not started)

### Target Coverage
- Overall: >80% line coverage
- Critical paths: >95% coverage
- All codecs: Property tests + fuzzing
- Public API: Integration tests + edge cases

## Contributing Tests

When adding tests:

1. **Public API changes** → Add to root `*_test.go`
2. **Pure Go internals** → Add to `internal/*/\*_test.go`
3. **New codec** → Create `internal/codec/<name>/<name>_test.go`
4. **Performance** → Add benchmarks
5. **Edge cases** → Add to `edge_case_test.go` or specific test file
6. **Fuzzing** → Add fuzz function, run for 1M+ executions

## Questions?

- Public API vs Internal: Ask "Would a user call this?"
  - Yes → Root level test
  - No → Internal test

- Where to add test: Ask "What am I testing?"
  - User API → Root
  - Frame format → internal/frame
  - Codec logic → internal/codec
  - Graph system → internal/graph

---

**Last Updated**: November 1, 2025
**Status**: Phase 1 complete, Phase 2 planned
