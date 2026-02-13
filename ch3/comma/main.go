// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

// See page 73.

// Comma prints its argument numbers with a comma at each power of 1000.
//
// Example:
//
//	$ go build gopl.io/ch3/comma
//	$ ./comma 1 12 123 1234 1234567890
//	1
//	12
//	123
//	1,234
//	1,234,567,890
package main

import (
	"fmt"
	"os"
)

func main() {
	for i := 1; i < len(os.Args); i++ {
		fmt.Printf("  %s\n", comma(os.Args[i]))
	}
}

// !+
// comma inserts commas in a non-negative decimal integer string.
func comma(s string) string {
	recResult := commaRecursive(s, "")
	iterResult := commaIterative(s)
	if recResult != iterResult {
		panic(fmt.Sprintf("results differ: %s != %s", recResult, iterResult))
	}
	return recResult
}

func commaRecursive(s string, acc string) string {
	n := len(s)
	if n <= 3 {
		if acc == "" {
			return s
		}
		return s + "," + acc
	}
	newAcc := s[n-3:]
	if acc != "" {
		newAcc += "," + acc
	}
	return commaRecursive(s[:n-3], newAcc)
}

func commaIterative(s string) string {
	var ret []rune
	runes := []rune(s)
	reverseRunes(runes)
	for i, r := range runes {
		if i > 0 && i%3 == 0 {
			ret = append(ret, ',')
		}
		ret = append(ret, r)
	}
	reverseRunes(ret)
	return string(ret)
}

func reverseRunes(r []rune) {
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
}

//!-
