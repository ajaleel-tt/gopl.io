// Exercise 5.8: ElementByID finds the first HTML element with the given id,
// stopping the traversal as soon as a match is found.
package main

import (
	"fmt"
	"net/http"
	"os"

	"golang.org/x/net/html"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: elementbyid <url> <id>")
		os.Exit(1)
	}
	url, id := os.Args[1], os.Args[2]

	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "elementbyid: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "elementbyid: %v\n", err)
		os.Exit(1)
	}

	if n := ElementByID(doc, id); n != nil {
		fmt.Printf("found <%s> with id=%q\n", n.Data, id)
	} else {
		fmt.Printf("no element with id=%q\n", id)
	}
}

// forEachNode visits each node in the tree rooted at n, calling pre before
// visiting children and post after. If either callback returns false,
// traversal stops and forEachNode returns false.
func forEachNode(n *html.Node, pre, post func(n *html.Node) bool) bool {
	if pre != nil && !pre(n) {
		return false
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if !forEachNode(c, pre, post) {
			return false
		}
	}
	if post != nil && !post(n) {
		return false
	}
	return true
}

// ElementByID returns the first element in doc whose id attribute equals id,
// or nil if no such element exists.
func ElementByID(doc *html.Node, id string) *html.Node {
	var found *html.Node
	forEachNode(doc, func(n *html.Node) bool {
		if n.Type != html.ElementNode {
			return true
		}
		for _, a := range n.Attr {
			if a.Key == "id" && a.Val == id {
				found = n
				return false
			}
		}
		return true
	}, nil)
	return found
}
