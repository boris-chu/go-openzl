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

// TestCompressWithDict_ExternalDictionary tests external dictionary compression
func TestCompressWithDict_ExternalDictionary(t *testing.T) {
	t.Log("=== External Dictionary Compression (CompressWithDict) ===\n")

	t.Run("SingleFile_WithExternalDict", func(t *testing.T) {
		t.Log("Test 1: Single File with External Dictionary")

		// Generate CSV data
		var sb strings.Builder
		sb.WriteString("id,name,email,phone,city,state,status\n")
		for i := 0; i < 10; i++ {
			sb.WriteString("1,John Doe,john@example.com,555-1234,New York,NY,active\n")
			sb.WriteString("2,Jane Smith,jane@example.com,555-5678,Los Angeles,CA,active\n")
		}
		data := []byte(sb.String())

		t.Logf("   Input: %d bytes (%.2f KB)", len(data), float64(len(data))/1024)

		// Train tiny dictionary
		trainer := dicttrainer.New()
		trainer.AddData(data)
		dict := trainer.Train(50) // Only 50 bytes
		t.Logf("   Dictionary: %d bytes\n", len(dict))

		// Compress WITHOUT dictionary
		compressed1, err := purgo.CompressSmart(data)
		if err != nil {
			t.Fatalf("CompressSmart failed: %v", err)
		}
		ratio1 := float64(len(data)) / float64(len(compressed1))
		t.Logf("   CompressSmart (no dict):    %d bytes → %.2f× compression", len(compressed1), ratio1)

		// Compress WITH embedded dictionary
		compressed2, err := purgo.CompressSmartWithDict(data, dict)
		if err != nil {
			t.Fatalf("CompressSmartWithDict failed: %v", err)
		}
		ratio2 := float64(len(data)) / float64(len(compressed2))
		t.Logf("   CompressSmartWithDict:      %d bytes → %.2f× compression (dict embedded)", len(compressed2), ratio2)

		// Compress WITH external dictionary
		compressed3, err := purgo.CompressWithDict(data, dict)
		if err != nil {
			t.Fatalf("CompressWithDict failed: %v", err)
		}
		ratio3 := float64(len(data)) / float64(len(compressed3))
		t.Logf("   CompressWithDict (external): %d bytes → %.2f× compression (dict NOT embedded)", len(compressed3), ratio3)

		// Decompress with external dictionary
		decompressed, err := purgo.DecompressWithDict(compressed3, dict)
		if err != nil {
			t.Fatalf("DecompressWithDict failed: %v", err)
		}

		if !bytes.Equal(decompressed, data) {
			t.Fatal("Roundtrip decompression mismatch")
		}

		t.Logf("\n   Analysis:")
		t.Logf("   - External dict is %d bytes smaller than embedded", len(compressed2)-len(compressed3))
		t.Logf("   - Savings from NOT embedding: %.0f%%", float64(len(compressed2)-len(compressed3))/float64(len(compressed2))*100)

		if len(compressed3) < len(compressed2) {
			t.Logf("   ✅ External dictionary is smaller!")
		} else {
			t.Logf("   ⚠️  External dictionary not smaller (frame overhead)")
		}
	})

	t.Run("BatchCompression_10Files", func(t *testing.T) {
		t.Log("\nTest 2: Batch Compression (10 Files, 1 Shared Dictionary)")

		// Generate 10 CSV files
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

		totalInputSize := len(files[0]) * 10
		t.Logf("   Input: 10 files × %d bytes = %d bytes (%.2f KB)\n",
			len(files[0]), totalInputSize, float64(totalInputSize)/1024)

		// Train dictionary
		trainer := dicttrainer.New()
		trainer.AddData(files[0])
		dict := trainer.Train(500) // 500-byte dictionary
		t.Logf("   Dictionary: %d bytes (stored ONCE)\n", len(dict))

		// Scenario 1: CompressSmart (no dictionary)
		totalSize1 := 0
		for i := 0; i < 10; i++ {
			compressed, _ := purgo.CompressSmart(files[i])
			totalSize1 += len(compressed)
		}
		ratio1 := float64(totalInputSize) / float64(totalSize1)
		t.Logf("   Scenario 1: CompressSmart (no dict)")
		t.Logf("   - Total compressed: %d bytes (%.2f KB)", totalSize1, float64(totalSize1)/1024)
		t.Logf("   - Ratio: %.2f×\n", ratio1)

		// Scenario 2: CompressSmartWithDict (dictionary embedded in EACH file)
		totalSize2 := 0
		for i := 0; i < 10; i++ {
			compressed, _ := purgo.CompressSmartWithDict(files[i], dict)
			totalSize2 += len(compressed)
		}
		ratio2 := float64(totalInputSize) / float64(totalSize2)
		dictOverhead2 := totalSize2 - totalSize1
		t.Logf("   Scenario 2: CompressSmartWithDict (dict embedded per-file)")
		t.Logf("   - Total compressed: %d bytes (%.2f KB)", totalSize2, float64(totalSize2)/1024)
		t.Logf("   - Dictionary overhead: %d bytes (%.2f KB, embedded 10 times)", dictOverhead2, float64(dictOverhead2)/1024)
		t.Logf("   - Ratio: %.2f×\n", ratio2)

		// Scenario 3: CompressWithDict (dictionary stored ONCE externally)
		compressedFiles := make([][]byte, 10)
		totalSize3 := 0
		for i := 0; i < 10; i++ {
			var err error
			compressedFiles[i], err = purgo.CompressWithDict(files[i], dict)
			if err != nil {
				t.Fatalf("CompressWithDict file %d failed: %v", i, err)
			}
			totalSize3 += len(compressedFiles[i])
		}
		effectiveSize3 := totalSize3 + len(dict) // Add dictionary (stored once)
		ratio3 := float64(totalInputSize) / float64(effectiveSize3)
		t.Logf("   Scenario 3: CompressWithDict (external dict, stored ONCE)")
		t.Logf("   - 10 compressed files: %d bytes (%.2f KB)", totalSize3, float64(totalSize3)/1024)
		t.Logf("   - Dictionary (stored once): %d bytes (%.2f KB)", len(dict), float64(len(dict))/1024)
		t.Logf("   - Total storage: %d bytes (%.2f KB)", effectiveSize3, float64(effectiveSize3)/1024)
		t.Logf("   - Ratio: %.2f×\n", ratio3)

		// Verify all files decompress correctly
		t.Log("   Verifying roundtrip decompression...")
		for i := 0; i < 10; i++ {
			decompressed, err := purgo.DecompressWithDict(compressedFiles[i], dict)
			if err != nil {
				t.Fatalf("DecompressWithDict file %d failed: %v", i, err)
			}
			if !bytes.Equal(decompressed, files[i]) {
				t.Fatalf("File %d roundtrip mismatch", i)
			}
		}
		t.Log("   ✅ All 10 files decompress correctly\n")

		// Summary
		t.Log("   === RESULTS ===")
		t.Logf("   CompressSmart (no dict):         %.2f× compression", ratio1)
		t.Logf("   CompressSmartWithDict (embedded): %.2f× compression (WORSE! Dict overhead)", ratio2)
		t.Logf("   CompressWithDict (external):      %.2f× compression", ratio3)

		if ratio3 > ratio1 {
			improvement := (ratio3 / ratio1) - 1
			t.Logf("\n   ✅ EXTERNAL DICTIONARY WINS!")
			t.Logf("   Improvement over no-dict: %.0f%% better", improvement*100)
			t.Logf("   Improvement over embedded: %.0f%% better", ((ratio3/ratio2)-1)*100)
		} else {
			t.Logf("\n   ⚠️  External dictionary not better than no-dict")
			t.Logf("   Reason: Not enough files to amortize dictionary overhead")
		}

		// Target verification
		if ratio3 >= 50.0 {
			t.Logf("\n   🔥 Target achieved: %.2f× compression (target was 50-70×)", ratio3)
		} else if ratio3 >= 36.0 {
			t.Logf("\n   ⚠️  Below target: %.2f× compression (target was 50-70×)", ratio3)
			t.Logf("   Still better than CompressSmart (%.2f×)", ratio1)
		} else {
			t.Logf("\n   ❌ Below no-dict baseline: %.2f× vs %.2f×", ratio3, ratio1)
		}
	})

	t.Run("RealCSV_WithExternalDict", func(t *testing.T) {
		t.Log("\nTest 3: Real CSV File (test-recovery-keys.csv)")

		// Load test CSV
		data, err := os.ReadFile("../docs/test-recovery-keys.csv")
		if err != nil {
			t.Skip("test-recovery-keys.csv not found")
			return
		}

		t.Logf("   Input: %d bytes (%.2f KB)", len(data), float64(len(data))/1024)

		// Train dictionary
		trainer := dicttrainer.New()
		trainer.AddData(data)
		dict := trainer.Train(500)
		t.Logf("   Dictionary: %d bytes\n", len(dict))

		// No dict
		compressed1, _ := purgo.CompressSmart(data)
		ratio1 := float64(len(data)) / float64(len(compressed1))
		t.Logf("   CompressSmart:         %d bytes → %.2f× compression", len(compressed1), ratio1)

		// Embedded dict
		compressed2, _ := purgo.CompressSmartWithDict(data, dict)
		ratio2 := float64(len(data)) / float64(len(compressed2))
		t.Logf("   CompressSmartWithDict: %d bytes → %.2f× compression", len(compressed2), ratio2)

		// External dict
		compressed3, err := purgo.CompressWithDict(data, dict)
		if err != nil {
			t.Fatalf("CompressWithDict failed: %v", err)
		}
		ratio3 := float64(len(data)) / float64(len(compressed3))
		t.Logf("   CompressWithDict:      %d bytes → %.2f× compression", len(compressed3), ratio3)

		// Decompress
		decompressed, err := purgo.DecompressWithDict(compressed3, dict)
		if err != nil {
			t.Fatalf("DecompressWithDict failed: %v", err)
		}

		if !bytes.Equal(decompressed, data) {
			t.Fatal("Roundtrip mismatch")
		}

		t.Logf("\n   Savings from external dict: %d bytes (not embedding)", len(compressed2)-len(compressed3))

		if len(compressed3) < len(compressed1) {
			t.Logf("   ✅ External dict helps! %.0f%% better than no-dict",
				((ratio3/ratio1)-1)*100)
		} else {
			t.Logf("   ⚠️  External dict not better (dict overhead > benefit for single file)")
		}
	})
}

// TestDecompressWithDict_ErrorCases tests error handling
func TestDecompressWithDict_ErrorCases(t *testing.T) {
	t.Log("=== Error Handling Tests ===")

	// Use longer data with repeated patterns to ensure non-empty dictionary
	var sb strings.Builder
	for i := 0; i < 10; i++ {
		sb.WriteString("test data for compression, repeated pattern ")
	}
	data := []byte(sb.String())

	trainer := dicttrainer.New()
	trainer.AddData(data)
	dict := trainer.Train(50) // Request 50 bytes

	if len(dict) == 0 {
		t.Skip("Dictionary training produced empty result")
	}

	// Compress with external dict
	compressed, err := purgo.CompressWithDict(data, dict)
	if err != nil {
		t.Fatalf("CompressWithDict failed: %v", err)
	}

	t.Run("WrongDictionary", func(t *testing.T) {
		wrongDict := []byte("this is a wrong dictionary, completely different!")

		_, err := purgo.DecompressWithDict(compressed, wrongDict)
		// This might succeed but produce wrong data, or fail during decompression
		// We can't easily detect wrong dictionary without checksum
		if err != nil {
			t.Logf("   ✅ Detected wrong dictionary: %v", err)
		} else {
			t.Log("   ⚠️  Wrong dictionary not detected (data corruption possible)")
		}
	})

	t.Run("EmptyDictionary", func(t *testing.T) {
		_, err := purgo.DecompressWithDict(compressed, []byte{})
		if err == nil {
			t.Fatal("Expected error with empty dictionary")
		}
		t.Logf("   ✅ Empty dictionary rejected: %v", err)
	})

	t.Run("DecompressRegularFrameWithDict", func(t *testing.T) {
		// Compress WITHOUT external dict
		regularCompressed, _ := purgo.CompressSmart(data)

		// Try to decompress with external dict API
		_, err := purgo.DecompressWithDict(regularCompressed, dict)
		if err == nil {
			t.Fatal("Expected error when using DecompressWithDict on regular frame")
		}
		t.Logf("   ✅ Correctly rejected regular frame: %v", err)
	})

	t.Run("DecompressExternalDictFrameWithoutDict", func(t *testing.T) {
		// Try to decompress external-dict frame with regular Decompress
		_, err := purgo.Decompress(compressed)
		if err == nil {
			t.Fatal("Expected error when using Decompress on external-dict frame")
		}
		t.Logf("   ✅ Correctly rejected external-dict frame: %v", err)
	})
}
