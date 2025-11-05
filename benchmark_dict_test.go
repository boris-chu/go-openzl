// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package openzl_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/boris-chu/go-openzl/dicttrainer"
	"github.com/boris-chu/go-openzl/internal/codec"
	"github.com/klauspost/compress/zstd"
)

// TestDictionaryCompressionBenchmark compares dictionary LZ77 vs Zstd on various datasets
func TestDictionaryCompressionBenchmark(t *testing.T) {
	t.Log("=== OpenZL Dictionary vs Zstd Compression Benchmark ===\n")

	// Test 1: test-recovery-keys.csv
	t.Run("RecoveryKeysCSV", func(t *testing.T) {
		benchmarkRecoveryKeys(t)
	})

	// Test 2: Large CSV
	t.Run("LargeCSV", func(t *testing.T) {
		benchmarkLargeCSV(t)
	})

	// Test 3: JSON
	t.Run("JSON", func(t *testing.T) {
		benchmarkJSON(t)
	})

	// Test 4: Source Code
	t.Run("SourceCode", func(t *testing.T) {
		benchmarkSourceCode(t)
	})
}

func benchmarkRecoveryKeys(t *testing.T) {
	t.Log("1. Recovery Keys CSV (test-recovery-keys.csv)")

	// Load data
	data, err := os.ReadFile("docs/test-recovery-keys.csv")
	if err != nil {
		t.Skip("test-recovery-keys.csv not found")
		return
	}

	t.Logf("   Input size: %d bytes (%.2f KB)", len(data), float64(len(data))/1024)

	// Benchmark 1: LZ77 WITHOUT dictionary
	lz77 := codec.NewLZ77()
	dst1 := make([]byte, len(data)*2)
	compressed1, err := lz77.Encode(dst1, data, nil)
	if err != nil {
		t.Fatalf("LZ77 encode error: %v", err)
	}
	ratio1 := float64(len(data)) / float64(compressed1)
	t.Logf("   LZ77 (no dict):  %d bytes (%.2f KB) → %.2f× compression", compressed1, float64(compressed1)/1024, ratio1)

	// Benchmark 2: LZ77 WITH trained dictionary
	t.Log("   Training custom dictionary...")
	trainer := dicttrainer.New()
	trainer.AddData(data)
	dict := trainer.Train(500) // Small 500-byte dictionary
	t.Logf("   Dictionary size: %d bytes", len(dict))

	lz77dict := codec.NewLZ77WithDict(dict)
	dst2 := make([]byte, len(data)*2)
	compressed2, err := lz77dict.Encode(dst2, data, nil)
	if err != nil {
		t.Fatalf("LZ77+dict encode error: %v", err)
	}
	ratio2 := float64(len(data)) / float64(compressed2)
	improvement2 := (ratio2 / ratio1) * 100
	t.Logf("   LZ77 (500B dict): %d bytes (%.2f KB) → %.2f× compression (%.0f%% better)", compressed2, float64(compressed2)/1024, ratio2, improvement2-100)

	// Benchmark 3: Zstd Level 11 (similar to Brotli Level 11)
	encoder, _ := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	compressed3 := encoder.EncodeAll(data, nil)
	encoder.Close()
	ratio3 := float64(len(data)) / float64(len(compressed3))
	t.Logf("   Zstd Level 11:    %d bytes (%.2f KB) → %.2f× compression", len(compressed3), float64(len(compressed3))/1024, ratio3)

	// Benchmark 4: LZ77 with pre-trained CSV dictionary (30KB)
	csvDict, err := os.ReadFile("/tmp/csv-dict-30kb.bin")
	if err == nil {
		lz77csvdict := codec.NewLZ77WithDict(csvDict)
		dst4 := make([]byte, len(data)*2)
		compressed4, err := lz77csvdict.Encode(dst4, data, nil)
		if err == nil {
			ratio4 := float64(len(data)) / float64(compressed4)
			improvement4 := (ratio4 / ratio1) * 100
			t.Logf("   LZ77 (30KB dict): %d bytes (%.2f KB) → %.2f× compression (%.0f%% better)", compressed4, float64(compressed4)/1024, ratio4, improvement4-100)
		}
	}

	// Summary
	t.Logf("\n   Summary:")
	t.Logf("   - Custom 500B dict: %.0f%% improvement over no dict", (ratio2/ratio1-1)*100)
	t.Logf("   - Zstd Level 11: %.2f× compression", ratio3)
}

func benchmarkLargeCSV(t *testing.T) {
	t.Log("2. Large Repetitive CSV (125KB)")

	// Generate large CSV
	var sb strings.Builder
	sb.WriteString("id,name,email,phone,city,state,country,status\n")
	for i := 0; i < 1000; i++ {
		sb.WriteString("1,John Doe,john@example.com,555-1234,New York,NY,USA,active\n")
		sb.WriteString("2,Jane Smith,jane@example.com,555-5678,Los Angeles,CA,USA,active\n")
	}
	data := []byte(sb.String())

	t.Logf("   Input size: %d bytes (%.2f KB)", len(data), float64(len(data))/1024)

	// LZ77 without dict
	lz77 := codec.NewLZ77()
	dst1 := make([]byte, len(data)*2)
	compressed1, _ := lz77.Encode(dst1, data, nil)
	ratio1 := float64(len(data)) / float64(compressed1)
	t.Logf("   LZ77 (no dict):   %d bytes (%.2f KB) → %.2f× compression", compressed1, float64(compressed1)/1024, ratio1)

	// LZ77 with trained dict
	trainer := dicttrainer.New()
	trainer.AddData(data)
	dict := trainer.Train(1024) // 1KB dict
	lz77dict := codec.NewLZ77WithDict(dict)
	dst2 := make([]byte, len(data)*2)
	compressed2, _ := lz77dict.Encode(dst2, data, nil)
	ratio2 := float64(len(data)) / float64(compressed2)
	t.Logf("   LZ77 (1KB dict):  %d bytes (%.2f KB) → %.2f× compression", compressed2, float64(compressed2)/1024, ratio2)

	// Zstd
	encoder, _ := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	compressed3 := encoder.EncodeAll(data, nil)
	encoder.Close()
	ratio3 := float64(len(data)) / float64(len(compressed3))
	t.Logf("   Zstd Level 11:    %d bytes (%.2f KB) → %.2f× compression", len(compressed3), float64(len(compressed3))/1024, ratio3)

	t.Logf("\n   Summary: LZ77+dict achieves %.0f%% of Zstd compression", (ratio2/ratio3)*100)
}

func benchmarkJSON(t *testing.T) {
	t.Log("3. JSON API Response")

	// Realistic JSON
	data := []byte(`{"id":1,"name":"John Doe","email":"john@example.com","phone":"555-1234","address":"123 Main St","city":"New York","state":"NY","zip":"10001","country":"USA","status":"active","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-11-02T00:00:00Z","metadata":{"source":"api","version":"v1","tags":["customer","premium"]}}`)

	t.Logf("   Input size: %d bytes", len(data))

	// LZ77 without dict
	lz77 := codec.NewLZ77()
	dst1 := make([]byte, len(data)*2)
	compressed1, _ := lz77.Encode(dst1, data, nil)
	ratio1 := float64(len(data)) / float64(compressed1)
	t.Logf("   LZ77 (no dict):    %d bytes → %.2f× compression", compressed1, ratio1)

	// LZ77 with JSON dict
	jsonDict, err := os.ReadFile("/tmp/json-dict-20kb.bin")
	if err == nil {
		lz77dict := codec.NewLZ77WithDict(jsonDict)
		dst2 := make([]byte, len(data)*2)
		compressed2, _ := lz77dict.Encode(dst2, data, nil)
		ratio2 := float64(len(data)) / float64(compressed2)
		t.Logf("   LZ77 (JSON dict):  %d bytes → %.2f× compression", compressed2, ratio2)
	}

	// Zstd
	encoder, _ := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	compressed3 := encoder.EncodeAll(data, nil)
	encoder.Close()
	ratio3 := float64(len(data)) / float64(len(compressed3))
	t.Logf("   Zstd Level 11:     %d bytes → %.2f× compression", len(compressed3), ratio3)
}

func benchmarkSourceCode(t *testing.T) {
	t.Log("4. TypeScript Source Code")

	// Sample TypeScript
	data := []byte(`
interface User {
  id: number;
  name: string;
  email: string;
}

async function fetchUser(id: number): Promise<User> {
  const response = await fetch("https://api.example.com/users/" + id);
  const data = await response.json();
  return data;
}

export { User, fetchUser };
`)

	t.Logf("   Input size: %d bytes", len(data))

	// LZ77 without dict
	lz77 := codec.NewLZ77()
	dst1 := make([]byte, len(data)*2)
	compressed1, _ := lz77.Encode(dst1, data, nil)
	ratio1 := float64(len(data)) / float64(compressed1)
	t.Logf("   LZ77 (no dict):       %d bytes → %.2f× compression", compressed1, ratio1)

	// LZ77 with source code dict
	codeDict, err := os.ReadFile("/tmp/source-code-dict-40kb.bin")
	if err == nil {
		lz77dict := codec.NewLZ77WithDict(codeDict)
		dst2 := make([]byte, len(data)*2)
		compressed2, _ := lz77dict.Encode(dst2, data, nil)
		ratio2 := float64(len(data)) / float64(compressed2)
		t.Logf("   LZ77 (source dict):  %d bytes → %.2f× compression", compressed2, ratio2)
	}

	// Zstd
	encoder, _ := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	compressed3 := encoder.EncodeAll(data, nil)
	encoder.Close()
	ratio3 := float64(len(data)) / float64(len(compressed3))
	t.Logf("   Zstd Level 11:        %d bytes → %.2f× compression", len(compressed3), ratio3)
}

// Example function to demonstrate public API usage
func ExampleTrainer() {
	// Train a dictionary on CSV data
	trainer := dicttrainer.New()
	trainer.AddData([]byte("id,name,email\n1,John,john@example.com\n2,Jane,jane@example.com"))
	dict := trainer.Train(100) // 100-byte dictionary

	// Save dictionary
	os.WriteFile("my-dict.bin", dict, 0644)

	// Use with LZ77
	lz77 := codec.NewLZ77WithDict(dict)
	dst := make([]byte, 1024)
	data := []byte("3,Bob,bob@example.com")
	compressed, _ := lz77.Encode(dst, data, nil)

	fmt.Printf("Compressed %d bytes to %d bytes\n", len(data), compressed)
}
