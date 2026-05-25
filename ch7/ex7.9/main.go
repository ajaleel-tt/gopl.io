// Exercise 7.9: serve the track list as an HTML table where each column
// header is a link; clicking sends an HTTP request that updates the
// server-side multi-tier sort.
//
// Server state is package-level for simplicity — there's one shared sort
// history for all clients. A real app would key state by session.
package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"
)

type Track struct {
	Title  string
	Artist string
	Album  string
	Year   int
	Length time.Duration
}

func length(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic(s)
	}
	return d
}

type lessFunc func(x, y *Track) bool

// columnDef pairs a comparator with a display name so we can render headers
// and look up comparators from URL query parameters.
type columnDef struct {
	Key   string // url query value, e.g. "title"
	Label string // display text, e.g. "Title"
	Less  lessFunc
}

var columns = []columnDef{
	{"title", "Title", func(x, y *Track) bool { return x.Title < y.Title }},
	{"artist", "Artist", func(x, y *Track) bool { return x.Artist < y.Artist }},
	{"album", "Album", func(x, y *Track) bool { return x.Album < y.Album }},
	{"year", "Year", func(x, y *Track) bool { return x.Year < y.Year }},
	{"length", "Length", func(x, y *Track) bool { return x.Length < y.Length }},
}

func columnByKey(key string) *columnDef {
	for i := range columns {
		if columns[i].Key == key {
			return &columns[i]
		}
	}
	return nil
}

// MultiSorter — same shape as ex7.8, but with column keys so we can dedupe
// reliably (function values can't be compared with ==).
type MultiSorter struct {
	tracks []*Track
	keys   []string // priority order; primary at index 0
}

func (m *MultiSorter) Sort(tracks []*Track) {
	m.tracks = tracks
	sort.Sort(m)
}

func (m *MultiSorter) Len() int      { return len(m.tracks) }
func (m *MultiSorter) Swap(i, j int) { m.tracks[i], m.tracks[j] = m.tracks[j], m.tracks[i] }

func (m *MultiSorter) Less(i, j int) bool {
	p, q := m.tracks[i], m.tracks[j]
	for k := 0; k < len(m.keys)-1; k++ {
		less := columnByKey(m.keys[k]).Less
		switch {
		case less(p, q):
			return true
		case less(q, p):
			return false
		}
	}
	if len(m.keys) == 0 {
		return false
	}
	return columnByKey(m.keys[len(m.keys)-1]).Less(p, q)
}

func (m *MultiSorter) ClickColumn(key string) {
	for i, k := range m.keys {
		if k == key {
			m.keys = append(m.keys[:i], m.keys[i+1:]...)
			break
		}
	}
	m.keys = append([]string{key}, m.keys...)
}

func sample() []*Track {
	return []*Track{
		{"Go", "Delilah", "From the Roots Up", 2012, length("3m38s")},
		{"Go", "Moby", "Moby", 1992, length("3m37s")},
		{"Go Ahead", "Alicia Keys", "As I Am", 2007, length("4m36s")},
		{"Ready 2 Go", "Martin Solveig", "Smash", 2011, length("4m24s")},
	}
}

// Server bundles the mutable state. Methods take a pointer receiver so
// handler state survives requests.
type Server struct {
	mu     sync.Mutex
	tracks []*Track
	sorter *MultiSorter
	tmpl   *template.Template
}

func NewServer() *Server {
	return &Server{
		tracks: sample(),
		sorter: &MultiSorter{},
		tmpl:   template.Must(template.New("table").Parse(tableTemplate)),
	}
}

func (s *Server) Handler(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if col := r.URL.Query().Get("sort"); col != "" {
		if columnByKey(col) != nil {
			s.sorter.ClickColumn(col)
			s.sorter.Sort(s.tracks)
		}
	}

	data := struct {
		Columns  []columnDef
		Tracks   []*Track
		SortKeys []string
	}{
		Columns:  columns,
		Tracks:   s.tracks,
		SortKeys: s.sorter.keys,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

const tableTemplate = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>Tracks</title>
  <style>
    body { font-family: sans-serif; }
    table { border-collapse: collapse; }
    th, td { border: 1px solid #888; padding: 4px 8px; text-align: left; }
    th a { text-decoration: none; color: #06c; }
    .sort-keys { color: #666; margin: 1em 0; }
  </style>
</head>
<body>
  <h1>Tracks</h1>
  <div class="sort-keys">
    {{if .SortKeys}}
      Sort priority (primary → tertiary): {{range $i, $k := .SortKeys}}{{if $i}}, {{end}}{{$k}}{{end}}
    {{else}}
      No sort applied yet — click a column header.
    {{end}}
  </div>
  <table>
    <thead>
      <tr>
        {{range .Columns}}
        <th><a href="?sort={{.Key}}">{{.Label}}</a></th>
        {{end}}
      </tr>
    </thead>
    <tbody>
      {{range .Tracks}}
      <tr>
        <td>{{.Title}}</td>
        <td>{{.Artist}}</td>
        <td>{{.Album}}</td>
        <td>{{.Year}}</td>
        <td>{{.Length}}</td>
      </tr>
      {{end}}
    </tbody>
  </table>
</body>
</html>
`

func main() {
	srv := NewServer()
	http.HandleFunc("/", srv.Handler)
	addr := ":8080"
	fmt.Printf("listening on http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
