// Exercise 5.14: BFS-walk the local filesystem from a root directory.
// Demonstrates breadthFirst on a tree (no cycles to worry about, in the
// absence of symlinks).
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: ex5.14 <root>...")
		os.Exit(1)
	}
	roots := os.Args[1:]
	breadthFirst(makeWalker(os.Stdout), roots)
}

// breadthFirst calls f for each item in the worklist. Any items returned by f
// are added to the worklist. f is called at most once for each item.
func breadthFirst(f func(item string) []string, worklist []string) {
	seen := make(map[string]bool)
	for len(worklist) > 0 {
		items := worklist
		worklist = nil
		for _, item := range items {
			if !seen[item] {
				seen[item] = true
				worklist = append(worklist, f(item)...)
			}
		}
	}
}

// makeWalker returns a worklist function that prints each visited path to w
// and returns the path's children (empty if the path isn't a directory).
func makeWalker(w io.Writer) func(string) []string {
	return func(path string) []string {
		fmt.Fprintln(w, path)
		entries, err := os.ReadDir(path)
		if err != nil {
			// Either not a directory, or unreadable — treat as a leaf.
			return nil
		}
		children := make([]string, 0, len(entries))
		for _, e := range entries {
			children = append(children, filepath.Join(path, e.Name()))
		}
		return children
	}
}
