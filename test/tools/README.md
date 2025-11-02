# OpenZL Test Tools

**Purpose**: Testing infrastructure for Pure Go implementation

**Important**: These are **test tools only** - NOT part of the Pure Go implementation!

---

## Overview

This directory contains C programs used for testing the Pure Go OpenZL implementation:

```
test/tools/
├── fixture_generator/    # Generates test frames using C library
│   ├── main.c           # Frame generation code
│   ├── Makefile         # Build system
│   └── README.md        # Documentation
└── validator/           # (Future) Validates Pure Go output
    └── main.c           # Frame validation code
```

---

## Fixture Generator

### What It Does

Generates OpenZL compressed frames using the **C library** to create known-good test files. Our Pure Go parser is then tested against these fixtures to ensure 100% compatibility.

### Why C?

**We need something to test against!**

- Our Pure Go implementation needs validation
- C library is the reference implementation
- Fixtures prove our Pure Go code can read real OpenZL frames

### What's Pure Go vs C

| Component | Language | Purpose | Part of Implementation? |
|-----------|----------|---------|------------------------|
| **Frame Reader** | Pure Go | Read OpenZL frames | ✅ YES |
| **Frame Writer** | Pure Go | Write OpenZL frames | ✅ YES |
| **Codecs** | Pure Go | Compress/decompress | ✅ YES |
| **Fixture Generator** | C | Generate test files | ❌ NO (test tool) |
| **Validator** | C | Validate Go output | ❌ NO (test tool) |

**Everything we ship is Pure Go!** C is only for test fixture generation.

---

## Building the Fixture Generator

### Prerequisites

```bash
# OpenZL C library already vendored
ls vendor/openzl/lib/libopenzl.a   # Should exist
```

### Build

```bash
cd test/tools/fixture_generator
make
```

This compiles `fixture_generator` using the vendored OpenZL C library.

---

## Generating Fixtures

```bash
cd test/tools/fixture_generator
make generate
```

**Output**: Creates ~10 test fixtures in `test/fixtures/frames/`:

```
test/fixtures/frames/
├── minimal.bin                   # Simplest frame
├── with_uncompressed_size.bin   # Frame with size field
├── with_compressed_size.bin     # Frame with compressed size
├── with_both_sizes.bin          # Both size fields
├── with_frame_checksum.bin      # Frame checksum
├── with_content_checksum.bin    # Content checksum
├── with_all_fields.bin          # All optional fields
├── empty_input.bin              # Edge case: empty
├── large_input.bin              # 1MB input
└── numeric_array.bin            # Typed int64 array
```

---

## Using Fixtures in Tests

### Go Test Example

```go
// internal/frame/reader_test.go
func TestParseMinimalFrame(t *testing.T) {
    // Load C-generated fixture
    data, err := os.ReadFile("../../test/fixtures/frames/minimal.bin")
    require.NoError(t, err)

    // Parse with Pure Go
    frame, err := ParseFrame(data)
    require.NoError(t, err)

    // Validate
    assert.Equal(t, uint32(0x5A4C0001), frame.Header.Magic)
    assert.NotNil(t, frame.Graph)
    assert.NotEmpty(t, frame.Payload)
}
```

### Test All Fixtures

```go
func TestParseAllFixtures(t *testing.T) {
    fixtures, err := filepath.Glob("../../test/fixtures/frames/*.bin")
    require.NoError(t, err)

    for _, fixture := range fixtures {
        t.Run(filepath.Base(fixture), func(t *testing.T) {
            data, err := os.ReadFile(fixture)
            require.NoError(t, err)

            frame, err := ParseFrame(data)
            require.NoError(t, err, "Failed to parse %s", fixture)

            // Basic validation
            assert.NotNil(t, frame)
            assert.NotNil(t, frame.Header)
        })
    }
}
```

---

## Testing Flow

### Phase 1: Read Fixtures (Parser Validation)

```
┌─────────────────────────────────────────────────┐
│  Step 1: Generate fixtures with C library      │
│  $ make generate                                 │
└─────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────┐
│  Step 2: Test Pure Go parser                   │
│  $ go test ./internal/frame/...                 │
│                                                  │
│  Pure Go reads C-generated frames               │
│  ✅ Proves Pure Go can read real OpenZL frames  │
└─────────────────────────────────────────────────┘
```

### Phase 2: Write + Validate (Writer Validation)

```
┌─────────────────────────────────────────────────┐
│  Step 1: Pure Go writes frames                 │
│  output := WriteFrame(frame)                    │
└─────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────┐
│  Step 2: C validator reads Pure Go output      │
│  $ ./validator test/outputs/go_*.bin            │
│                                                  │
│  C library validates Go-written frames          │
│  ✅ Proves C can read Pure Go frames            │
└─────────────────────────────────────────────────┘
```

---

## FAQ

### Q: Why not just write Pure Go tests without C?

**A**: We need to validate against the **reference implementation**. Without C fixtures, how do we know our Pure Go parser is correct? The C library defines what a valid OpenZL frame looks like.

### Q: Is the C code part of our Pure Go implementation?

**A**: NO! The C code is **test infrastructure only**. Our implementation is 100% Pure Go. The C fixture generator is like using a JPEG library to create test images for your Pure Go JPEG decoder - it's not part of your decoder.

### Q: Do we ship the C fixture generator?

**A**: NO! Users don't need it. We run it **once during development** to generate test files, commit those fixtures to git, and users run Go tests against the fixtures. The C tool stays in `test/tools/` and isn't distributed.

### Q: Can we test without the C library at all?

**A**: Not recommended. We need a source of truth. But once we have fixtures, we can:
1. Generate fixtures with C (one-time)
2. Commit fixtures to git
3. Run Go tests against fixtures (no C needed)
4. CI/CD only needs Go (fixtures are checked in)

### Q: What if we change the frame format?

**A**: Regenerate fixtures with `make generate`. But OpenZL's frame format is stable, so this should be rare.

---

## Makefile Targets

```bash
make           # Build fixture generator
make generate  # Build + generate all fixtures
make clean     # Remove generated files
make help      # Show help
```

---

## Next Steps

1. **Build fixture generator**:
   ```bash
   cd test/tools/fixture_generator
   make
   ```

2. **Generate test fixtures**:
   ```bash
   make generate
   ```

3. **Verify fixtures created**:
   ```bash
   ls ../../fixtures/frames/
   # Should see: minimal.bin, with_checksum.bin, etc.
   ```

4. **Begin Pure Go parser implementation**:
   ```bash
   # Create internal/frame/reader.go
   # Write tests using fixtures
   # Implement parser in Pure Go
   ```

---

## Reference

- [COMPATIBILITY_TEST_FRAMEWORK.md](../../docs/pure-go-planning/COMPATIBILITY_TEST_FRAMEWORK.md) - Full testing strategy
- [FRAME_FORMAT_SPEC.md](../../docs/pure-go-planning/FRAME_FORMAT_SPEC.md) - Binary format specification
- [PURE_GO_ARCHITECTURE.md](../../docs/pure-go-planning/PURE_GO_ARCHITECTURE.md) - Package design

---

**Remember**: This is test infrastructure, not our implementation! 🚀
**Our implementation is 100% Pure Go!** ✅
