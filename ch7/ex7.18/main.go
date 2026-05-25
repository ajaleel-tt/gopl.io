// Exercise 7.18: build a generic node tree from an XML token stream.
//
// Parse consumes all top-level tokens and returns the resulting forest
// (typically one *Element plus surrounding whitespace CharData). The
// program reads XML from stdin and prints the tree to stdout.
//
// Notes / decisions:
//   - The xml.CharData returned by Token() shares an internal buffer
//     that is reused on the next call; we copy it into our own CharData
//     value before retaining it.
//   - Comments, processing instructions, and directives are dropped:
//     Node is defined as CharData | *Element and nothing else.
package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
)

// Node is satisfied by CharData and *Element.
type Node interface{}

type CharData string

type Element struct {
	Type     xml.Name
	Attr     []xml.Attr
	Children []Node
}

// Parse reads XML tokens from r and returns the resulting top-level
// nodes. A well-formed document yields a single *Element as the root,
// possibly surrounded by whitespace CharData.
func Parse(r io.Reader) ([]Node, error) {
	dec := xml.NewDecoder(r)

	// Stack of currently-open elements. nil top means "top level".
	var stack []*Element
	var top []Node // collects top-level nodes when stack is empty

	// append appends n to the current parent, or to top if at root.
	add := func(n Node) {
		if len(stack) == 0 {
			top = append(top, n)
			return
		}
		parent := stack[len(stack)-1]
		parent.Children = append(parent.Children, n)
	}

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch tok := tok.(type) {
		case xml.StartElement:
			// Copy Attr — Decoder reuses the backing array across tokens.
			attrs := make([]xml.Attr, len(tok.Attr))
			copy(attrs, tok.Attr)
			el := &Element{Type: tok.Name, Attr: attrs}
			add(el)
			stack = append(stack, el)
		case xml.EndElement:
			if len(stack) == 0 {
				return nil, fmt.Errorf("unmatched </%s>", tok.Name.Local)
			}
			stack = stack[:len(stack)-1]
		case xml.CharData:
			// Decoder reuses the underlying slice — must copy.
			text := make([]byte, len(tok))
			copy(text, tok)
			add(CharData(text))
		}
		// xml.Comment, xml.ProcInst, xml.Directive: ignored by design.
	}
	if len(stack) != 0 {
		open := make([]string, len(stack))
		for i, e := range stack {
			open[i] = e.Type.Local
		}
		return nil, fmt.Errorf("unclosed elements: %s", strings.Join(open, ", "))
	}
	return top, nil
}

// ---- pretty-print, for the demo main ----

// Format writes a human-readable, indented view of the tree. Whitespace-only
// CharData is skipped to keep output tidy.
func Format(w io.Writer, nodes []Node, indent string) {
	for _, n := range nodes {
		switch n := n.(type) {
		case *Element:
			fmt.Fprintf(w, "%s<%s", indent, qname(n.Type))
			for _, a := range n.Attr {
				fmt.Fprintf(w, " %s=%q", qname(a.Name), a.Value)
			}
			if len(n.Children) == 0 {
				fmt.Fprintln(w, "/>")
				continue
			}
			fmt.Fprintln(w, ">")
			Format(w, n.Children, indent+"  ")
			fmt.Fprintf(w, "%s</%s>\n", indent, qname(n.Type))
		case CharData:
			s := strings.TrimSpace(string(n))
			if s != "" {
				fmt.Fprintf(w, "%s%q\n", indent, s)
			}
		}
	}
}

func qname(n xml.Name) string {
	if n.Space == "" {
		return n.Local
	}
	return n.Space + ":" + n.Local
}

func main() {
	nodes, err := Parse(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse:", err)
		os.Exit(1)
	}
	Format(os.Stdout, nodes, "")
}
