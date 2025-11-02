// Frame Inspector - Dumps detailed information about OpenZL frames
// This helps us understand the actual binary format

#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <openzl/openzl.h>

void print_hex(const uint8_t* data, size_t size, const char* label) {
    printf("%s (%zu bytes):\n", label, size);
    for (size_t i = 0; i < size; i++) {
        if (i > 0 && i % 16 == 0) printf("\n");
        printf("%02x ", data[i]);
    }
    printf("\n\n");
}

int main(int argc, char** argv) {
    if (argc != 2) {
        fprintf(stderr, "Usage: %s <frame.bin>\n", argv[0]);
        return 1;
    }

    // Read frame file
    FILE* f = fopen(argv[1], "rb");
    if (!f) {
        fprintf(stderr, "Failed to open %s\n", argv[1]);
        return 1;
    }

    fseek(f, 0, SEEK_END);
    size_t frame_size = ftell(f);
    fseek(f, 0, SEEK_SET);

    uint8_t* frame_data = malloc(frame_size);
    fread(frame_data, 1, frame_size, f);
    fclose(f);

    printf("=================================\n");
    printf("OpenZL Frame Inspector\n");
    printf("=================================\n\n");

    printf("Frame file: %s\n", argv[1]);
    printf("Frame size: %zu bytes\n\n", frame_size);

    // Print hex dump
    print_hex(frame_data, frame_size, "Raw frame data");

    // Get decompressed size
    ZL_Report decompressed_size_report = ZL_getDecompressedSize(frame_data, frame_size);
    if (ZL_isError(decompressed_size_report)) {
        printf("ERROR: Failed to get decompressed size\n");
        free(frame_data);
        return 1;
    }
    size_t decompressed_size = ZL_validResult(decompressed_size_report);
    printf("Decompressed size: %zu bytes\n\n", decompressed_size);

    // Get compressed size
    ZL_Report compressed_size_report = ZL_getCompressedSize(frame_data, frame_size);
    if (ZL_isError(compressed_size_report)) {
        printf("ERROR: Failed to get compressed size\n");
    } else {
        size_t compressed_size = ZL_validResult(compressed_size_report);
        printf("Compressed size (from frame): %zu bytes\n\n", compressed_size);
    }

    // Decompress
    uint8_t* decompressed = malloc(decompressed_size);
    ZL_Report result = ZL_decompress(
        decompressed,
        decompressed_size,
        frame_data,
        frame_size
    );

    if (ZL_isError(result)) {
        printf("ERROR: Decompression failed\n");
        free(frame_data);
        free(decompressed);
        return 1;
    }

    size_t actual_decompressed = ZL_validResult(result);
    printf("Decompressed %zu bytes successfully\n\n", actual_decompressed);

    print_hex(decompressed, actual_decompressed, "Decompressed data");

    printf("Decompressed as ASCII:\n");
    for (size_t i = 0; i < actual_decompressed; i++) {
        if (decompressed[i] >= 32 && decompressed[i] < 127) {
            printf("%c", decompressed[i]);
        } else {
            printf(".");
        }
    }
    printf("\n\n");

    free(frame_data);
    free(decompressed);

    printf("=================================\n");
    printf("Frame inspection complete!\n");
    printf("=================================\n");

    return 0;
}
