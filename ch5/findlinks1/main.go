// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

// See page 122.
//!+main

// Findlinks1 prints the links in an HTML document read from standard input.
package main

import (
	"fmt"
	"os"

	"golang.org/x/net/html"
)

func main() {
	doc, err := html.Parse(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "findlinks1: %v\n", err)
		os.Exit(1)
	}
	for _, link := range visit(nil, doc) {
		fmt.Println(link)
	}
	for name, count := range countElements(doc, make(map[string]int)) {
		fmt.Printf("%4d %s\n", count, name)
	}
	printText(doc)
}

//!-main

// !+visit
// visit appends to links each link found in n and returns the result.
// It is fully tail recursive: a "to-do" slice carries pending nodes
// so that every recursive call is in tail position.
func visit(links []string, n *html.Node) []string {
	return visitTail(links, n, nil)
}

var linkAttrs = map[string]string{
	"a":      "href",
	"link":   "href",
	"img":    "src",
	"script": "src",
	"iframe": "src",
	"audio":  "src",
	"video":  "src",
	"source": "src",
}

func visitTail(links []string, n *html.Node, todo []*html.Node) []string {
	if n.Type == html.ElementNode {
		if key, ok := linkAttrs[n.Data]; ok {
			for _, a := range n.Attr {
				if a.Key == key {
					links = append(links, a.Val)
				}
			}
		}
	}
	// Queue the sibling for later, then recurse into the child.
	if n.NextSibling != nil {
		todo = append(todo, n.NextSibling)
	}
	if n.FirstChild != nil {
		return visitTail(links, n.FirstChild, todo)
	}
	// No child — pop the next pending node.
	if len(todo) == 0 {
		return links
	}
	next := todo[len(todo)-1]
	todo = todo[:len(todo)-1]
	return visitTail(links, next, todo)
}

//!-visit

// printText prints the content of all text nodes in the HTML document tree,
// skipping <script> and <style> elements.
func printText(n *html.Node) {
	if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
		return
	}
	if n.Type == html.TextNode {
		fmt.Println(n.Data)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		printText(c)
	}
}

// countElements populates counts with the number of elements of each
// name (p, div, span, ...) in the tree rooted at n.
func countElements(n *html.Node, counts map[string]int) map[string]int {
	if n.Type == html.ElementNode {
		counts[n.Data]++
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		countElements(c, counts)
	}
	return counts
}

/*
//!+html
package html

type Node struct {
	Type                    NodeType
	Data                    string
	Attr                    []Attribute
	FirstChild, NextSibling *Node
}

type NodeType int32

const (
	ErrorNode NodeType = iota
	TextNode
	DocumentNode
	ElementNode
	CommentNode
	DoctypeNode
)

type Attribute struct {
	Key, Val string
}

func Parse(r io.Reader) (*Node, error)
//!-html
*/
