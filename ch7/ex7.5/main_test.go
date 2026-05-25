package main

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestLimitReaderLessThanLimit(t *testing.T) {
	src := strings.NewReader("short")
	lr := LimitReader(src, 100)

	got, err := io.ReadAll(lr)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(got) != "short" {
		t.Errorf("got %q, want %q", got, "short")
	}
}

func TestLimitReaderCapsAtLimit(t *testing.T) {
	src := strings.NewReader("the quick brown fox")
	lr := LimitReader(src, 9)

	got, err := io.ReadAll(lr)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(got) != "the quick" {
		t.Errorf("got %q, want %q", got, "the quick")
	}
}

func TestLimitReaderExactlyAtLimit(t *testing.T) {
	src := strings.NewReader("hello")
	lr := LimitReader(src, 5)

	got, err := io.ReadAll(lr)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestLimitReaderZeroLimit(t *testing.T) {
	src := strings.NewReader("anything")
	lr := LimitReader(src, 0)

	got, err := io.ReadAll(lr)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d bytes, want 0", len(got))
	}
}

func TestLimitReaderNegativeLimit(t *testing.T) {
	src := strings.NewReader("anything")
	lr := LimitReader(src, -5)

	got, _ := io.ReadAll(lr)
	if len(got) != 0 {
		t.Errorf("got %d bytes, want 0 for negative limit", len(got))
	}
}

func TestLimitReaderInChunks(t *testing.T) {
	src := strings.NewReader("abcdefghij")
	lr := LimitReader(src, 7)

	var collected []byte
	buf := make([]byte, 3)
	for {
		n, err := lr.Read(buf)
		collected = append(collected, buf[:n]...)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
	}
	if string(collected) != "abcdefg" {
		t.Errorf("collected = %q, want %q", collected, "abcdefg")
	}
}

func TestLimitReaderDoesNotConsumeBeyondLimit(t *testing.T) {
	// After LimitReader returns EOF, the underlying reader should still
	// have its untouched tail available.
	src := strings.NewReader("ABCDEFGHIJ")
	lr := LimitReader(src, 4)

	consumed, err := io.ReadAll(lr)
	if err != nil {
		t.Fatalf("ReadAll on limited: %v", err)
	}
	if string(consumed) != "ABCD" {
		t.Errorf("limited reader gave %q, want %q", consumed, "ABCD")
	}

	// The underlying source should still hold "EFGHIJ".
	rest, err := io.ReadAll(src)
	if err != nil {
		t.Fatalf("ReadAll on underlying source: %v", err)
	}
	if string(rest) != "EFGHIJ" {
		t.Errorf("underlying source had %q remaining, want %q", rest, "EFGHIJ")
	}
}

func TestLimitReaderShorterSource(t *testing.T) {
	// If the underlying source ends before the limit is reached, we should
	// surface the source's EOF normally.
	src := strings.NewReader("hi")
	lr := LimitReader(src, 100)

	got, err := io.ReadAll(lr)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(got) != "hi" {
		t.Errorf("got %q, want %q", got, "hi")
	}
}

func TestLimitReaderParityWithStdlib(t *testing.T) {
	const content = "the quick brown fox jumps over the lazy dog"
	const n = 17

	ours, err := io.ReadAll(LimitReader(strings.NewReader(content), n))
	if err != nil {
		t.Fatalf("ours: %v", err)
	}
	theirs, err := io.ReadAll(io.LimitReader(strings.NewReader(content), n))
	if err != nil {
		t.Fatalf("stdlib: %v", err)
	}
	if !bytes.Equal(ours, theirs) {
		t.Errorf("byte streams differ: ours=%q stdlib=%q", ours, theirs)
	}
}

// errReader returns a fixed error on Read. Used to verify error propagation.
type errReader struct{ err error }

func (e *errReader) Read(p []byte) (int, error) { return 0, e.err }

func TestLimitReaderPropagatesUnderlyingError(t *testing.T) {
	want := errors.New("boom")
	lr := LimitReader(&errReader{err: want}, 100)

	buf := make([]byte, 10)
	_, err := lr.Read(buf)
	if !errors.Is(err, want) {
		t.Errorf("got err = %v, want %v", err, want)
	}
}
