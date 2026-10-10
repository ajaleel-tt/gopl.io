// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sort"
	"strings"
	"testing"
)

// TestMirror stands up a fixed in-memory site and checks that mirror():
//   - fetches every reachable same-domain page (and no off-domain page),
//   - writes each to a URL-derived path on disk,
//   - rewrites same-domain links to relative local paths, leaving external ones.
//
// Site graph (all same-domain except the external link on "/"):
//
//	/           -> /a.html, /b.html, http://external.example/x
//	/a.html     -> /b.html, /sub/c.html
//	/b.html     -> /a.html            (cycle: dedup)
//	/sub/c.html -> /a.html            (cross-directory: relative rewrite)
func TestMirror(t *testing.T) {
	graph := map[string]string{
		"/":           `<a href="/a.html">A</a><a href="/b.html">B</a><a href="http://external.example/x">ext</a>`,
		"/a.html":     `<a href="/b.html">B</a><a href="/sub/c.html">C</a>`,
		"/b.html":     `<a href="/a.html">A</a>`,
		"/sub/c.html": `<a href="/a.html">A</a>`,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := graph[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<html><body>%s</body></html>", body)
	}))
	defer srv.Close()

	out := t.TempDir()
	mirrored, err := mirror(srv.URL, out, 20)
	if err != nil {
		t.Fatal(err)
	}

	// 1. Exactly the same-domain pages were mirrored — the external one wasn't.
	want := []string{
		srv.URL,             // seed (path "")
		srv.URL + "/a.html",
		srv.URL + "/b.html",
		srv.URL + "/sub/c.html",
	}
	sort.Strings(mirrored)
	sort.Strings(want)
	if fmt.Sprint(mirrored) != fmt.Sprint(want) {
		t.Errorf("mirrored set wrong\n got:  %v\n want: %v", mirrored, want)
	}

	// 2. Each expected file exists on disk; the external host has no directory.
	mustParse := func(s string) *url.URL {
		u, err := url.Parse(s)
		if err != nil {
			t.Fatal(err)
		}
		return u
	}
	for _, u := range want {
		if _, err := os.Stat(localPath(out, mustParse(u))); err != nil {
			t.Errorf("expected mirrored file missing for %s: %v", u, err)
		}
	}
	if _, err := os.Stat(out + "/external.example"); !os.IsNotExist(err) {
		t.Errorf("external.example should not have been fetched/created")
	}

	// 3. Links were rewritten correctly.
	read := func(u string) string {
		b, err := os.ReadFile(localPath(out, mustParse(u)))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	index := read(srv.URL)
	if !strings.Contains(index, `href="a.html"`) || !strings.Contains(index, `href="b.html"`) {
		t.Errorf("index links not rewritten to relative local paths:\n%s", index)
	}
	if !strings.Contains(index, `href="http://external.example/x"`) {
		t.Errorf("external link should be left absolute:\n%s", index)
	}
	// /sub/c.html links up to /a.html -> "../a.html" (crosses a directory).
	c := read(srv.URL + "/sub/c.html")
	if !strings.Contains(c, `href="../a.html"`) {
		t.Errorf("cross-directory link not rewritten to ../a.html:\n%s", c)
	}
}
