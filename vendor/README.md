# Vendor Directory

This directory will contain the OpenZL C library source code.

## Setup

### Option 1: Git Submodule (Recommended)

```bash
git submodule add https://github.com/facebook/openzl.git vendor/openzl
git submodule update --init --recursive
```

### Option 2: Manual Download

```bash
cd vendor
git clone --depth 1 https://github.com/facebook/openzl.git
cd openzl
git checkout <specific-release-tag>
```

## Building OpenZL

Once the source is in place, build the library:

```bash
# From project root
make build-openzl
```

This will:
1. Build the OpenZL C library
2. Create `vendor/openzl/lib/libopenzl.a`
3. Make headers available in `vendor/openzl/include/`

## Current Status

- [ ] OpenZL source added
- [ ] Library built successfully
- [ ] CGO can link against it

## Version Tracking

We vendor a specific version of OpenZL to ensure compatibility:

- **Target Version**: Latest stable release from https://github.com/facebook/openzl
- **Compatibility**: Bindings target OpenZL C API version compatible with release tags

## Note

The `vendor/openzl/` directory is in `.gitignore` (will be once submodule is added).
Users of this library will need to initialize submodules or the build system will
download and build OpenZL automatically.
