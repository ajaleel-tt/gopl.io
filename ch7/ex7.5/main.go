// Exercise 7.5: LimitReader wraps an io.Reader and reports EOF after at most
// n bytes have been read from it.
package main

import (
	"fmt"
	"io"
	"strings"
)

type limitReader struct {
	r io.Reader // underlying source
	n int64     // bytes remaining before we signal EOF
}

// LimitReader returns a Reader that reads from r but stops with EOF after
// n bytes. If n <= 0, the returned Reader is empty.
func LimitReader(r io.Reader, n int64) io.Reader {
	return &limitReader{r: r, n: n}
}

func (lr *limitReader) Read(p []byte) (int, error) {
	if lr.n <= 0 {
		return 0, io.EOF
	}
	// Don't ask the underlying reader for more bytes than we're allowed to
	// surface; trim the destination slice instead.
	if int64(len(p)) > lr.n {
		p = p[:lr.n]
	}
	n, err := lr.r.Read(p)
	lr.n -= int64(n)
	return n, err
}

func main() {
	source := strings.NewReader("the quick brown fox jumps over the lazy dog")
	limited := LimitReader(source, 19)

	data, err := io.ReadAll(limited)
	fmt.Printf("read %d bytes: %q (err=%v)\n", len(data), data, err)
}
