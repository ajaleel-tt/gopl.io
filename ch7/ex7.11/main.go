// Exercise 7.11: extend the book's http3a inventory server with CRUD
// handlers — /create, /read, /update, /delete — plus the original /list.
//
// The exercise warns that concurrent writes are now possible, so the
// database carries a sync.Mutex. Every handler locks it for the duration
// of the request; reads and writes serialize together.
//
// Try:
//   curl 'localhost:8000/list'
//   curl 'localhost:8000/read?item=socks'
//   curl 'localhost:8000/create?item=hat&price=15'
//   curl 'localhost:8000/update?item=socks&price=6'
//   curl 'localhost:8000/delete?item=shoes'
package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
)

type database struct {
	mu    sync.Mutex
	items map[string]int
}

func (db *database) list(w http.ResponseWriter, req *http.Request) {
	db.mu.Lock()
	defer db.mu.Unlock()
	for item, price := range db.items {
		fmt.Fprintf(w, "%s: $%d\n", item, price)
	}
}

func (db *database) read(w http.ResponseWriter, req *http.Request) {
	item := req.URL.Query().Get("item")
	db.mu.Lock()
	defer db.mu.Unlock()
	price, ok := db.items[item]
	if !ok {
		http.Error(w, fmt.Sprintf("no such item: %q", item), http.StatusNotFound)
		return
	}
	fmt.Fprintf(w, "$%d\n", price)
}

func (db *database) create(w http.ResponseWriter, req *http.Request) {
	item, price, err := parseItemPrice(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	if _, ok := db.items[item]; ok {
		http.Error(w, fmt.Sprintf("item already exists: %q", item), http.StatusConflict)
		return
	}
	db.items[item] = price
	fmt.Fprintf(w, "created %s at $%d\n", item, price)
}

func (db *database) update(w http.ResponseWriter, req *http.Request) {
	item, price, err := parseItemPrice(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	if _, ok := db.items[item]; !ok {
		http.Error(w, fmt.Sprintf("no such item: %q", item), http.StatusNotFound)
		return
	}
	db.items[item] = price
	fmt.Fprintf(w, "updated %s to $%d\n", item, price)
}

func (db *database) delete(w http.ResponseWriter, req *http.Request) {
	item := req.URL.Query().Get("item")
	if item == "" {
		http.Error(w, "missing item", http.StatusBadRequest)
		return
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	if _, ok := db.items[item]; !ok {
		http.Error(w, fmt.Sprintf("no such item: %q", item), http.StatusNotFound)
		return
	}
	delete(db.items, item)
	fmt.Fprintf(w, "deleted %s\n", item)
}

// parseItemPrice pulls item and price out of the query, validating both.
// Price must be a non-negative integer (cents would be nicer, but the
// book's database uses whole-dollar ints).
func parseItemPrice(req *http.Request) (item string, price int, err error) {
	item = req.URL.Query().Get("item")
	if item == "" {
		return "", 0, fmt.Errorf("missing item")
	}
	raw := req.URL.Query().Get("price")
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
	return item, price, nil
}

func main() {
	db := &database{items: map[string]int{"shoes": 50, "socks": 5}}
	mux := http.NewServeMux()
	mux.HandleFunc("/list", db.list)
	mux.HandleFunc("/read", db.read)
	mux.HandleFunc("/create", db.create)
	mux.HandleFunc("/update", db.update)
	mux.HandleFunc("/delete", db.delete)
	log.Fatal(http.ListenAndServe("localhost:8000", mux))
}
