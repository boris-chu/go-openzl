// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package purgo_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/boris-chu/go-openzl/dicttrainer"
	"github.com/boris-chu/go-openzl/purgo"
	"github.com/klauspost/compress/zstd"
)

// TestCompressSmartWithDict_Pipeline tests Dictionary LZ77 → Huffman pipeline
func TestCompressSmartWithDict_Pipeline(t *testing.T) {
	t.Log("=== Testing Dictionary LZ77 → Huffman Pipeline ===\n")

	t.Run("CSV_WithTrainedDict", func(t *testing.T) {
		testCSVPipeline(t)
	})

	t.Run("JSON_WithTrainedDict", func(t *testing.T) {
		testJSONPipeline(t)
	})

	t.Run("LargeCSV_WithTrainedDict", func(t *testing.T) {
		testLargeCSVPipeline(t)
	})

	t.Run("SourceCode_WithTrainedDict", func(t *testing.T) {
		testSourceCodePipeline(t)
	})
}

func testCSVPipeline(t *testing.T) {
	t.Log("1. CSV Data with Trained Dictionary")

	// Load test-recovery-keys.csv
	data, err := os.ReadFile("../docs/test-recovery-keys.csv")
	if err != nil {
		t.Skip("test-recovery-keys.csv not found")
		return
	}

	t.Logf("   Input size: %d bytes (%.2f KB)", len(data), float64(len(data))/1024)

	// Benchmark 1: CompressSmart WITHOUT dictionary (LZ77 → Huffman)
	compressed1, err := purgo.CompressSmart(data)
	if err != nil {
		t.Fatalf("CompressSmart failed: %v", err)
	}
	ratio1 := float64(len(data)) / float64(len(compressed1))
	t.Logf("   LZ77→Huffman (no dict):   %d bytes → %.2f× compression", len(compressed1), ratio1)

	// Verify decompression
	decompressed1, err := purgo.Decompress(compressed1)
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}
	if !bytes.Equal(decompressed1, data) {
		t.Fatal("Decompressed data mismatch")
	}

	// Benchmark 2: Train custom dictionary and use CompressSmartWithDict
	t.Log("   Training custom 500-byte dictionary...")
	trainer := dicttrainer.New()
	trainer.AddData(data)
	dict := trainer.Train(500)
	t.Logf("   Dictionary trained: %d bytes", len(dict))

	compressed2, err := purgo.CompressSmartWithDict(data, dict)
	if err != nil {
		t.Fatalf("CompressSmartWithDict failed: %v", err)
	}
	ratio2 := float64(len(data)) / float64(len(compressed2))
	improvement := (ratio2 / ratio1) * 100
	t.Logf("   Dict+LZ77→Huffman (500B):  %d bytes → %.2f× compression (%.0f%% of no-dict)", len(compressed2), ratio2, improvement)

	// Verify decompression with dictionary
	decompressed2, err := purgo.Decompress(compressed2)
	if err != nil {
		t.Fatalf("Decompress with dict failed: %v", err)
	}
	if !bytes.Equal(decompressed2, data) {
		t.Fatal("Decompressed data with dict mismatch")
	}

	// Benchmark 3: Zstd Level 11 (for comparison)
	encoder, _ := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	compressed3 := encoder.EncodeAll(data, nil)
	encoder.Close()
	ratio3 := float64(len(data)) / float64(len(compressed3))
	t.Logf("   Zstd Level 11:             %d bytes → %.2f× compression", len(compressed3), ratio3)

	// Benchmark 4: Try pre-trained CSV dictionary if available
	csvDict, err := os.ReadFile("/tmp/csv-dict-30kb.bin")
	if err == nil {
		compressed4, err := purgo.CompressSmartWithDict(data, csvDict)
		if err == nil {
			ratio4 := float64(len(data)) / float64(len(compressed4))
			improvement4 := (ratio4 / ratio1) * 100
			t.Logf("   Dict+LZ77→Huffman (30KB):  %d bytes → %.2f× compression (%.0f%% of no-dict)", len(compressed4), ratio4, improvement4)

			// Verify decompression
			decompressed4, err := purgo.Decompress(compressed4)
			if err != nil {
				t.Errorf("Decompress with 30KB dict failed: %v", err)
			} else if !bytes.Equal(decompressed4, data) {
				t.Error("Decompressed data with 30KB dict mismatch")
			}
		}
	}

	t.Logf("\n   Summary:")
	t.Logf("   - Dictionary LZ77→Huffman pipeline working: ✅")
	t.Logf("   - Compression ratio: %.2f× (%.0f%% of Zstd)", ratio2, (ratio2/ratio3)*100)
	t.Logf("   - Roundtrip decompression: ✅")
}

func testJSONPipeline(t *testing.T) {
	t.Log("2. JSON Data with Trained Dictionary")

	// Realistic JSON with repeated field names
	data := []byte(`{"id":1,"name":"John Doe","email":"john@example.com","phone":"555-1234","address":"123 Main St","city":"New York","state":"NY","zip":"10001","country":"USA","status":"active","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-11-02T00:00:00Z","metadata":{"source":"api","version":"v1","tags":["customer","premium"]}}
{"id":2,"name":"Jane Smith","email":"jane@example.com","phone":"555-5678","address":"456 Oak Ave","city":"Los Angeles","state":"CA","zip":"90001","country":"USA","status":"active","created_at":"2024-01-15T00:00:00Z","updated_at":"2024-11-02T00:00:00Z","metadata":{"source":"api","version":"v1","tags":["customer","basic"]}}
{"id":3,"name":"Bob Johnson","email":"bob@example.com","phone":"555-9012","address":"789 Pine Rd","city":"Chicago","state":"IL","zip":"60601","country":"USA","status":"inactive","created_at":"2024-02-01T00:00:00Z","updated_at":"2024-11-02T00:00:00Z","metadata":{"source":"api","version":"v1","tags":["lead"]}}`)

	t.Logf("   Input size: %d bytes (%.2f KB)", len(data), float64(len(data))/1024)

	// Without dictionary
	compressed1, err := purgo.CompressSmart(data)
	if err != nil {
		t.Fatalf("CompressSmart failed: %v", err)
	}
	ratio1 := float64(len(data)) / float64(len(compressed1))
	t.Logf("   LZ77→Huffman (no dict):   %d bytes → %.2f× compression", len(compressed1), ratio1)

	// Train JSON dictionary
	trainer := dicttrainer.New()
	trainer.AddData(data)
	dict := trainer.Train(1024) // 1KB dict for JSON
	t.Logf("   Dictionary trained: %d bytes", len(dict))

	// With dictionary
	compressed2, err := purgo.CompressSmartWithDict(data, dict)
	if err != nil {
		t.Fatalf("CompressSmartWithDict failed: %v", err)
	}
	ratio2 := float64(len(data)) / float64(len(compressed2))
	improvement := (ratio2 / ratio1) * 100
	t.Logf("   Dict+LZ77→Huffman (1KB):  %d bytes → %.2f× compression (%.0f%% of no-dict)", len(compressed2), ratio2, improvement)

	// Verify decompression
	decompressed2, err := purgo.Decompress(compressed2)
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}
	if !bytes.Equal(decompressed2, data) {
		t.Fatal("Decompressed data mismatch")
	}

	// Zstd comparison
	encoder, _ := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	compressed3 := encoder.EncodeAll(data, nil)
	encoder.Close()
	ratio3 := float64(len(data)) / float64(len(compressed3))
	t.Logf("   Zstd Level 11:            %d bytes → %.2f× compression", len(compressed3), ratio3)

	// Try pre-trained JSON dict
	jsonDict, err := os.ReadFile("/tmp/json-dict-20kb.bin")
	if err == nil {
		compressed4, err := purgo.CompressSmartWithDict(data, jsonDict)
		if err == nil {
			ratio4 := float64(len(data)) / float64(len(compressed4))
			t.Logf("   Dict+LZ77→Huffman (20KB): %d bytes → %.2f× compression (%.0f%% of no-dict)", len(compressed4), ratio4, (ratio4/ratio1)*100)
		}
	}

	t.Logf("\n   Summary: Pipeline achieves %.2f× compression (%.0f%% of Zstd)", ratio2, (ratio2/ratio3)*100)
}

func testLargeCSVPipeline(t *testing.T) {
	t.Log("3. Large Repetitive CSV (125KB)")

	// Generate large CSV
	var sb strings.Builder
	sb.WriteString("id,name,email,phone,city,state,country,status\n")
	for i := 0; i < 1000; i++ {
		sb.WriteString("1,John Doe,john@example.com,555-1234,New York,NY,USA,active\n")
		sb.WriteString("2,Jane Smith,jane@example.com,555-5678,Los Angeles,CA,USA,active\n")
	}
	data := []byte(sb.String())

	t.Logf("   Input size: %d bytes (%.2f KB)", len(data), float64(len(data))/1024)

	// Without dictionary
	compressed1, err := purgo.CompressSmart(data)
	if err != nil {
		t.Fatalf("CompressSmart failed: %v", err)
	}
	ratio1 := float64(len(data)) / float64(len(compressed1))
	t.Logf("   LZ77→Huffman (no dict):   %d bytes → %.2f× compression", len(compressed1), ratio1)

	// Train dictionary
	trainer := dicttrainer.New()
	trainer.AddData(data)
	dict := trainer.Train(2048) // 2KB dict
	t.Logf("   Dictionary trained: %d bytes", len(dict))

	// With dictionary
	compressed2, err := purgo.CompressSmartWithDict(data, dict)
	if err != nil {
		t.Fatalf("CompressSmartWithDict failed: %v", err)
	}
	ratio2 := float64(len(data)) / float64(len(compressed2))
	t.Logf("   Dict+LZ77→Huffman (2KB):  %d bytes → %.2f× compression (%.0f%% of no-dict)", len(compressed2), ratio2, (ratio2/ratio1)*100)

	// Verify decompression
	decompressed2, err := purgo.Decompress(compressed2)
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}
	if !bytes.Equal(decompressed2, data) {
		t.Fatal("Decompressed data mismatch")
	}

	// Zstd comparison
	encoder, _ := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	compressed3 := encoder.EncodeAll(data, nil)
	encoder.Close()
	ratio3 := float64(len(data)) / float64(len(compressed3))
	t.Logf("   Zstd Level 11:            %d bytes → %.2f× compression", len(compressed3), ratio3)

	t.Logf("\n   Target: 20-30× compression")
	if ratio2 >= 20.0 {
		t.Logf("   ✅ Target achieved: %.2f× compression", ratio2)
	} else {
		t.Logf("   ⚠️  Below target: %.2f× compression (%.0f%% of target)", ratio2, (ratio2/20.0)*100)
	}
}

func testSourceCodePipeline(t *testing.T) {
	t.Log("4. TypeScript Source Code")

	// Sample TypeScript with repeated patterns
	data := []byte(`interface User {
  id: number;
  name: string;
  email: string;
  phone: string;
  address: string;
}

interface Product {
  id: number;
  name: string;
  price: number;
  category: string;
}

async function fetchUser(id: number): Promise<User> {
  const response = await fetch(\` + "`https://api.example.com/users/${id}`" + `);
  const data = await response.json();
  return data;
}

async function fetchProduct(id: number): Promise<Product> {
  const response = await fetch(\` + "`https://api.example.com/products/${id}`" + `);
  const data = await response.json();
  return data;
}

export { User, Product, fetchUser, fetchProduct };
`)

	t.Logf("   Input size: %d bytes", len(data))

	// Without dictionary
	compressed1, err := purgo.CompressSmart(data)
	if err != nil {
		t.Fatalf("CompressSmart failed: %v", err)
	}
	ratio1 := float64(len(data)) / float64(len(compressed1))
	t.Logf("   LZ77→Huffman (no dict):        %d bytes → %.2f× compression", len(compressed1), ratio1)

	// Train source code dictionary
	trainer := dicttrainer.New()
	trainer.AddData(data)
	dict := trainer.Train(512) // 512B dict
	t.Logf("   Dictionary trained: %d bytes", len(dict))

	// With dictionary
	compressed2, err := purgo.CompressSmartWithDict(data, dict)
	if err != nil {
		t.Fatalf("CompressSmartWithDict failed: %v", err)
	}
	ratio2 := float64(len(data)) / float64(len(compressed2))
	t.Logf("   Dict+LZ77→Huffman (512B):     %d bytes → %.2f× compression (%.0f%% of no-dict)", len(compressed2), ratio2, (ratio2/ratio1)*100)

	// Verify decompression
	decompressed2, err := purgo.Decompress(compressed2)
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}
	if !bytes.Equal(decompressed2, data) {
		t.Fatal("Decompressed data mismatch")
	}

	// Zstd comparison
	encoder, _ := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	compressed3 := encoder.EncodeAll(data, nil)
	encoder.Close()
	ratio3 := float64(len(data)) / float64(len(compressed3))
	t.Logf("   Zstd Level 11:                 %d bytes → %.2f× compression", len(compressed3), ratio3)

	t.Logf("\n   Summary: Pipeline achieves %.2f× compression (%.0f%% of Zstd)", ratio2, (ratio2/ratio3)*100)
}

// TestCompressSmartWithDict_Roundtrip verifies dictionary persistence in Frame v22
func TestCompressSmartWithDict_Roundtrip(t *testing.T) {
	t.Log("Testing dictionary persistence in Frame v22 format")

	data := []byte("The quick brown fox jumps over the lazy dog. The quick brown fox.")

	// Train dictionary
	trainer := dicttrainer.New()
	trainer.AddData(data)
	dict := trainer.Train(20)

	// Compress with dictionary
	compressed, err := purgo.CompressSmartWithDict(data, dict)
	if err != nil {
		t.Fatalf("CompressSmartWithDict failed: %v", err)
	}

	t.Logf("   Input: %d bytes", len(data))
	t.Logf("   Dictionary: %d bytes", len(dict))
	t.Logf("   Compressed: %d bytes", len(compressed))

	// Decompress (dictionary should be embedded in frame)
	decompressed, err := purgo.Decompress(compressed)
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}

	if !bytes.Equal(decompressed, data) {
		t.Fatal("Roundtrip decompression mismatch")
	}

	t.Log("   ✅ Dictionary roundtrip successful")
}

// TestCompressSmartWithDict_NilDictionary verifies nil dictionary behaves like CompressSmart
func TestCompressSmartWithDict_NilDictionary(t *testing.T) {
	data := []byte("Sample data for compression testing.")

	// Compress with nil dictionary
	compressed1, err := purgo.CompressSmartWithDict(data, nil)
	if err != nil {
		t.Fatalf("CompressSmartWithDict(nil) failed: %v", err)
	}

	// Compress with CompressSmart
	compressed2, err := purgo.CompressSmart(data)
	if err != nil {
		t.Fatalf("CompressSmart failed: %v", err)
	}

	// Both should produce same result
	if len(compressed1) != len(compressed2) {
		t.Errorf("Nil dictionary should behave like CompressSmart: %d vs %d bytes", len(compressed1), len(compressed2))
	}

	// Verify decompression
	decompressed, err := purgo.Decompress(compressed1)
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}

	if !bytes.Equal(decompressed, data) {
		t.Fatal("Decompressed data mismatch")
	}

	t.Log("   ✅ Nil dictionary works correctly")
}
