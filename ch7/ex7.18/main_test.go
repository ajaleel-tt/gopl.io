package main

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"
)

func parseStr(t *testing.T, s string) []Node {
	t.Helper()
	nodes, err := Parse(strings.NewReader(s))
	if err != nil {
		t.Fatalf("Parse: %v\ninput: %s", err, s)
	}
	return nodes
}

// findRoot returns the single *Element among top-level nodes, failing
// the test if there isn't exactly one.
func findRoot(t *testing.T, nodes []Node) *Element {
	t.Helper()
	var root *Element
	for _, n := range nodes {
		if e, ok := n.(*Element); ok {
			if root != nil {
				t.Fatal("multiple top-level elements")
			}
			root = e
		}
	}
	if root == nil {
		t.Fatal("no top-level element")
	}
	return root
}

func TestParse_SimpleTree(t *testing.T) {
	nodes := parseStr(t, `<a><b>hi</b><c x="1"/></a>`)
	root := findRoot(t, nodes)

	if root.Type.Local != "a" {
		t.Errorf("root local = %q, want %q", root.Type.Local, "a")
	}
	if len(root.Children) != 2 {
		t.Fatalf("root has %d children, want 2", len(root.Children))
	}

	b, ok := root.Children[0].(*Element)
	if !ok || b.Type.Local != "b" {
		t.Fatalf("first child = %#v, want <b>", root.Children[0])
	}
	if len(b.Children) != 1 {
		t.Fatalf("<b> children = %d, want 1", len(b.Children))
	}
	if cd, ok := b.Children[0].(CharData); !ok || string(cd) != "hi" {
		t.Errorf("<b> text = %#v, want CharData \"hi\"", b.Children[0])
	}

	c, ok := root.Children[1].(*Element)
	if !ok || c.Type.Local != "c" {
		t.Fatalf("second child = %#v, want <c>", root.Children[1])
	}
	if len(c.Attr) != 1 || c.Attr[0].Name.Local != "x" || c.Attr[0].Value != "1" {
		t.Errorf("<c> attrs = %#v, want [x=\"1\"]", c.Attr)
	}
}

func TestParse_CharDataIsCopied(t *testing.T) {
	// If the decoder's buffer reuse leaks through, later CharData would
	// overwrite earlier values stored in the tree. A document with many
	// adjacent text nodes is the classic reproducer.
	doc := `<r><x>aaaa</x><x>bbbb</x><x>cccc</x><x>dddd</x></r>`
	root := findRoot(t, parseStr(t, doc))

	var got []string
	for _, ch := range root.Children {
		el := ch.(*Element)
		got = append(got, string(el.Children[0].(CharData)))
	}
	want := []string{"aaaa", "bbbb", "cccc", "dddd"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("child %d = %q, want %q (CharData buffer reuse leaked through?)", i, got[i], want[i])
		}
	}
}

func TestParse_AttrIsCopied(t *testing.T) {
	// xml.Decoder reuses Attr storage too; ensure two adjacent elements
	// don't share an Attr slice.
	doc := `<r><x a="1"/><x a="2"/></r>`
	root := findRoot(t, parseStr(t, doc))
	a1 := root.Children[0].(*Element).Attr
	a2 := root.Children[1].(*Element).Attr
	if a1[0].Value != "1" || a2[0].Value != "2" {
		t.Errorf("attrs got %v, %v; want 1 and 2", a1, a2)
	}
}

func TestParse_NamespacesPreserved(t *testing.T) {
	doc := `<root xmlns:x="http://example.com/x">
              <x:item x:lang="en">hello</x:item>
            </root>`
	root := findRoot(t, parseStr(t, doc))
	var item *Element
	for _, ch := range root.Children {
		if e, ok := ch.(*Element); ok && e.Type.Local == "item" {
			item = e
			break
		}
	}
	if item == nil {
		t.Fatal("did not find <x:item>")
	}
	if item.Type.Space != "http://example.com/x" {
		t.Errorf("item namespace = %q, want %q", item.Type.Space, "http://example.com/x")
	}
	// The xmlns:x declaration shows up as an attr in the namespace
	// "http://www.w3.org/2000/xmlns/" — we just want to confirm
	// element-name namespace was resolved by the decoder.
	_ = xml.Name{}
}

func TestParse_CommentsAndPIIgnored(t *testing.T) {
	doc := `<?xml version="1.0"?>
            <!-- a comment -->
            <r><!--inner--><a/></r>`
	root := findRoot(t, parseStr(t, doc))
	for _, ch := range root.Children {
		switch ch.(type) {
		case *Element, CharData:
			// fine
		default:
			t.Errorf("unexpected child type %T (comments/PIs should be dropped)", ch)
		}
	}
}

func TestParse_UnmatchedClosingTagErrors(t *testing.T) {
	// "</orphan>" with no opener should error from the xml package
	// before our Parse even reaches the EndElement branch, but in
	// either path we expect an error rather than a panic.
	if _, err := Parse(strings.NewReader(`<a></a></b>`)); err == nil {
		t.Error("expected error for unmatched closing tag")
	}
}

func TestParse_TruncatedDocErrors(t *testing.T) {
	if _, err := Parse(strings.NewReader(`<a><b>`)); err == nil {
		t.Error("expected error for unclosed elements")
	}
}

func TestFormatRoundTrip(t *testing.T) {
	doc := `<a><b x="1">hi</b><c/></a>`
	nodes := parseStr(t, doc)
	var buf bytes.Buffer
	Format(&buf, nodes, "")
	out := buf.String()
	for _, want := range []string{`<a>`, `<b x="1">`, `"hi"`, `<c/>`, `</a>`} {
		if !strings.Contains(out, want) {
			t.Errorf("Format output missing %q:\n%s", want, out)
		}
	}
}
