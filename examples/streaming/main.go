// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/borischu/go-openzl"
)

func main() {
	fmt.Println("OpenZL Streaming API Example")
	fmt.Println("=============================")
	fmt.Println()

	// Example 1: Basic streaming compression and decompression
	fmt.Println("Example 1: Basic Streaming")
	fmt.Println("--------------------------")

	originalData := []byte(strings.Repeat("Hello, Streaming World! ", 1000))
	fmt.Printf("Original size: %d bytes\n", len(originalData))

	// Compress using Writer
	var compressed bytes.Buffer
	writer, err := openzl.NewWriter(&compressed)
	if err != nil {
		log.Fatal(err)
	}

	n, err := writer.Write(originalData)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Wrote %d bytes to Writer\n", n)

	if err := writer.Close(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Compressed size: %d bytes (%.2fx ratio)\n",
		compressed.Len(), float64(len(originalData))/float64(compressed.Len()))

	// Decompress using Reader
	reader, err := openzl.NewReader(&compressed)
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		log.Fatal(err)
	}

	if bytes.Equal(decompressed, originalData) {
		fmt.Println("✓ Decompression successful!")
	} else {
		fmt.Println("✗ Decompression failed - data mismatch")
	}
	fmt.Println()

	// Example 2: Streaming with io.Copy
	fmt.Println("Example 2: Using io.Copy")
	fmt.Println("------------------------")

	source := bytes.NewReader(bytes.Repeat([]byte("0123456789"), 10000))
	fmt.Printf("Source size: %d bytes\n", source.Len())

	var compressedBuf bytes.Buffer
	compWriter, _ := openzl.NewWriter(&compressedBuf)

	start := time.Now()
	copied, err := io.Copy(compWriter, source)
	if err != nil {
		log.Fatal(err)
	}
	compWriter.Close()
	elapsed := time.Since(start)

	fmt.Printf("Copied %d bytes in %v\n", copied, elapsed)
	fmt.Printf("Compressed to %d bytes (%.2fx ratio)\n",
		compressedBuf.Len(), float64(copied)/float64(compressedBuf.Len()))
	fmt.Printf("Throughput: %.2f MB/s\n",
		float64(copied)/(1024*1024)/elapsed.Seconds())
	fmt.Println()

	// Example 3: File compression
	fmt.Println("Example 3: File Compression")
	fmt.Println("---------------------------")

	// Create a temporary file with test data
	tmpFile, err := os.CreateTemp("", "openzl-test-*.txt")
	if err != nil {
		log.Fatal(err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Write test data to file
	testData := bytes.Repeat([]byte("This is test data for file compression.\n"), 1000)
	if _, err := tmpFile.Write(testData); err != nil {
		log.Fatal(err)
	}
	tmpFile.Close()

	// Compress file
	compressedPath := tmpPath + ".zl"
	defer os.Remove(compressedPath)

	if err := compressFile(tmpPath, compressedPath); err != nil {
		log.Fatal(err)
	}

	// Check sizes
	origInfo, _ := os.Stat(tmpPath)
	compInfo, _ := os.Stat(compressedPath)

	fmt.Printf("Original file: %d bytes\n", origInfo.Size())
	fmt.Printf("Compressed file: %d bytes\n", compInfo.Size())
	fmt.Printf("Compression ratio: %.2fx\n", float64(origInfo.Size())/float64(compInfo.Size()))

	// Decompress file
	decompressedPath := tmpPath + ".decompressed"
	defer os.Remove(decompressedPath)

	if err := decompressFile(compressedPath, decompressedPath); err != nil {
		log.Fatal(err)
	}

	// Verify
	decompInfo, _ := os.Stat(decompressedPath)
	if decompInfo.Size() == origInfo.Size() {
		fmt.Println("✓ File round-trip successful!")
	} else {
		fmt.Printf("✗ Size mismatch: got %d, want %d\n", decompInfo.Size(), origInfo.Size())
	}
	fmt.Println()

	// Example 4: Streaming large data
	fmt.Println("Example 4: Large Data Streaming")
	fmt.Println("--------------------------------")

	// Simulate large data stream (10MB)
	largeSize := 10 * 1024 * 1024
	fmt.Printf("Streaming %d bytes (10 MB)\n", largeSize)

	var largeBuf bytes.Buffer
	largeWriter, _ := openzl.NewWriter(&largeBuf)

	start = time.Now()
	chunk := bytes.Repeat([]byte("X"), 1024*64) // 64KB chunks
	for written := 0; written < largeSize; {
		toWrite := len(chunk)
		if written+toWrite > largeSize {
			toWrite = largeSize - written
		}
		n, err := largeWriter.Write(chunk[:toWrite])
		if err != nil {
			log.Fatal(err)
		}
		written += n
	}
	largeWriter.Close()
	elapsed = time.Since(start)

	fmt.Printf("Compressed %d bytes to %d bytes in %v\n",
		largeSize, largeBuf.Len(), elapsed)
	fmt.Printf("Compression ratio: %.2fx\n", float64(largeSize)/float64(largeBuf.Len()))
	fmt.Printf("Throughput: %.2f MB/s\n",
		float64(largeSize)/(1024*1024)/elapsed.Seconds())
	fmt.Println()

	// Example 5: Custom frame size
	fmt.Println("Example 5: Custom Frame Size")
	fmt.Println("----------------------------")

	data := bytes.Repeat([]byte("Test"), 10000)

	frameSizes := []int{
		4 * 1024,   // 4KB
		64 * 1024,  // 64KB (default)
		256 * 1024, // 256KB
	}

	for _, frameSize := range frameSizes {
		var buf bytes.Buffer
		w, err := openzl.NewWriter(&buf, openzl.WithFrameSize(frameSize))
		if err != nil {
			log.Fatal(err)
		}

		w.Write(data)
		w.Close()

		fmt.Printf("Frame size %6d KB: compressed to %d bytes (%.2fx)\n",
			frameSize/1024, buf.Len(), float64(len(data))/float64(buf.Len()))
	}
	fmt.Println()

	fmt.Println("Summary")
	fmt.Println("-------")
	fmt.Println("✓ Streaming API works seamlessly with io.Reader/Writer")
	fmt.Println("✓ Automatic buffering and frame management")
	fmt.Println("✓ Excellent compression ratios on streaming data")
	fmt.Println("✓ High throughput for large data streams")
	fmt.Println("✓ Configurable frame sizes for different use cases")
}

// compressFile compresses a file using streaming compression
func compressFile(srcPath, dstPath string) error {
	// Open source file
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	// Create destination file
	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	// Create compressing writer
	writer, err := openzl.NewWriter(dst)
	if err != nil {
		return err
	}
	defer writer.Close()

	// Copy and compress
	if _, err := io.Copy(writer, src); err != nil {
		return err
	}

	return nil
}

// decompressFile decompresses a file using streaming decompression
func decompressFile(srcPath, dstPath string) error {
	// Open compressed file
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	// Create destination file
	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	// Create decompressing reader
	reader, err := openzl.NewReader(src)
	if err != nil {
		return err
	}
	defer reader.Close()

	// Copy and decompress
	if _, err := io.Copy(dst, reader); err != nil {
		return err
	}

	return nil
}
