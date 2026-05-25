package treesort

import (
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestStringNilTree(t *testing.T) {
	var nilTree *tree
	if got := nilTree.String(); got != "[]" {
		t.Errorf("(*tree)(nil).String() = %q, want %q", got, "[]")
	}
}

func TestStringSingleNode(t *testing.T) {
	root := &tree{value: 42}
	if got := root.String(); got != "[42]" {
		t.Errorf("single-node String() = %q, want %q", got, "[42]")
	}
}

func TestStringInOrderAfterInserts(t *testing.T) {
	cases := []struct {
		input []int
		want  string
	}{
		{[]int{}, "[]"},
		{[]int{5}, "[5]"},
		{[]int{3, 1, 2}, "[1 2 3]"},
		{[]int{5, 3, 7, 1, 4, 6, 8}, "[1 3 4 5 6 7 8]"},
		{[]int{2, 2, 2}, "[2 2 2]"},
		{[]int{1, -3, 0, -1}, "[-3 -1 0 1]"},
	}
	for _, tc := range cases {
		var root *tree
		for _, v := range tc.input {
			root = add(root, v)
		}
		if got := root.String(); got != tc.want {
			t.Errorf("after inserting %v, String() = %q, want %q",
				tc.input, got, tc.want)
		}
	}
}

func TestSortAndRevealConsistency(t *testing.T) {
	values := []int{5, 3, 7, 1, 4, 6, 8}
	sorted, treeStr := SortAndReveal(values)

	want := "[1 3 4 5 6 7 8]"
	if treeStr != want {
		t.Errorf("treeStr = %q, want %q", treeStr, want)
	}

	// The tree's String form must match the sorted slice in shape.
	if treeStr != formatSlice(sorted) {
		t.Errorf("tree String %q does not match sorted slice format %q",
			treeStr, formatSlice(sorted))
	}
}

func TestSortStillWorks(t *testing.T) {
	data := make([]int, 50)
	for i := range data {
		data[i] = rand.Int() % 50
	}
	Sort(data)
	if !sort.IntsAreSorted(data) {
		t.Errorf("not sorted: %v", data)
	}
}

func formatSlice(xs []int) string {
	parts := make([]string, len(xs))
	for i, v := range xs {
		parts[i] = strconv.Itoa(v)
	}
	return "[" + strings.Join(parts, " ") + "]"
}
