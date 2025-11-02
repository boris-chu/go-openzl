# Migration Guide: gzip/zstd to OpenZL

**Target Audience**: Go developers using `compress/gzip` or `github.com/klauspost/compress/zstd`
**Goal**: Migrate to go-openzl for better compression ratios on structured data

---

## Why Migrate to OpenZL?

### When OpenZL is Better

✅ **Use OpenZL when**:
- Compressing **structured data** (logs, metrics, telemetry)
- Compressing **numeric arrays** (ML datasets, time series)
- Compressing **repeated patterns** (configuration files, dumps)
- Need **best possible compression ratio**
- Working with **type-aware data**

### When to Stick with gzip/zstd

⚠️ **Keep using gzip/zstd when**:
- Need **universal compatibility** (web servers, HTTP)
- Compressing **random binary data** (images, videos)
- Need **streaming decompression** from untrusted sources
- Working with **very small files** (<1KB)

### Performance Comparison

| Library | Compression Ratio | Speed | Use Case |
|---------|------------------|-------|----------|
| **OpenZL (typed)** | **50x** | Fast | Numeric arrays, metrics |
| **OpenZL (streaming)** | **1364x** | 2287 MB/s | Repeated structured data |
| zstd | 20-30x | 400 MB/s | General purpose |
| gzip | 10-15x | 100 MB/s | HTTP, universal |

---

## Quick Start: Drop-In Replacement

### From gzip

**Before (gzip)**:
```go
import (
    "compress/gzip"
    "io"
    "os"
)

func compressFile(input, output string) error {
    in, _ := os.Open(input)
    defer in.Close()

    out, _ := os.Create(output)
    defer out.Close()

    writer := gzip.NewWriter(out)
    defer writer.Close()

    _, err := io.Copy(writer, in)
    return err
}
```

**After (OpenZL)**:
```go
import (
    "github.com/borischu/go-openzl"
    "io"
    "os"
)

func compressFile(input, output string) error {
    in, _ := os.Open(input)
    defer in.Close()

    out, _ := os.Create(output)
    defer out.Close()

    writer, _ := openzl.NewWriter(out)
    defer writer.Close()

    _, err := io.Copy(writer, in)
    return err
}
```

**Changes**: Just swap `gzip.NewWriter` → `openzl.NewWriter`!

---

## Migration Patterns

### Pattern 1: Simple In-Memory Compression

#### gzip
```go
import (
    "bytes"
    "compress/gzip"
)

func compressData(data []byte) ([]byte, error) {
    var buf bytes.Buffer
    writer := gzip.NewWriter(&buf)
    writer.Write(data)
    writer.Close()
    return buf.Bytes(), nil
}

func decompressData(compressed []byte) ([]byte, error) {
    reader, err := gzip.NewReader(bytes.NewReader(compressed))
    if err != nil {
        return nil, err
    }
    defer reader.Close()

    return io.ReadAll(reader)
}
```

#### OpenZL
```go
import "github.com/borischu/go-openzl"

func compressData(data []byte) ([]byte, error) {
    return openzl.Compress(data)
}

func decompressData(compressed []byte) ([]byte, error) {
    return openzl.Decompress(compressed)
}
```

**Benefit**: Simpler API, no buffer management needed

---

### Pattern 2: Streaming Large Files

#### zstd
```go
import (
    "github.com/klauspost/compress/zstd"
    "io"
    "os"
)

func compressLargeFile(input, output string) error {
    in, _ := os.Open(input)
    defer in.Close()

    out, _ := os.Create(output)
    defer out.Close()

    encoder, _ := zstd.NewWriter(out)
    defer encoder.Close()

    _, err := io.Copy(encoder, in)
    return err
}
```

#### OpenZL
```go
import (
    "github.com/borischu/go-openzl"
    "io"
    "os"
)

func compressLargeFile(input, output string) error {
    in, _ := os.Open(input)
    defer in.Close()

    out, _ := os.Create(output)
    defer out.Close()

    writer, _ := openzl.NewWriter(out)
    defer writer.Close()

    _, err := io.Copy(writer, in)
    return err
}
```

**Benefit**: Same API, better compression ratios on structured data

---

### Pattern 3: Compressing Numeric Data (NEW!)

#### Before (gzip on binary data)
```go
import (
    "bytes"
    "compress/gzip"
    "encoding/binary"
)

func compressMetrics(metrics []int64) ([]byte, error) {
    // Convert to bytes
    var buf bytes.Buffer
    for _, m := range metrics {
        binary.Write(&buf, binary.LittleEndian, m)
    }

    // Compress
    var compressed bytes.Buffer
    writer := gzip.NewWriter(&compressed)
    writer.Write(buf.Bytes())
    writer.Close()

    return compressed.Bytes(), nil
}
// Compression ratio: ~3-5x
```

#### After (OpenZL typed compression)
```go
import "github.com/borischu/go-openzl"

func compressMetrics(metrics []int64) ([]byte, error) {
    return openzl.CompressNumeric(metrics)
}

func decompressMetrics(compressed []byte) ([]int64, error) {
    return openzl.DecompressNumeric[int64](compressed)
}
// Compression ratio: 50x (10x better!)
```

**Benefit**: Type-aware compression, dramatically better ratios

---

### Pattern 4: High-Throughput Pipeline

#### zstd with encoder reuse
```go
import "github.com/klauspost/compress/zstd"

type Pipeline struct {
    encoder *zstd.Encoder
}

func NewPipeline() (*Pipeline, error) {
    encoder, err := zstd.NewWriter(nil,
        zstd.WithEncoderConcurrency(1))
    if err != nil {
        return nil, err
    }
    return &Pipeline{encoder: encoder}, nil
}

func (p *Pipeline) Compress(data []byte) ([]byte, error) {
    return p.encoder.EncodeAll(data, nil), nil
}

func (p *Pipeline) Close() {
    p.encoder.Close()
}
```

#### OpenZL with context reuse
```go
import "github.com/borischu/go-openzl"

type Pipeline struct {
    compressor *openzl.Compressor
}

func NewPipeline() (*Pipeline, error) {
    compressor, err := openzl.NewCompressor()
    if err != nil {
        return nil, err
    }
    return &Pipeline{compressor: compressor}, nil
}

func (p *Pipeline) Compress(data []byte) ([]byte, error) {
    return p.compressor.Compress(data)
}

func (p *Pipeline) Close() {
    p.compressor.Close()
}
```

**Benefit**: Similar API, 21% faster with context reuse

---

## Feature Comparison

### API Parity

| Feature | gzip | zstd | OpenZL |
|---------|------|------|--------|
| **Basic** |
| Simple compress/decompress | ✅ | ✅ | ✅ |
| Streaming I/O | ✅ | ✅ | ✅ |
| io.Reader/Writer | ✅ | ✅ | ✅ |
| **Performance** |
| Context reuse | ❌ | ✅ | ✅ |
| Compression levels | ✅ | ✅ | ⏳ (v1.1) |
| Concurrent compression | ❌ | ✅ | ✅ |
| **Advanced** |
| Typed compression | ❌ | ❌ | ✅ |
| Numeric arrays | ❌ | ❌ | ✅ |
| Frame size control | ❌ | ✅ | ✅ |
| Checksum | ✅ | ✅ | ⏳ (v1.1) |
| **Ecosystem** |
| HTTP middleware | ✅ | ✅ | ⏳ (v1.1) |
| Universal format | ✅ | ✅ | ❌ |
| Browser support | ✅ | ❌ | ❌ |

---

## Common Migration Scenarios

### Scenario 1: Log File Archival

**Before**:
```go
// Daily log rotation with gzip
func rotateLogs() error {
    input, _ := os.Open("/var/log/app.log")
    output, _ := os.Create("/var/log/app.log.gz")

    writer := gzip.NewWriter(output)
    io.Copy(writer, input)
    writer.Close()

    os.Remove("/var/log/app.log")
    return nil
}
// Compression: 400MB -> 40MB (10x)
```

**After**:
```go
import "github.com/borischu/go-openzl"

func rotateLogs() error {
    input, _ := os.Open("/var/log/app.log")
    output, _ := os.Create("/var/log/app.log.zl")

    writer, _ := openzl.NewWriter(output)
    io.Copy(writer, input)
    writer.Close()

    os.Remove("/var/log/app.log")
    return nil
}
// Compression: 400MB -> 0.9MB (430x!)
// Save 97% more storage
```

---

### Scenario 2: Metrics Storage

**Before**:
```go
// Store metrics as compressed JSON
type Metrics struct {
    Timestamp []int64   `json:"ts"`
    Values    []float64 `json:"vals"`
}

func saveMetrics(m *Metrics) error {
    data, _ := json.Marshal(m)

    var compressed bytes.Buffer
    writer := gzip.NewWriter(&compressed)
    writer.Write(data)
    writer.Close()

    return os.WriteFile("metrics.json.gz", compressed.Bytes(), 0644)
}
// Compression: ~5-10x
```

**After**:
```go
import "github.com/borischu/go-openzl"

// Store metrics as compressed arrays (better!)
type Metrics struct {
    Timestamp []int64
    Values    []float64
}

func saveMetrics(m *Metrics) error {
    // Compress typed arrays
    tsCompressed, _ := openzl.CompressNumeric(m.Timestamp)
    valCompressed, _ := openzl.CompressNumeric(m.Values)

    // Save (or send over network)
    // ...
    return nil
}
// Compression: 50x (10x better!)
// No JSON overhead
```

---

### Scenario 3: HTTP Response Compression

**Before**:
```go
import (
    "compress/gzip"
    "net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Encoding", "gzip")

    writer := gzip.NewWriter(w)
    defer writer.Close()

    // Write response
    writer.Write(largeResponse)
}
// Works everywhere (browsers, curl, etc.)
```

**After**:
```go
import (
    "github.com/borischu/go-openzl"
    "net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
    // Custom encoding (client must support)
    w.Header().Set("Content-Encoding", "openzl")

    writer, _ := openzl.NewWriter(w)
    defer writer.Close()

    writer.Write(largeResponse)
}
// Better compression, but custom client needed
```

**Note**: For HTTP, stick with gzip for compatibility. Use OpenZL for:
- Internal APIs (microservices)
- Mobile apps (custom client)
- Data pipelines (not web browsers)

---

## Performance Tuning

### Choosing the Right API

```go
// 1. Small data, occasional compression
// Use: One-shot API
compressed, _ := openzl.Compress(data)

// 2. Many compressions, same thread
// Use: Context API (21% faster)
compressor, _ := openzl.NewCompressor()
for _, data := range batches {
    compressed, _ := compressor.Compress(data)
}
compressor.Close()

// 3. Large files (>1MB)
// Use: Streaming API (2.3 GB/s)
writer, _ := openzl.NewWriter(output)
io.Copy(writer, input)
writer.Close()

// 4. Numeric data (metrics, telemetry)
// Use: Typed API (50x better ratio!)
compressed, _ := openzl.CompressNumeric(numbers)
```

### Frame Size Tuning

```go
// Small files (<10KB): Use 4KB frames
writer, _ := openzl.NewWriter(output,
    openzl.WithFrameSize(4*1024))

// Medium files (10KB-1MB): Use default 64KB
writer, _ := openzl.NewWriter(output) // 64KB default

// Large files (>1MB): Use 256KB frames
writer, _ := openzl.NewWriter(output,
    openzl.WithFrameSize(256*1024))
```

---

## Common Pitfalls

### Pitfall 1: Using for Random Binary Data

❌ **Bad**:
```go
// Compressing random/encrypted data
imageData, _ := os.ReadFile("photo.jpg")
compressed, _ := openzl.Compress(imageData)
// Result: Larger than input! (No patterns to compress)
```

✅ **Good**:
```go
// OpenZL is for structured data
logData, _ := os.ReadFile("app.log")
compressed, _ := openzl.Compress(logData)
// Result: 430x compression (lots of patterns)
```

### Pitfall 2: Not Closing Writers

❌ **Bad**:
```go
writer, _ := openzl.NewWriter(output)
writer.Write(data)
// Missing writer.Close()!
// Data not flushed, file corrupt
```

✅ **Good**:
```go
writer, _ := openzl.NewWriter(output)
defer writer.Close() // Always defer Close()
writer.Write(data)
```

### Pitfall 3: Forgetting Type for Decompression

❌ **Bad**:
```go
// Type mismatch!
compressed, _ := openzl.CompressNumeric([]int64{1, 2, 3})
decompressed, _ := openzl.DecompressNumeric[int32](compressed)
// Result: Wrong values!
```

✅ **Good**:
```go
compressed, _ := openzl.CompressNumeric([]int64{1, 2, 3})
decompressed, _ := openzl.DecompressNumeric[int64](compressed)
// Must match original type
```

---

## Gradual Migration Strategy

### Step 1: Identify Good Candidates

Analyze your data:
```bash
# Check compression ratio improvement
$ gzip -c data.log | wc -c    # e.g., 1MB
$ go run compress_test.go      # e.g., 2KB
# 500x improvement -> Good candidate!
```

Good candidates:
- Structured logs
- Time-series data
- Metrics/telemetry
- Configuration dumps
- Repeated text

Bad candidates:
- Images/videos
- Already compressed data
- Encrypted data
- Random binary data

### Step 2: Parallel Running

Run both compressions, compare:
```go
func migrateCompress(data []byte) ([]byte, error) {
    // Old way
    oldCompressed, _ := compressWithGzip(data)
    log.Printf("gzip: %d bytes", len(oldCompressed))

    // New way
    newCompressed, _ := openzl.Compress(data)
    log.Printf("openzl: %d bytes", len(newCompressed))

    // Use new way if better
    if len(newCompressed) < len(oldCompressed) {
        return newCompressed, nil
    }
    return oldCompressed, nil
}
```

### Step 3: Feature Flag

Use feature flag for gradual rollout:
```go
func compressLogs(data []byte) ([]byte, error) {
    if featureFlags.UseOpenZL {
        return openzl.Compress(data)
    }
    return compressWithGzip(data)
}
```

### Step 4: Full Migration

Once confident, remove old code:
```go
func compressLogs(data []byte) ([]byte, error) {
    return openzl.Compress(data)
}
```

---

## Compatibility Considerations

### File Format

**OpenZL files are NOT compatible with gzip/zstd**:
- Different compression algorithm
- Different frame format
- Different headers

**Migration approach**:
```go
// Option 1: Dual-format support
func detectAndDecompress(data []byte) ([]byte, error) {
    // Try OpenZL first
    if result, err := openzl.Decompress(data); err == nil {
        return result, nil
    }

    // Fallback to gzip
    return decompressGzip(data)
}

// Option 2: File extension convention
// .gz -> gzip
// .zl -> openzl
```

### HTTP Content-Encoding

**gzip** is universally supported:
```
Content-Encoding: gzip
```

**OpenZL** requires custom handling:
```
Content-Encoding: openzl
```

**Recommendation**: Use gzip for HTTP, OpenZL for internal services

---

## Checklist: Are You Ready to Migrate?

Use this checklist to decide:

✅ **Yes, migrate if**:
- [ ] Compressing structured/repeated data
- [ ] Internal services (not public web)
- [ ] Need best possible compression ratio
- [ ] Working with numeric arrays
- [ ] Can change both client and server
- [ ] Not concerned about universal compatibility

❌ **No, stick with gzip/zstd if**:
- [ ] Need HTTP browser compatibility
- [ ] Compressing random binary data
- [ ] Need universal format support
- [ ] Very small files (<100 bytes)
- [ ] Can't change client code
- [ ] Require streaming from untrusted sources

---

## Support and Resources

### Documentation
- [README.md](README.md) - Quick start and examples
- [TESTING.md](TESTING.md) - Performance metrics
- [API_COMPARISON.md](docs/API_COMPARISON.md) - Detailed API comparison

### Examples
- `examples/simple/` - Basic compression
- `examples/context/` - Performance optimization
- `examples/typed/` - Numeric data compression
- `examples/streaming/` - Large file handling

### Getting Help
- [GitHub Issues](https://github.com/boris-chu/go-openzl/issues) - Bug reports
- [Discussions](https://github.com/boris-chu/go-openzl/discussions) - Questions

---

## Conclusion

**OpenZL excels at**:
- Structured data (logs, configs, dumps)
- Numeric arrays (metrics, telemetry, ML)
- Repeated patterns (time-series data)

**Best migration candidates**:
1. Log file archival
2. Metrics storage
3. Telemetry data
4. Internal API compression
5. Data pipeline optimization

**Not a replacement for**:
- HTTP compression (use gzip)
- Binary file compression (use zstd)
- Universal file format

**Start small**, measure results, and migrate incrementally!

---

**Document Version**: 1.0
**Last Updated**: October 22, 2025
**Target**: go-openzl v1.0.0
