// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

// Crawl (Exercise 8.6) is the concurrent web crawler with depth limiting.
// With -depth=N, only URLs reachable by at most N links from a seed are
// fetched (seeds are depth 0; links found on a depth-d page are depth d+1).
//
// It builds on crawl2's design — a counting semaphore bounds concurrent
// fetches, and an explicit count of pending worklist sends gives clean
// termination, which depth-limiting now makes possible (the crawl is finite).
//
// Usage:
//
//	go run . -depth=3 https://golang.org
package main

import (
	"flag"
	"fmt"
	"log"

	"gopl.io/ch5/links"
)

// item is a URL paired with its depth: the number of links followed from a
// seed to reach it.
type item struct {
	url   string
	depth int
}

// crawlSite crawls outward from seeds, fetching only URLs within maxDepth links
// and running at most `parallelism` fetches at once. It returns the set of URLs
// fetched (deduplicated) and terminates once no work remains.
func crawlSite(seeds []string, maxDepth, parallelism int) []string {
	tokens := make(chan struct{}, parallelism) // counting semaphore

	// crawl fetches one item's links, tagging each with the next depth.
	crawl := func(it item) []item {
		tokens <- struct{}{} // acquire: bound concurrent fetches
		list, err := links.Extract(it.url)
		<-tokens // release
		if err != nil {
			log.Print(err)
		}
		next := make([]item, 0, len(list))
		for _, link := range list {
			next = append(next, item{url: link, depth: it.depth + 1})
		}
		return next
	}

	worklist := make(chan []item)
	n := 1 // number of pending sends to worklist (starts with the seed batch)
	go func() {
		seed := make([]item, len(seeds))
		for i, u := range seeds {
			seed[i] = item{url: u, depth: 0}
		}
		worklist <- seed
	}()

	// The main goroutine owns seen/fetched/n — no other goroutine touches
	// them, so no locking is needed.
	var fetched []string
	seen := make(map[string]bool)
	for ; n > 0; n-- {
		for _, it := range <-worklist {
			if it.depth > maxDepth || seen[it.url] {
				continue // too deep, or already crawled
			}
			seen[it.url] = true
			fetched = append(fetched, it.url)
			fmt.Printf("[depth %d] %s\n", it.depth, it.url)
			n++
			go func(it item) {
				worklist <- crawl(it)
			}(it)
		}
	}
	return fetched
}

func main() {
	maxDepth := flag.Int("depth", 3, "maximum number of links to follow from a seed")
	parallelism := flag.Int("parallelism", 20, "maximum concurrent fetches")
	flag.Parse()

	if len(flag.Args()) == 0 {
		log.Fatal("usage: crawl [-depth=N] [-parallelism=N] url...")
	}
	crawlSite(flag.Args(), *maxDepth, *parallelism)
}
