// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package codec

import (
	"encoding/binary"
	"fmt"
)

// LZ77 implements the LZ77 dictionary compression algorithm.
//
// LZ77 is a lossless compression algorithm that replaces repeated occurrences
// of data with references to earlier occurrences. This is the foundation of
// gzip, zlib, and many other compression formats.
//
// Algorithm:
//  1. Maintain a sliding window of recent data (default 32KB)
//  2. For each position, search for longest match in window
//  3. Output either:
//     - Literal: single byte (no match found)
//     - Match: (distance, length) pair pointing to previous occurrence
//
// Example:
//
//	Input:  "Hello, Hello, World!"
//	Output: "Hello, " + Match(7,6) + " World!"
//	        (Match points back 7 bytes, copies 6 bytes)
//
// This is critical for JSON/text compression:
//   - Repeated field names: "password_id" appears 141 times → 1 + 140 references
//   - Common prefixes: "CN=COMPUTER", "DC=ladpss,DC=org"
//   - UUID patterns: only suffix varies
//
// Expected performance:
//   - JSON: 7-10x compression (before entropy coding)
//   - Text: 3-5x compression
//   - Binary: 1.5-3x compression
//
// Combined with Huffman/FSE:
//   - JSON: 15-20x total compression (competitive with zstd)
type LZ77 struct {
	windowSize int // Sliding window size (default 32KB)
	maxMatch   int // Maximum match length (default 258)
	minMatch   int // Minimum match length (default 3)
}

// NewLZ77 creates a new LZ77 codec with default parameters.
func NewLZ77() *LZ77 {
	return &LZ77{
		windowSize: 32 * 1024, // 32KB window (same as gzip)
		maxMatch:   258,       // Maximum match length (DEFLATE standard)
		minMatch:   3,         // Minimum match length (shorter matches waste space)
	}
}

// NewLZ77WithWindow creates an LZ77 codec with custom window size.
func NewLZ77WithWindow(windowSize int) *LZ77 {
	return &LZ77{
		windowSize: windowSize,
		maxMatch:   258,
		minMatch:   3,
	}
}

// ID returns the codec identifier
func (c *LZ77) ID() ID {
	return IDLZ77
}

// Name returns the codec name
func (c *LZ77) Name() string {
	return "LZ77"
}

// PreservesSize returns false because LZ77 changes size.
//
// LZ77 compresses data by finding repeated patterns and replacing them
// with shorter back-references. The output size depends on how much
// redundancy exists in the input.
func (c *LZ77) PreservesSize() bool {
	return false
}

// Token represents either a literal or a match in LZ77 encoding
type Token struct {
	isLiteral bool   // true = literal, false = match
	literal   byte   // literal byte value (if isLiteral)
	distance  uint16 // match distance (if !isLiteral)
	length    uint16 // match length (if !isLiteral)
}

// Encode compresses data using LZ77 dictionary compression.
//
// Output format (token stream):
//
//	[num_tokens(4)] [tokens...]
//	Each token:
//	  - Literal: [type=0(1)] [byte(1)]
//	  - Match:   [type=1(1)] [distance(2)] [length(2)]
//
// This preserves the order of literals and matches, making decode trivial.
func (c *LZ77) Encode(dst, src, params []byte) (int, error) {
	if len(src) == 0 {
		// Empty input
		binary.LittleEndian.PutUint32(dst[0:], 0) // num_tokens
		return 4, nil
	}

	// Build hash table for fast string matching
	hash := NewHashTable(c.windowSize)

	var tokens []Token

	pos := 0
	for pos < len(src) {
		// Find longest match in sliding window
		bestDist, bestLen := c.findMatch(src, pos, hash)

		if bestLen >= c.minMatch {
			// Found a good match - emit it
			tokens = append(tokens, Token{
				isLiteral: false,
				distance:  uint16(bestDist),
				length:    uint16(bestLen),
			})
			// Update hash table for all positions in match
			for i := 0; i < bestLen && pos < len(src); i++ {
				hash.Insert(src, pos)
				pos++
			}
		} else {
			// No match - emit literal byte
			tokens = append(tokens, Token{
				isLiteral: true,
				literal:   src[pos],
			})
			hash.Insert(src, pos)
			pos++
		}
	}

	// Encode output
	return c.encodeTokens(dst, tokens)
}

// Decode decompresses LZ77-encoded data back to original.
//
// The decoder is simple: read tokens and execute them in order.
func (c *LZ77) Decode(dst, src, params []byte) (int, error) {
	if len(src) < 4 {
		return 0, fmt.Errorf("lz77: input too small (need at least 4 bytes)")
	}

	// Parse header
	numTokens := binary.LittleEndian.Uint32(src[0:])
	if numTokens == 0 {
		return 0, nil // Empty output
	}

	// Decode tokens
	outPos := 0
	srcPos := 4

	for i := uint32(0); i < numTokens; i++ {
		if srcPos >= len(src) {
			return 0, fmt.Errorf("lz77: unexpected end of input at token %d", i)
		}

		tokenType := src[srcPos]
		srcPos++

		if tokenType == 0 {
			// Literal
			if srcPos >= len(src) {
				return 0, fmt.Errorf("lz77: unexpected end of input reading literal")
			}
			if outPos >= len(dst) {
				return 0, ErrBufferTooSmall
			}
			dst[outPos] = src[srcPos]
			outPos++
			srcPos++
		} else {
			// Match
			if srcPos+4 > len(src) {
				return 0, fmt.Errorf("lz77: unexpected end of input reading match")
			}
			distance := binary.LittleEndian.Uint16(src[srcPos:])
			length := binary.LittleEndian.Uint16(src[srcPos+2:])
			srcPos += 4

			// Validate distance
			if int(distance) > outPos {
				return 0, fmt.Errorf("lz77: invalid distance %d at position %d", distance, outPos)
			}

			// Copy from earlier position
			copyPos := outPos - int(distance)
			for j := 0; j < int(length); j++ {
				if outPos >= len(dst) {
					return 0, ErrBufferTooSmall
				}
				dst[outPos] = dst[copyPos]
				outPos++
				copyPos++
			}
		}
	}

	return outPos, nil
}

// findMatch searches for the longest match in the sliding window.
//
// Uses hash table for O(1) candidate lookup instead of O(n) linear search.
//
// Returns: (distance, length) of best match, or (0, 0) if no match found.
func (c *LZ77) findMatch(src []byte, pos int, hash *HashTable) (int, int) {
	if pos+c.minMatch > len(src) {
		return 0, 0 // Not enough data for minimum match
	}

	// Get candidates from hash table
	candidates := hash.Lookup(src, pos)

	bestDist := 0
	bestLen := 0

	for _, candPos := range candidates {
		// Check if candidate is within window
		dist := pos - candPos
		if dist <= 0 || dist > c.windowSize {
			continue
		}

		// Calculate match length
		matchLen := 0
		for matchLen < c.maxMatch &&
			pos+matchLen < len(src) &&
			candPos+matchLen < len(src) &&
			src[pos+matchLen] == src[candPos+matchLen] {
			matchLen++
		}

		if matchLen > bestLen {
			bestLen = matchLen
			bestDist = dist
		}
	}

	return bestDist, bestLen
}

// encodeTokens writes tokens to output buffer in token stream format.
//
// Format:
//
//	[num_tokens(4)] [token1] [token2] ...
//
// Each token:
//   - Literal: [type=0(1)] [byte(1)]          = 2 bytes
//   - Match:   [type=1(1)] [distance(2)] [length(2)] = 5 bytes
func (c *LZ77) encodeTokens(dst []byte, tokens []Token) (int, error) {
	// Calculate required size
	requiredSize := 4 // num_tokens header
	for _, token := range tokens {
		if token.isLiteral {
			requiredSize += 2 // type + literal byte
		} else {
			requiredSize += 5 // type + distance + length
		}
	}

	if len(dst) < requiredSize {
		return 0, ErrBufferTooSmall
	}

	// Write number of tokens
	binary.LittleEndian.PutUint32(dst[0:], uint32(len(tokens)))
	offset := 4

	// Write each token
	for _, token := range tokens {
		if token.isLiteral {
			// Literal: type=0, then byte
			dst[offset] = 0
			dst[offset+1] = token.literal
			offset += 2
		} else {
			// Match: type=1, then distance, then length
			dst[offset] = 1
			binary.LittleEndian.PutUint16(dst[offset+1:], token.distance)
			binary.LittleEndian.PutUint16(dst[offset+3:], token.length)
			offset += 5
		}
	}

	return offset, nil
}

// HashTable provides fast string matching for LZ77 compression.
//
// Maps 3-byte sequences to their positions in the input.
// Uses simple hash function: (b0 << 16) | (b1 << 8) | b2
type HashTable struct {
	table    map[uint32][]int
	maxChain int // Maximum positions to store per hash
}

// NewHashTable creates a new hash table for LZ77 matching
func NewHashTable(windowSize int) *HashTable {
	return &HashTable{
		table:    make(map[uint32][]int, windowSize/4),
		maxChain: 16, // Limit chain length to avoid slowdown
	}
}

// Insert adds a position to the hash table
func (h *HashTable) Insert(data []byte, pos int) {
	if pos+3 > len(data) {
		return // Need at least 3 bytes for hash
	}

	hash := h.hash3(data[pos : pos+3])
	chain := h.table[hash]

	// Limit chain length
	if len(chain) >= h.maxChain {
		// Remove oldest entry
		chain = chain[1:]
	}

	h.table[hash] = append(chain, pos)
}

// Lookup finds candidate positions for matching
func (h *HashTable) Lookup(data []byte, pos int) []int {
	if pos+3 > len(data) {
		return nil
	}

	hash := h.hash3(data[pos : pos+3])
	return h.table[hash]
}

// hash3 computes hash of 3-byte sequence
func (h *HashTable) hash3(b []byte) uint32 {
	return (uint32(b[0]) << 16) | (uint32(b[1]) << 8) | uint32(b[2])
}
