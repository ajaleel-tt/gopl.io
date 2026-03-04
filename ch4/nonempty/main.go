// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

// See page 91.

//!+nonempty

// Nonempty is an example of an in-place slice algorithm.
package main

import "fmt"

// nonempty returns a slice holding only the non-empty strings.
// The underlying array is modified during the call.
func nonempty(strings []string) []string {
	i := 0
	for _, s := range strings {
		if s != "" {
			strings[i] = s
			i++
		}
	}
	return strings[:i]
}

//!-nonempty

func main() {
	//!+main
	data := []string{"one", "", "three"}
	fmt.Printf("%q\n", nonempty(data)) // `["one" "three"]`
	fmt.Printf("%q\n", data)           // `["one" "three" "three"]`
	//!-main
}

// !+alt
//
//goland:noinspection GoUnusedFunction
func nonempty2(strings []string) []string {
	out := strings[:0] // zero-length slice of original
	for _, s := range strings {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

//!-alt

// removeAdjacentDups removes adjacent duplicates from a string slice in place.
//
//goland:noinspection GoUnusedFunction
func removeAdjacentDups(strings []string) []string {
	if len(strings) == 0 {
		return strings
	}
	i := 0
	for j := 1; j < len(strings); j++ {
		if strings[j] != strings[i] {
			i++
			strings[i] = strings[j]
		}
	}
	return strings[:i+1]
}
