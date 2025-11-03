// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package purgo

import (
	"os"
	"testing"
)

// TestDetectFormat_CSV verifies CSV detection using real BitLocker test file.
func TestDetectFormat_CSV(t *testing.T) {
	// Read test CSV file
	data, err := os.ReadFile("../docs/test-bitlocker.csv")
	if err != nil {
		t.Skipf("Skipping CSV test: %v", err)
		return
	}

	format := DetectFormat(data)
	if format != FormatCSV {
		t.Errorf("DetectFormat() = %v, want %v for CSV file", format, FormatCSV)
	}

	t.Logf("✅ Correctly detected CSV format (file size: %d bytes)", len(data))
}

// TestDetectFormat_JSON verifies JSON object detection.
func TestDetectFormat_JSON(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{
			name: "json object",
			data: `{"name": "Alice", "age": 30, "city": "New York"}`,
		},
		{
			name: "json array",
			data: `[{"id": 1}, {"id": 2}, {"id": 3}]`,
		},
		{
			name: "nested json",
			data: `{
				"users": [
					{"name": "Bob", "email": "bob@example.com"},
					{"name": "Carol", "email": "carol@example.com"}
				]
			}`,
		},
		{
			name: "json with whitespace",
			data: `  { "key" : "value" }  `,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format := DetectFormat([]byte(tt.data))
			if format != FormatJSON {
				t.Errorf("DetectFormat() = %v, want %v for JSON: %s", format, FormatJSON, tt.data)
			}
		})
	}
}

// TestDetectFormat_Text verifies plain text detection.
func TestDetectFormat_Text(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{
			name: "plain text",
			data: "This is plain text without special structure.",
		},
		{
			name: "multi-line text",
			data: "Line 1\nLine 2\nLine 3",
		},
		{
			name: "text with numbers",
			data: "Error 404: Page not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format := DetectFormat([]byte(tt.data))
			if format != FormatText {
				t.Errorf("DetectFormat() = %v, want %v for text: %s", format, FormatText, tt.data)
			}
		})
	}
}

// TestDetectFormat_Binary verifies binary data detection.
func TestDetectFormat_Binary(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "binary with null bytes",
			data: []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD},
		},
		{
			name: "high non-printable ratio",
			data: []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, // PNG header
		},
		{
			name: "mixed binary",
			data: append([]byte("HEADER"), []byte{0x00, 0x00, 0xFF, 0xFF, 0x00, 0x00, 0xFF, 0xFF}...),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format := DetectFormat(tt.data)
			if format != FormatBinary {
				t.Errorf("DetectFormat() = %v, want %v for binary data", format, FormatBinary)
			}
		})
	}
}

// TestDetectFormat_Empty verifies empty input handling.
func TestDetectFormat_Empty(t *testing.T) {
	format := DetectFormat([]byte{})
	if format != FormatUnknown {
		t.Errorf("DetectFormat() = %v, want %v for empty input", format, FormatUnknown)
	}
}

// TestDetectFormat_EdgeCases verifies edge case handling.
func TestDetectFormat_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		expected DataFormat
	}{
		{
			name:     "json-like but not json",
			data:     "{not valid json",
			expected: FormatText,
		},
		{
			name:     "csv-like but inconsistent",
			data:     "a,b,c\n1,2\n3,4,5,6",
			expected: FormatText, // Inconsistent comma counts
		},
		{
			name:     "single line csv",
			data:     "a,b,c",
			expected: FormatText, // Need at least 2 lines
		},
		{
			name:     "json array empty",
			data:     "[]",
			expected: FormatText, // Empty arrays don't benefit from JSON segmentation
		},
		{
			name:     "json object empty",
			data:     "{}",
			expected: FormatText, // Empty objects don't benefit from JSON segmentation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format := DetectFormat([]byte(tt.data))
			if format != tt.expected {
				t.Errorf("DetectFormat() = %v, want %v for: %s", format, tt.expected, tt.data)
			}
		})
	}
}

// TestDataFormat_String verifies String() method.
func TestDataFormat_String(t *testing.T) {
	tests := []struct {
		format   DataFormat
		expected string
	}{
		{FormatUnknown, "Unknown"},
		{FormatJSON, "JSON"},
		{FormatCSV, "CSV"},
		{FormatText, "Text"},
		{FormatBinary, "Binary"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.format.String()
			if got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestIsJSON_Comprehensive verifies JSON detection logic.
func TestIsJSON_Comprehensive(t *testing.T) {
	tests := []struct {
		name   string
		data   string
		isJSON bool
	}{
		// Valid JSON
		{name: "object with quotes and colons", data: `{"key": "value"}`, isJSON: true},
		{name: "array with quotes", data: `["a", "b", "c"]`, isJSON: true},
		{name: "nested array", data: `[[1, 2], [3, 4]]`, isJSON: true},

		// Invalid JSON
		{name: "object without colon", data: `{"key" "value"}`, isJSON: false},
		{name: "object without quotes", data: `{key: value}`, isJSON: false},
		{name: "mismatched braces", data: `{"key": "value"]`, isJSON: false},
		{name: "array without quotes or brackets", data: `[1, 2, 3]`, isJSON: false}, // No quotes, no nested brackets
		{name: "empty object", data: `{}`, isJSON: false},                            // No structure
		{name: "too short", data: `{`, isJSON: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isJSON([]byte(tt.data))
			if got != tt.isJSON {
				t.Errorf("isJSON(%q) = %v, want %v", tt.data, got, tt.isJSON)
			}
		})
	}
}

// TestIsCSV_Comprehensive verifies CSV detection logic.
func TestIsCSV_Comprehensive(t *testing.T) {
	tests := []struct {
		name  string
		data  string
		isCSV bool
	}{
		// Valid CSV
		{
			name:  "basic csv",
			data:  "a,b,c\n1,2,3\n4,5,6",
			isCSV: true,
		},
		{
			name:  "csv with header",
			data:  "Name,Age,City\nAlice,30,NYC\nBob,25,LA",
			isCSV: true,
		},
		{
			name: "csv with many rows",
			data: "col1,col2,col3\n" +
				"1,2,3\n" +
				"4,5,6\n" +
				"7,8,9\n" +
				"10,11,12",
			isCSV: true,
		},

		// Invalid CSV
		{
			name:  "no commas",
			data:  "just text\nno structure",
			isCSV: false,
		},
		{
			name:  "inconsistent commas",
			data:  "a,b,c\n1,2\n3,4,5,6",
			isCSV: false, // <80% consistency
		},
		{
			name:  "single line",
			data:  "a,b,c",
			isCSV: false, // Need at least 2 lines
		},
		{
			name:  "too short",
			data:  "a,b",
			isCSV: false, // <10 bytes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCSV([]byte(tt.data))
			if got != tt.isCSV {
				t.Errorf("isCSV(%q) = %v, want %v", tt.data, got, tt.isCSV)
			}
		})
	}
}

// TestIsBinary_Comprehensive verifies binary detection logic.
func TestIsBinary_Comprehensive(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		isBinary bool
	}{
		// Binary data
		{
			name:     "high non-printable ratio",
			data:     []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD, 0xFC, 0xFB, 0xFA},
			isBinary: true, // 100% non-printable
		},
		{
			name:     "mixed with 30% non-printable",
			data:     append([]byte("Hello"), []byte{0x00, 0x00, 0xFF, 0xFF}...),
			isBinary: true, // 4/9 = 44% > 20%
		},

		// Text data
		{
			name:     "pure ascii text",
			data:     []byte("Hello, World!"),
			isBinary: false, // 0% non-printable
		},
		{
			name:     "text with newlines",
			data:     []byte("Line 1\nLine 2\nLine 3"),
			isBinary: false, // Newlines are allowed
		},
		{
			name:     "text with tabs",
			data:     []byte("Column1\tColumn2\tColumn3"),
			isBinary: false, // Tabs are allowed
		},

		// Edge cases
		{
			name:     "empty",
			data:     []byte{},
			isBinary: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBinary(tt.data)
			if got != tt.isBinary {
				t.Errorf("isBinary() = %v, want %v for data: %v", got, tt.isBinary, tt.data)
			}
		})
	}
}

// BenchmarkDetectFormat benchmarks format detection speed.
func BenchmarkDetectFormat(b *testing.B) {
	// Use real CSV file if available
	data, err := os.ReadFile("../docs/test-bitlocker.csv")
	if err != nil {
		// Fall back to synthetic CSV
		data = []byte("Name,Age,City\nAlice,30,NYC\nBob,25,LA\nCarol,35,SF")
	}

	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		_ = DetectFormat(data)
	}
}

// BenchmarkDetectFormat_LargeFile benchmarks detection on large files.
func BenchmarkDetectFormat_LargeFile(b *testing.B) {
	// Create 1MB CSV-like data
	data := make([]byte, 1024*1024)
	for i := range data {
		if i%50 == 0 {
			data[i] = '\n'
		} else if i%10 == 0 {
			data[i] = ','
		} else {
			data[i] = 'a' + byte(i%26)
		}
	}

	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		_ = DetectFormat(data)
	}
}

// TestSegmentCSV_Simple verifies CSV segmentation on simple data.
func TestSegmentCSV_Simple(t *testing.T) {
	tests := []struct {
		name         string
		data         string
		expectedCols int
	}{
		{
			name:         "basic csv",
			data:         "Name,Age,City\nAlice,30,NYC\nBob,25,LA\nCarol,35,SF",
			expectedCols: 3,
		},
		{
			name:         "repeated values",
			data:         "Status,Count\nactive,1\nactive,2\nactive,3\nactive,4\nactive,5",
			expectedCols: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments, err := SegmentCSV([]byte(tt.data))
			if err != nil {
				t.Fatalf("SegmentCSV() error = %v", err)
			}

			if len(segments) != tt.expectedCols {
				t.Errorf("SegmentCSV() returned %d segments, want %d", len(segments), tt.expectedCols)
			}

			for i, seg := range segments {
				t.Logf("  Column %d: %d bytes → %s", i+1, len(seg.Data), seg.CodecName)
			}
		})
	}
}

// TestSegmentCSV_EdgeCases verifies edge case handling.
func TestSegmentCSV_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		expected int
	}{
		{
			name:     "empty",
			data:     "",
			expected: 0,
		},
		{
			name:     "header only",
			data:     "Name,Age,City",
			expected: 0, // Need at least one data row
		},
		{
			name:     "single column",
			data:     "Name\nAlice\nBob",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments, err := SegmentCSV([]byte(tt.data))
			if err != nil {
				t.Fatalf("SegmentCSV() error = %v", err)
			}

			if len(segments) != tt.expected {
				t.Errorf("SegmentCSV() returned %d segments, want %d", len(segments), tt.expected)
			}
		})
	}
}

// TestSegmentJSON_Simple verifies JSON segmentation.
func TestSegmentJSON_Simple(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{
			name: "json object",
			data: `{"name": "Alice", "age": 30}`,
		},
		{
			name: "json array",
			data: `[{"id": 1}, {"id": 2}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments, err := SegmentJSON([]byte(tt.data))
			if err != nil {
				t.Fatalf("SegmentJSON() error = %v", err)
			}

			if len(segments) == 0 {
				t.Error("SegmentJSON() returned 0 segments, want at least 1")
			}

			for i, seg := range segments {
				t.Logf("  Segment %d: %d bytes → %s", i+1, len(seg.Data), seg.CodecName)
			}

			// JSON should use LZ77 (dictionary compression for repeated keys)
			if segments[0].CodecName != "LZ77" {
				t.Errorf("JSON codec = %s, want LZ77", segments[0].CodecName)
			}
		})
	}
}

// TestAnalyzeColumnPattern verifies column pattern detection.
func TestAnalyzeColumnPattern(t *testing.T) {
	tests := []struct {
		name         string
		data         string
		expectedName string
	}{
		// Constant detection
		{
			name:         "constant values",
			data:         "USA\nUSA\nUSA\nUSA\nUSA",
			expectedName: "Constant",
		},
		{
			name:         "constant single value",
			data:         "true",
			expectedName: "Constant",
		},

		// Delta detection (sequential numbers)
		{
			name:         "sequential IDs",
			data:         "1001\n1002\n1003\n1004\n1005",
			expectedName: "Delta",
		},
		{
			name:         "sequential timestamps",
			data:         "1609459200\n1609545600\n1609632000",
			expectedName: "Delta",
		},
		{
			name:         "negative sequential",
			data:         "-10\n-9\n-8\n-7\n-6",
			expectedName: "Delta",
		},

		// RLE detection (high repetition ≥80%, but not 100%)
		// Note: Repetition ratio = 1 - (unique_values / total_lines)
		{
			name: "high repetition 90%",
			data: `active
active
active
active
active
active
active
active
active
inactive`,
			expectedName: "RLE", // 2 unique values out of 10 lines = 1 - (2/10) = 80% repetition
		},
		{
			name: "status column 80%",
			data: `enabled
enabled
enabled
enabled
enabled
enabled
enabled
enabled
disabled
disabled`,
			expectedName: "RLE", // 2 unique out of 10 = 1 - (2/10) = 80% repetition
		},

		// Bitpack detection (small integers 0-255, non-sequential)
		{
			name:         "small non-sequential integers",
			data:         "10\n50\n100\n200\n255\n3",
			expectedName: "Bitpack",
		},
		{
			name:         "random byte values",
			data:         "42\n7\n99\n13\n200\n5",
			expectedName: "Bitpack",
		},

		// Numeric (Transpose) detection
		{
			name:         "large non-sequential integers",
			data:         "1000\n5000\n3000\n2000\n4000", // Large values, not sequential
			expectedName: "Transpose",
		},
		{
			name:         "float column",
			data:         "3.14\n2.71\n1.41\n0.5",
			expectedName: "Transpose",
		},
		{
			name:         "mixed positive negative",
			data:         "100\n-50\n75\n-25\n0",
			expectedName: "Transpose",
		},

		// UUID/LZ77 detection
		{
			name:         "uuid pattern",
			data:         "{12345678-1234-1234-1234-123456789012}\n{ABCDEFGH-ABCD-ABCD-ABCD-ABCDEFGHIJKL}",
			expectedName: "LZ77",
		},
		{
			name:         "email addresses",
			data:         "alice@example.com\nbob@example.org\ncarol@test.net",
			expectedName: "LZ77",
		},

		// Note: FSE/Huffman are typically used as final-stage entropy coders after other codecs
		// For standalone use on short text, LZ77 is often chosen as it's more general-purpose

		// General text (LZ77 default)
		{
			name:         "general text",
			data:         "Alice\nBob\nCarol\nDave",
			expectedName: "LZ77",
		},
		{
			name:         "mixed text",
			data:         "Product A\nItem 123\nService XYZ",
			expectedName: "LZ77",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, codecName := analyzeColumnPattern([]byte(tt.data))
			if codecName != tt.expectedName {
				t.Errorf("analyzeColumnPattern() = %s, want %s", codecName, tt.expectedName)
			}
		})
	}
}

// TestAnalyzeColumnPattern_EdgeCases tests edge cases.
func TestAnalyzeColumnPattern_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		data         string
		expectedName string
	}{
		{
			name:         "empty",
			data:         "",
			expectedName: "Identity",
		},
		{
			name:         "single line",
			data:         "value",
			expectedName: "Constant",
		},
		{
			name:         "two identical",
			data:         "same\nsame",
			expectedName: "Constant",
		},
		{
			name:         "two numbers not sequential",
			data:         "100\n200",
			expectedName: "Transpose", // Need 3 for Delta detection
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, codecName := analyzeColumnPattern([]byte(tt.data))
			if codecName != tt.expectedName {
				t.Errorf("analyzeColumnPattern() = %s, want %s", codecName, tt.expectedName)
			}
		})
	}
}

// TestSegmentCSV_PublicExamples tests CSV segmentation with public data.
func TestSegmentCSV_PublicExamples(t *testing.T) {
	tests := []struct {
		name           string
		csv            string
		expectedCodecs []string
	}{
		{
			name: "sales data",
			csv: `Region,Quarter,Sales,Status
North,Q1,1000,active
North,Q2,1050,active
North,Q3,1100,active
South,Q1,800,active
South,Q2,850,active`,
			expectedCodecs: []string{"LZ77", "LZ77", "Delta", "Constant"}, // Region(60% repetition=LZ77), Quarter(text with letters), Sales(Delta), Status(Constant)
		},
		{
			name: "user activity",
			csv: `UserID,Timestamp,Action
1001,1609459200,login
1002,1609545600,logout
1003,1609632000,login
1004,1609718400,logout`,
			expectedCodecs: []string{"Delta", "Delta", "LZ77"}, // UserID(Delta), Timestamp(Delta), Action(LZ77)
		},
		{
			name: "product inventory",
			csv: `SKU,Price,Stock,Warehouse
A-001,19.99,50,NYC
A-002,29.99,30,NYC
A-003,39.99,20,NYC`,
			expectedCodecs: []string{"LZ77", "Transpose", "Bitpack", "Constant"}, // SKU(LZ77), Price(float/numeric), Stock(small ints 0-255), Warehouse(constant)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments, err := SegmentCSV([]byte(tt.csv))
			if err != nil {
				t.Fatalf("SegmentCSV() error = %v", err)
			}

			if len(segments) != len(tt.expectedCodecs) {
				t.Errorf("SegmentCSV() returned %d segments, want %d", len(segments), len(tt.expectedCodecs))
			}

			for i, expectedCodec := range tt.expectedCodecs {
				if i >= len(segments) {
					break
				}
				if segments[i].CodecName != expectedCodec {
					t.Errorf("Segment %d codec = %s, want %s", i, segments[i].CodecName, expectedCodec)
				}
			}
		})
	}
}

// TestIsNumeric tests numeric detection.
func TestIsNumeric(t *testing.T) {
	tests := []struct {
		data     string
		expected bool
	}{
		{"42", true},
		{"3.14", true},
		{"-100", true},
		{"+50", true},
		{"0", true},
		{"abc", false},
		{"12.34.56", false},
		{"", false},
		{"  123  ", true},
		{"-3.14", true},
	}

	for _, tt := range tests {
		t.Run(tt.data, func(t *testing.T) {
			got := isNumeric([]byte(tt.data))
			if got != tt.expected {
				t.Errorf("isNumeric(%q) = %v, want %v", tt.data, got, tt.expected)
			}
		})
	}
}

// TestParseInt64 tests integer parsing.
func TestParseInt64(t *testing.T) {
	tests := []struct {
		data    string
		want    int64
		wantErr bool
	}{
		{"42", 42, false},
		{"-100", -100, false},
		{"+50", 50, false},
		{"0", 0, false},
		{"  123  ", 123, false},
		{"abc", 0, true},
		{"12.34", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.data, func(t *testing.T) {
			got, err := parseInt64([]byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Errorf("parseInt64(%q) error = %v, wantErr %v", tt.data, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseInt64(%q) = %d, want %d", tt.data, got, tt.want)
			}
		})
	}
}
