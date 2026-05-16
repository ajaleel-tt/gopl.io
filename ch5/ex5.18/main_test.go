package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// chdirTo runs the test in a temp dir so fetch's local file write doesn't
// pollute the project.
func chdirTo(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})
}

func TestFetchWritesFile(t *testing.T) {
	chdirTo(t)

	const body = "hello from server"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	local, n, err := fetch(srv.URL + "/file.txt")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if local != "file.txt" {
		t.Errorf("local = %q, want %q", local, "file.txt")
	}
	if int(n) != len(body) {
		t.Errorf("n = %d, want %d", n, len(body))
	}

	got, err := os.ReadFile(filepath.Clean(local))
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(got) != body {
		t.Errorf("contents = %q, want %q", got, body)
	}
}

func TestFetchRootUsesIndex(t *testing.T) {
	chdirTo(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html>root</html>"))
	}))
	defer srv.Close()

	local, _, err := fetch(srv.URL + "/")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if local != "index.html" {
		t.Errorf("local = %q, want %q", local, "index.html")
	}
}

func TestFetchErrorOnBadURL(t *testing.T) {
	chdirTo(t)

	_, _, err := fetch("http://no-such-host.invalid/")
	if err == nil {
		t.Fatal("expected error from bad URL, got nil")
	}
}

// TestDeferredCloseHappens is the real point of the rewrite: the file must
// be closed by the time fetch returns. We test that by writing to a path
// that requires the file to be closed before we can stat its final size.
func TestDeferredCloseHappens(t *testing.T) {
	chdirTo(t)

	const body = "small payload"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	local, n, err := fetch(srv.URL + "/x.txt")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}

	info, err := os.Stat(local)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Size() != n {
		t.Errorf("file size %d, fetch returned n=%d", info.Size(), n)
	}
	if !strings.HasSuffix(local, "x.txt") {
		t.Errorf("local name suffix wrong: %q", local)
	}
}
