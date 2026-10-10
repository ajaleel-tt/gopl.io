// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"testing"
)

// TestDepthLimiting crawls a fixed in-memory link graph and checks that each
// -depth value fetches exactly the URLs within that many links of the seed.
//
// Graph (seed is "/", at depth 0):
//
//	/  -> /a, /b          (depth 1)
//	/a -> /c              (depth 2)
//	/b -> /c, /d          (depth 2; /c is shared, so deduplicated)
//	/c -> /e              (depth 3)
//	/d, /e -> (no links)
func TestDepthLimiting(t *testing.T) {
	graph := map[string][]string{
		"/":  {"/a", "/b"},
		"/a": {"/c"},
		"/b": {"/c", "/d"},
		"/c": {"/e"},
		"/d": {},
		"/e": {},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, link := range graph[r.URL.Path] {
			fmt.Fprintf(w, `<a href="%s">link</a>`, link)
		}
	}))
	defer srv.Close()

	// abs maps a graph path to the absolute URL the crawler records. The seed
	// is srv.URL itself (no trailing slash); links resolve to srv.URL+path.
	abs := func(p string) string {
		if p == "/" {
			return srv.URL
		}
		return srv.URL + p
	}

	tests := []struct {
		depth int
		want  []string
	}{
		{0, []string{abs("/")}},
		{1, []string{abs("/"), abs("/a"), abs("/b")}},
		{2, []string{abs("/"), abs("/a"), abs("/b"), abs("/c"), abs("/d")}},
		{3, []string{abs("/"), abs("/a"), abs("/b"), abs("/c"), abs("/d"), abs("/e")}},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("depth=%d", tt.depth), func(t *testing.T) {
			got := crawlSite([]string{srv.URL}, tt.depth, 20)
			sort.Strings(got)
			want := append([]string(nil), tt.want...)
			sort.Strings(want)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("depth=%d fetched the wrong set\n got:  %v\n want: %v", tt.depth, got, want)
			}
		})
	}
}
