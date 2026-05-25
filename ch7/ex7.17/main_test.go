package main

import (
	"bytes"
	"encoding/xml"
	"reflect"
	"strings"
	"testing"
)

func mustParse(t *testing.T, s string) selector {
	t.Helper()
	sel, err := parseSelector(s)
	if err != nil {
		t.Fatalf("parseSelector(%q): %v", s, err)
	}
	return sel
}

func TestParseSelector(t *testing.T) {
	cases := []struct {
		in   string
		want selector
	}{
		{"div", selector{name: "div"}},
		{"*", selector{}},
		{"#page", selector{id: "page"}},
		{".wide", selector{classes: []string{"wide"}}},
		{"div#page", selector{name: "div", id: "page"}},
		{"div.wide", selector{name: "div", classes: []string{"wide"}}},
		{"div#page.wide", selector{name: "div", id: "page", classes: []string{"wide"}}},
		{"div.wide.featured", selector{name: "div", classes: []string{"wide", "featured"}}},
		{"a[href]", selector{name: "a", attrs: []attrConstraint{{name: "href"}}}},
		{`a[href=/x]`, selector{name: "a", attrs: []attrConstraint{{name: "href", value: "/x", exact: true}}}},
		{`a[href="/x y"]`, selector{name: "a", attrs: []attrConstraint{{name: "href", value: "/x y", exact: true}}}},
		{`div#main.wide[lang=en]`, selector{
			name: "div", id: "main", classes: []string{"wide"},
			attrs: []attrConstraint{{name: "lang", value: "en", exact: true}},
		}},
	}
	for _, c := range cases {
		got, err := parseSelector(c.in)
		if err != nil {
			t.Errorf("parseSelector(%q): %v", c.in, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("parseSelector(%q) = %#v, want %#v", c.in, got, c.want)
		}
	}
}

func TestParseSelectorErrors(t *testing.T) {
	bad := []string{
		"",
		"div#",
		"div.",
		"div[",
		"div[]",
		"div#a#b",
		"div?bogus",
	}
	for _, s := range bad {
		if _, err := parseSelector(s); err == nil {
			t.Errorf("parseSelector(%q) = nil error, want failure", s)
		}
	}
}

// makeEl builds an xml.StartElement for selector matching tests.
func makeEl(name string, attrs map[string]string) xml.StartElement {
	el := xml.StartElement{Name: xml.Name{Local: name}}
	for k, v := range attrs {
		el.Attr = append(el.Attr, xml.Attr{Name: xml.Name{Local: k}, Value: v})
	}
	return el
}

func TestSelectorMatches(t *testing.T) {
	div := makeEl("div", map[string]string{"id": "page", "class": "wide featured", "lang": "en"})
	span := makeEl("span", nil)

	cases := []struct {
		sel  string
		el   xml.StartElement
		want bool
	}{
		{"div", div, true},
		{"span", div, false},
		{"#page", div, true},
		{"#other", div, false},
		{".wide", div, true},
		{".missing", div, false},
		{".wide.featured", div, true},
		{".wide.bogus", div, false},
		{"div#page.wide", div, true},
		{"div#page.bogus", div, false},
		{"*", span, true},
		{"[lang]", div, true},
		{"[lang=en]", div, true},
		{"[lang=fr]", div, false},
		{"[missing]", div, false},
	}
	for _, c := range cases {
		s := mustParse(t, c.sel)
		if got := s.matches(c.el); got != c.want {
			t.Errorf("%q.matches(%s) = %v, want %v", c.sel, c.el.Name.Local, got, c.want)
		}
	}
}

const sample = `
<html>
  <body>
    <div id="page" class="wide featured">
      <h1>Title</h1>
      <p class="lead">Lead paragraph.</p>
      <p>Normal paragraph.</p>
    </div>
    <div id="sidebar" class="narrow">
      <p>Sidebar text.</p>
      <a href="/about">about</a>
      <a href="/contact" rel="nofollow">contact</a>
    </div>
  </body>
</html>
`

func TestStream(t *testing.T) {
	cases := []struct {
		name     string
		selArgs  []string
		wantHas  []string
		wantNone []string
	}{
		{
			name:    "by element name (original behavior)",
			selArgs: []string{"div", "p"},
			wantHas: []string{"Lead paragraph", "Normal paragraph", "Sidebar text"},
		},
		{
			name:     "by id only",
			selArgs:  []string{"#page"},
			wantHas:  []string{"Lead paragraph", "Normal paragraph"},
			wantNone: []string{"Sidebar text"},
		},
		{
			name:     "by class only",
			selArgs:  []string{".narrow"},
			wantHas:  []string{"Sidebar text", "about", "contact"},
			wantNone: []string{"Lead paragraph"},
		},
		{
			name:    "tag + class chain",
			selArgs: []string{"div.wide", "p.lead"},
			wantHas: []string{"Lead paragraph"},
			wantNone: []string{
				"Normal paragraph", // p but not p.lead
				"Sidebar text",     // wrong outer div
			},
		},
		{
			name:    "attribute presence",
			selArgs: []string{"a[rel]"},
			wantHas: []string{"contact"},
			wantNone: []string{
				"about", // a, but no rel
			},
		},
		{
			name:    "attribute equality",
			selArgs: []string{`a[href="/about"]`},
			wantHas: []string{"about"},
			wantNone: []string{"contact"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sels := make([]selector, len(c.selArgs))
			for i, a := range c.selArgs {
				sels[i] = mustParse(t, a)
			}
			var out bytes.Buffer
			if err := stream(strings.NewReader(sample), &out, sels); err != nil {
				t.Fatalf("stream: %v", err)
			}
			body := out.String()
			for _, s := range c.wantHas {
				if !strings.Contains(body, s) {
					t.Errorf("missing %q in output:\n%s", s, body)
				}
			}
			for _, s := range c.wantNone {
				if strings.Contains(body, s) {
					t.Errorf("unexpected %q in output:\n%s", s, body)
				}
			}
		})
	}
}
