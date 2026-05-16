package main

import (
	"fmt"
	"strings"
	"testing"
)

// isValidTopoOrder reports whether order is a valid topological ordering of m:
// every prerequisite must appear before any course that depends on it, and
// every node in m (and every prerequisite mentioned) must appear exactly once.
func isValidTopoOrder(m map[string]map[string]bool, order []string) error {
	pos := make(map[string]int, len(order))
	for i, name := range order {
		if _, dup := pos[name]; dup {
			return fmt.Errorf("duplicate node in order: %s", name)
		}
		pos[name] = i
	}

	for course, deps := range m {
		ci, ok := pos[course]
		if !ok {
			return fmt.Errorf("course %q missing from order", course)
		}
		for dep := range deps {
			di, ok := pos[dep]
			if !ok {
				return fmt.Errorf("prerequisite %q (of %q) missing from order", dep, course)
			}
			if di >= ci {
				return fmt.Errorf("prerequisite %q at index %d does not precede %q at index %d",
					dep, di, course, ci)
			}
		}
	}
	return nil
}

func TestTopoSortNondeterministicButValid(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 200; i++ {
		order := topoSort(prereqs)
		if err := isValidTopoOrder(prereqs, order); err != nil {
			t.Fatalf("invalid topo order on run %d: %v\norder=%v", i, err, order)
		}
		seen[strings.Join(order, "|")] = true
	}
	if len(seen) < 2 {
		t.Logf("only %d distinct ordering(s) observed across 200 runs — "+
			"map iteration nondeterminism is statistical, not guaranteed", len(seen))
	} else {
		t.Logf("observed %d distinct valid orderings across 200 runs", len(seen))
	}
}

func TestTopoSortIncludesAllNodes(t *testing.T) {
	order := topoSort(prereqs)

	want := make(map[string]bool)
	for course, deps := range prereqs {
		want[course] = true
		for dep := range deps {
			want[dep] = true
		}
	}

	got := make(map[string]bool, len(order))
	for _, name := range order {
		got[name] = true
	}

	for name := range want {
		if !got[name] {
			t.Errorf("missing node in order: %s", name)
		}
	}
	if len(got) != len(want) {
		t.Errorf("got %d unique nodes, want %d", len(got), len(want))
	}
}
