// Exercise 5.17: variadic ElementsByTagName.
package main

import (
	"fmt"
	"net/http"
	"os"

	"golang.org/x/net/html"
)

// ElementsByTagName returns all element nodes in the tree rooted at doc whose
// tag name matches one of the given names. If no names are supplied, the
// result is empty. The order of results is document order (depth-first,
// preorder).
func ElementsByTagName(doc *html.Node, names ...string) []*html.Node {
	if len(names) == 0 {
		return nil
	}
	want := make(map[string]bool, len(names))
	for _, n := range names {
		want[n] = true
	}

	var result []*html.Node
	var visit func(n *html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && want[n.Data] {
			result = append(result, n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(doc)
	return result
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: ex5.17 <url> <tag>...")
		os.Exit(1)
	}
	url := os.Args[1]
	tags := os.Args[2:]

	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse: %v\n", err)
		os.Exit(1)
	}

	matches := ElementsByTagName(doc, tags...)
	fmt.Printf("found %d match(es) for %v\n", len(matches), tags)
	for _, n := range matches {
		fmt.Printf("  <%s>", n.Data)
		for _, a := range n.Attr {
			fmt.Printf(" %s=%q", a.Key, a.Val)
		}
		fmt.Println()
	}
}
