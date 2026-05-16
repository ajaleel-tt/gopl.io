package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeTree builds a directory tree under root described by paths (relative to
// root). Paths ending in "/" are directories; others are files.
func makeTree(t *testing.T, root string, paths []string) {
	t.Helper()
	for _, p := range paths {
		full := filepath.Join(root, p)
		if strings.HasSuffix(p, "/") {
			if err := os.MkdirAll(full, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// runWalk runs breadthFirst from root and returns the visited paths in order,
// each made relative to root for stable comparison.
func runWalk(t *testing.T, root string) []string {
	t.Helper()
	var buf bytes.Buffer
	breadthFirst(makeWalker(&buf), []string{root})

	var rel []string
	for _, line := range strings.Split(strings.TrimRight(buf.String(), "\n"), "\n") {
		r, err := filepath.Rel(root, line)
		if err != nil {
			t.Fatalf("rel %q: %v", line, err)
		}
		rel = append(rel, r)
	}
	return rel
}

func TestBFSOrderShallowBeforeDeep(t *testing.T) {
	root := t.TempDir()
	makeTree(t, root, []string{
		"a.txt",
		"b.txt",
		"sub/",
		"sub/c.txt",
		"sub/deep/",
		"sub/deep/d.txt",
	})

	visited := runWalk(t, root)

	// Build position map.
	pos := make(map[string]int, len(visited))
	for i, p := range visited {
		pos[p] = i
	}

	// Every shallow item must precede any deeper item.
	depths := map[string]int{
		".":              0,
		"a.txt":          1,
		"b.txt":          1,
		"sub":            1,
		"sub/c.txt":      2,
		"sub/deep":       2,
		"sub/deep/d.txt": 3,
	}
	for a, da := range depths {
		for b, db := range depths {
			if da < db {
				if pos[a] >= pos[b] {
					t.Errorf("BFS violated: %q (depth %d, pos %d) should precede %q (depth %d, pos %d)\nvisited: %v",
						a, da, pos[a], b, db, pos[b], visited)
				}
			}
		}
	}
}

func TestBFSVisitsEveryNodeOnce(t *testing.T) {
	root := t.TempDir()
	paths := []string{
		"x.txt",
		"y.txt",
		"d1/",
		"d1/m.txt",
		"d1/n.txt",
		"d2/",
		"d2/d3/",
		"d2/d3/leaf.txt",
	}
	makeTree(t, root, paths)

	visited := runWalk(t, root)
	got := make(map[string]int)
	for _, p := range visited {
		got[p]++
	}

	want := append([]string{"."}, paths...)
	for i, p := range want {
		want[i] = filepath.Clean(strings.TrimSuffix(p, "/"))
	}

	for _, p := range want {
		if got[p] != 1 {
			t.Errorf("%q visited %d times, want 1", p, got[p])
		}
	}
	if len(got) != len(want) {
		t.Errorf("visited %d distinct paths, want %d\nvisited: %v", len(got), len(want), visited)
	}
}

func TestBFSEmptyDir(t *testing.T) {
	root := t.TempDir()
	visited := runWalk(t, root)
	if len(visited) != 1 || visited[0] != "." {
		t.Errorf("empty dir: got %v, want [\".\"]", visited)
	}
}

func TestBFSSingleFile(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "lone.txt")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	breadthFirst(makeWalker(&buf), []string{file})

	got := strings.TrimSpace(buf.String())
	if got != file {
		t.Errorf("single file: got %q, want %q", got, file)
	}
}
