// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

// Exercise 4.6: squashSpaces squashes each run of adjacent Unicode spaces
// in a UTF-8 encoded []byte slice into a single ASCII space.
package main

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

func main() {
	// Test cases
	tests := []string{
		"hello   world",
		"a\u2003\u2003b", // EM SPACE (3 bytes each)
		"  leading",
		"trailing  ",
		"a\t\n\rb", // tabs, newlines
		"no spaces",
		"   ", // only spaces
		"",    // empty
	}

	for _, s := range tests {
		input := []byte(s)
		result := squashSpaces(input)
		fmt.Printf("%q -> %q\n", s, result)
	}
}

// squashSpaces replaces runs of adjacent Unicode whitespace with a single ASCII space.
// It operates in-place on the input slice and returns the shortened slice.
func squashSpaces(data []byte) []byte {
	w := 0           // write position
	inSpace := false // currently in a whitespace run?

	for r := 0; r < len(data); {
		runeValue, runeWidth := utf8.DecodeRune(data[r:])

		if unicode.IsSpace(runeValue) {
			if !inSpace {
				data[w] = ' ' // emit single ASCII space
				w++
				inSpace = true
			}
			r += runeWidth // skip this space
		} else {
			inSpace = false
			if w != r {
				copy(data[w:w+runeWidth], data[r:r+runeWidth])
			}
			w += runeWidth
			r += runeWidth
		}
	}
	return data[:w]
}
