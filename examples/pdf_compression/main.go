package main

import (
	"fmt"
	"os"
	"time"

	"github.com/borischu/go-openzl"
	"github.com/klauspost/compress/zstd"
)

func main() {
	// Read the PDF file
	pdfPath := "../test.pdf"
	data, err := os.ReadFile(pdfPath)
	if err != nil {
		fmt.Printf("Error reading PDF: %v\n", err)
		return
	}

	fmt.Printf("PDF Compression Comparison\n")
	fmt.Printf("===========================\n\n")
	fmt.Printf("Original file: %s\n", pdfPath)
	fmt.Printf("Original size: %d bytes (%.2f KB)\n\n", len(data), float64(len(data))/1024)

	// Test OpenZL
	fmt.Printf("OpenZL Compression:\n")
	fmt.Printf("-------------------\n")
	startOpenZL := time.Now()
	compressedOpenZL, err := openzl.Compress(data)
	compressTimeOpenZL := time.Since(startOpenZL)
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
	} else {
		startDecomp := time.Now()
		decompressed, err := openzl.Decompress(compressedOpenZL)
		decompTimeOpenZL := time.Since(startDecomp)

		if err != nil {
			fmt.Printf("  Decompression error: %v\n", err)
		} else {
			ratio := float64(len(data)) / float64(len(compressedOpenZL))
			fmt.Printf("  Compressed size: %d bytes (%.2f KB)\n", len(compressedOpenZL), float64(len(compressedOpenZL))/1024)
			fmt.Printf("  Compression ratio: %.2fx\n", ratio)
			fmt.Printf("  Compress time: %v\n", compressTimeOpenZL)
			fmt.Printf("  Decompress time: %v\n", decompTimeOpenZL)
			fmt.Printf("  Round-trip time: %v\n", compressTimeOpenZL+decompTimeOpenZL)
			fmt.Printf("  Compress throughput: %.2f MB/s\n", float64(len(data))/1024/1024/compressTimeOpenZL.Seconds())
			fmt.Printf("  Decompress throughput: %.2f MB/s\n", float64(len(data))/1024/1024/decompTimeOpenZL.Seconds())

			// Verify data integrity
			if len(decompressed) != len(data) {
				fmt.Printf("  ⚠️  WARNING: Decompressed size mismatch\n")
			} else {
				match := true
				for i := range data {
					if data[i] != decompressed[i] {
						match = false
						break
					}
				}
				if match {
					fmt.Printf("  ✓ Data integrity verified\n")
				} else {
					fmt.Printf("  ⚠️  WARNING: Data mismatch\n")
				}
			}
		}
	}

	fmt.Printf("\nZstd Compression:\n")
	fmt.Printf("-----------------\n")
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		fmt.Printf("  Error creating encoder: %v\n", err)
		return
	}
	defer encoder.Close()

	startZstd := time.Now()
	compressedZstd := encoder.EncodeAll(data, nil)
	compressTimeZstd := time.Since(startZstd)

	decoder, err := zstd.NewReader(nil)
	if err != nil {
		fmt.Printf("  Error creating decoder: %v\n", err)
		return
	}
	defer decoder.Close()

	startDecompZstd := time.Now()
	decompressedZstd, err := decoder.DecodeAll(compressedZstd, nil)
	decompTimeZstd := time.Since(startDecompZstd)

	if err != nil {
		fmt.Printf("  Decompression error: %v\n", err)
	} else {
		ratio := float64(len(data)) / float64(len(compressedZstd))
		fmt.Printf("  Compressed size: %d bytes (%.2f KB)\n", len(compressedZstd), float64(len(compressedZstd))/1024)
		fmt.Printf("  Compression ratio: %.2fx\n", ratio)
		fmt.Printf("  Compress time: %v\n", compressTimeZstd)
		fmt.Printf("  Decompress time: %v\n", decompTimeZstd)
		fmt.Printf("  Round-trip time: %v\n", compressTimeZstd+decompTimeZstd)
		fmt.Printf("  Compress throughput: %.2f MB/s\n", float64(len(data))/1024/1024/compressTimeZstd.Seconds())
		fmt.Printf("  Decompress throughput: %.2f MB/s\n", float64(len(data))/1024/1024/decompTimeZstd.Seconds())

		// Verify data integrity
		if len(decompressedZstd) != len(data) {
			fmt.Printf("  ⚠️  WARNING: Decompressed size mismatch\n")
		} else {
			match := true
			for i := range data {
				if data[i] != decompressedZstd[i] {
					match = false
					break
				}
			}
			if match {
				fmt.Printf("  ✓ Data integrity verified\n")
			} else {
				fmt.Printf("  ⚠️  WARNING: Data mismatch\n")
			}
		}
	}

	// Comparison
	fmt.Printf("\nComparison Summary:\n")
	fmt.Printf("===================\n")
	if len(compressedOpenZL) > 0 && len(compressedZstd) > 0 {
		ratioOpenZL := float64(len(data)) / float64(len(compressedOpenZL))
		ratioZstd := float64(len(data)) / float64(len(compressedZstd))

		fmt.Printf("Compressed Size:\n")
		if len(compressedOpenZL) < len(compressedZstd) {
			savings := float64(len(compressedZstd)-len(compressedOpenZL)) / float64(len(compressedZstd)) * 100
			fmt.Printf("  OpenZL: %d bytes (%.2fx) ✓ Winner!\n", len(compressedOpenZL), ratioOpenZL)
			fmt.Printf("  Zstd:   %d bytes (%.2fx) - %.1f%% larger\n", len(compressedZstd), ratioZstd, savings)
		} else {
			savings := float64(len(compressedOpenZL)-len(compressedZstd)) / float64(len(compressedOpenZL)) * 100
			fmt.Printf("  OpenZL: %d bytes (%.2fx)\n", len(compressedOpenZL), ratioOpenZL)
			fmt.Printf("  Zstd:   %d bytes (%.2fx) ✓ Winner! - %.1f%% smaller\n", len(compressedZstd), ratioZstd, savings)
		}

		fmt.Printf("\nCompression Speed:\n")
		throughputOpenZL := float64(len(data)) / 1024 / 1024 / compressTimeOpenZL.Seconds()
		throughputZstd := float64(len(data)) / 1024 / 1024 / compressTimeZstd.Seconds()
		if throughputOpenZL > throughputZstd {
			speedup := throughputOpenZL / throughputZstd
			fmt.Printf("  OpenZL: %.2f MB/s ✓ Winner! (%.2fx faster)\n", throughputOpenZL, speedup)
			fmt.Printf("  Zstd:   %.2f MB/s\n", throughputZstd)
		} else {
			speedup := throughputZstd / throughputOpenZL
			fmt.Printf("  OpenZL: %.2f MB/s\n", throughputOpenZL)
			fmt.Printf("  Zstd:   %.2f MB/s ✓ Winner! (%.2fx faster)\n", throughputZstd, speedup)
		}
	}
}
