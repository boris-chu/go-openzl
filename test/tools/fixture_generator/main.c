// Copyright (c) 2025 Boris Chu
// C Fixture Generator for OpenZL Pure Go Implementation Testing
//
// This tool generates OpenZL compressed frames for testing the Pure Go parser.
// It uses the OpenZL C library to create known-good test fixtures that validate
// our Pure Go implementation can correctly read real OpenZL frames.

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <sys/stat.h>
#include <openzl/openzl.h>

// Color output for terminal
#define COLOR_GREEN "\033[0;32m"
#define COLOR_YELLOW "\033[0;33m"
#define COLOR_RESET "\033[0m"

// Helper to create directory if it doesn't exist
static void ensure_dir(const char* path) {
    mkdir(path, 0755);
}

// Helper to write binary file
static int write_file(const char* path, const void* data, size_t size) {
    FILE* f = fopen(path, "wb");
    if (!f) {
        fprintf(stderr, "Failed to open %s for writing: ", path);
        perror("");
        return -1;
    }

    size_t written = fwrite(data, 1, size, f);
    fclose(f);

    if (written != size) {
        fprintf(stderr, "Failed to write all bytes to %s\n", path);
        return -1;
    }

    return 0;
}

// Helper to get error message
static const char* get_error(ZL_CCtx* cctx, ZL_Report result) {
    const char* msg = ZL_CCtx_getErrorContextString(cctx, result);
    return msg ? msg : "unknown error";
}

// Minimal frame - simplest possible
int gen_minimal_frame(const char* output_dir) {
    uint8_t input[] = {0x01, 0x02, 0x03, 0x04};
    uint8_t compressed[1024];

    ZL_CCtx* cctx = ZL_CCtx_create();
    if (!cctx) {
        fprintf(stderr, "Failed to create compression context\n");
        return -1;
    }

    // Set format version (required) - ignore return value warning for test tool
    (void)ZL_CCtx_setParameter(cctx, ZL_CParam_formatVersion, ZL_MAX_FORMAT_VERSION);

    // Compress
    ZL_Report result = ZL_CCtx_compress(
        cctx,
        compressed,
        sizeof(compressed),
        input,
        sizeof(input)
    );

    if (ZL_isError(result)) {
        fprintf(stderr, "Compression failed: %s\n", get_error(cctx, result));
        ZL_CCtx_free(cctx);
        return -1;
    }

    size_t compressed_size = ZL_validResult(result);

    char path[512];
    snprintf(path, sizeof(path), "%s/frames/minimal.bin", output_dir);
    int ret = write_file(path, compressed, compressed_size);

    if (ret == 0) {
        printf("  " COLOR_GREEN "✓" COLOR_RESET " Generated minimal.bin (%zu bytes)\n",
               compressed_size);
    }

    ZL_CCtx_free(cctx);
    return ret;
}

// Frame with checksums
int gen_frame_with_checksums(const char* output_dir) {
    uint8_t input[] = "Hello OpenZL!";
    uint8_t compressed[1024];

    ZL_CCtx* cctx = ZL_CCtx_create();
    if (!cctx) return -1;

    (void)ZL_CCtx_setParameter(cctx, ZL_CParam_formatVersion, ZL_MAX_FORMAT_VERSION);
    (void)ZL_CCtx_setParameter(cctx, ZL_CParam_compressedChecksum, 1);
    (void)ZL_CCtx_setParameter(cctx, ZL_CParam_contentChecksum, 1);

    ZL_Report result = ZL_CCtx_compress(
        cctx,
        compressed,
        sizeof(compressed),
        input,
        sizeof(input) - 1  // Exclude null terminator
    );

    if (ZL_isError(result)) {
        fprintf(stderr, "Compression failed: %s\n", get_error(cctx, result));
        ZL_CCtx_free(cctx);
        return -1;
    }

    size_t compressed_size = ZL_validResult(result);

    char path[512];
    snprintf(path, sizeof(path), "%s/frames/with_checksums.bin", output_dir);
    int ret = write_file(path, compressed, compressed_size);

    if (ret == 0) {
        printf("  " COLOR_GREEN "✓" COLOR_RESET " Generated with_checksums.bin (%zu bytes)\n",
               compressed_size);
    }

    ZL_CCtx_free(cctx);
    return ret;
}

// Large input (1MB)
int gen_large_input_frame(const char* output_dir) {
    size_t input_size = 1024 * 1024;
    uint8_t* input = malloc(input_size);
    if (!input) {
        fprintf(stderr, "Failed to allocate 1MB for large input\n");
        return -1;
    }

    // Fill with pattern
    for (size_t i = 0; i < input_size; i++) {
        input[i] = (uint8_t)(i % 256);
    }

    size_t compressed_bound = input_size * 2 + 1024;
    uint8_t* compressed = malloc(compressed_bound);
    if (!compressed) {
        free(input);
        return -1;
    }

    ZL_CCtx* cctx = ZL_CCtx_create();
    if (!cctx) {
        free(input);
        free(compressed);
        return -1;
    }

    (void)ZL_CCtx_setParameter(cctx, ZL_CParam_formatVersion, ZL_MAX_FORMAT_VERSION);

    ZL_Report result = ZL_CCtx_compress(
        cctx,
        compressed,
        compressed_bound,
        input,
        input_size
    );

    if (ZL_isError(result)) {
        fprintf(stderr, "Large input compression failed: %s\n", get_error(cctx, result));
        ZL_CCtx_free(cctx);
        free(input);
        free(compressed);
        return -1;
    }

    size_t compressed_size = ZL_validResult(result);

    char path[512];
    snprintf(path, sizeof(path), "%s/frames/large_input.bin", output_dir);
    int ret = write_file(path, compressed, compressed_size);

    if (ret == 0) {
        printf("  " COLOR_GREEN "✓" COLOR_RESET " Generated large_input.bin (%zu bytes from 1MB input)\n",
               compressed_size);
    }

    ZL_CCtx_free(cctx);
    free(input);
    free(compressed);
    return ret;
}

// Numeric array (typed compression)
int gen_numeric_array_frame(const char* output_dir) {
    int64_t input[] = {100, 101, 102, 103, 104, 105, 106, 107, 108, 109};
    uint8_t compressed[1024];

    ZL_CCtx* cctx = ZL_CCtx_create();
    if (!cctx) return -1;

    (void)ZL_CCtx_setParameter(cctx, ZL_CParam_formatVersion, ZL_MAX_FORMAT_VERSION);

    // Create typed reference for numeric array
    ZL_TypedRef* tref = ZL_TypedRef_createNumeric(
        input,
        sizeof(int64_t),
        sizeof(input) / sizeof(int64_t)
    );

    if (!tref) {
        ZL_CCtx_free(cctx);
        return -1;
    }

    ZL_Report result = ZL_CCtx_compressTypedRef(
        cctx,
        compressed,
        sizeof(compressed),
        tref
    );

    ZL_TypedRef_free(tref);

    if (ZL_isError(result)) {
        fprintf(stderr, "Typed compression failed: %s\n", get_error(cctx, result));
        ZL_CCtx_free(cctx);
        return -1;
    }

    size_t compressed_size = ZL_validResult(result);

    char path[512];
    snprintf(path, sizeof(path), "%s/frames/numeric_array.bin", output_dir);
    int ret = write_file(path, compressed, compressed_size);

    if (ret == 0) {
        printf("  " COLOR_GREEN "✓" COLOR_RESET " Generated numeric_array.bin (%zu bytes from 10 int64s)\n",
               compressed_size);
    }

    ZL_CCtx_free(cctx);
    return ret;
}

int main(int argc, char** argv) {
    if (argc != 2) {
        fprintf(stderr, "Usage: %s <output_directory>\n", argv[0]);
        fprintf(stderr, "Example: %s test/fixtures\n", argv[0]);
        return 1;
    }

    const char* output_dir = argv[1];

    printf("\n");
    printf("OpenZL C Fixture Generator\n");
    printf("==========================\n");
    printf("Generating test fixtures for Pure Go parser validation...\n\n");

    // Ensure output directories exist
    char frames_dir[1024];
    snprintf(frames_dir, sizeof(frames_dir), "%s/frames", output_dir);
    ensure_dir(output_dir);
    ensure_dir(frames_dir);

    printf("Output directory: %s\n\n", frames_dir);

    int count = 0;
    int failed = 0;

    // Generate fixtures
    if (gen_minimal_frame(output_dir) == 0) count++; else failed++;
    if (gen_frame_with_checksums(output_dir) == 0) count++; else failed++;
    if (gen_large_input_frame(output_dir) == 0) count++; else failed++;
    if (gen_numeric_array_frame(output_dir) == 0) count++; else failed++;

    printf("\n");
    if (failed == 0) {
        printf(COLOR_GREEN "✓ Success!" COLOR_RESET " Generated %d test fixtures in %s/frames/\n",
               count, output_dir);
    } else {
        printf(COLOR_YELLOW "⚠ Warning!" COLOR_RESET " Generated %d fixtures, %d failed\n",
               count, failed);
    }
    printf("\n");
    printf("These fixtures can now be used to test the Pure Go frame parser.\n");
    printf("See: internal/frame/reader_test.go\n");
    printf("\n");

    return failed > 0 ? 1 : 0;
}
