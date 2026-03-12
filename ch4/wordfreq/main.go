// Wordfreq reports the frequency of each word in an input text file.
package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		_, err := fmt.Fprintf(os.Stderr, "usage: wordfreq <filename>\n")
		if err != nil {
			return
		}
		os.Exit(1)
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		_, err := fmt.Fprintf(os.Stderr, "wordfreq: %v\n", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			return
		}
	}(f)

	counts := make(map[string]int)

	input := bufio.NewScanner(f)
	input.Split(bufio.ScanWords)
	for input.Scan() {
		counts[input.Text()]++
	}
	if err := input.Err(); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "wordfreq: %v\n", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}

	fmt.Printf("word\tcount\n")
	for word, n := range counts {
		fmt.Printf("%s\t%d\n", word, n)
	}
}
