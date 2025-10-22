# Examples

This directory contains example programs demonstrating go-openzl usage.

## Current Status

Examples will be added as features are implemented:

- [ ] **simple**: Basic compress/decompress (Phase 1)
- [ ] **context**: Reusable compression contexts (Phase 2)
- [ ] **numeric**: Typed compression for numeric arrays (Phase 3)
- [ ] **streaming**: Streaming compression with io.Reader/Writer (Phase 4)
- [ ] **file**: Compress and decompress files
- [ ] **benchmark**: Compare with other compression libraries

## Running Examples

Once implemented, run examples with:

```bash
go run examples/simple/main.go
go run examples/context/main.go
```

Or build all examples:

```bash
make examples
```
