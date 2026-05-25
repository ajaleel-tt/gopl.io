// Exercise 7.4: a simple strings.NewReader, used to feed an in-memory HTML
// string into golang.org/x/net/html.Parse (which takes an io.Reader).
package main

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/net/html"
)

// StringReader implements io.Reader, reading bytes sequentially from a string.
type StringReader struct {
	s   string
	pos int
}

// NewReader returns a new *StringReader reading from s.
func NewReader(s string) *StringReader {
	return &StringReader{s: s}
}

// Read copies bytes from the current position in r.s into p, advances the
// position, and returns the number of bytes copied. When the entire string
// has been consumed, subsequent calls return (0, io.EOF).
func (r *StringReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.s) {
		return 0, io.EOF
	}
	n = copy(p, r.s[r.pos:])
	r.pos += n
	return n, nil
}

func main() {
	const page = `
<html>
  <head><title>Demo</title></head>
  <body>
    <h1>Hello</h1>
    <p>Some text <a href="https://example.com">with a link</a>.</p>
  </body>
</html>`

	doc, err := html.Parse(NewReader(page))
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse: %v\n", err)
		os.Exit(1)
	}

	walk(doc, 0)
}

// walk prints the element tags of doc indented by depth, demonstrating that
// the HTML parser successfully consumed our StringReader.
func walk(n *html.Node, depth int) {
	if n.Type == html.ElementNode {
		fmt.Printf("%*s<%s>\n", depth*2, "", n.Data)
		depth++
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c, depth)
	}
}
