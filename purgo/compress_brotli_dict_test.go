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
)

// TestBrotliDictionary_Comparison tests Brotli's standard 120KB dictionary vs our trained dictionaries
func TestBrotliDictionary_Comparison(t *testing.T) {
	t.Log("=== Brotli Standard Dictionary vs Trained Dictionary ===\n")

	// Load Brotli's standard dictionary
	brotliDict, err := os.ReadFile("/tmp/brotli-dict.bin")
	if err != nil {
		t.Skip("Brotli dictionary not found (run: curl -o /tmp/brotli-dict.bin https://raw.githubusercontent.com/google/brotli/master/c/common/dictionary.bin)")
		return
	}

	t.Logf("Brotli dictionary: %d bytes (%.2f KB)\n", len(brotliDict), float64(len(brotliDict))/1024)

	t.Run("CSV_BrotliVsTrained", func(t *testing.T) {
		t.Log("Test 1: CSV Data - Brotli Dict vs Trained Dict")

		// Load test CSV
		csvData, err := os.ReadFile("../docs/test-recovery-keys.csv")
		if err != nil {
			t.Skip("test-recovery-keys.csv not found")
			return
		}

		t.Logf("   Input: %d bytes (%.2f KB)\n", len(csvData), float64(len(csvData))/1024)

		// Scenario 1: No dictionary (baseline)
		compressed1, _ := purgo.CompressSmart(csvData)
		ratio1 := float64(len(csvData)) / float64(len(compressed1))
		t.Logf("   1. CompressSmart (no dict):    %d bytes → %.2f× compression", len(compressed1), ratio1)

		// Scenario 2: Brotli standard dictionary (120KB)
		compressed2, err := purgo.CompressWithDict(csvData, brotliDict)
		if err != nil {
			t.Logf("   2. Brotli dict (120KB):        FAILED: %v", err)
		} else {
			ratio2 := float64(len(csvData)) / float64(len(compressed2))
			t.Logf("   2. Brotli dict (120KB):        %d bytes → %.2f× compression", len(compressed2), ratio2)

			// Verify decompression
			decompressed2, err := purgo.DecompressWithDict(compressed2, brotliDict)
			if err != nil {
				t.Errorf("   Decompression failed: %v", err)
			} else if !bytes.Equal(decompressed2, csvData) {
				t.Error("   Roundtrip mismatch with Brotli dict")
			} else {
				t.Log("   ✅ Roundtrip successful")
			}
		}

		// Scenario 3: Small trained dictionary (500 bytes)
		trainer := dicttrainer.New()
		trainer.AddData(csvData)
		trainedDict := trainer.Train(500)
		t.Logf("   Trained dictionary: %d bytes\n", len(trainedDict))

		compressed3, err := purgo.CompressWithDict(csvData, trainedDict)
		if err != nil {
			t.Fatalf("   3. Trained dict (500B):        FAILED: %v", err)
		}
		ratio3 := float64(len(csvData)) / float64(len(compressed3))
		t.Logf("   3. Trained dict (500B):        %d bytes → %.2f× compression", len(compressed3), ratio3)

		// Summary
		t.Log("\n   === COMPARISON ===")
		if len(compressed2) > 0 {
			brotliOverhead := len(compressed2) - len(compressed1)
			trainedOverhead := len(compressed3) - len(compressed1)
			t.Logf("   Brotli dict overhead:   %+d bytes (120KB dict in frame)", brotliOverhead)
			t.Logf("   Trained dict overhead:  %+d bytes (500B dict in frame)", trainedOverhead)

			if len(compressed3) < len(compressed2) {
				savings := len(compressed2) - len(compressed3)
				t.Logf("\n   ✅ Trained dict is %d bytes smaller (%.0f%% better)",
					savings, float64(savings)/float64(len(compressed2))*100)
			} else {
				t.Logf("\n   ⚠️  Brotli dict is smaller (unexpected!)")
			}
		}
	})

	t.Run("JSON_BrotliVsTrained", func(t *testing.T) {
		t.Log("\nTest 2: JSON Data - Brotli Dict vs Trained Dict")

		// Realistic JSON
		jsonData := []byte(`{"users":[{"id":1,"name":"John Doe","email":"john@example.com","phone":"555-1234","address":{"street":"123 Main St","city":"New York","state":"NY","zip":"10001"}},{"id":2,"name":"Jane Smith","email":"jane@example.com","phone":"555-5678","address":{"street":"456 Oak Ave","city":"Los Angeles","state":"CA","zip":"90001"}},{"id":3,"name":"Bob Johnson","email":"bob@example.com","phone":"555-9012","address":{"street":"789 Pine Rd","city":"Chicago","state":"IL","zip":"60601"}}]}`)

		t.Logf("   Input: %d bytes (%.2f KB)\n", len(jsonData), float64(len(jsonData))/1024)

		// No dictionary
		compressed1, _ := purgo.CompressSmart(jsonData)
		ratio1 := float64(len(jsonData)) / float64(len(compressed1))
		t.Logf("   1. No dict:           %d bytes → %.2f× compression", len(compressed1), ratio1)

		// Brotli dictionary
		compressed2, _ := purgo.CompressWithDict(jsonData, brotliDict)
		ratio2 := float64(len(jsonData)) / float64(len(compressed2))
		t.Logf("   2. Brotli (120KB):    %d bytes → %.2f× compression", len(compressed2), ratio2)

		// Trained dictionary
		trainer := dicttrainer.New()
		trainer.AddData(jsonData)
		trainedDict := trainer.Train(500)
		compressed3, _ := purgo.CompressWithDict(jsonData, trainedDict)
		ratio3 := float64(len(jsonData)) / float64(len(compressed3))
		t.Logf("   3. Trained (500B):    %d bytes → %.2f× compression", len(compressed3), ratio3)

		// Best result
		if ratio3 > ratio2 && ratio3 > ratio1 {
			t.Logf("\n   ✅ Trained dict wins: %.2f× vs Brotli %.2f× vs no-dict %.2f×", ratio3, ratio2, ratio1)
		} else if ratio2 > ratio3 && ratio2 > ratio1 {
			t.Logf("\n   ⚠️  Brotli dict wins: %.2f× vs Trained %.2f× vs no-dict %.2f×", ratio2, ratio3, ratio1)
		} else {
			t.Logf("\n   ⚠️  No dict wins: %.2f× vs Brotli %.2f× vs Trained %.2f×", ratio1, ratio2, ratio3)
		}
	})

	t.Run("TextData_BrotliVsTrained", func(t *testing.T) {
		t.Log("\nTest 3: English Text - Brotli Dict vs Trained Dict")

		// English text (where Brotli should be good - it's 59% English words)
		textData := []byte(`The quick brown fox jumps over the lazy dog. This is a test of the emergency broadcast system.
In computing, compression algorithms are used to reduce the size of data.
The most common algorithms include Huffman coding, LZ77, and arithmetic coding.
Brotli is a compression algorithm developed by Google that uses a dictionary of common English words.
The standard Brotli dictionary contains approximately 13,000 common words and phrases.
Dictionary-based compression works by replacing common patterns with references to a pre-built dictionary.
This approach is particularly effective for text data with repeated phrases and common vocabulary.
However, specialized dictionaries trained on specific data types often outperform generic dictionaries.`)

		t.Logf("   Input: %d bytes (%.2f KB)\n", len(textData), float64(len(textData))/1024)

		// No dictionary
		compressed1, _ := purgo.CompressSmart(textData)
		ratio1 := float64(len(textData)) / float64(len(compressed1))
		t.Logf("   1. No dict:           %d bytes → %.2f× compression", len(compressed1), ratio1)

		// Brotli dictionary (should be good for English text!)
		compressed2, _ := purgo.CompressWithDict(textData, brotliDict)
		ratio2 := float64(len(textData)) / float64(len(compressed2))
		t.Logf("   2. Brotli (120KB):    %d bytes → %.2f× compression", len(compressed2), ratio2)

		// Trained dictionary
		trainer := dicttrainer.New()
		trainer.AddData(textData)
		trainedDict := trainer.Train(500)
		compressed3, _ := purgo.CompressWithDict(textData, trainedDict)
		ratio3 := float64(len(textData)) / float64(len(compressed3))
		t.Logf("   3. Trained (500B):    %d bytes → %.2f× compression", len(compressed3), ratio3)

		// Analysis
		t.Log("\n   === ANALYSIS ===")
		t.Logf("   Brotli is 59%% English words (13,000 words)")
		t.Logf("   Should be optimal for English text")

		if ratio2 > ratio3 {
			t.Logf("\n   ✅ Brotli wins on English text: %.2f× vs %.2f× (as expected)", ratio2, ratio3)
		} else {
			t.Logf("\n   ⚠️  Trained dict beats Brotli even on English: %.2f× vs %.2f× (surprising!)", ratio3, ratio2)
		}
	})

	t.Run("BatchCompression_BrotliVsTrained", func(t *testing.T) {
		t.Log("\nTest 4: Batch Compression (10 CSV files) - Brotli vs Trained")

		// Generate 10 CSV files
		files := make([][]byte, 10)
		for i := 0; i < 10; i++ {
			var sb strings.Builder
			sb.WriteString("id,name,email,phone,city,state,status\n")
			for j := 0; j < 100; j++ {
				sb.WriteString("1,John Doe,john@example.com,555-1234,New York,NY,active\n")
			}
			files[i] = []byte(sb.String())
		}

		totalInputSize := len(files[0]) * 10
		t.Logf("   Input: 10 files × %d bytes = %d bytes (%.2f KB)\n",
			len(files[0]), totalInputSize, float64(totalInputSize)/1024)

		// Train dictionary on first file
		trainer := dicttrainer.New()
		trainer.AddData(files[0])
		trainedDict := trainer.Train(500)
		t.Logf("   Trained dictionary: %d bytes", len(trainedDict))
		t.Logf("   Brotli dictionary:  %d bytes (%.0f× larger)\n", len(brotliDict), float64(len(brotliDict))/float64(len(trainedDict)))

		// No dictionary
		totalSize1 := 0
		for i := 0; i < 10; i++ {
			compressed, _ := purgo.CompressSmart(files[i])
			totalSize1 += len(compressed)
		}
		ratio1 := float64(totalInputSize) / float64(totalSize1)
		t.Logf("   1. No dict:           %d bytes → %.2f× compression", totalSize1, ratio1)

		// Brotli dictionary (external)
		totalSize2 := 0
		for i := 0; i < 10; i++ {
			compressed, _ := purgo.CompressWithDict(files[i], brotliDict)
			totalSize2 += len(compressed)
		}
		effectiveSize2 := totalSize2 + len(brotliDict)
		ratio2 := float64(totalInputSize) / float64(effectiveSize2)
		t.Logf("   2. Brotli (external):  %d bytes + 120KB dict = %d bytes → %.2f× compression",
			totalSize2, effectiveSize2, ratio2)

		// Trained dictionary (external)
		totalSize3 := 0
		for i := 0; i < 10; i++ {
			compressed, _ := purgo.CompressWithDict(files[i], trainedDict)
			totalSize3 += len(compressed)
		}
		effectiveSize3 := totalSize3 + len(trainedDict)
		ratio3 := float64(totalInputSize) / float64(effectiveSize3)
		t.Logf("   3. Trained (external): %d bytes + 500B dict = %d bytes → %.2f× compression",
			totalSize3, effectiveSize3, ratio3)

		// Summary
		t.Log("\n   === SUMMARY ===")
		if ratio3 > ratio2 {
			improvement := (ratio3 / ratio2) - 1
			t.Logf("   ✅ Trained dict wins: %.2f× vs %.2f× (%.0f%% better)", ratio3, ratio2, improvement*100)
			t.Logf("   Reason: Trained on actual CSV data, not generic web content")
		} else {
			t.Logf("   ⚠️  Brotli dict wins: %.2f× vs %.2f×", ratio2, ratio3)
		}

		// Dictionary efficiency analysis
		t.Log("\n   === DICTIONARY EFFICIENCY ===")
		brotliOverhead := len(brotliDict)
		trainedOverhead := len(trainedDict)
		brotliSavings := totalSize1 - totalSize2
		trainedSavings := totalSize1 - totalSize3

		t.Logf("   Brotli:  120KB dict → %d bytes savings = %.2f× overhead/savings ratio",
			brotliSavings, float64(brotliOverhead)/float64(brotliSavings))
		t.Logf("   Trained: 500B dict → %d bytes savings = %.2f× overhead/savings ratio",
			trainedSavings, float64(trainedOverhead)/float64(trainedSavings))

		if float64(trainedOverhead)/float64(trainedSavings) < float64(brotliOverhead)/float64(brotliSavings) {
			t.Log("\n   ✅ Trained dict is more efficient (better overhead/savings ratio)")
		}
	})
}
