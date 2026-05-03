package main

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func parse(t *testing.T, s string) *html.Node {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(s))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return doc
}

func TestElementByIDFound(t *testing.T) {
	doc := parse(t, `
<html><body>
  <div id="outer">
    <p id="target">hello</p>
  </div>
</body></html>`)

	n := ElementByID(doc, "target")
	if n == nil {
		t.Fatal("expected to find element, got nil")
	}
	if n.Data != "p" {
		t.Errorf("expected <p>, got <%s>", n.Data)
	}
}

func TestElementByIDNotFound(t *testing.T) {
	doc := parse(t, `<html><body><p id="other">x</p></body></html>`)

	if n := ElementByID(doc, "missing"); n != nil {
		t.Errorf("expected nil, got <%s>", n.Data)
	}
}

func TestElementByIDFirstMatch(t *testing.T) {
	doc := parse(t, `
<html><body>
  <p id="dup">first</p>
  <p id="dup">second</p>
</body></html>`)

	n := ElementByID(doc, "dup")
	if n == nil {
		t.Fatal("expected match, got nil")
	}
	if n.FirstChild == nil || n.FirstChild.Data != "first" {
		t.Errorf("expected first match, got %q", n.FirstChild.Data)
	}
}

func TestForEachNodeStopsEarly(t *testing.T) {
	doc := parse(t, `<html><body><a/><b/><c/><d/></body></html>`)

	var visited []string
	forEachNode(doc, func(n *html.Node) bool {
		if n.Type == html.ElementNode {
			visited = append(visited, n.Data)
			if n.Data == "b" {
				return false
			}
		}
		return true
	}, nil)

	for _, name := range visited {
		if name == "c" || name == "d" {
			t.Errorf("traversal continued after stop: visited %v", visited)
			break
		}
	}
}

func TestForEachNodeFullTraversal(t *testing.T) {
	doc := parse(t, `<html><body><p>x</p></body></html>`)

	count := 0
	forEachNode(doc, func(n *html.Node) bool {
		count++
		return true
	}, nil)

	if count == 0 {
		t.Error("expected to visit at least one node")
	}
}
