// Exercise 7.12: render /list as an HTML table using html/template.
// CRUD endpoints carry over from ex7.11.
//
// This version splits the program into two layers:
//
//   - Store: pure data layer. Knows about items and prices; knows nothing
//     about HTTP. Returns sentinel errors (ErrNotFound, ErrExists) at the
//     seam.
//   - Server: HTTP layer. Parses requests, calls into a Store, formats
//     responses, maps Store errors to HTTP status codes.
//
// Server depends on the Store *interface*, so MemStore could be swapped
// for a Postgres-backed store (or a fake in tests) without touching the
// handlers. The mutex lives inside MemStore — that's a MemStore
// implementation detail, not a contract the Server cares about.
package main

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"
	"sync"
)

// ---- storage layer ----

var (
	ErrNotFound = errors.New("item not found")
	ErrExists   = errors.New("item already exists")
)

type Item struct {
	Name  string
	Price int
}

type Store interface {
	List() []Item
	Get(name string) (int, error)
	Create(name string, price int) error
	Update(name string, price int) error
	Delete(name string) error
}

type MemStore struct {
	mu    sync.Mutex
	items map[string]int
}

func NewMemStore(seed map[string]int) *MemStore {
	items := make(map[string]int, len(seed))
	for k, v := range seed {
		items[k] = v
	}
	return &MemStore{items: items}
}

func (s *MemStore) List() []Item {
	s.mu.Lock()
	out := make([]Item, 0, len(s.items))
	for k, v := range s.items {
		out = append(out, Item{k, v})
	}
	s.mu.Unlock()
	return out
}

func (s *MemStore) Get(name string) (int, error) {
	s.mu.Lock()
	price, ok := s.items[name]
	s.mu.Unlock()
	if !ok {
		return 0, ErrNotFound
	}
	return price, nil
}

// Create/Update/Delete must hold the lock across the check-then-modify
// — releasing between would let two goroutines both pass the check and
// race on the write.

func (s *MemStore) Create(name string, price int) error {
	s.mu.Lock()
	_, exists := s.items[name]
	if !exists {
		s.items[name] = price
	}
	s.mu.Unlock()
	if exists {
		return ErrExists
	}
	return nil
}

func (s *MemStore) Update(name string, price int) error {
	s.mu.Lock()
	_, ok := s.items[name]
	if ok {
		s.items[name] = price
	}
	s.mu.Unlock()
	if !ok {
		return ErrNotFound
	}
	return nil
}

func (s *MemStore) Delete(name string) error {
	s.mu.Lock()
	_, ok := s.items[name]
	if ok {
		delete(s.items, name)
	}
	s.mu.Unlock()
	if !ok {
		return ErrNotFound
	}
	return nil
}

// ---- HTTP layer ----

const listTemplate = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>Inventory</title>
  <style>
    body { font-family: sans-serif; }
    table { border-collapse: collapse; }
    th, td { border: 1px solid #888; padding: 4px 12px; text-align: left; }
    th { background: #eee; }
    td.price { text-align: right; }
  </style>
</head>
<body>
  <h1>Inventory</h1>
  <table>
    <thead><tr><th>Item</th><th>Price</th></tr></thead>
    <tbody>
      {{range .}}
      <tr><td>{{.Name}}</td><td class="price">${{.Price}}</td></tr>
      {{end}}
    </tbody>
  </table>
</body>
</html>
`

type Server struct {
	store Store
	tmpl  *template.Template
}

func NewServer(store Store) *Server {
	return &Server{
		store: store,
		tmpl:  template.Must(template.New("list").Parse(listTemplate)),
	}
}

func (s *Server) Routes(mux *http.ServeMux) {
	mux.HandleFunc("/list", s.list)
	mux.HandleFunc("/read", s.read)
	mux.HandleFunc("/create", s.create)
	mux.HandleFunc("/update", s.update)
	mux.HandleFunc("/delete", s.delete)
}

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	items := s.store.List()
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.Execute(w, items); err != nil {
		log.Printf("template: %v", err)
	}
}

func (s *Server) read(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("item")
	if name == "" {
		http.Error(w, "missing item", http.StatusBadRequest)
		return
	}
	price, err := s.store.Get(name)
	if err != nil {
		writeStoreError(w, name, err)
		return
	}
	fmt.Fprintf(w, "$%d\n", price)
}

func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	name, price, err := parseItemPrice(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.store.Create(name, price); err != nil {
		writeStoreError(w, name, err)
		return
	}
	fmt.Fprintf(w, "created %s at $%d\n", name, price)
}

func (s *Server) update(w http.ResponseWriter, r *http.Request) {
	name, price, err := parseItemPrice(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.store.Update(name, price); err != nil {
		writeStoreError(w, name, err)
		return
	}
	fmt.Fprintf(w, "updated %s to $%d\n", name, price)
}

func (s *Server) delete(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("item")
	if name == "" {
		http.Error(w, "missing item", http.StatusBadRequest)
		return
	}
	if err := s.store.Delete(name); err != nil {
		writeStoreError(w, name, err)
		return
	}
	fmt.Fprintf(w, "deleted %s\n", name)
}

// writeStoreError is the one place that knows how Store errors map to
// HTTP status codes. Adding a new Store error means adding one case
// here, not changing every handler.
func writeStoreError(w http.ResponseWriter, name string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		http.Error(w, fmt.Sprintf("no such item: %q", name), http.StatusNotFound)
	case errors.Is(err, ErrExists):
		http.Error(w, fmt.Sprintf("item already exists: %q", name), http.StatusConflict)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func parseItemPrice(r *http.Request) (name string, price int, err error) {
	name = r.URL.Query().Get("item")
	if name == "" {
		return "", 0, fmt.Errorf("missing item")
	}
	raw := r.URL.Query().Get("price")
	if raw == "" {
		return "", 0, fmt.Errorf("missing price")
	}
	price, err = strconv.Atoi(raw)
	if err != nil {
		return "", 0, fmt.Errorf("invalid price %q", raw)
	}
	if price < 0 {
		return "", 0, fmt.Errorf("invalid price %d (must be >= 0)", price)
	}
	return name, price, nil
}

// ---- entry point ----

func main() {
	store := NewMemStore(map[string]int{"shoes": 50, "socks": 5})
	srv := NewServer(store)
	mux := http.NewServeMux()
	srv.Routes(mux)
	log.Fatal(http.ListenAndServe("localhost:8000", mux))
}
