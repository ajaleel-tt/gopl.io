package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mustParse(t *testing.T, s string) *url.URL {
	t.Helper()
	u, err := url.Parse(s)
	if err != nil {
		t.Fatalf("parse %s: %v", s, err)
	}
	return u
}

func TestLocalPath(t *testing.T) {
	cases := []struct {
		url  string
		want string
	}{
		{"http://example.com/", filepath.Join("m", "example.com", "index.html")},
		{"http://example.com", filepath.Join("m", "example.com", "index.html")},
		{"http://example.com/foo.html", filepath.Join("m", "example.com", "foo.html")},
		{"http://example.com/a/b/c.html", filepath.Join("m", "example.com", "a", "b", "c.html")},
		{"http://example.com/a/b/", filepath.Join("m", "example.com", "a", "b", "index.html")},
	}
	for _, tc := range cases {
		got := localPath("m", mustParse(t, tc.url))
		if got != tc.want {
			t.Errorf("localPath(%q) = %q, want %q", tc.url, got, tc.want)
		}
	}
}

func TestCrawlMirrorsSameDomainOnly(t *testing.T) {
	// Off-domain server should NEVER be hit. Track that.
	var offDomainHits int32
	off := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		offDomainHits++
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body>off domain</body></html>`))
	}))
	defer off.Close()

	mux := http.NewServeMux()
	var on *httptest.Server
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body>
<a href="/page2.html">page2</a>
<a href="` + off.URL + `/foo">offsite</a>
</body></html>`))
	})
	mux.HandleFunc("/page2.html", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body>page two</body></html>`))
	})
	on = httptest.NewServer(mux)
	defer on.Close()

	mirrorDir := t.TempDir()
	seedURL := mustParse(t, on.URL)
	allowed := map[string]bool{seedURL.Host: true}
	crawler := makeCrawler(allowed, mirrorDir, 0)
	breadthFirst(crawler, []string{on.URL})

	// On-domain root saved as index.html.
	rootPath := filepath.Join(mirrorDir, seedURL.Host, "index.html")
	if _, err := os.Stat(rootPath); err != nil {
		t.Errorf("expected %s to exist: %v", rootPath, err)
	}

	// On-domain page2 saved.
	page2Path := filepath.Join(mirrorDir, seedURL.Host, "page2.html")
	if _, err := os.Stat(page2Path); err != nil {
		t.Errorf("expected %s to exist: %v", page2Path, err)
	}

	// Off-domain not saved AND off-domain server not hit.
	offHost := mustParse(t, off.URL).Host
	offDir := filepath.Join(mirrorDir, offHost)
	if _, err := os.Stat(offDir); !os.IsNotExist(err) {
		t.Errorf("off-domain dir %s should not exist; stat err: %v", offDir, err)
	}
	if offDomainHits != 0 {
		t.Errorf("off-domain server received %d requests, want 0", offDomainHits)
	}
}

func TestCrawlSavesNonHTML(t *testing.T) {
	// HTML page links to a CSS file on the same domain; both should be saved.
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><a href="/style.css">css</a></body></html>`))
	})
	mux.HandleFunc("/style.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		_, _ = w.Write([]byte(`body { color: red; }`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	mirrorDir := t.TempDir()
	host := mustParse(t, srv.URL).Host
	allowed := map[string]bool{host: true}
	breadthFirst(makeCrawler(allowed, mirrorDir, 0), []string{srv.URL})

	cssPath := filepath.Join(mirrorDir, host, "style.css")
	data, err := os.ReadFile(cssPath)
	if err != nil {
		t.Fatalf("expected css file at %s: %v", cssPath, err)
	}
	if !strings.Contains(string(data), "color: red") {
		t.Errorf("css contents wrong: %q", data)
	}
}

func TestCrawlNestedPaths(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><a href="/a/b/c.html">deep</a></body></html>`))
	})
	mux.HandleFunc("/a/b/c.html", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body>deep page</body></html>`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	mirrorDir := t.TempDir()
	host := mustParse(t, srv.URL).Host
	allowed := map[string]bool{host: true}
	breadthFirst(makeCrawler(allowed, mirrorDir, 0), []string{srv.URL})

	deep := filepath.Join(mirrorDir, host, "a", "b", "c.html")
	if _, err := os.Stat(deep); err != nil {
		t.Errorf("expected %s to exist: %v", deep, err)
	}
}
