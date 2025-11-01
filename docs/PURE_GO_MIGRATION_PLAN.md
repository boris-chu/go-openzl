# OpenZL Pure Go Migration Plan

**Mission**: Migrate from CGO bindings to a pure Go implementation of Meta's OpenZL compression framework.

**Why**: Eliminate CGO dependency for better cross-compilation, faster debugging, easier deployment, and full control over optimizations.

**Challenge**: Port 370,000 lines of C/C++17 code (592 C files, 651 headers, 128 codecs) to idiomatic Go.

**Timeline**: 12-18 months (phased approach, production-ready increments)

---

## Architecture Analysis

### Current State (v0.1.0)
```
Go API Layer (2,000 lines)
    ↓
CGO Bindings (500 lines)
    ↓
Meta's OpenZL C Library (370,000 lines)
    ├── Core Compression Engine
    ├── 128 Codecs (delta, entropy, bitpack, etc.)
    ├── Graph System (DAG-based compression)
    ├── Frame Format & Serialization
    └── Universal Decompressor
```

### Target State (v3.0.0 - Pure Go)
```
Go API Layer (idiomatic, same as current)
    ↓
Pure Go OpenZL Implementation (est. 100,000-150,000 lines)
    ├── Core Engine (Go native)
    ├── Codecs (Go implementations)
    ├── Graph System (Go DAG)
    ├── Frame Format (Go encoding)
    └── Universal Decompressor (Go)
```

### OpenZL Core Components

**From C library analysis:**

1. **Compression Engine** (`src/openzl/compress/`)
   - `cctx.c` - Compression context
   - `cgraph.c` - Compression graph
   - `segmenter.c` - Data segmentation
   - `selector.c` - Codec selection
   - Lines: ~15,000

2. **Decompression Engine** (`src/openzl/decompress/`)
   - Universal decompressor
   - Frame parsing
   - Graph execution
   - Lines: ~10,000

3. **Codecs** (`src/openzl/codecs/`, 128 codec implementations)
   - `delta/` - Delta encoding
   - `entropy/` - Entropy coding
   - `bitpack/` - Bit packing
   - `zigzag/` - ZigZag encoding
   - `transpose/` - Transpose operations
   - `quantize/` - Quantization
   - `zstd/` - Zstd integration
   - `rolz/` - ROLZ compression
   - `float_deconstruct/` - Float encoding
   - Lines: ~26,500 (codec implementations only)

4. **Frame Format** (`src/openzl/frame/`)
   - Binary serialization
   - Metadata encoding
   - Lines: ~5,000

5. **Graph System** (`src/openzl/compress/graphs/`)
   - DAG construction
   - Node composition
   - Transform pipelines
   - Lines: ~8,000

6. **Support Infrastructure**
   - Error handling
   - Memory management
   - Portability layer
   - Introspection
   - Lines: ~10,000

**Total Core**: ~75,000 lines (excluding benchmarks, tests, tools)

---

## Phased Migration Strategy

### Phase 0: Foundation & Research (2 months)
**Goal**: Deep understanding and infrastructure setup

#### Tasks:
1. **Study OpenZL Algorithm** (3 weeks)
   - Read whitepaper thoroughly
   - Document compression graph model
   - Understand codec composition
   - Map data flow through system

2. **Analyze C Codebase** (3 weeks)
   - Create dependency graph
   - Identify critical paths
   - Document codec interfaces
   - Find reusable Go libraries

3. **Setup Development Infrastructure** (2 weeks)
   - Dual build system (CGO + Pure Go)
   - Compatibility test suite
   - Performance benchmarking framework
   - CI/CD for both implementations

#### Deliverables:
- [x] Architecture documentation
- [x] Component dependency map
- [ ] Codec interface specifications
- [ ] Development workflow
- [ ] Test framework

**Effort**: 1 engineer × 2 months

---

### Phase 1: Frame Format & Scaffolding (2 months)
**Goal**: Pure Go frame reading/writing, no compression yet

#### What to Build:
1. **Frame Format Parser** (Go)
   - Binary frame structure
   - Metadata parsing
   - Header/footer handling
   - Checksum validation

2. **Basic Scaffolding**
   - Context types (CCtx, DCtx)
   - Error types and handling
   - Buffer management
   - Memory pooling (sync.Pool)

3. **Compatibility Layer**
   - Can read frames created by C library
   - Can write frames readable by C library
   - Pass-through compression (use C for compression, Go for framing)

#### Success Metrics:
- Parse all valid OpenZL frames
- Write frames compatible with C decompressor
- 100% frame format compatibility tests pass

**Effort**: 1 engineer × 2 months
**Lines of Code**: ~5,000 Go

---

### Phase 2: Simple Codecs (3 months)
**Goal**: Implement simplest codecs in pure Go

#### Priority Codecs (Easiest First):
1. **Identity Codec** (passthrough) - 1 day
2. **Constant Codec** - 3 days
3. **Delta Codec** - 1 week
4. **ZigZag Codec** - 1 week
5. **Bitpack Codec** - 2 weeks
6. **Transpose Codec** - 2 weeks
7. **Quantize Codec** - 2 weeks

#### Integration:
- Codec registry system
- Codec interface in Go
- Compression graph (simplified)
- Basic selector (choose codec)

#### Success Metrics:
- 7 codecs implemented and tested
- Can compress/decompress simple numeric arrays
- Performance within 2x of C implementation
- Full compatibility with C library frames

**Effort**: 1-2 engineers × 3 months
**Lines of Code**: ~15,000 Go

---

### Phase 3: Entropy Coding (3 months)
**Goal**: Implement entropy codecs (hardest part)

#### Codecs to Implement:
1. **FSE (Finite State Entropy)** - Can port from Klaus Post's library?
2. **Huffman Coding** - Can port from Klaus Post's library?
3. **ANS (Asymmetric Numeral Systems)**
4. **Range Coding**

#### Challenge:
These are complex, performance-critical algorithms. Options:
- **Option A**: Port from Klaus Post's `klauspost/compress` (FSE, Huffman exist!)
- **Option B**: Implement from scratch using OpenZL C code as reference
- **Option C**: Hybrid - use Klaus Post where available, implement rest

**Recommended**: Option A - leverage Klaus Post's battle-tested Go implementations

#### Success Metrics:
- All entropy codecs working
- Performance within 1.5x of C
- Can compress structured data

**Effort**: 2 engineers × 3 months
**Lines of Code**: ~20,000 Go (or less if using Klaus Post libs)

---

### Phase 4: Graph System (2 months)
**Goal**: Full compression graph DAG system

#### What to Build:
1. **Graph Construction**
   - Node composition
   - Edge connections
   - Type checking

2. **Graph Execution**
   - Forward pass (compression)
   - Backward pass (decompression)
   - Streaming support

3. **Graph Serialization**
   - Encode graph to frame
   - Decode graph from frame
   - Graph compatibility

#### Success Metrics:
- Multi-codec graphs work
- Can execute arbitrary codec DAGs
- Full compatibility with C graph format

**Effort**: 1-2 engineers × 2 months
**Lines of Code**: ~10,000 Go

---

### Phase 5: Advanced Codecs (4 months)
**Goal**: Implement remaining complex codecs

#### Codecs:
1. **LZ (Lempel-Ziv)** - 4 weeks
2. **ROLZ (Reduced Offset LZ)** - 4 weeks
3. **Float Deconstruct** - 3 weeks
4. **Dispatch by Tag** - 3 weeks
5. **Merge Sorted** - 2 weeks
6. **Range Pack** - 2 weeks
7. **Flatpack** - 2 weeks

#### Integration with Zstd:
- Use Klaus Post's `zstd` package
- Wrap as OpenZL codec
- Ensure compatibility

#### Success Metrics:
- All major codecs implemented
- Full feature parity with C library
- Performance within 1.2x of C

**Effort**: 2-3 engineers × 4 months
**Lines of Code**: ~30,000 Go

---

### Phase 6: Optimization & Production (3 months)
**Goal**: Make it faster than C implementation

#### Optimizations:
1. **Profiling**
   - CPU profiling (pprof)
   - Memory profiling
   - Identify hot paths

2. **Go-Specific Optimizations**
   - Buffer pooling (sync.Pool)
   - Avoid allocations in hot paths
   - Use unsafe where necessary
   - Assembly for critical loops (AMD64, ARM64)

3. **Concurrency**
   - Parallel codec execution
   - Goroutine-based streaming
   - Lock-free data structures

4. **SIMD**
   - Use `golang.org/x/sys/cpu` for feature detection
   - Assembly SIMD for codecs (AVX2, NEON)
   - Port Klaus Post's SIMD techniques

#### Klaus Post Patterns to Apply:
- **Buffer Pooling**: Reuse buffers via sync.Pool
- **Inline-Friendly**: Small functions for compiler inlining
- **Reduced Allocations**: Pre-allocate, reuse slices
- **Stateless Mode**: Option for low memory (many concurrent compressors)

#### Success Metrics:
- **Compression**: Match or beat C performance
- **Decompression**: 1.2x faster than C (Go typically excels here)
- **Memory**: Lower allocations than C
- **Concurrency**: 4x faster on 8-core machines

**Effort**: 2 engineers × 3 months
**Lines of Code**: ~5,000 Go (optimizations)

---

### Phase 7: Complete Feature Parity (2 months)
**Goal**: Everything the C library can do

#### Features:
1. **Selector System**
   - Adaptive codec selection
   - ML-based selection
   - Custom selectors

2. **Segmenter System**
   - Data segmentation strategies
   - Adaptive segmentation

3. **Custom Transforms**
   - User-defined codecs
   - Plugin system

4. **Advanced Parameters**
   - Compression levels
   - Checksum options
   - Format versions

5. **Streaming API**
   - Streaming compression
   - Streaming decompression
   - Frame chunking

#### Success Metrics:
- 100% API coverage
- All C library features available
- Full documentation

**Effort**: 1-2 engineers × 2 months
**Lines of Code**: ~10,000 Go

---

## Migration Strategy: Dual Implementation

### Hybrid Approach (Recommended)

During migration, maintain **both implementations**:

```
go-openzl/
├── README.md
├── go.mod
├── openzl.go              # Public API (stable)
│
├── cgo/                   # CGO implementation (current)
│   ├── compressor.go
│   ├── decompressor.go
│   └── internal/cgo/
│
├── pure/                  # Pure Go implementation (new)
│   ├── compressor.go
│   ├── decompressor.go
│   ├── frame/            # Frame format
│   ├── codecs/           # Codec implementations
│   │   ├── delta/
│   │   ├── entropy/
│   │   ├── bitpack/
│   │   └── ...
│   ├── graph/            # Graph system
│   └── internal/
│
└── build_tags.go         # Build tag selection
```

### Build Tags

Users choose implementation at build time:

```go
// build_tags.go

//go:build !purego
// +build !purego

package openzl

import "github.com/borischu/go-openzl/cgo"

type Compressor = cgo.Compressor
type Decompressor = cgo.Decompressor
```

```go
// build_tags_pure.go

//go:build purego
// +build purego

package openzl

import "github.com/borischu/go-openzl/pure"

type Compressor = pure.Compressor
type Decompressor = pure.Decompressor
```

**Usage:**

```bash
# Use CGO version (default, fastest during migration)
go build

# Use Pure Go version
go build -tags purego
```

### Testing Both Implementations

```go
// compatibility_test.go

func TestCGOPureCompatibility(t *testing.T) {
    data := []byte("test data")

    // Compress with CGO
    cgoComp, _ := cgo.NewCompressor()
    compressed, _ := cgoComp.Compress(data)

    // Decompress with Pure Go
    pureDecomp, _ := pure.NewDecompressor()
    decompressed, _ := pureDecomp.Decompress(compressed)

    assert.Equal(t, data, decompressed)
}
```

---

## Timeline Summary

| Phase | Duration | Cumulative | Description |
|-------|----------|------------|-------------|
| 0: Foundation | 2 months | 2 mo | Research, architecture, setup |
| 1: Frame Format | 2 months | 4 mo | Frame parsing, scaffolding |
| 2: Simple Codecs | 3 months | 7 mo | Delta, zigzag, bitpack, etc. |
| 3: Entropy Coding | 3 months | 10 mo | FSE, Huffman, ANS (use Klaus Post) |
| 4: Graph System | 2 months | 12 mo | Compression DAG |
| 5: Advanced Codecs | 4 months | 16 mo | LZ, ROLZ, float, etc. |
| 6: Optimization | 3 months | 19 mo | Performance tuning |
| 7: Feature Parity | 2 months | 21 mo | Complete features |

**Total**: 21 months (1.75 years)

**With 2 engineers**: 12-15 months (1-1.25 years)

---

## Effort Estimates

### By Lines of Code:
- **C Library Core**: ~75,000 lines
- **Estimated Pure Go**: ~100,000 lines (Go is more verbose)
- **Effective Productivity**: 100-200 LOC/day for complex compression code

### By Engineer-Months:
- **Phase 0**: 2 EM (research)
- **Phase 1**: 2 EM (framing)
- **Phase 2**: 4 EM (simple codecs)
- **Phase 3**: 6 EM (entropy)
- **Phase 4**: 3 EM (graph)
- **Phase 5**: 10 EM (advanced codecs)
- **Phase 6**: 6 EM (optimization)
- **Phase 7**: 3 EM (features)

**Total**: 36 engineer-months

**With 2 engineers**: 18 months
**With 1 engineer**: 36 months (not recommended)

---

## Risk Mitigation

### Technical Risks:

1. **Performance**: Pure Go slower than C
   - **Mitigation**: Assembly for hot paths, learn from Klaus Post

2. **Complexity**: Some codecs very complex
   - **Mitigation**: Phase approach, leverage existing Go libraries

3. **Compatibility**: Frames not compatible
   - **Mitigation**: Extensive compatibility testing, hybrid implementation

4. **Maintenance**: C library evolves
   - **Mitigation**: Keep CGO version, sync features periodically

### Project Risks:

1. **Timeline Slippage**: Underestimate complexity
   - **Mitigation**: Conservative estimates, phase gates

2. **Team Size**: Not enough engineers
   - **Mitigation**: Start with foundation, recruit contributors

3. **Motivation**: Long project, burnout
   - **Mitigation**: Celebrate milestones, release incrementally

---

## Success Criteria

### Phase Gates (Must Pass to Continue):

**Phase 1**:
- ✅ Parse all C-generated frames
- ✅ Write frames C can decompress
- ✅ 100% frame compatibility tests

**Phase 2**:
- ✅ 7 simple codecs working
- ✅ Compress/decompress numeric data
- ✅ Performance within 2x of C

**Phase 3**:
- ✅ Entropy codecs working
- ✅ Performance within 1.5x of C
- ✅ Compress structured data

**Phase 4**:
- ✅ Multi-codec graphs work
- ✅ Full graph compatibility

**Phase 5**:
- ✅ All codecs implemented
- ✅ Feature parity with C

**Phase 6**:
- ✅ Performance matches or beats C
- ✅ Production-ready

**Phase 7**:
- ✅ 100% feature coverage
- ✅ Complete documentation

### Final Release Criteria (v3.0.0):

- ✅ **Performance**: Within 1.2x of C compression, faster decompression
- ✅ **Compatibility**: 100% compatible with C library frames
- ✅ **Features**: Complete feature parity
- ✅ **Quality**: 100% test coverage, fuzz tested
- ✅ **Documentation**: Complete API docs, migration guide
- ✅ **Production**: Used in production for 3 months
- ✅ **Community**: 10+ external contributors

---

## Quick Wins from Klaus Post

While building Pure Go, adopt these patterns immediately:

### 1. Buffer Pooling
```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 128*1024)
    },
}

func (c *Compressor) Compress(src []byte) ([]byte, error) {
    buf := bufferPool.Get().([]byte)
    defer bufferPool.Put(buf)
    // ... use buf
}
```

### 2. Reduce Allocations
```go
// Before
func compress(data []byte) []byte {
    result := make([]byte, 0)
    // ... append to result
    return result
}

// After (Klaus Post style)
func CompressTo(dst, src []byte) (int, error) {
    // User provides buffer, no allocation
    n := doCompression(dst, src)
    return n, nil
}
```

### 3. Stateless Mode
```go
type Compressor struct {
    stateless bool
    ctx       *context
}

func WithStateless() Option {
    return func(c *Compressor) error {
        c.stateless = true
        return nil
    }
}
```

### 4. Inline-Friendly Functions
```go
// Small functions that compiler can inline
func zigzagEncode(x int64) uint64 {
    return uint64((x << 1) ^ (x >> 63))
}

func zigzagDecode(x uint64) int64 {
    return int64((x >> 1) ^ -(x & 1))
}
```

---

## Recommended First Steps

### Week 1-2: Foundation
1. Create `PURE_GO_ARCHITECTURE.md` document
2. Map all 128 codecs to complexity tiers
3. Set up dual build system (CGO + Pure Go)
4. Create compatibility test framework

### Week 3-4: Frame Format
1. Implement frame parser in pure Go
2. Test against 1000+ C-generated frames
3. Implement frame writer in pure Go
4. Verify C can decompress Go-written frames

### Week 5-8: First Codec (Delta)
1. Implement Delta codec in pure Go
2. Benchmark against C implementation
3. Optimize to within 2x of C
4. Document codec interface pattern

### Month 3: Build Momentum
1. Implement 3 more simple codecs
2. Create codec registry system
3. Build simple graph executor
4. Compress first real data!

---

## Long-Term Vision

### v1.0.0 (Current - CGO)
- ✅ CGO bindings to C library
- ✅ Idiomatic Go API
- ✅ Production-ready

### v2.0.0 (Hybrid - 6 months)
- ✅ CGO implementation (default)
- ✅ Pure Go implementation (partial, experimental)
- ✅ Build tags to choose
- Use cases: Simple codecs in pure Go

### v3.0.0 (Pure Go - 18 months)
- ✅ Pure Go implementation (default, faster!)
- ✅ CGO implementation (optional, fallback)
- ✅ Full feature parity
- ✅ Better performance than C

### v4.0.0+ (Future)
- Remove CGO implementation entirely
- Pure Go is the only implementation
- Focus on Go-specific optimizations
- Advanced features beyond C library

---

## Community & Collaboration

### Open Source Strategy:

1. **Transparency**: Public roadmap, weekly updates
2. **Incremental Releases**: Ship phases as they complete
3. **Contributor Friendly**: Good first issues, mentorship
4. **Documentation**: Architecture docs, codec guides
5. **Communication**: Discord/Slack for contributors

### Recruit Contributors:

- **Klaus Post**: Ask for advice, entropy codec collaboration
- **Go Community**: HN post, /r/golang, Gophers Slack
- **Compression Experts**: Reach out to academia, industry
- **Meta Engineers**: Invite OpenZL C library authors

---

## Decision Points

### Should You Proceed?

**✅ YES, if:**
- You have 12-18 months timeline
- You can dedicate 1-2 engineers
- You want to learn compression algorithms deeply
- You want full control over implementation
- Cross-compilation is critical
- You want to be a Go compression expert

**❌ NO, if:**
- You need it done in < 6 months
- You can't dedicate engineering resources
- CGO is acceptable for your use case
- You just want OpenZL bindings
- You don't care about implementation details

### Hybrid Approach (Recommended):

**Start the journey, but keep CGO:**

1. Begin Phase 0-1 (Foundation, Frame Format) - 4 months
2. Assess progress and interest
3. If going well, continue to Phase 2-3
4. Always have CGO as fallback
5. Release incremental pure Go improvements
6. v2.0 has both, v3.0 is pure Go by default

This way, you:
- ✅ Make progress toward pure Go
- ✅ Don't risk breaking current users
- ✅ Can stop anytime and still have value
- ✅ Learn compression algorithms
- ✅ Build expertise gradually

---

## Conclusion

**This is a massive, ambitious undertaking** - but it's **achievable** with:
- ✅ Phased approach (7 phases over 18 months)
- ✅ Dual implementation (CGO + Pure Go)
- ✅ Leveraging existing Go libraries (Klaus Post)
- ✅ Strong testing and compatibility
- ✅ Incremental releases
- ✅ Community collaboration

**You'll emerge with:**
- 🏆 Pure Go OpenZL implementation
- 🏆 Deep compression expertise
- 🏆 One of the most advanced Go compression libraries
- 🏆 Potentially faster than the C version
- 🏆 Full control over optimizations

**Next Steps:**
1. Review this plan
2. Decide on commitment level
3. Start Phase 0 (Foundation) - 2 months
4. Assess and adjust
5. Keep building!

Let's do this! 🚀
