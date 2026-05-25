package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"
)

func TestCountingWriterBasic(t *testing.T) {
	var buf bytes.Buffer
	w, n := CountingWriter(&buf)

	body := []byte("hello world")
	written, err := w.Write(body)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if written != len(body) {
		t.Errorf("Write returned %d, want %d", written, len(body))
	}
	if *n != int64(len(body)) {
		t.Errorf("counter = %d, want %d", *n, len(body))
	}
	if buf.String() != string(body) {
		t.Errorf("underlying buffer = %q, want %q", buf.String(), body)
	}
}

func TestCountingWriterAccumulates(t *testing.T) {
	var buf bytes.Buffer
	w, n := CountingWriter(&buf)

	parts := []string{"foo", "bar", "baz"}
	total := 0
	for _, p := range parts {
		w.Write([]byte(p))
		total += len(p)
		if *n != int64(total) {
			t.Errorf("after writing %d bytes, counter = %d, want %d", total, *n, total)
		}
	}
	if buf.String() != "foobarbaz" {
		t.Errorf("underlying buffer = %q, want %q", buf.String(), "foobarbaz")
	}
}

func TestCountingWriterViaFprintf(t *testing.T) {
	var buf bytes.Buffer
	w, n := CountingWriter(&buf)

	fmt.Fprintf(w, "answer is %d", 42)
	want := "answer is 42"
	if buf.String() != want {
		t.Errorf("buffer = %q, want %q", buf.String(), want)
	}
	if *n != int64(len(want)) {
		t.Errorf("counter = %d, want %d", *n, len(want))
	}
}

func TestCountingWriterDiscard(t *testing.T) {
	// Wrapping io.Discard — bytes go nowhere, but we still count them.
	w, n := CountingWriter(io.Discard)
	for i := 0; i < 5; i++ {
		fmt.Fprintln(w, "noise")
	}
	want := int64(5 * len("noise\n"))
	if *n != want {
		t.Errorf("counter wrapping io.Discard = %d, want %d", *n, want)
	}
}

// errAfterWriter writes n bytes successfully then returns an error.
type errAfterWriter struct {
	allow int
	err   error
}

func (e *errAfterWriter) Write(p []byte) (int, error) {
	if len(p) <= e.allow {
		e.allow -= len(p)
		return len(p), nil
	}
	wrote := e.allow
	e.allow = 0
	return wrote, e.err
}

// TestCountingWriterPartialWrite verifies the counter reflects bytes
// actually written, not bytes offered, when the underlying writer fails.
func TestCountingWriterPartialWrite(t *testing.T) {
	errBoom := errors.New("boom")
	inner := &errAfterWriter{allow: 5, err: errBoom}

	w, n := CountingWriter(inner)

	written, err := w.Write([]byte("hello world"))   // 11 bytes offered
	if !errors.Is(err, errBoom) {
		t.Errorf("expected errBoom, got %v", err)
	}
	if written != 5 {
		t.Errorf("Write returned %d, want 5 (partial)", written)
	}
	if *n != 5 {
		t.Errorf("counter = %d, want 5 (partial)", *n)
	}
}

// TestCountingWriterReturnedPointerIsLive confirms that the returned *int64
// stays in sync with subsequent writes — the caller doesn't need to ask
// again for an updated value.
func TestCountingWriterReturnedPointerIsLive(t *testing.T) {
	var buf bytes.Buffer
	w, n := CountingWriter(&buf)

	w.Write([]byte("a"))
	first := *n

	w.Write([]byte("bcd"))
	second := *n

	if first != 1 {
		t.Errorf("first read = %d, want 1", first)
	}
	if second != 4 {
		t.Errorf("second read = %d, want 4", second)
	}
}

// TestMultipleCountingWritersIndependent verifies that two separate
// CountingWriter calls produce independent counters.
func TestMultipleCountingWritersIndependent(t *testing.T) {
	var b1, b2 bytes.Buffer
	w1, n1 := CountingWriter(&b1)
	w2, n2 := CountingWriter(&b2)

	w1.Write([]byte("ab"))
	w2.Write([]byte("cdef"))

	if *n1 != 2 {
		t.Errorf("n1 = %d, want 2", *n1)
	}
	if *n2 != 4 {
		t.Errorf("n2 = %d, want 4", *n2)
	}
}
