package main

import (
	"io"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestReadFullStringInOneCall(t *testing.T) {
	r := NewReader("hello")
	buf := make([]byte, 10)

	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("first Read returned err = %v, want nil", err)
	}
	if n != 5 {
		t.Errorf("first Read returned n = %d, want 5", n)
	}
	if string(buf[:n]) != "hello" {
		t.Errorf("got %q, want %q", buf[:n], "hello")
	}

	// Next read should be EOF.
	n, err = r.Read(buf)
	if err != io.EOF {
		t.Errorf("second Read err = %v, want io.EOF", err)
	}
	if n != 0 {
		t.Errorf("second Read n = %d, want 0", n)
	}
}

func TestReadInChunks(t *testing.T) {
	r := NewReader("hello world")
	buf := make([]byte, 3)

	var collected []byte
	for {
		n, err := r.Read(buf)
		collected = append(collected, buf[:n]...)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
	}
	if string(collected) != "hello world" {
		t.Errorf("collected = %q, want %q", collected, "hello world")
	}
}

func TestEmptyStringReturnsImmediateEOF(t *testing.T) {
	r := NewReader("")
	buf := make([]byte, 10)

	n, err := r.Read(buf)
	if err != io.EOF {
		t.Errorf("Read on empty string err = %v, want io.EOF", err)
	}
	if n != 0 {
		t.Errorf("Read on empty string n = %d, want 0", n)
	}
}

// TestInteropWithReadAll verifies our reader works with standard library
// consumers that take io.Reader.
func TestInteropWithReadAll(t *testing.T) {
	const want = "the quick brown fox jumps over the lazy dog"
	r := NewReader(want)

	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	if string(got) != want {
		t.Errorf("io.ReadAll got %q, want %q", got, want)
	}
}

// TestInteropWithHTMLParse is the exercise's stated goal: feed an in-memory
// string into html.Parse via our NewReader.
func TestInteropWithHTMLParse(t *testing.T) {
	const page = `<html><body><h1 id="title">Hi</h1></body></html>`
	doc, err := html.Parse(NewReader(page))
	if err != nil {
		t.Fatalf("html.Parse: %v", err)
	}

	// Walk for an <h1> with id="title".
	var found bool
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "h1" {
			for _, a := range n.Attr {
				if a.Key == "id" && a.Val == "title" {
					found = true
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(doc)

	if !found {
		t.Error("html.Parse via NewReader did not produce expected DOM")
	}
}

// TestParityWithStringsNewReader cross-checks our implementation against the
// standard library by reading the same content through both readers and
// asserting they produce identical byte streams.
func TestParityWithStringsNewReader(t *testing.T) {
	const content = "compare me byte-for-byte"

	ours, err := io.ReadAll(NewReader(content))
	if err != nil {
		t.Fatalf("our reader: %v", err)
	}
	theirs, err := io.ReadAll(strings.NewReader(content))
	if err != nil {
		t.Fatalf("strings.NewReader: %v", err)
	}
	if string(ours) != string(theirs) {
		t.Errorf("byte streams differ: ours=%q stdlib=%q", ours, theirs)
	}
}

func TestReadWithSmallerBufferAdvancesPosition(t *testing.T) {
	r := NewReader("abc")
	buf := make([]byte, 1)

	for i, want := range "abc" {
		n, err := r.Read(buf)
		if n != 1 {
			t.Errorf("step %d: n = %d, want 1", i, n)
		}
		if err != nil {
			t.Errorf("step %d: err = %v, want nil", i, err)
		}
		if buf[0] != byte(want) {
			t.Errorf("step %d: got %c, want %c", i, buf[0], want)
		}
	}

	_, err := r.Read(buf)
	if err != io.EOF {
		t.Errorf("final read err = %v, want io.EOF", err)
	}
}
