// format_analyzer.go - Analyze OpenZL frame format empirically
package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <frame.bin>\n", os.Args[0])
		os.Exit(1)
	}

	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=================================")
	fmt.Println("OpenZL Frame Format Analyzer")
	fmt.Println("=================================\n")

	fmt.Printf("Frame size: %d bytes\n\n", len(data))

	// Hex dump with annotations
	fmt.Println("Byte-by-byte analysis:")
	fmt.Println("Offset | Hex  | Dec  | Interpretation")
	fmt.Println("-------|------|------|---------------")

	for i := 0; i < len(data) && i < 24; i++ {
		b := data[i]
		fmt.Printf("  %02d   | 0x%02x | %4d | ", i, b, b)

		switch i {
		case 0, 1, 2, 3:
			if i == 3 {
				magic := binary.LittleEndian.Uint32(data[0:4])
				fmt.Printf("Magic = 0x%08X", magic)
			} else {
				fmt.Printf("Magic byte %d", i)
			}
		case 4:
			fmt.Printf("Version major")
		case 5:
			fmt.Printf("Version minor")
		case 6:
			flags := b
			fmt.Printf("Flags = 0b%08b", flags)
		default:
			// Try to interpret as varint
			if (b & 0x80) == 0 {
				fmt.Printf("Varint = %d (1 byte)", b)
			} else {
				fmt.Printf("Varint continuation bit set")
			}
		}
		fmt.Println()
	}

	fmt.Println("\nLooking for decompressed size = 4...")
	fmt.Println("Checking all single-byte values:")
	for i := 7; i < len(data) && i < 15; i++ {
		if data[i] == 4 {
			fmt.Printf("  Found 0x04 at offset %d!\n", i)
		}
	}

	fmt.Println("\nFull hex dump (first 24 bytes):")
	for i := 0; i < len(data) && i < 24; i++ {
		if i > 0 && i%16 == 0 {
			fmt.Println()
		}
		fmt.Printf("%02x ", data[i])
	}
	fmt.Println()
}
