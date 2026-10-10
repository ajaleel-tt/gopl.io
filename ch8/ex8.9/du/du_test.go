// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestPerRootTotals builds a known tree under several roots and checks that
// each root's total is computed independently — including a nested file
// attributed to its top-level root, and an empty root reported as zero.
//
//	A/  f1 (100B), sub/f2 (200B)   -> 2 files, 300 bytes
//	B/  f3 (50B)                    -> 1 file,  50 bytes
//	C/  (empty)                     -> 0 files,  0 bytes
func TestPerRootTotals(t *testing.T) {
	base := t.TempDir()
	rootA := filepath.Join(base, "A")
	rootB := filepath.Join(base, "B")
	rootC := filepath.Join(base, "C")
	writeSized(t, filepath.Join(rootA, "f1"), 100)
	writeSized(t, filepath.Join(rootA, "sub", "f2"), 200)
	writeSized(t, filepath.Join(rootB, "f3"), 50)
	if err := os.MkdirAll(rootC, 0o755); err != nil {
		t.Fatal(err)
	}

	got := duRoots([]string{rootA, rootB, rootC}, 20, nil, nil)
	want := map[string]total{
		rootA: {nfiles: 2, nbytes: 300},
		rootB: {nfiles: 1, nbytes: 50},
		rootC: {nfiles: 0, nbytes: 0},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("per-root totals wrong\n got:  %v\n want: %v", got, want)
	}
}

func writeSized(t *testing.T, path string, n int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, n), 0o644); err != nil {
		t.Fatal(err)
	}
}
