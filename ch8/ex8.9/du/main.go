// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

// Du (Exercise 8.9) computes disk usage like du2/du3, but keeps a *separate*
// running total for each root directory named on the command line, printing
// them periodically with -v and once at the end.
//
// Each file size is tagged with the root it descended from, so the single
// aggregator goroutine can bucket sizes per root. A counting semaphore bounds
// concurrent directory reads (as in du3).
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// fileSize is a file's size tagged with the root directory it belongs to.
type fileSize struct {
	root string
	size int64
}

type total struct {
	nfiles, nbytes int64
}

// duRoots walks each root concurrently (at most `parallelism` directory reads
// at a time) and accumulates a separate total per root. If tick is non-nil it
// calls report(totals) on each tick. It returns the final per-root totals.
func duRoots(roots []string, parallelism int, tick <-chan time.Time, report func(map[string]total)) map[string]total {
	sema := make(chan struct{}, parallelism)
	fileSizes := make(chan fileSize)
	var wg sync.WaitGroup

	// walkDir carries `root` (the top-level directory this descent started
	// from) down the recursion so every file it finds is attributed correctly.
	var walkDir func(root, dir string)
	walkDir = func(root, dir string) {
		defer wg.Done()
		sema <- struct{}{} // acquire: bound concurrent directory reads
		entries, err := os.ReadDir(dir)
		<-sema // release
		if err != nil {
			fmt.Fprintf(os.Stderr, "du: %v\n", err)
			return
		}
		for _, entry := range entries {
			if entry.IsDir() {
				wg.Add(1)
				go walkDir(root, filepath.Join(dir, entry.Name()))
			} else if info, err := entry.Info(); err == nil {
				fileSizes <- fileSize{root: root, size: info.Size()}
			}
		}
	}

	for _, root := range roots {
		wg.Add(1)
		go walkDir(root, root)
	}
	go func() { wg.Wait(); close(fileSizes) }()

	// This goroutine is the sole owner of `totals` — no locking needed.
	totals := make(map[string]total)
	for _, root := range roots {
		totals[root] = total{} // ensure every root reports, even if empty
	}
	for {
		select {
		case fs, ok := <-fileSizes:
			if !ok {
				return totals // all walks finished
			}
			t := totals[fs.root]
			t.nfiles++
			t.nbytes += fs.size
			totals[fs.root] = t
		case <-tick:
			if report != nil {
				report(totals)
			}
		}
	}
}

func printTotals(roots []string, totals map[string]total) {
	for _, root := range roots { // iterate the slice for stable order
		t := totals[root]
		fmt.Printf("%-30s %7d files  %6.2f GB\n", root, t.nfiles, float64(t.nbytes)/1e9)
	}
}

func main() {
	verbose := flag.Bool("v", false, "show per-root progress periodically")
	parallelism := flag.Int("parallelism", 20, "maximum concurrent directory reads")
	flag.Parse()
	roots := flag.Args()
	if len(roots) == 0 {
		roots = []string{"."}
	}

	// A nil tick channel disables the tick case in duRoots's select, so with
	// -v off there is simply no periodic reporting.
	var tick <-chan time.Time
	if *verbose {
		tick = time.Tick(500 * time.Millisecond)
	}
	totals := duRoots(roots, *parallelism, tick, func(t map[string]total) {
		printTotals(roots, t)
		fmt.Println("----")
	})
	printTotals(roots, totals)
}
