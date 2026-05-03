// Exercise 5.7: Pretty-print HTML documents with comments, text, and attributes.
package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/html"
)

func main() {
	for _, url := range os.Args[1:] {
		if err := outline(url, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "prettyprint: %v\n", err)
		}
	}
}

func outline(url string, w io.Writer) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return err
	}

	prettyPrint(w, doc)
	return nil
}

var (
	out   io.Writer
	depth int
)

func prettyPrint(w io.Writer, doc *html.Node) {
	out = w
	depth = 0
	forEachNode(doc, startNode, endNode)
}

func startNode(n *html.Node) {
	switch n.Type {
	case html.ElementNode:
		attrs := formatAttrs(n)
		if n.FirstChild == nil {
			fmt.Fprintf(out, "%s<%s%s/>\n", indent(depth), n.Data, attrs)
		} else {
			fmt.Fprintf(out, "%s<%s%s>\n", indent(depth), n.Data, attrs)
		}
		depth++
	case html.TextNode:
		text := strings.TrimSpace(n.Data)
		if text == "" {
			return
		}
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				fmt.Fprintf(out, "%s%s\n", indent(depth), line)
			}
		}
	case html.CommentNode:
		fmt.Fprintf(out, "%s<!--%s-->\n", indent(depth), n.Data)
	case html.DoctypeNode:
		fmt.Fprintf(out, "<!DOCTYPE %s>\n", n.Data)
	}
}

func endNode(n *html.Node) {
	if n.Type != html.ElementNode {
		return
	}
	depth--
	if n.FirstChild != nil {
		fmt.Fprintf(out, "%s</%s>\n", indent(depth), n.Data)
	}
}

func indent(depth int) string {
	return strings.Repeat("  ", depth)
}

func formatAttrs(n *html.Node) string {
	if len(n.Attr) == 0 {
		return ""
	}
	var b strings.Builder
	for _, a := range n.Attr {
		fmt.Fprintf(&b, " %s='%s'", a.Key, a.Val)
	}
	return b.String()
}

func forEachNode(n *html.Node, pre, post func(n *html.Node)) {
	if pre != nil {
		pre(n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		forEachNode(c, pre, post)
	}
	if post != nil {
		post(n)
	}
}
