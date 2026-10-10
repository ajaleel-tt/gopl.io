// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

// Crawl (Exercise 8.10) is the §8.6 concurrent crawler with cancellation.
//
// The exercise suggests http.Request.Cancel, but that field is deprecated
// (since Go 1.7); its modern replacement is context. So a single context does
// both jobs at once:
//   - crawler-wide cancellation — the loop and workers bail on ctx.Done()
//     (this is du4's done/cancelled(), unified), and
//   - per-request cancellation — each GET is built with
//     http.NewRequestWithContext, so cancelling ctx aborts an in-flight fetch
//     instead of waiting for a slow server (this is what req.Cancel did).
//
// Pressing return cancels the whole crawl and stops it promptly.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/net/html"
)

// tokens is a counting semaphore limiting concurrent fetches (as in crawl2).
var tokens = make(chan struct{}, 20)

// crawl fetches url and returns the links found on it. Both the semaphore
// acquire and the HTTP request honor ctx, so a cancelled crawl unwinds promptly
// rather than finishing in-flight work.
func crawl(ctx context.Context, url string) []string {
	fmt.Println(url)
	select {
	case tokens <- struct{}{}: // acquire a fetch permit...
	case <-ctx.Done():
		return nil // ...or bail out if we've been cancelled
	}
	defer func() { <-tokens }()

	list, err := extract(ctx, url)
	if err != nil {
		log.Print(err)
	}
	return list
}

// extract issues a cancellable GET for url and returns the hrefs it links to.
// Building the request with NewRequestWithContext is the modern equivalent of
// the exercise's http.Request.Cancel: cancelling ctx aborts the request even
// while it is in flight.
func extract(ctx context.Context, url string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err // wraps context.Canceled when ctx is cancelled mid-flight
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("getting %s: %s", url, resp.Status)
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parsing %s as HTML: %v", url, err)
	}
	var links []string
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key != "href" {
					continue
				}
				if link, err := resp.Request.URL.Parse(a.Val); err == nil {
					links = append(links, link.String())
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(doc)
	return links, nil
}

// crawlAll runs crawl2's bounded, self-terminating loop with cancellation woven
// through: the main select bails on ctx.Done(), and each worker's send is
// guarded so nothing blocks forever when we tear down.
func crawlAll(ctx context.Context, seeds []string) {
	worklist := make(chan []string)
	n := 1
	go func() {
		select {
		case worklist <- seeds:
		case <-ctx.Done():
		}
	}()

	// seen is owned by this goroutine alone.
	seen := make(map[string]bool)
	for ; n > 0; n-- {
		var list []string
		select {
		case list = <-worklist:
		case <-ctx.Done():
			return // cancelled: stop dispatching new work
		}
		for _, link := range list {
			if seen[link] {
				continue
			}
			seen[link] = true
			n++
			go func(link string) {
				found := crawl(ctx, link)
				select {
				case worklist <- found:
				case <-ctx.Done(): // don't leak the send if we've been cancelled
				}
			}(link)
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // release ctx resources on normal completion

	// Cancel the entire crawl when the user presses return.
	go func() {
		os.Stdin.Read(make([]byte, 1))
		cancel()
	}()

	crawlAll(ctx, os.Args[1:])
}
