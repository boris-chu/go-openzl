// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package codec

import (
	"encoding/binary"
	"fmt"
)

// Prefix is a codec that extracts common prefixes from consecutive strings.
//
// Algorithm:
// - For each consecutive pair of strings, find the longest common prefix
// - Output: prefix length + unmatched suffix
// - This is highly effective for URL lists, file paths, and domain names
//
// Example:
//
//	Input:  ["/usr/local/bin/gcc", "/usr/local/bin/g++", "/usr/local/lib/libz.a"]
//	Process:
//	  String 0: prefix=0, suffix="/usr/local/bin/gcc" (first string, no prefix)
//	  String 1: prefix=15 ("/usr/local/bin/"), suffix="g++"
//	  String 2: prefix=11 ("/usr/local/"), suffix="lib/libz.a"
//	Compression: 58 bytes → ~35 bytes (40% savings)
//
// Wire Format:
//
//	[numStrings: 4 bytes (uint32)]
//	For each string:
//	  [prefixLen: 2 bytes (uint16)] - How many bytes to reuse from previous string
//	  [suffixLen: 2 bytes (uint16)] - How many new bytes follow
//	  [suffix: suffixLen bytes]     - The unmatched suffix
//
// Use Cases:
// - URL lists with common base (https://api.example.com/v1/...)
// - File paths (/usr/local/bin/..., /home/user/documents/...)
// - Domain names (mail.google.com, drive.google.com, docs.google.com)
// - Log files with repeated prefixes
type Prefix struct{}

// NewPrefix creates a new Prefix codec.
func NewPrefix() *Prefix {
	return &Prefix{}
}

// ID returns the codec identifier.
func (p *Prefix) ID() ID {
	return IDPrefix
}

// Name returns the human-readable name of the codec.
func (p *Prefix) Name() string {
	return "Prefix"
}

// PreservesSize returns false since Prefix changes output size.
func (p *Prefix) PreservesSize() bool {
	return false
}

// Encode compresses data by extracting common prefixes from consecutive strings.
//
// Input format (src):
//
//	[numStrings: 4 bytes (uint32)]
//	For each string:
//	  [length: 4 bytes (uint32)]
//	  [data: length bytes]
//
// Output format (dst):
//
//	[numStrings: 4 bytes (uint32)]
//	For each string:
//	  [prefixLen: 2 bytes (uint16)]
//	  [suffixLen: 2 bytes (uint16)]
//	  [suffix: suffixLen bytes]
//
// Params: None required
func (p *Prefix) Encode(dst, src, params []byte) (int, error) {
	if len(src) < 4 {
		return 0, fmt.Errorf("prefix: input too small (need at least 4 bytes for count)")
	}

	// Read number of strings
	numStrings := binary.LittleEndian.Uint32(src[0:4])
	if numStrings == 0 {
		// Empty input: just write count
		binary.LittleEndian.PutUint32(dst[0:4], 0)
		return 4, nil
	}

	// Parse input strings
	strings := make([][]byte, numStrings)
	pos := 4
	for i := uint32(0); i < numStrings; i++ {
		if pos+4 > len(src) {
			return 0, fmt.Errorf("prefix: incomplete string %d header", i)
		}
		strLen := binary.LittleEndian.Uint32(src[pos : pos+4])
		pos += 4

		if pos+int(strLen) > len(src) {
			return 0, fmt.Errorf("prefix: incomplete string %d data (need %d bytes, have %d)", i, strLen, len(src)-pos)
		}
		strings[i] = src[pos : pos+int(strLen)]
		pos += int(strLen)
	}

	// Encode with prefix extraction
	outPos := 0

	// Write number of strings
	if outPos+4 > len(dst) {
		return 0, ErrBufferTooSmall
	}
	binary.LittleEndian.PutUint32(dst[outPos:], numStrings)
	outPos += 4

	// Process each string
	var prevString []byte
	for i, currentString := range strings {
		// Find common prefix length with previous string
		var prefixLen uint16
		if i > 0 {
			prefixLen = uint16(commonPrefixLen(prevString, currentString))
		}

		// Suffix is the part after the prefix
		suffixLen := uint16(len(currentString)) - prefixLen

		// Write prefix length (2 bytes)
		if outPos+2 > len(dst) {
			return 0, ErrBufferTooSmall
		}
		binary.LittleEndian.PutUint16(dst[outPos:], prefixLen)
		outPos += 2

		// Write suffix length (2 bytes)
		if outPos+2 > len(dst) {
			return 0, ErrBufferTooSmall
		}
		binary.LittleEndian.PutUint16(dst[outPos:], suffixLen)
		outPos += 2

		// Write suffix data
		if outPos+int(suffixLen) > len(dst) {
			return 0, ErrBufferTooSmall
		}
		copy(dst[outPos:], currentString[prefixLen:])
		outPos += int(suffixLen)

		prevString = currentString
	}

	return outPos, nil
}

// Decode decompresses data by reconstructing strings from prefix lengths and suffixes.
//
// Input format (src):
//
//	[numStrings: 4 bytes (uint32)]
//	For each string:
//	  [prefixLen: 2 bytes (uint16)]
//	  [suffixLen: 2 bytes (uint16)]
//	  [suffix: suffixLen bytes]
//
// Output format (dst):
//
//	[numStrings: 4 bytes (uint32)]
//	For each string:
//	  [length: 4 bytes (uint32)]
//	  [data: length bytes]
//
// Params: None required
func (p *Prefix) Decode(dst, src, params []byte) (int, error) {
	if len(src) < 4 {
		return 0, fmt.Errorf("prefix: compressed data too small")
	}

	// Read number of strings
	numStrings := binary.LittleEndian.Uint32(src[0:4])
	if numStrings == 0 {
		// Empty input: just write count
		binary.LittleEndian.PutUint32(dst[0:4], 0)
		return 4, nil
	}

	inPos := 4
	outPos := 0

	// Write number of strings
	if outPos+4 > len(dst) {
		return 0, ErrBufferTooSmall
	}
	binary.LittleEndian.PutUint32(dst[outPos:], numStrings)
	outPos += 4

	// Reconstruct each string
	var prevString []byte
	for i := uint32(0); i < numStrings; i++ {
		// Read prefix length
		if inPos+2 > len(src) {
			return 0, fmt.Errorf("prefix: incomplete prefix length for string %d", i)
		}
		prefixLen := binary.LittleEndian.Uint16(src[inPos:])
		inPos += 2

		// Read suffix length
		if inPos+2 > len(src) {
			return 0, fmt.Errorf("prefix: incomplete suffix length for string %d", i)
		}
		suffixLen := binary.LittleEndian.Uint16(src[inPos:])
		inPos += 2

		// Validate prefix length
		if int(prefixLen) > len(prevString) {
			return 0, fmt.Errorf("prefix: invalid prefix length %d (previous string length %d)", prefixLen, len(prevString))
		}

		// Calculate total string length
		totalLen := uint32(prefixLen) + uint32(suffixLen)

		// Write string length
		if outPos+4 > len(dst) {
			return 0, ErrBufferTooSmall
		}
		binary.LittleEndian.PutUint32(dst[outPos:], totalLen)
		outPos += 4

		// Write prefix (from previous string)
		if outPos+int(prefixLen) > len(dst) {
			return 0, ErrBufferTooSmall
		}
		if prefixLen > 0 {
			copy(dst[outPos:], prevString[:prefixLen])
			outPos += int(prefixLen)
		}

		// Read and write suffix
		if inPos+int(suffixLen) > len(src) {
			return 0, fmt.Errorf("prefix: incomplete suffix data for string %d", i)
		}
		if outPos+int(suffixLen) > len(dst) {
			return 0, ErrBufferTooSmall
		}
		copy(dst[outPos:], src[inPos:inPos+int(suffixLen)])
		outPos += int(suffixLen)
		inPos += int(suffixLen)

		// Save current string for next iteration
		prevString = dst[outPos-int(totalLen) : outPos]
	}

	return outPos, nil
}

// String returns a human-readable name for the codec.
func (p *Prefix) String() string {
	return "Prefix"
}

// commonPrefixLen returns the length of the common prefix between two byte slices.
func commonPrefixLen(a, b []byte) int {
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}

	for i := 0; i < minLen; i++ {
		if a[i] != b[i] {
			return i
		}
	}

	return minLen
}
