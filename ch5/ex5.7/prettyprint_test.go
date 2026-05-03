package main

import (
	"bytes"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestPrettyPrintParseable(t *testing.T) {
	input := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<h1 class="title">Hello</h1>
<p>Some <em>text</em> here.</p>
<img src="pic.png" alt="a picture">
<!-- a comment -->
<br>
<div id="empty"></div>
</body>
</html>`

	doc, err := html.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parsing input: %v", err)
	}

	var buf bytes.Buffer
	prettyPrint(&buf, doc)
	output := buf.String()

	_, err = html.Parse(strings.NewReader(output))
	if err != nil {
		t.Fatalf("pretty-printed output is not parseable:\n%s\nerror: %v", output, err)
	}
}

func TestSelfClosingElements(t *testing.T) {
	input := `<html><body><img src="x.png"><br></body></html>`

	doc, err := html.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	prettyPrint(&buf, doc)
	output := buf.String()

	if !strings.Contains(output, "<img") {
		t.Error("expected <img in output")
	}
	if !strings.Contains(output, "<br/>") {
		t.Error("expected <br/> in output")
	}
	if strings.Contains(output, "</br>") {
		t.Error("should not have </br> for self-closing element")
	}
}

func TestAttributes(t *testing.T) {
	input := `<html><body><a href="http://example.com" id="link">click</a></body></html>`

	doc, err := html.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	prettyPrint(&buf, doc)
	output := buf.String()

	if !strings.Contains(output, "href='http://example.com'") {
		t.Errorf("expected href attribute in output, got:\n%s", output)
	}
	if !strings.Contains(output, "id='link'") {
		t.Errorf("expected id attribute in output, got:\n%s", output)
	}
}

func TestCommentNodes(t *testing.T) {
	input := `<html><body><!-- hello world --></body></html>`

	doc, err := html.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	prettyPrint(&buf, doc)
	output := buf.String()

	if !strings.Contains(output, "<!-- hello world -->") {
		t.Errorf("expected comment in output, got:\n%s", output)
	}
}

func TestTextNodes(t *testing.T) {
	input := `<html><body><p>Hello world</p></body></html>`

	doc, err := html.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	prettyPrint(&buf, doc)
	output := buf.String()

	if !strings.Contains(output, "Hello world") {
		t.Errorf("expected text in output, got:\n%s", output)
	}
}
