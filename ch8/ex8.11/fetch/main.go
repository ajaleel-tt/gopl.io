// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

// Fetch (Exercise 8.11) requests several URLs concurrently and returns the
// first successful response, cancelling the other in-flight requests.
//
// It combines two ideas:
//   - mirroredQuery (§8.4.4): a buffered result channel so the losing
//     goroutines can always send and exit — no goroutine leak; and
//   - cancellation (ex8.10): a shared context so that as soon as a winner
//     arrives, the losers' HTTP requests are aborted mid-flight rather than
//     left running to completion (wasted bandwidth and load).
//
// Usage:
//
//	go run . https://mirror1/x https://mirror2/x https://mirror3/x
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

// result is one request's outcome.
type result struct {
	url  string
	body []byte
	err  error
}

// get issues a cancellable GET; cancelling ctx aborts it even in flight.
func get(ctx context.Context, url string) result {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return result{url: url, err: err}
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return result{url: url, err: err} // wraps context.Canceled if we lost the race
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return result{url: url, err: fmt.Errorf("%s: %s", url, resp.Status)}
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result{url: url, err: err}
	}
	return result{url: url, body: body}
}

// fetch requests every url concurrently and returns the first successful one,
// cancelling the rest. If all requests fail, it returns their errors.
func fetch(urls ...string) (result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // fires on return → aborts every still-running request

	// Buffered to len(urls): once we return with a winner, the losing
	// goroutines can still send their (cancelled) result without blocking, so
	// none leak — mirroredQuery's trick.
	ch := make(chan result, len(urls))
	for _, url := range urls {
		go func(url string) { ch <- get(ctx, url) }(url)
	}

	var errs []string
	for range urls {
		if r := <-ch; r.err == nil {
			return r, nil // first success; defer cancel() aborts the others
		} else {
			errs = append(errs, r.err.Error())
		}
	}
	return result{}, fmt.Errorf("all %d requests failed: %s", len(urls), strings.Join(errs, "; "))
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: fetch url [url...]")
	}
	r, err := fetch(os.Args[1:]...)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s won (%d bytes)\n", r.url, len(r.body))
}
