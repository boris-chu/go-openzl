# Contributing to go-openzl

First off, thank you for considering contributing to go-openzl! This project aims to bring Meta's OpenZL compression framework to the Go ecosystem, and we need community help to make it successful.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [How Can I Contribute?](#how-can-i-contribute)
- [Development Setup](#development-setup)
- [Coding Standards](#coding-standards)
- [Commit Messages](#commit-messages)
- [Pull Request Process](#pull-request-process)
- [License and Attribution](#license-and-attribution)

## Code of Conduct

This project follows the [Go Community Code of Conduct](https://go.dev/conduct). Please be respectful and constructive in all interactions.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check existing issues to avoid duplicates. When creating a bug report, include:

- **Clear title and description**
- **Steps to reproduce** the issue
- **Expected behavior** vs actual behavior
- **Environment details** (Go version, OS, architecture)
- **Code sample** if applicable
- **Error messages and stack traces**

### Suggesting Enhancements

Enhancement suggestions are welcome! Please:

- Use a clear and descriptive title
- Provide detailed description of the proposed enhancement
- Explain why this enhancement would be useful
- Provide code examples if applicable

### Your First Code Contribution

Unsure where to begin? Look for issues labeled:

- `good first issue` - Simple issues perfect for newcomers
- `help wanted` - Issues where we need community help
- `documentation` - Help improve our docs

### Areas We Need Help

1. **Core Implementation** (High Priority)
   - CGO bindings for OpenZL C API
   - Memory management and lifecycle handling
   - Error translation from C to Go

2. **Testing**
   - Unit tests for all API functions
   - Integration tests with real data
   - Fuzz testing for robustness
   - Benchmarks and performance tests

3. **Documentation**
   - API documentation with examples
   - Usage guides and tutorials
   - Performance best practices
   - Migration guides from other compression libraries

4. **Infrastructure**
   - CI/CD pipelines (GitHub Actions)
   - Cross-platform testing (Linux, macOS, Windows)
   - Build scripts for OpenZL C library
   - Release automation

5. **Advanced Features**
   - Streaming API (io.Reader/Writer)
   - Concurrent compression workers
   - Custom compression graphs
   - Performance optimization

## Development Setup

### Prerequisites

- Go 1.21 or later
- CGO enabled (`CGO_ENABLED=1`)
- C11 compiler (gcc, clang)
- C++17 compiler (g++, clang++)
- Git with submodules support

### Setup Instructions

```bash
# Clone the repository
git clone https://github.com/yourusername/go-openzl.git
cd go-openzl

# Initialize git submodules (when available)
git submodule update --init --recursive

# Build the OpenZL C library (when build scripts are ready)
make build-openzl

# Run tests
go test ./...

# Run tests with race detector
go test -race ./...

# Run benchmarks
go test -bench=. ./benchmarks/

# Check code coverage
go test -cover ./...
```

## Coding Standards

### Go Code Style

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` for formatting (enforced by CI)
- Use `go vet` to catch common issues
- Use `golangci-lint` for comprehensive linting

### Code Organization

```go
// Package comment with overview
package openzl

import (
    "errors"
    "fmt"
    // Standard library imports first

    "github.com/external/package"
    // External imports second

    "github.com/yourusername/go-openzl/internal/cgo"
    // Internal imports last
)

// Exported types and functions have godoc comments
// that start with the name of the thing being documented.

// Compressor handles OpenZL compression operations.
// It wraps the underlying C compression context and provides
// a safe, idiomatic Go interface.
type Compressor struct {
    // unexported fields
}

// NewCompressor creates a new Compressor instance.
// It returns an error if the underlying C context cannot be created.
func NewCompressor() (*Compressor, error) {
    // implementation
}
```

### Error Handling

- Always check and handle errors
- Use meaningful error messages
- Wrap errors with context using `fmt.Errorf` with `%w`
- Define sentinel errors as package-level vars

```go
var (
    ErrInvalidInput = errors.New("openzl: invalid input")
    ErrBufferTooSmall = errors.New("openzl: buffer too small")
)

func (c *Compressor) Compress(dst, src []byte) (int, error) {
    if len(src) == 0 {
        return 0, ErrInvalidInput
    }

    n, err := c.compress(dst, src)
    if err != nil {
        return 0, fmt.Errorf("compress failed: %w", err)
    }

    return n, nil
}
```

### Testing

- Write tests for all public APIs
- Use table-driven tests where appropriate
- Test error conditions, not just happy paths
- Include benchmarks for performance-critical code

```go
func TestCompressor_Compress(t *testing.T) {
    tests := []struct {
        name    string
        input   []byte
        wantErr bool
    }{
        {
            name:    "empty input",
            input:   []byte{},
            wantErr: true,
        },
        {
            name:    "small input",
            input:   []byte("hello world"),
            wantErr: false,
        },
        // more test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            c, err := NewCompressor()
            if err != nil {
                t.Fatalf("NewCompressor() error = %v", err)
            }
            defer c.Close()

            // test implementation
        })
    }
}
```

### Documentation

- All exported types, functions, methods, and constants must have godoc comments
- Comments should be complete sentences
- Include usage examples in doc comments where helpful
- Use `Example*` functions for runnable examples

```go
// Compress compresses src into dst using OpenZL.
//
// It returns the number of bytes written to dst and any error encountered.
// If dst is too small, it returns ErrBufferTooSmall.
//
// Example usage:
//
//     c, _ := openzl.NewCompressor()
//     defer c.Close()
//
//     src := []byte("hello world")
//     dst := make([]byte, openzl.CompressBound(len(src)))
//
//     n, err := c.Compress(dst, src)
//     if err != nil {
//         log.Fatal(err)
//     }
//
//     compressed := dst[:n]
func (c *Compressor) Compress(dst, src []byte) (int, error) {
    // implementation
}
```

## Commit Messages

We follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

### Format

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks
- `perf`: Performance improvements

### Examples

```
feat: add basic compression API

Implement ZL_CCtx_compress wrapper with Go-friendly interface.
Includes memory management and error handling.

feat(typed): add numeric array compression

Implement typed compression for numeric arrays with proper
alignment and endianness handling.

fix: prevent memory leak in Compressor.Close

Ensure C context is properly freed even if Close is called multiple times.

docs: add compression examples to README

test: add fuzz tests for decompression

chore: update dependencies
```

## Pull Request Process

1. **Fork and Branch**
   - Fork the repository
   - Create a feature branch from `main`
   - Use descriptive branch names (e.g., `feat/streaming-api`, `fix/memory-leak`)

2. **Make Changes**
   - Write clean, tested, documented code
   - Follow coding standards
   - Add/update tests as needed
   - Update documentation if applicable

3. **Test Locally**
   ```bash
   # Format code
   go fmt ./...

   # Run linters
   golangci-lint run

   # Run all tests
   go test ./...

   # Run with race detector
   go test -race ./...

   # Check coverage
   go test -cover ./...
   ```

4. **Commit**
   - Write clear commit messages
   - Make atomic commits (one logical change per commit)
   - Sign your commits if possible

5. **Push and Create PR**
   - Push your branch to your fork
   - Create a Pull Request to `main`
   - Fill out the PR template completely
   - Link related issues

6. **PR Review**
   - Address reviewer feedback promptly
   - Keep discussions constructive and respectful
   - Update your PR based on feedback
   - Don't force-push after review has started (unless asked)

7. **Merge**
   - PRs require at least one approval
   - All CI checks must pass
   - Maintainers will merge once ready

## License and Attribution

### Code License

All contributions are licensed under the [BSD 3-Clause License](LICENSE), the same license as the project itself.

By contributing, you agree that your contributions will be licensed under this license.

### Attribution Requirements

#### For Your Contributions

When you contribute code, you automatically retain copyright to your contribution, but you grant us (and everyone) the right to use it under the BSD 3-Clause License.

Your contribution will be attributed in:
- Git commit history (use your real name or consistent pseudonym)
- GitHub contribution graphs
- Release notes and changelogs
- CONTRIBUTORS file (for significant contributions)

#### Copyright Headers

New files should include a copyright header:

```go
// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package openzl
```

For files with substantial contributions from multiple people:

```go
// Copyright (c) 2025 Boris Chu and contributors
// Copyright (c) 2025 Jane Developer
// SPDX-License-Identifier: BSD-3-Clause

package openzl
```

#### Attribution for OpenZL

Since this project wraps Meta's OpenZL library:

- Keep OpenZL attribution in LICENSE file
- Reference OpenZL in documentation
- Link to upstream OpenZL project
- Follow OpenZL's BSD 3-Clause License terms

#### Using go-openzl in Your Projects

If you use go-openzl in your project:

**Required:**
- Include a copy of our LICENSE file (or reference it)
- Maintain copyright notices

**Appreciated (but not required):**
- Mention go-openzl in your documentation
- Link to this repository
- Star the project on GitHub

Example attribution in your README:

```markdown
This project uses [go-openzl](https://github.com/yourusername/go-openzl)
for compression, which provides Go bindings to Meta's OpenZL library.
```

### Contributor License Agreement

We do **not** require a separate CLA. By submitting a pull request, you are agreeing to license your contribution under the project's BSD 3-Clause License.

## Questions?

- **Discussions**: Use [GitHub Discussions](https://github.com/yourusername/go-openzl/discussions) for questions
- **Issues**: Use [GitHub Issues](https://github.com/yourusername/go-openzl/issues) for bugs and features
- **Email**: Contact the maintainers directly for sensitive matters

## Recognition

Significant contributors will be:
- Added to a CONTRIBUTORS file
- Mentioned in release notes
- Credited in documentation
- Given recognition in the community

Thank you for contributing to go-openzl! 🎉
