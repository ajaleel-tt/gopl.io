// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

// See page 83.

// The sha256 command computes the SHA256 hash (an array) of a string.
package main

import (
	"crypto/sha256"
	"crypto/sha512"
	"flag"
	"fmt"
	"os"
)

const usage = `Usage: sha256 [OPTIONS]
Compute SHA hash of strings.

Options:
  -algo string
        Hash algorithm to use: sha256, sha384, or sha512 (default "sha256")
`

var algo = flag.String(
	"algo", "sha256", "Hash algorithm: sha256, sha384, or sha512",
)

func main() {
	flag.Usage = func() {
		_, _ = fmt.Fprint(os.Stderr, usage)
	}
	flag.Parse()

	if flag.NArg() < 1 {
		_, _ = fmt.Fprintln(os.Stderr, "Error: no input provided")
		flag.Usage()
		os.Exit(1)
	}

	input := []byte(flag.Arg(0))

	switch *algo {
	case "sha256":
		fmt.Printf("%x\n", sha256.Sum256(input))
	case "sha384":
		fmt.Printf("%x\n", sha512.Sum384(input))
	case "sha512":
		fmt.Printf("%x\n", sha512.Sum512(input))
	default:
		_, _ = fmt.Fprintf(os.Stderr, "Unknown algorithm: %s\n", *algo)
		flag.Usage()
		os.Exit(1)
	}
}

//goland:noinspection GoUnusedFunction
func compareBytes(a [32]byte, b [32]byte) [32]byte {
	var c [32]byte
	for i := 0; i < 32; i++ {
		c[i] = a[i] ^ b[i]
	}
	return c
}

//!-
