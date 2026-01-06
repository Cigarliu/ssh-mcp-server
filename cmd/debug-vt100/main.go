package main

import (
	"fmt"
	"github.com/vito/vt100"
)

func main() {
	// Create a VT100 emulator
	term := vt100.NewVT100(30, 120)

	// Simulate htop output with ANSI sequences
	htopOutput := "\x1b[100m\x1b[1mTasks: 86\x1b[0m"

	// Write to emulator
	term.Write([]byte(htopOutput))

	// Read content
	content := term.Content
	format := term.Format

	fmt.Println("=== VT100 Content Analysis ===")
	fmt.Printf("Content rows: %d\n", len(content))
	fmt.Printf("Format rows: %d\n", len(format))

	// Print first row
	if len(content) > 0 {
		row := content[0]
		fmt.Printf("\nFirst row content (%d chars):\n  ", len(row))
		for i, r := range row {
			if r == 0 {
				continue
			}
			if i > 0 && i%20 == 0 {
				fmt.Printf("\n  ")
			}
			fmt.Printf("[%c:%d] ", r, r)
		}
		fmt.Println()
	}

	// Print hex representation
	fmt.Printf("\nFirst row as hex:\n  ")
	for i, r := range content[0] {
		if r == 0 {
			continue
		}
		fmt.Printf("%02X ", r)
		if i > 0 && i%20 == 0 {
			fmt.Printf("\n  ")
		}
	}
	fmt.Println()

	// Check for 'B' character
	bCount := 0
	for y := 0; y < len(content); y++ {
		for x := 0; x < len(content[y]); x++ {
			if content[y][x] == 'B' {
				bCount++
			}
		}
	}
	fmt.Printf("\nTotal 'B' characters found: %d\n", bCount)
}
