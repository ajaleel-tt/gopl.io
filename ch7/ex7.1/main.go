// Exercise 7.1: WordCounter and LineCounter, both io.Writer implementations
// using the same single-int-as-counter pattern as ByteCounter.
//
// Caveat: counts are accumulated per Write call. If a word or line is split
// across two Write calls (e.g., "hel" then "lo"), it will count as two units
// instead of one. ByteCounter doesn't have this problem because bytes never
// span calls. A fragmentation-safe version would need to buffer the trailing
// partial token between calls; that's beyond the scope of this exercise.
package main

import (
	"bufio"
	"bytes"
	"fmt"
)

type WordCounter int

func (c *WordCounter) Write(p []byte) (int, error) {
	s := bufio.NewScanner(bytes.NewReader(p))
	s.Split(bufio.ScanWords)
	for s.Scan() {
		*c++
	}
	return len(p), nil
}

type LineCounter int

func (c *LineCounter) Write(p []byte) (int, error) {
	s := bufio.NewScanner(bytes.NewReader(p))
	s.Split(bufio.ScanLines)
	for s.Scan() {
		*c++
	}
	return len(p), nil
}

func main() {
	var w WordCounter
	w.Write([]byte("hello there world"))
	fmt.Printf("words: %d\n", w) // 3

	w = 0
	fmt.Fprintf(&w, "the quick brown fox jumps over the lazy dog")
	fmt.Printf("words: %d\n", w) // 9

	var l LineCounter
	l.Write([]byte("line one\nline two\nline three\n"))
	fmt.Printf("lines: %d\n", l) // 3

	l = 0
	fmt.Fprintf(&l, "first\nsecond\nthird")
	fmt.Printf("lines: %d\n", l) // 3 (final line has no trailing newline)
}
