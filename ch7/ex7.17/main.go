// Exercise 7.17: xmlselect with CSS-style simple selectors.
//
// Each command-line argument is a simple selector that can combine:
//   name              element local name (e.g. "div")
//   #id               id attribute equals "id"
//   .class            class attribute (space-separated tokens) contains "class"
//   [attr]            element has attribute "attr"
//   [attr=value]      attribute "attr" equals "value"  (value may be quoted)
//
// A bare "*" or an empty name part matches any element. Multiple
// constraints stack, e.g. div#page.wide[lang=en].
//
// Like the original xmlselect, multiple selector arguments describe a
// descendant chain (in order, not necessarily contiguous).
package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
)

// ----- selector model -----

type selector struct {
	name    string // "" means any
	id      string // "" means don't constrain
	classes []string
	attrs   []attrConstraint
}

type attrConstraint struct {
	name  string
	value string // "" + hasValue=false means "attribute exists, any value"
	exact bool
}

func (s *selector) matches(el xml.StartElement) bool {
	if s.name != "" && s.name != el.Name.Local {
		return false
	}
	if s.id != "" {
		v, ok := attrOf(el, "id")
		if !ok || v != s.id {
			return false
		}
	}
	if len(s.classes) > 0 {
		v, ok := attrOf(el, "class")
		if !ok {
			return false
		}
		have := strings.Fields(v)
		for _, want := range s.classes {
			if !containsString(have, want) {
				return false
			}
		}
	}
	for _, c := range s.attrs {
		v, ok := attrOf(el, c.name)
		if !ok {
			return false
		}
		if c.exact && v != c.value {
			return false
		}
	}
	return true
}

func attrOf(el xml.StartElement, name string) (string, bool) {
	for _, a := range el.Attr {
		if a.Name.Local == name {
			return a.Value, true
		}
	}
	return "", false
}

func containsString(xs []string, x string) bool {
	for _, s := range xs {
		if s == x {
			return true
		}
	}
	return false
}

// ----- selector parser -----

// parseSelector parses a single CSS simple selector. It returns an
// error rather than panicking so bad CLI input can be reported clearly.
func parseSelector(s string) (selector, error) {
	if s == "" {
		return selector{}, fmt.Errorf("empty selector")
	}
	var sel selector
	i := 0

	// Optional name (or '*').
	if s[0] != '#' && s[0] != '.' && s[0] != '[' {
		name, j := readIdent(s, i)
		if name == "" {
			return sel, fmt.Errorf("invalid character %q at offset %d in %q", s[i], i, s)
		}
		if name != "*" {
			sel.name = name
		}
		i = j
	}

	for i < len(s) {
		switch s[i] {
		case '#':
			id, j := readIdent(s, i+1)
			if id == "" {
				return sel, fmt.Errorf("empty #id in %q", s)
			}
			if sel.id != "" {
				return sel, fmt.Errorf("multiple #id in %q", s)
			}
			sel.id = id
			i = j
		case '.':
			cls, j := readIdent(s, i+1)
			if cls == "" {
				return sel, fmt.Errorf("empty .class in %q", s)
			}
			sel.classes = append(sel.classes, cls)
			i = j
		case '[':
			end := strings.IndexByte(s[i:], ']')
			if end < 0 {
				return sel, fmt.Errorf("unclosed '[' in %q", s)
			}
			body := s[i+1 : i+end]
			c, err := parseAttr(body)
			if err != nil {
				return sel, err
			}
			sel.attrs = append(sel.attrs, c)
			i += end + 1
		default:
			return sel, fmt.Errorf("unexpected %q at offset %d in %q", s[i], i, s)
		}
	}
	return sel, nil
}

func parseAttr(body string) (attrConstraint, error) {
	body = strings.TrimSpace(body)
	if eq := strings.IndexByte(body, '='); eq >= 0 {
		name := strings.TrimSpace(body[:eq])
		val := strings.TrimSpace(body[eq+1:])
		// Strip surrounding quotes if present.
		if n := len(val); n >= 2 && (val[0] == '"' && val[n-1] == '"' || val[0] == '\'' && val[n-1] == '\'') {
			val = val[1 : n-1]
		}
		if name == "" {
			return attrConstraint{}, fmt.Errorf("empty attribute name in [%s]", body)
		}
		return attrConstraint{name: name, value: val, exact: true}, nil
	}
	if body == "" {
		return attrConstraint{}, fmt.Errorf("empty []")
	}
	return attrConstraint{name: body}, nil
}

// isIdentChar accepts the conservative set used in CSS-ish identifiers
// plus ':' so XML namespace-prefixed names work.
func isIdentChar(c byte) bool {
	return c == '_' || c == '-' || c == ':' ||
		c == '*' ||
		(c >= '0' && c <= '9') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z')
}

// readIdent consumes a run of ident chars starting at i. It returns the
// run and the index past it. If no chars are consumed, the second
// return is i unchanged.
func readIdent(s string, i int) (string, int) {
	j := i
	for j < len(s) && isIdentChar(s[j]) {
		j++
	}
	return s[i:j], j
}

// ----- main loop -----

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: xmlselect <selector>...")
		os.Exit(2)
	}
	sels := make([]selector, len(os.Args)-1)
	for i, a := range os.Args[1:] {
		s, err := parseSelector(a)
		if err != nil {
			fmt.Fprintf(os.Stderr, "xmlselect: %v\n", err)
			os.Exit(2)
		}
		sels[i] = s
	}
	if err := stream(os.Stdin, os.Stdout, sels); err != nil {
		fmt.Fprintf(os.Stderr, "xmlselect: %v\n", err)
		os.Exit(1)
	}
}

// stream is the testable core: read XML from r, write matches to w.
func stream(r io.Reader, w io.Writer, sels []selector) error {
	dec := xml.NewDecoder(r)
	var stack []xml.StartElement
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		switch tok := tok.(type) {
		case xml.StartElement:
			stack = append(stack, tok)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if chainMatches(stack, sels) {
				path := make([]string, len(stack))
				for i, e := range stack {
					path[i] = e.Name.Local
				}
				fmt.Fprintf(w, "%s: %s\n", strings.Join(path, " "), tok)
			}
		}
	}
}

// chainMatches reports whether the selectors appear in order along the
// ancestor stack — the same descendant semantics as the original
// xmlselect, but each step compares with the richer selector type.
func chainMatches(stack []xml.StartElement, sels []selector) bool {
	for len(sels) <= len(stack) {
		if len(sels) == 0 {
			return true
		}
		if sels[0].matches(stack[0]) {
			sels = sels[1:]
		}
		stack = stack[1:]
	}
	return false
}
