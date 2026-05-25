package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer(t *testing.T) (*httptest.Server, *Server) {
	t.Helper()
	srv := NewServer()
	ts := httptest.NewServer(http.HandlerFunc(srv.Handler))
	t.Cleanup(ts.Close)
	return ts, srv
}

func getBody(t *testing.T, url string) string {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("GET %s: status %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(body)
}

func TestInitialPageRendersAllTracks(t *testing.T) {
	ts, _ := newTestServer(t)
	body := getBody(t, ts.URL)

	for _, want := range []string{"Delilah", "Moby", "Alicia Keys", "Martin Solveig"} {
		if !strings.Contains(body, want) {
			t.Errorf("initial page missing %q", want)
		}
	}
	// Should contain a table.
	if !strings.Contains(body, "<table>") {
		t.Error("response does not contain a <table>")
	}
}

func TestColumnHeadersAreLinks(t *testing.T) {
	ts, _ := newTestServer(t)
	body := getBody(t, ts.URL)

	for _, col := range columns {
		wantHref := `href="?sort=` + col.Key + `"`
		wantLabel := ">" + col.Label + "</a>"
		if !strings.Contains(body, wantHref) {
			t.Errorf("missing link for column %q (no %s)", col.Key, wantHref)
		}
		if !strings.Contains(body, wantLabel) {
			t.Errorf("missing label for column %q (no %s)", col.Key, wantLabel)
		}
	}
}

// TestClickSortsTracks verifies that issuing a ?sort= request actually
// reorders the rendered rows.
func TestClickSortsTracks(t *testing.T) {
	ts, _ := newTestServer(t)
	body := getBody(t, ts.URL+"?sort=year")

	// In the rendered HTML, rows appear in document order, so the first
	// year value to appear should be 1992 (Moby's "Go") after a Year sort.
	mobyPos := strings.Index(body, "Moby")
	delilahPos := strings.Index(body, "Delilah")
	if mobyPos < 0 || delilahPos < 0 {
		t.Fatalf("missing expected artists; body=%s", body)
	}
	if mobyPos >= delilahPos {
		t.Errorf("after ?sort=year, Moby (1992) should appear before Delilah (2012)")
	}
}

// TestClickStateAcrossRequests verifies the click history is preserved
// across requests — primary becomes secondary after a new click.
func TestClickStateAcrossRequests(t *testing.T) {
	ts, srv := newTestServer(t)

	getBody(t, ts.URL+"?sort=year")  // primary: year
	getBody(t, ts.URL+"?sort=title") // primary: title, secondary: year

	srv.mu.Lock()
	keys := append([]string{}, srv.sorter.keys...)
	srv.mu.Unlock()

	want := []string{"title", "year"}
	if !equalStrings(keys, want) {
		t.Errorf("after two clicks, keys = %v, want %v", keys, want)
	}

	body := getBody(t, ts.URL) // re-render without click — order should reflect existing sort
	// "Go" titles tied, year breaks tie: 1992 (Moby) before 2012 (Delilah)
	mobyPos := strings.Index(body, "Moby")
	delilahPos := strings.Index(body, "Delilah")
	if mobyPos >= delilahPos {
		t.Errorf("multi-tier sort not applied: Moby (1992) should precede Delilah (2012)")
	}
}

func TestClickReClickPromotes(t *testing.T) {
	ts, srv := newTestServer(t)

	getBody(t, ts.URL+"?sort=year")   // [year]
	getBody(t, ts.URL+"?sort=title")  // [title, year]
	getBody(t, ts.URL+"?sort=year")   // [year, title] — promoted, not duplicated

	srv.mu.Lock()
	keys := append([]string{}, srv.sorter.keys...)
	srv.mu.Unlock()

	want := []string{"year", "title"}
	if !equalStrings(keys, want) {
		t.Errorf("after re-click, keys = %v, want %v", keys, want)
	}
	if len(keys) != 2 {
		t.Errorf("re-click duplicated key: %v", keys)
	}
}

func TestInvalidColumnIgnored(t *testing.T) {
	ts, srv := newTestServer(t)
	getBody(t, ts.URL+"?sort=does-not-exist")

	srv.mu.Lock()
	defer srv.mu.Unlock()
	if len(srv.sorter.keys) != 0 {
		t.Errorf("invalid column should not be added to keys: %v", srv.sorter.keys)
	}
}

func TestSortKeysDisplayed(t *testing.T) {
	ts, _ := newTestServer(t)

	body := getBody(t, ts.URL)
	if !strings.Contains(body, "No sort applied") {
		t.Error("initial page should show 'No sort applied' notice")
	}

	body = getBody(t, ts.URL+"?sort=year")
	if !strings.Contains(body, "year") {
		t.Error("after click, page should display the sort key 'year'")
	}
}

func equalStrings(a, b []string) bool {
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
