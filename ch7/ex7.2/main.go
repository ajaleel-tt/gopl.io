// Exercise 7.2: CountingWriter wraps an io.Writer and exposes a live byte
// counter through a returned *int64.
package main

import (
	"fmt"
	"io"
	"os"
)

// countingWriter delegates Write to an inner writer and keeps a running tally
// of bytes actually written (as reported by the inner writer).
type countingWriter struct {
	w     io.Writer
	count int64
}

func (cw *countingWriter) Write(p []byte) (int, error) {
	n, err := cw.w.Write(p)
	cw.count += int64(n)
	return n, err
}

// CountingWriter returns an io.Writer that wraps w and a pointer to an
// int64 that, at any moment, reflects the total number of bytes that have
// been successfully written to the returned writer.
func CountingWriter(w io.Writer) (io.Writer, *int64) {
	cw := &countingWriter{w: w}
	return cw, &cw.count
}

func main() {
	wrapped, n := CountingWriter(os.Stdout)
	fmt.Fprintln(wrapped, "hello, counted world")
	fmt.Printf("wrote %d bytes\n", *n)
}
