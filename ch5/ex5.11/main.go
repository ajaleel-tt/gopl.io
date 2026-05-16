// Exercise 5.11: topoSort extended to detect and report cycles.
package main

import (
	"fmt"
	"strings"
)

// acyclicPrereqs is the original DAG.
var acyclicPrereqs = map[string]map[string]bool{
	"algorithms": {"data structures": true},
	"calculus":   {"linear algebra": true},

	"compilers": {
		"data structures":       true,
		"formal languages":      true,
		"computer organization": true,
	},

	"data structures":       {"discrete math": true},
	"databases":             {"data structures": true},
	"discrete math":         {"intro to programming": true},
	"formal languages":      {"discrete math": true},
	"networks":              {"operating systems": true},
	"operating systems":     {"data structures": true, "computer organization": true},
	"programming languages": {"data structures": true, "computer organization": true},
}

// cyclicPrereqs adds "calculus" as a prerequisite of "linear algebra",
// creating a cycle: calculus -> linear algebra -> calculus.
var cyclicPrereqs = func() map[string]map[string]bool {
	m := make(map[string]map[string]bool, len(acyclicPrereqs)+1)
	for k, v := range acyclicPrereqs {
		copyDeps := make(map[string]bool, len(v))
		for d := range v {
			copyDeps[d] = true
		}
		m[k] = copyDeps
	}
	m["linear algebra"] = map[string]bool{"calculus": true}
	return m
}()

func main() {
	fmt.Println("--- acyclic ---")
	order, err := topoSort(acyclicPrereqs)
	if err != nil {
		fmt.Println("error:", err)
	} else {
		for i, course := range order {
			fmt.Printf("%d:\t%s\n", i+1, course)
		}
	}

	fmt.Println("\n--- cyclic ---")
	if _, err := topoSort(cyclicPrereqs); err != nil {
		fmt.Println("error:", err)
	} else {
		fmt.Println("no cycle detected (unexpected)")
	}
}

// Node colors for DFS.
const (
	white = iota // unvisited
	gray         // on the recursion stack
	black        // fully processed
)

// topoSort returns a topological ordering of m, or an error describing a
// cycle if one exists.
func topoSort(m map[string]map[string]bool) ([]string, error) {
	var order []string
	color := make(map[string]int)
	var stack []string

	var visit func(name string) error
	visit = func(name string) error {
		switch color[name] {
		case gray:
			return fmt.Errorf("cycle detected: %s", formatCycle(stack, name))
		case black:
			return nil
		}
		color[name] = gray
		stack = append(stack, name)

		for dep := range m[name] {
			if err := visit(dep); err != nil {
				return err
			}
		}

		stack = stack[:len(stack)-1]
		color[name] = black
		order = append(order, name)
		return nil
	}

	for course := range m {
		if err := visit(course); err != nil {
			return nil, err
		}
	}
	return order, nil
}

// formatCycle renders the cycle path from the first occurrence of name in
// stack through the rest of the stack and back to name.
func formatCycle(stack []string, name string) string {
	for i, n := range stack {
		if n == name {
			path := append([]string{}, stack[i:]...)
			path = append(path, name)
			return strings.Join(path, " -> ")
		}
	}
	return name + " -> ... -> " + name
}
