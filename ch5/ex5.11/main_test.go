package main

import (
	"strings"
	"testing"
)

func TestTopoSortAcyclic(t *testing.T) {
	order, err := topoSort(acyclicPrereqs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pos := make(map[string]int, len(order))
	for i, name := range order {
		pos[name] = i
	}
	for course, deps := range acyclicPrereqs {
		ci := pos[course]
		for dep := range deps {
			if pos[dep] >= ci {
				t.Errorf("prereq %q at %d does not precede %q at %d",
					dep, pos[dep], course, ci)
			}
		}
	}
}

func TestTopoSortCyclic(t *testing.T) {
	_, err := topoSort(cyclicPrereqs)
	if err == nil {
		t.Fatal("expected cycle error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "cycle") {
		t.Errorf("expected error to mention 'cycle', got: %v", err)
	}
	if !strings.Contains(msg, "calculus") || !strings.Contains(msg, "linear algebra") {
		t.Errorf("expected cycle to mention both 'calculus' and 'linear algebra', got: %v", err)
	}
}

func TestTopoSortSelfLoop(t *testing.T) {
	m := map[string]map[string]bool{
		"a": {"a": true},
	}
	_, err := topoSort(m)
	if err == nil {
		t.Fatal("expected cycle error for self-loop, got nil")
	}
	if !strings.Contains(err.Error(), "a -> a") {
		t.Errorf("expected self-loop to be reported as 'a -> a', got: %v", err)
	}
}

func TestTopoSortLongerCycle(t *testing.T) {
	// a -> b -> c -> a
	m := map[string]map[string]bool{
		"a": {"b": true},
		"b": {"c": true},
		"c": {"a": true},
	}
	_, err := topoSort(m)
	if err == nil {
		t.Fatal("expected cycle error, got nil")
	}
	for _, n := range []string{"a", "b", "c"} {
		if !strings.Contains(err.Error(), n) {
			t.Errorf("expected cycle to mention %q, got: %v", n, err)
		}
	}
}
