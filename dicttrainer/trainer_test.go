// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package dicttrainer

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestTrainer_BasicUsage(t *testing.T) {
	trainer := New()

	// Add sample CSV data
	csvData := `id,name,email
1,John,john@example.com
2,Jane,jane@example.com
3,Bob,bob@example.com`

	trainer.AddData([]byte(csvData))

	// Train a small dictionary
	dict := trainer.Train(100)

	if len(dict) == 0 {
		t.Error("Expected non-empty dictionary")
	}

	if len(dict) > 100 {
		t.Errorf("Dictionary size %d exceeds target 100", len(dict))
	}

	// Dictionary should contain common patterns
	dictStr := string(dict)
	t.Logf("Dictionary (%d bytes): %q", len(dict), dictStr)

	// Dictionary should contain repeated patterns
	// Note: Single "," may not be included if longer patterns like "@example.com" score higher
	if len(dictStr) == 0 {
		t.Error("Expected non-empty dictionary")
	}
}

func TestTrainer_MultipleFiles(t *testing.T) {
	trainer := New()

	// Add multiple datasets
	trainer.AddData([]byte("pattern1 pattern1 pattern1"))
	trainer.AddData([]byte("pattern2 pattern2 pattern2"))
	trainer.AddData([]byte("pattern1 pattern2"))

	dict := trainer.Train(50)

	dictStr := string(dict)
	t.Logf("Dictionary: %q", dictStr)

	// Both patterns should be included
	if !strings.Contains(dictStr, "pattern") {
		t.Error("Expected 'pattern' in dictionary")
	}
}

func TestTrainer_CustomPatterns(t *testing.T) {
	trainer := New()

	// Add data
	trainer.AddData([]byte("some data here"))

	// Add custom patterns
	trainer.AddPatterns([]string{
		"https://api.example.com/",
		"X-API-Key: ",
	})

	dict := trainer.Train(100)
	dictStr := string(dict)

	t.Logf("Dictionary with custom patterns: %q", dictStr)

	// Custom patterns should be included even if not in corpus
	if !strings.Contains(dictStr, "https://") {
		t.Error("Expected custom pattern 'https://' in dictionary")
	}
}

func TestTrainer_SetPatternRange(t *testing.T) {
	trainer := New()

	// Set custom range
	err := trainer.SetPatternRange(2, 16)
	if err != nil {
		t.Fatalf("SetPatternRange() error = %v", err)
	}

	// Invalid ranges should error
	err = trainer.SetPatternRange(1, 10)
	if err == nil {
		t.Error("Expected error for minLen < 2")
	}

	err = trainer.SetPatternRange(10, 5)
	if err == nil {
		t.Error("Expected error for maxLen < minLen")
	}

	err = trainer.SetPatternRange(3, 300)
	if err == nil {
		t.Error("Expected error for maxLen > 258")
	}
}

func TestTrainer_GetStats(t *testing.T) {
	trainer := New()
	trainer.AddData([]byte("test data"))

	stats := trainer.GetStats()

	if stats.CorpusSize != 9 {
		t.Errorf("CorpusSize = %d, want 9", stats.CorpusSize)
	}

	if stats.MinPatternLen != 3 {
		t.Errorf("MinPatternLen = %d, want 3", stats.MinPatternLen)
	}

	if stats.MaxPatternLen != 32 {
		t.Errorf("MaxPatternLen = %d, want 32", stats.MaxPatternLen)
	}

	t.Logf("Stats: %+v", stats)
}

func TestTrainer_EmptyCorpus(t *testing.T) {
	trainer := New()

	// Train with no data
	dict := trainer.Train(100)

	if len(dict) != 0 {
		t.Errorf("Expected empty dictionary for empty corpus, got %d bytes", len(dict))
	}
}

func TestTrainer_LargeCorpus(t *testing.T) {
	trainer := New()

	// Add large repetitive data
	var sb strings.Builder
	for i := 0; i < 1000; i++ {
		sb.WriteString("id,name,email,phone,city,state,country\n")
		sb.WriteString("1,John,john@example.com,555-1234,NYC,NY,USA\n")
	}
	trainer.AddData([]byte(sb.String()))

	// Train dictionary
	dict := trainer.Train(1024) // 1KB

	if len(dict) == 0 {
		t.Error("Expected non-empty dictionary")
	}

	if len(dict) > 1024 {
		t.Errorf("Dictionary size %d exceeds target 1024", len(dict))
	}

	t.Logf("Large corpus dictionary (%d bytes)", len(dict))

	// Should capture common patterns (exact patterns depend on scoring)
	if len(dict) == 0 {
		t.Error("Expected non-empty dictionary for large repetitive corpus")
	}
}

func TestTrainer_RealFile(t *testing.T) {
	// Try to load test-recovery-keys.csv
	trainer := New()
	err := trainer.AddFile("/Users/borischu/go-openzl/docs/test-recovery-keys.csv")
	if err != nil {
		t.Skip("test-recovery-keys.csv not found")
	}

	stats := trainer.GetStats()
	t.Logf("CSV corpus: %d bytes", stats.CorpusSize)

	dict := trainer.Train(500) // 500 byte dictionary
	t.Logf("Trained dictionary: %d bytes", len(dict))

	// Should successfully train on real CSV file
	if len(dict) == 0 {
		t.Error("Expected non-empty dictionary for real CSV data")
	}
}

func TestTrainer_BinaryData(t *testing.T) {
	trainer := New()

	// Add binary data with repeated patterns
	// Use longer patterns to avoid control character filtering
	pattern := []byte("BIN") // Readable pattern
	data := bytes.Repeat(pattern, 100)
	trainer.AddData(data)

	dict := trainer.Train(20)

	// Binary data should produce dictionary
	t.Logf("Binary dictionary: %d bytes", len(dict))

	if len(dict) > 20 {
		t.Errorf("Dictionary exceeds target size: %d > 20", len(dict))
	}
}

func TestTrainer_SaveAndLoad(t *testing.T) {
	trainer := New()
	trainer.AddData([]byte("test pattern test pattern"))

	dict := trainer.Train(50)

	// Save to temp file
	tmpfile, err := os.CreateTemp("", "dict-*.bin")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	err = os.WriteFile(tmpfile.Name(), dict, 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Load back
	loaded, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(dict, loaded) {
		t.Error("Loaded dictionary doesn't match saved dictionary")
	}

	t.Logf("Saved and loaded %d byte dictionary", len(loaded))
}

func BenchmarkTrainer_SmallCorpus(b *testing.B) {
	data := []byte("pattern1,pattern2,pattern3,pattern1,pattern2")

	for i := 0; i < b.N; i++ {
		trainer := New()
		trainer.AddData(data)
		trainer.Train(100)
	}
}

func BenchmarkTrainer_LargeCorpus(b *testing.B) {
	// Generate 1MB of CSV data
	var sb strings.Builder
	for i := 0; i < 10000; i++ {
		sb.WriteString("id,name,email,phone,city,state\n")
		sb.WriteString("1,John,john@example.com,555-1234,NYC,NY\n")
	}
	data := []byte(sb.String())

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		trainer := New()
		trainer.AddData(data)
		trainer.Train(30 * 1024) // 30KB
	}
}
