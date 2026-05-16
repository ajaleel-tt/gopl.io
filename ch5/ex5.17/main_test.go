package main

import (
	"sort"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func parseHTML(t *testing.T, s string) *html.Node {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(s))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return doc
}

func tagsOf(nodes []*html.Node) []string {
	tags := make([]string, len(nodes))
	for i, n := range nodes {
		tags[i] = n.Data
	}
	return tags
}

func TestElementsByTagNameSingle(t *testing.T) {
	doc := parseHTML(t, `
<html><body>
  <img src="a.png">
  <p>hello</p>
  <img src="b.png">
  <div><img src="c.png"></div>
</body></html>`)

	imgs := ElementsByTagName(doc, "img")
	if len(imgs) != 3 {
		t.Errorf("expected 3 imgs, got %d (tags: %v)", len(imgs), tagsOf(imgs))
	}
	for _, n := range imgs {
		if n.Data != "img" {
			t.Errorf("non-img in results: %s", n.Data)
		}
	}
}

func TestElementsByTagNameMultiple(t *testing.T) {
	doc := parseHTML(t, `
<html><body>
  <h1>a</h1>
  <p>p</p>
  <h2>b</h2>
  <div><h3>c</h3></div>
  <h4>d</h4>
  <h5>not asked for</h5>
</body></html>`)

	headings := ElementsByTagName(doc, "h1", "h2", "h3", "h4")
	got := tagsOf(headings)
	sort.Strings(got)
	want := []string{"h1", "h2", "h3", "h4"}
	if !equalStringSlice(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestElementsByTagNameNoNames(t *testing.T) {
	doc := parseHTML(t, `<html><body><p>x</p></body></html>`)
	if got := ElementsByTagName(doc); len(got) != 0 {
		t.Errorf("expected no results with no names, got %d", len(got))
	}
}

func TestElementsByTagNameUnknownName(t *testing.T) {
	doc := parseHTML(t, `<html><body><p>x</p></body></html>`)
	if got := ElementsByTagName(doc, "marquee"); len(got) != 0 {
		t.Errorf("expected no results, got %v", tagsOf(got))
	}
}

func TestElementsByTagNameDuplicateNames(t *testing.T) {
	doc := parseHTML(t, `<html><body><p>1</p><p>2</p></body></html>`)
	once := ElementsByTagName(doc, "p")
	twice := ElementsByTagName(doc, "p", "p")
	if len(once) != len(twice) {
		t.Errorf("duplicate names produced %d results, single produced %d",
			len(twice), len(once))
	}
}

func TestElementsByTagNameDocumentOrder(t *testing.T) {
	doc := parseHTML(t, `
<html><body>
  <span id="1"></span>
  <div><span id="2"></span></div>
  <span id="3"></span>
</body></html>`)

	spans := ElementsByTagName(doc, "span")
	var ids []string
	for _, n := range spans {
		for _, a := range n.Attr {
			if a.Key == "id" {
				ids = append(ids, a.Val)
			}
		}
	}
	want := []string{"1", "2", "3"}
	if !equalStringSlice(ids, want) {
		t.Errorf("got ids %v, want %v (document order)", ids, want)
	}
}

func TestElementsByTagNameDeepNesting(t *testing.T) {
	doc := parseHTML(t, `<html><body><div><div><div><a/></div></div></div></body></html>`)
	if got := ElementsByTagName(doc, "a"); len(got) != 1 {
		t.Errorf("expected 1 deep <a>, got %d", len(got))
	}
}

func equalStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
