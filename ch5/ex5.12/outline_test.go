package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func newServer(body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, body)
	}))
}

func TestOutlineNesting(t *testing.T) {
	ts := newServer(`<html><body><p><span></span></p></body></html>`)
	defer ts.Close()

	var buf bytes.Buffer
	if err := outline(&buf, ts.URL); err != nil {
		t.Fatalf("outline: %v", err)
	}
	got := buf.String()

	// html.Parse inserts an implicit <head>, so the tree is
	// html(0) > body(1) > p(2) > span(3) — 6 spaces of indent.
	if !strings.Contains(got, "      <span>") {
		t.Errorf("expected <span> indented 6 spaces, got:\n%s", got)
	}
	if !strings.Contains(got, "<html>") {
		t.Errorf("expected <html> with no indent, got:\n%s", got)
	}
}

// TestConcurrentNoInterference is the real point of moving depth into
// a local variable: two outlines running in parallel must not clobber
// each other's depth counter.
func TestConcurrentNoInterference(t *testing.T) {
	ts := newServer(`<html><body><div><p></p></div></body></html>`)
	defer ts.Close()

	var wg sync.WaitGroup
	results := make([]string, 8)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			var buf bytes.Buffer
			if err := outline(&buf, ts.URL); err != nil {
				t.Errorf("outline: %v", err)
				return
			}
			results[i] = buf.String()
		}(i)
	}
	wg.Wait()

	for i := 1; i < len(results); i++ {
		if results[i] != results[0] {
			t.Errorf("concurrent run %d differs from run 0:\nrun 0:\n%s\nrun %d:\n%s",
				i, results[0], i, results[i])
		}
	}
}

func TestOutlineBalanced(t *testing.T) {
	ts := newServer(`<html><body><a><b></b></a></body></html>`)
	defer ts.Close()

	var buf bytes.Buffer
	if err := outline(&buf, ts.URL); err != nil {
		t.Fatalf("outline: %v", err)
	}
	out := buf.String()

	for _, tag := range []string{"html", "body", "a", "b"} {
		opens := strings.Count(out, "<"+tag+">")
		closes := strings.Count(out, "</"+tag+">")
		if opens != closes {
			t.Errorf("tag %q: %d open vs %d close in:\n%s", tag, opens, closes, out)
		}
	}
}
