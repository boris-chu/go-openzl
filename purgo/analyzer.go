// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package purgo

import (
	"bytes"
)

// DataFormat represents the detected format of input data
type DataFormat int

const (
	// FormatUnknown means format could not be detected
	FormatUnknown DataFormat = iota
	// FormatJSON represents JSON data ({ or [ started)
	FormatJSON
	// FormatCSV represents comma-separated values
	FormatCSV
	// FormatText represents plain text
	FormatText
	// FormatBinary represents binary data
	FormatBinary
)

// String returns human-readable format name
func (f DataFormat) String() string {
	switch f {
	case FormatJSON:
		return "JSON"
	case FormatCSV:
		return "CSV"
	case FormatText:
		return "Text"
	case FormatBinary:
		return "Binary"
	default:
		return "Unknown"
	}
}

// DetectFormat analyzes data and determines its likely format.
//
// Detection is fast (analyzes first 4KB only) and uses simple heuristics:
//   - JSON: Starts with { or [, has : and "
//   - CSV: Has consistent delimiters (,) per line
//   - Text: Mostly printable ASCII
//   - Binary: High percentage of non-printable bytes
//
// Returns FormatUnknown if format cannot be determined.
func DetectFormat(data []byte) DataFormat {
	if len(data) == 0 {
		return FormatUnknown
	}

	// Analyze first 4KB (or less if file is smaller)
	sampleSize := len(data)
	if sampleSize > 4096 {
		sampleSize = 4096
	}
	sample := data[:sampleSize]

	// Try JSON detection first (most specific)
	if isJSON(sample) {
		return FormatJSON
	}

	// Try CSV detection
	if isCSV(sample) {
		return FormatCSV
	}

	// Check if binary (high non-printable byte count)
	if isBinary(sample) {
		return FormatBinary
	}

	// Default to text
	return FormatText
}

// isJSON detects if data appears to be JSON format
func isJSON(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) < 2 {
		return false
	}

	// Must start with { or [
	first := trimmed[0]
	if first != '{' && first != '[' {
		return false
	}

	// Must have matching close brace
	last := trimmed[len(trimmed)-1]
	if (first == '{' && last != '}') || (first == '[' && last != ']') {
		return false
	}

	// Must have JSON structural characters
	hasQuotes := bytes.Contains(trimmed, []byte(`"`))
	hasColons := bytes.Contains(trimmed, []byte(`:`))

	// JSON objects must have both quotes and colons
	// JSON arrays might not have colons, but should have quotes or numbers
	if first == '{' {
		return hasQuotes && hasColons
	}

	// JSON arrays: need quotes (strings) OR nested brackets (nested arrays/objects)
	// Simple numeric arrays like [1,2,3] are not detected as JSON
	hasNestedBrackets := bytes.Count(trimmed, []byte(`[`)) > 1 ||
		bytes.Contains(trimmed, []byte(`{`))
	return hasQuotes || hasNestedBrackets
}

// isCSV detects if data appears to be CSV format
func isCSV(data []byte) bool {
	if len(data) < 10 {
		return false
	}

	// Split into lines
	lines := bytes.Split(data, []byte("\n"))
	if len(lines) < 2 {
		return false
	}

	// Count commas in first few lines
	var commaCounts []int
	for i := 0; i < len(lines) && i < 10; i++ {
		line := bytes.TrimSpace(lines[i])
		if len(line) == 0 {
			continue
		}
		commaCount := bytes.Count(line, []byte(","))
		if commaCount > 0 {
			commaCounts = append(commaCounts, commaCount)
		}
	}

	if len(commaCounts) < 2 {
		return false
	}

	// CSV should have consistent comma counts per line
	firstCount := commaCounts[0]
	consistentCount := 0
	for _, count := range commaCounts {
		if count == firstCount {
			consistentCount++
		}
	}

	// At least 80% of lines should have same comma count
	return float64(consistentCount)/float64(len(commaCounts)) >= 0.8
}

// isBinary detects if data appears to be binary (not text)
func isBinary(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	// Count non-printable bytes
	nonPrintable := 0
	for _, b := range data {
		// Printable ASCII: 32-126, plus tab(9), newline(10), carriage return(13)
		if (b < 32 || b > 126) && b != 9 && b != 10 && b != 13 {
			nonPrintable++
		}
	}

	// If >20% non-printable, likely binary
	return float64(nonPrintable)/float64(len(data)) > 0.2
}

// Codec name constants (must match internal/codec/codec.go)
const (
	codecNameIdentity  = "Identity"
	codecNameConstant  = "Constant"
	codecNameDelta     = "Delta"
	codecNameZigZag    = "ZigZag"
	codecNameBitpack   = "Bitpack"
	codecNameTranspose = "Transpose"
	codecNameQuantize  = "Quantize"
	codecNameFSE       = "FSE"
	codecNameHuffman   = "Huffman"
	codecNameLZ77      = "LZ77"
	codecNameRLE       = "RLE"
	codecNameZstd      = "Zstd"
)

// Codec ID constants (must match internal/codec/codec.go)
const (
	codecIDIdentity  uint16 = 0
	codecIDConstant  uint16 = 1
	codecIDDelta     uint16 = 2
	codecIDZigZag    uint16 = 3
	codecIDBitpack   uint16 = 4
	codecIDTranspose uint16 = 5
	codecIDQuantize  uint16 = 6
	codecIDFSE       uint16 = 10
	codecIDHuffman   uint16 = 11
	codecIDLZ77      uint16 = 12
	codecIDRLE       uint16 = 13
	codecIDZstd      uint16 = 20
)

// Segment represents a portion of data with a suggested codec
type Segment struct {
	Data      []byte
	CodecID   uint16 // Suggested codec ID (matches codec.ID type)
	CodecName string // Human-readable codec name
}

// SegmentCSV analyzes CSV data and returns segments per column.
//
// Each column is analyzed separately to choose the optimal codec:
//   - Repeated values (e.g., domain names) → RLE
//   - UUID/passwords with patterns → LZ77
//   - Sequential numbers → Delta
//
// Returns one segment per column, preserving column order.
func SegmentCSV(data []byte) ([]Segment, error) {
	if len(data) == 0 {
		return nil, nil
	}

	// Parse CSV structure
	lines := bytes.Split(data, []byte("\n"))
	if len(lines) < 2 {
		return nil, nil // Need header + at least one data row
	}

	// Get column count from first line (header)
	header := bytes.TrimSpace(lines[0])
	numCols := bytes.Count(header, []byte(",")) + 1

	// Extract column data
	columns := make([][]byte, numCols)
	for i := range columns {
		columns[i] = make([]byte, 0, len(data)/numCols)
	}

	// Parse each row and extract column values
	for i := 1; i < len(lines); i++ {
		line := bytes.TrimSpace(lines[i])
		if len(line) == 0 {
			continue // Skip empty lines
		}

		fields := bytes.Split(line, []byte(","))
		for colIdx := 0; colIdx < numCols && colIdx < len(fields); colIdx++ {
			// Append field value + delimiter (to preserve structure)
			columns[colIdx] = append(columns[colIdx], fields[colIdx]...)
			columns[colIdx] = append(columns[colIdx], '\n')
		}
	}

	// Analyze each column and create segments
	segments := make([]Segment, 0, numCols)
	for colIdx, colData := range columns {
		if len(colData) == 0 {
			continue
		}

		// Analyze column characteristics
		codecID, codecName := analyzeColumnPattern(colData)

		segments = append(segments, Segment{
			Data:      colData,
			CodecID:   codecID,
			CodecName: codecName,
		})

		_ = colIdx // Keep for debugging
	}

	return segments, nil
}

// analyzeColumnPattern analyzes a column's data pattern and suggests a codec.
//
// Detection strategies (in priority order):
//  1. Constant: All values identical → Constant codec
//  2. Delta: Sequential numeric values → Delta (with ZigZag for signed)
//  3. Bitpack: Small integers (0-255) → Bitpack
//  4. RLE: High repetition (≥80%) → RLE
//  5. Numeric: All numbers → Transpose (multi-byte pattern optimization)
//  6. UUID/Text patterns: → LZ77 (dictionary compression)
//  7. Text with low entropy: → FSE or Huffman (entropy coding)
//  8. Default: General text → LZ77
func analyzeColumnPattern(data []byte) (codecID uint16, codecName string) {
	if len(data) == 0 {
		return codecIDIdentity, codecNameIdentity
	}

	// Parse column into lines
	lines := bytes.Split(data, []byte("\n"))
	nonEmptyLines := make([][]byte, 0, len(lines))
	for _, line := range lines {
		if len(line) > 0 {
			nonEmptyLines = append(nonEmptyLines, line)
		}
	}

	if len(nonEmptyLines) == 0 {
		return codecIDIdentity, codecNameIdentity
	}

	// Strategy 1: Constant - All values identical
	if isConstantColumn(nonEmptyLines) {
		return codecIDConstant, codecNameConstant
	}

	// Strategy 2: Delta - Sequential numeric values (IDs, timestamps)
	// Note: For signed integers, Delta codec internally uses ZigZag encoding
	if isDeltaColumn(nonEmptyLines) {
		return codecIDDelta, codecNameDelta
	}

	// Strategy 3: Bitpack - Small integers (0-255 range)
	if isBitpackColumn(nonEmptyLines) {
		return codecIDBitpack, codecNameBitpack
	}

	// Strategy 4: RLE - High repetition (≥80% same values)
	repetitionRatio := calculateRepetition(nonEmptyLines)
	if repetitionRatio >= 0.8 {
		return codecIDRLE, codecNameRLE
	}

	// Strategy 5: Numeric - All numbers (int/float)
	if isNumericColumn(nonEmptyLines) {
		return codecIDTranspose, codecNameTranspose
	}

	// Strategy 6: UUID/Pattern - Dictionary compression
	if hasUUIDPattern(data) {
		return codecIDLZ77, codecNameLZ77
	}

	// Strategy 7: Low entropy text - Entropy coding
	// FSE and Huffman are typically used as final stage after other codecs
	// For standalone use, prefer FSE for slightly better compression on low-entropy data
	if hasLowEntropy(nonEmptyLines) {
		return codecIDFSE, codecNameFSE
	}

	// Strategy 8: Default - General text (LZ77 dictionary compression)
	return codecIDLZ77, codecNameLZ77
}

// isConstantColumn checks if all values are identical
func isConstantColumn(lines [][]byte) bool {
	if len(lines) == 0 {
		return false
	}

	first := string(lines[0])
	for i := 1; i < len(lines); i++ {
		if string(lines[i]) != first {
			return false
		}
	}
	return true
}

// isDeltaColumn checks if values are sequential numbers (Delta codec candidate)
func isDeltaColumn(lines [][]byte) bool {
	if len(lines) < 3 {
		return false // Need at least 3 values to detect sequence
	}

	// Try parsing first 3 values as integers
	var values [3]int64
	for i := 0; i < 3 && i < len(lines); i++ {
		val, err := parseInt64(lines[i])
		if err != nil {
			return false // Not numeric
		}
		values[i] = val
	}

	// Check if differences are consistent (sequential)
	diff1 := values[1] - values[0]
	diff2 := values[2] - values[1]

	// Allow small variance (±1) for semi-sequential data
	if diff1 == 0 || diff2 == 0 {
		return false // Not sequential
	}

	variance := diff1 - diff2
	if variance < -1 || variance > 1 {
		return false // Not sequential enough
	}

	return true
}

// isBitpackColumn checks if values are small integers suitable for bitpacking (0-255)
func isBitpackColumn(lines [][]byte) bool {
	if len(lines) < 3 {
		return false // Need enough data to justify bitpacking
	}

	// Check if all values are small integers (0-255)
	for i := 0; i < min(20, len(lines)); i++ {
		val, err := parseInt64(lines[i])
		if err != nil {
			return false // Not an integer
		}
		if val < 0 || val > 255 {
			return false // Outside bitpack range
		}
	}

	return true
}

// isNumericColumn checks if all values are numbers (int or float)
func isNumericColumn(lines [][]byte) bool {
	if len(lines) == 0 {
		return false
	}

	// Check first 10 lines (or all if fewer)
	checkCount := min(10, len(lines))
	for i := 0; i < checkCount; i++ {
		if !isNumeric(lines[i]) {
			return false
		}
	}

	return true
}

// hasLowEntropy checks if data has low entropy (repeated characters/patterns)
// This suggests FSE or Huffman encoding would be effective
func hasLowEntropy(lines [][]byte) bool {
	if len(lines) < 10 {
		return false // Need enough data to justify entropy coding
	}

	// Count character frequency across all lines
	charFreq := make(map[byte]int)
	totalChars := 0

	for _, line := range lines {
		for _, b := range line {
			charFreq[b]++
			totalChars++
		}
	}

	if totalChars < 50 {
		return false // Need enough chars to measure entropy
	}

	// If a single character appears >50% of the time, it's low entropy
	for _, count := range charFreq {
		if float64(count)/float64(totalChars) > 0.5 {
			return true
		}
	}

	// If very few unique characters (<8) with enough data, it's low entropy
	if len(charFreq) < 8 && totalChars > 100 {
		return true
	}

	return false
}

// hasUUIDPattern checks for UUID-like patterns
func hasUUIDPattern(data []byte) bool {
	// UUIDs have format: {XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX}
	return bytes.Contains(data, []byte("-")) &&
		(bytes.Contains(data, []byte("{")) || len(data) > 100)
}

// calculateRepetition calculates the repetition ratio (1.0 = all same, 0.0 = all unique)
func calculateRepetition(lines [][]byte) float64 {
	if len(lines) == 0 {
		return 0
	}

	uniqueValues := make(map[string]bool)
	for _, line := range lines {
		uniqueValues[string(line)] = true
	}

	return 1.0 - (float64(len(uniqueValues)) / float64(len(lines)))
}

// parseInt64 parses a byte slice as an int64
func parseInt64(data []byte) (int64, error) {
	// Trim whitespace
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return 0, bytes.ErrTooLarge
	}

	// Simple integer parsing (supports negative)
	var result int64
	negative := false
	start := 0

	if trimmed[0] == '-' {
		negative = true
		start = 1
	} else if trimmed[0] == '+' {
		start = 1
	}

	for i := start; i < len(trimmed); i++ {
		if trimmed[i] < '0' || trimmed[i] > '9' {
			return 0, bytes.ErrTooLarge // Not a digit
		}
		result = result*10 + int64(trimmed[i]-'0')
	}

	if negative {
		result = -result
	}

	return result, nil
}

// isNumeric checks if a byte slice represents a number (int or float)
func isNumeric(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return false
	}

	hasDigit := false
	hasDot := false

	for i, b := range trimmed {
		if b >= '0' && b <= '9' {
			hasDigit = true
		} else if b == '.' && !hasDot {
			hasDot = true // Allow one decimal point
		} else if b == '-' || b == '+' {
			if i != 0 {
				return false // Sign only at start
			}
		} else {
			return false // Invalid character
		}
	}

	return hasDigit
}

// SegmentJSON analyzes JSON data and returns segments per field.
//
// Each field is analyzed separately to choose the optimal codec:
//   - Repeated field names → LZ77 (dictionary)
//   - Numeric arrays → Delta or Bitpack
//   - String values → LZ77
//
// Returns segments for field names and values separately.
func SegmentJSON(data []byte) ([]Segment, error) {
	if len(data) == 0 {
		return nil, nil
	}

	// For now, use LZ77 for entire JSON (most effective for JSON structure)
	// Future: Parse JSON and segment by field type
	return []Segment{
		{
			Data:      data,
			CodecID:   codecIDLZ77,
			CodecName: codecNameLZ77,
		},
	}, nil
}
