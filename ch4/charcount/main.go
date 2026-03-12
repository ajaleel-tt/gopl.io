// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

// See page 97.
//!+

// Charcount computes counts of Unicode characters.
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"unicode"
	"unicode/utf8"
)

func main() {
	counts := make(map[rune]int)    // counts of Unicode characters
	var utflen [utf8.UTFMax + 1]int // count of lengths of UTF-8 encodings
	invalid := 0                    // count of invalid UTF-8 characters
	categories := map[string]int{
		"letter":  0,
		"digit":   0,
		"space":   0,
		"punct":   0,
		"symbol":  0,
		"mark":    0,
		"control": 0,
	}

	in := bufio.NewReader(os.Stdin)
	for {
		r, n, err := in.ReadRune() // returns rune, nbytes, error
		if err == io.EOF {
			break
		}
		if err != nil {
			_, err := fmt.Fprintf(os.Stderr, "char count: %v\n", err)
			if err != nil {
				return
			}
			os.Exit(1)
		}
		if r == unicode.ReplacementChar && n == 1 {
			invalid++
			continue
		}
		counts[r]++
		utflen[n]++

		switch {
		case unicode.IsLetter(r):
			categories["letter"]++
		case unicode.IsDigit(r):
			categories["digit"]++
		case unicode.IsSpace(r):
			categories["space"]++
		case unicode.IsPunct(r):
			categories["punct"]++
		case unicode.IsSymbol(r):
			categories["symbol"]++
		case unicode.IsMark(r):
			categories["mark"]++
		case unicode.IsControl(r):
			categories["control"]++
		}
	}
	fmt.Printf("rune\tcount\n")
	for c, n := range counts {
		fmt.Printf("%q\t%d\n", c, n)
	}
	fmt.Print("\nlen\tcount\n")
	for i, n := range utflen {
		if i > 0 {
			fmt.Printf("%d\t%d\n", i, n)
		}
	}
	fmt.Print("\ncat\tcount\n")
	for cat, n := range categories {
		if n > 0 {
			fmt.Printf("%s\t%d\n", cat, n)
		}
	}
	if invalid > 0 {
		fmt.Printf("\n%d invalid UTF-8 characters\n", invalid)
	}
}

//!-
