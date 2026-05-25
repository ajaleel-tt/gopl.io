package main

import (
	"fmt"
	"testing"
)

func TestWordCounterBasic(t *testing.T) {
	cases := []struct {
		input string
		want  WordCounter
	}{
		{"", 0},
		{"hello", 1},
		{"hello world", 2},
		{"  spaces   between   words  ", 3},
		{"tab\tseparated\tword", 3},
		{"line1\nline2\nline3", 3},
		{"mixed \t whitespace\n separators", 3},
	}
	for _, tc := range cases {
		var w WordCounter
		w.Write([]byte(tc.input))
		if w != tc.want {
			t.Errorf("WordCounter(%q) = %d, want %d", tc.input, w, tc.want)
		}
	}
}

func TestWordCounterViaFprintf(t *testing.T) {
	var w WordCounter
	fmt.Fprintf(&w, "the quick brown fox jumps over the lazy dog")
	if w != 9 {
		t.Errorf("WordCounter via Fprintf = %d, want 9", w)
	}
}

func TestWordCounterReset(t *testing.T) {
	var w WordCounter
	w.Write([]byte("a b c"))
	w = 0
	w.Write([]byte("x y"))
	if w != 2 {
		t.Errorf("after reset, WordCounter = %d, want 2", w)
	}
}

func TestLineCounterBasic(t *testing.T) {
	cases := []struct {
		input string
		want  LineCounter
	}{
		{"", 0},
		{"one line", 1},
		{"one\ntwo\nthree", 3},
		{"one\ntwo\nthree\n", 3},
		{"\n\n\n", 3},
		{"hello world\nfoo bar", 2},
	}
	for _, tc := range cases {
		var l LineCounter
		l.Write([]byte(tc.input))
		if l != tc.want {
			t.Errorf("LineCounter(%q) = %d, want %d", tc.input, l, tc.want)
		}
	}
}

func TestLineCounterViaFprintf(t *testing.T) {
	var l LineCounter
	fmt.Fprintf(&l, "first\nsecond\nthird\n")
	if l != 3 {
		t.Errorf("LineCounter via Fprintf = %d, want 3", l)
	}
}

func TestWriteReturnsCorrectByteCount(t *testing.T) {
	// io.Writer contract: return n equal to the byte count consumed.
	for _, input := range []string{"", "x", "hello world", "multi\nline\ntext"} {
		var w WordCounter
		n, err := w.Write([]byte(input))
		if err != nil {
			t.Errorf("WordCounter.Write(%q) returned error: %v", input, err)
		}
		if n != len(input) {
			t.Errorf("WordCounter.Write(%q) returned n=%d, want %d", input, n, len(input))
		}

		var l LineCounter
		n, err = l.Write([]byte(input))
		if err != nil {
			t.Errorf("LineCounter.Write(%q) returned error: %v", input, err)
		}
		if n != len(input) {
			t.Errorf("LineCounter.Write(%q) returned n=%d, want %d", input, n, len(input))
		}
	}
}

// TestWordCounterFragmentationGotcha documents the per-Write limitation:
// a word split across two Write calls is counted twice.
func TestWordCounterFragmentationGotcha(t *testing.T) {
	var w WordCounter
	w.Write([]byte("hel"))
	w.Write([]byte("lo"))
	if w != 2 {
		t.Errorf("got %d, want 2 — fragmentation gotcha changed unexpectedly", w)
	}
}