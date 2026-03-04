// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

// See page 86.

// Rev reverses a slice.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"
)

func main() {
	//!+array
	a := [...]int{0, 1, 2, 3, 4, 5}
	reverse(a[:])
	fmt.Println(a) // "[5 4 3 2 1 0]"
	//!-array

	//!+utf8
	// Test UTF-8 reversal with multibyte characters
	utf8str := []byte("Hello, 世界!")
	fmt.Printf("Before: %s\n", utf8str)
	reverseUTF8(utf8str)
	fmt.Printf("After:  %s\n", utf8str) // "!界世 ,olleH"
	//!-utf8

	//!+slice
	s := []int{0, 1, 2, 3, 4, 5}
	// Rotate s left by two positions.
	reverse(s[:2])
	reverse(s[2:])
	reverse(s)
	fmt.Println(s) // "[2 3 4 5 0 1]"
	//!-slice

	// Interactive test of reverse.
	input := bufio.NewScanner(os.Stdin)
outer:
	for input.Scan() {
		var ints []int
		for _, s := range strings.Fields(input.Text()) {
			x, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, err)
				continue outer
			}
			ints = append(ints, int(x))
		}
		reverse(ints)
		fmt.Printf("%v\n", ints)
	}
	// NOTE: ignoring potential errors from input.Err()
}

// !+rev
// reverse reverses a slice of ints in place.
func reverse(s []int) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

//!-rev

// reverseArray reverses an array of ints in place using an array pointer.
//
//goland:noinspection GoUnusedFunction
func reverseArray(s *[6]int) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

// reverseUTF8 reverses a UTF-8 encoded byte slice in place without allocating.
// Strategy: move each rune from the back to its final position at the front.
func reverseUTF8(b []byte) {
	// Position where next reversed rune should go
	dest := 0

	for dest < len(b) {
		// Find the last rune in the unreversed portion
		_, size := utf8.DecodeLastRune(b[dest:])
		if size == 0 {
			break
		}

		// Rotate b[dest:] left by (len(b[dest:])-size) positions
		// This moves the last rune to position dest
		// Using triple-reverse: reverse first part, reverse second part, reverse all
		src := len(b) - size
		if src > dest {
			reverseBytes(b[dest:src])
			reverseBytes(b[src:])
			reverseBytes(b[dest:])
		}

		dest += size
	}
}

// reverseBytes reverses a byte slice in place.
func reverseBytes(b []byte) {
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
}

// rotate rotates a slice left by n positions in a single pass.
//
//goland:noinspection GoUnusedFunction
func rotate(s []int, n int) {
	if len(s) == 0 {
		return
	}
	n = n % len(s) //
	if n < 0 {
		n += len(s)
	}
	if n == 0 {
		return
	}

	count := 0
	for start := 0; count < len(s); start++ {
		current := start
		prev := s[start]
		for {
			next := (current + n) % len(s)
			prev, s[next] = s[next], prev
			current = next
			count++
			if current == start {
				break
			}
		}
	}
}
