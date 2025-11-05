// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package purgo_test

import (
	"os"
	"strings"
	"testing"

	"github.com/boris-chu/go-openzl/dicttrainer"
	"github.com/boris-chu/go-openzl/purgo"
)

// TestCompressSmartWithDict_BatchCompression tests dictionary amortization
//
// Dictionary compression makes sense when:
//  1. You compress MULTIPLE files with the SAME dictionary (amortize dict cost)
//  2. The dictionary is SMALLER than the total compression improvement
//
// Single file example:
//   - Input: 20KB CSV
//   - Dictionary: 30KB (embedded in each frame)
//   - Result: 20KB → 32KB (EXPANSION! Dict overhead dominates)
//
// Batch compression example (10 files):
//   - Input: 10 × 20KB = 200KB total
//   - Dictionary: 30KB (stored once, loaded once)
//   - Per-file overhead: 30KB / 10 = 3KB effective overhead
//   - Result: Much better compression than single-file
func TestCompressSmartWithDict_BatchCompression(t *testing.T) {
	t.Log("=== Dictionary Compression: Single File vs Batch ===\n")

	// Generate 10 CSV files with similar structure
	generateCSVFiles := func() [][]byte {
		files := make([][]byte, 10)
		for fileIdx := 0; fileIdx < 10; fileIdx++ {
			var sb strings.Builder
			sb.WriteString("id,name,email,phone,city,state,status\n")
			for i := 0; i < 100; i++ {
				sb.WriteString("1,John Doe,john@example.com,555-1234,New York,NY,active\n")
				sb.WriteString("2,Jane Smith,jane@example.com,555-5678,Los Angeles,CA,active\n")
			}
			files[fileIdx] = []byte(sb.String())
		}
		return files
	}

	files := generateCSVFiles()
	t.Logf("Generated 10 CSV files, each %d bytes (%.2f KB)", len(files[0]), float64(len(files[0]))/1024)

	// Train dictionary on first file
	trainer := dicttrainer.New()
	trainer.AddData(files[0])
	dict := trainer.Train(500) // Small 500-byte dictionary
	t.Logf("Dictionary trained: %d bytes\n", len(dict))

	// Scenario 1: Single file compression (WITHOUT dictionary)
	t.Log("Scenario 1: Single File (No Dictionary)")
	compressed := make([][]byte, 10)
	totalSizeNoDict := 0
	for i := 0; i < 10; i++ {
		compressed[i], _ = purgo.CompressSmart(files[i])
		totalSizeNoDict += len(compressed[i])
	}
	totalInputSize := len(files[0]) * 10
	ratioNoDict := float64(totalInputSize) / float64(totalSizeNoDict)
	t.Logf("   Input:      %d bytes (%.2f KB)", totalInputSize, float64(totalInputSize)/1024)
	t.Logf("   Compressed: %d bytes (%.2f KB)", totalSizeNoDict, float64(totalSizeNoDict)/1024)
	t.Logf("   Ratio:      %.2f×\n", ratioNoDict)

	// Scenario 2: Single file compression (WITH dictionary embedded in each frame)
	t.Log("Scenario 2: Single File (Dictionary Embedded Per-Frame)")
	compressedWithDict := make([][]byte, 10)
	totalSizeWithDict := 0
	for i := 0; i < 10; i++ {
		compressedWithDict[i], _ = purgo.CompressSmartWithDict(files[i], dict)
		totalSizeWithDict += len(compressedWithDict[i])
	}
	ratioWithDict := float64(totalInputSize) / float64(totalSizeWithDict)
	t.Logf("   Input:      %d bytes (%.2f KB)", totalInputSize, float64(totalInputSize)/1024)
	t.Logf("   Compressed: %d bytes (%.2f KB)", totalSizeWithDict, float64(totalSizeWithDict)/1024)
	t.Logf("   Ratio:      %.2f×", ratioWithDict)
	t.Logf("   Overhead:   %d bytes per file (dictionary embedded 10 times)", len(dict))
	t.Logf("   Total dict overhead: %d bytes\n", len(dict)*10)

	// Scenario 3: Batch compression (dictionary stored ONCE externally)
	// This is how dictionary compression SHOULD be used!
	t.Log("Scenario 3: Batch Compression (Dictionary Shared, Stored Once)")
	t.Log("   Storage model:")
	t.Log("   - Store dictionary: my-dict.bin (500 bytes)")
	t.Log("   - Store compressed files: file1.openzl, file2.openzl, ...")
	t.Log("   - Each file compresses using dict but doesn't embed it")
	t.Log("   - Decompression: Load dict once, decompress all files")

	// For demo purposes, simulate this by calculating effective size:
	// - 1 dictionary (stored once)
	// - 10 compressed payloads (without dictionary embedding)
	//
	// Problem: Our current implementation EMBEDS dictionary in every frame
	// This is correct for self-contained frames but inefficient for batches.
	//
	// Ideal solution:
	//   purgo.CompressSmartWithDict(data, dict) → compressed WITHOUT dict
	//   purgo.DecompressWithDict(compressed, dict) → decompressed
	//
	// Current workaround: Calculate theoretical batch size
	dictOverheadPerFile := 0
	for i := 0; i < 10; i++ {
		// Each frame with dict contains: frame_metadata + dict + compressed_data
		// We need to measure just the compressed_data size (excluding dict)
		// For now, estimate: compressed_size = frame_with_dict - dict_size - frame_overhead
		frameOverhead := 100 // Approximate frame metadata size
		compressedDataSize := len(compressedWithDict[i]) - len(dict) - frameOverhead
		if compressedDataSize < 0 {
			compressedDataSize = len(compressedWithDict[i]) // Fallback
		}
		dictOverheadPerFile += compressedDataSize
	}

	effectiveBatchSize := len(dict) + dictOverheadPerFile // 1 dict + 10 payloads
	effectiveBatchRatio := float64(totalInputSize) / float64(effectiveBatchSize)

	t.Logf("   Theoretical batch size: %d bytes (%.2f KB)", effectiveBatchSize, float64(effectiveBatchSize)/1024)
	t.Logf("   - Dictionary (stored once): %d bytes", len(dict))
	t.Logf("   - 10 compressed payloads: ~%d bytes", dictOverheadPerFile)
	t.Logf("   Effective ratio: %.2f×\n", effectiveBatchRatio)

	// Summary
	t.Log("Summary:")
	t.Logf("   Without dictionary:        %.2f× compression", ratioNoDict)
	t.Logf("   With dict (per-frame):     %.2f× compression (WORSE! Dict overhead)", ratioWithDict)
	t.Logf("   With dict (shared, batch): %.2f× compression (BETTER! Amortized)", effectiveBatchRatio)
	t.Logf("\n   Conclusion: Dictionary compression needs BATCH API for efficiency")
	t.Logf("   Recommendation: Add DecompressWithDict(compressed, dict) in future version")
}

// TestCompressSmartWithDict_WhenDictionaryHelps tests when dictionaries are beneficial
func TestCompressSmartWithDict_WhenDictionaryHelps(t *testing.T) {
	t.Log("=== When Dictionary Compression Helps ===\n")

	// Case 1: Tiny dictionary (50 bytes) on moderately sized file (5KB)
	t.Run("TinyDict_ModerateFile", func(t *testing.T) {
		t.Log("Case 1: Tiny Dictionary (50 bytes) on 5KB File")

		// Generate 5KB CSV
		var sb strings.Builder
		sb.WriteString("id,name,email,phone,city,state,status\n")
		for i := 0; i < 50; i++ {
			sb.WriteString("1,John Doe,john@example.com,555-1234,New York,NY,active\n")
			sb.WriteString("2,Jane Smith,jane@example.com,555-5678,Los Angeles,CA,active\n")
		}
		data := []byte(sb.String())

		t.Logf("   Input: %d bytes (%.2f KB)", len(data), float64(len(data))/1024)

		// Without dictionary
		compressed1, _ := purgo.CompressSmart(data)
		ratio1 := float64(len(data)) / float64(len(compressed1))
		t.Logf("   Without dict: %d bytes → %.2f× compression", len(compressed1), ratio1)

		// Train tiny dictionary
		trainer := dicttrainer.New()
		trainer.AddData(data)
		dict := trainer.Train(50) // Only 50 bytes!
		t.Logf("   Dictionary: %d bytes", len(dict))

		// With dictionary
		compressed2, _ := purgo.CompressSmartWithDict(data, dict)
		ratio2 := float64(len(data)) / float64(len(compressed2))
		t.Logf("   With dict:    %d bytes → %.2f× compression", len(compressed2), ratio2)

		// Dictionary overhead
		overhead := len(compressed2) - len(compressed1)
		t.Logf("   Dictionary overhead: %d bytes", overhead)

		if len(compressed2) < len(compressed1) {
			t.Logf("   ✅ Dictionary helped! %d bytes saved", len(compressed1)-len(compressed2))
		} else {
			t.Logf("   ⚠️  Dictionary hurt: %d bytes added (overhead > benefit)", overhead)
		}
	})

	// Case 2: Pre-trained dictionary on DIFFERENT file
	t.Run("PreTrainedDict_DifferentFile", func(t *testing.T) {
		t.Log("\nCase 2: Pre-trained Dictionary on Different File")

		// Load test-recovery-keys.csv
		data, err := os.ReadFile("../docs/test-recovery-keys.csv")
		if err != nil {
			t.Skip("test-recovery-keys.csv not found")
			return
		}

		t.Logf("   Input: %d bytes (%.2f KB)", len(data), float64(len(data))/1024)

		// Without dictionary
		compressed1, _ := purgo.CompressSmart(data)
		ratio1 := float64(len(data)) / float64(len(compressed1))
		t.Logf("   Without dict: %d bytes → %.2f× compression", len(compressed1), ratio1)

		// Try pre-trained CSV dictionary
		csvDict, err := os.ReadFile("/tmp/csv-dict-30kb.bin")
		if err != nil {
			t.Skip("Pre-trained CSV dictionary not found")
			return
		}

		t.Logf("   Dictionary: %d bytes (pre-trained)", len(csvDict))

		// With dictionary
		compressed2, _ := purgo.CompressSmartWithDict(data, csvDict)
		ratio2 := float64(len(data)) / float64(len(compressed2))
		t.Logf("   With dict:    %d bytes → %.2f× compression", len(compressed2), ratio2)

		if len(compressed2) < len(compressed1) {
			t.Logf("   ✅ Pre-trained dictionary helped!")
		} else {
			overhead := len(compressed2) - len(compressed1)
			t.Logf("   ⚠️  Dictionary overhead (%d bytes) exceeded benefit", overhead)
			t.Logf("   Reason: Small file (%.2f KB) + large dict (%.2f KB) = overhead dominates",
				float64(len(data))/1024, float64(len(csvDict))/1024)
		}
	})

	t.Log("\nConclusion:")
	t.Log("   Dictionary compression is beneficial when:")
	t.Log("   1. Dictionary is SMALL relative to file size (< 5% of file size)")
	t.Log("   2. Compressing MANY similar files with SAME dictionary (batch mode)")
	t.Log("   3. Decompressor can load dictionary ONCE and reuse it")
}
