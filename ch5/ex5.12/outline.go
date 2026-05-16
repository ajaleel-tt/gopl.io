// Exercise 5.12: startElement/endElement turned into anonymous functions
// that share a depth variable local to the outline function.
package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"golang.org/x/net/html"
)

func main() {
	for _, url := range os.Args[1:] {
		if err := outline(os.Stdout, url); err != nil {
			fmt.Fprintf(os.Stderr, "outline: %v\n", err)
		}
	}
}

func outline(w io.Writer, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return err
	}

	depth := 0
	startElement := func(n *html.Node) {
		if n.Type == html.ElementNode {
			fmt.Fprintf(w, "%*s<%s>\n", depth*2, "", n.Data)
			depth++
		}
	}
	endElement := func(n *html.Node) {
		if n.Type == html.ElementNode {
			depth--
			fmt.Fprintf(w, "%*s</%s>\n", depth*2, "", n.Data)
		}
	}

	forEachNode(doc, startElement, endElement)
	return nil
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
