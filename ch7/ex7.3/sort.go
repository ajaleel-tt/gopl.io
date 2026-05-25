// Exercise 7.3: treesort with a String method on *tree that reveals the
// in-order sequence of values (which, by BST invariant, is sorted).
package treesort

import (
	"bytes"
	"strconv"
)

type tree struct {
	value       int
	left, right *tree
}

// Sort sorts values in place.
func Sort(values []int) {
	var root *tree
	for _, v := range values {
		root = add(root, v)
	}
	appendValues(values[:0], root)
}

// SortAndReveal sorts values in place and returns the tree's String form.
// Exists so callers outside this package can observe the String method.
func SortAndReveal(values []int) (sorted []int, treeString string) {
	var root *tree
	for _, v := range values {
		root = add(root, v)
	}
	return appendValues(values[:0], root), root.String()
}

// appendValues appends the elements of t to values in order
// and returns the resulting slice.
func appendValues(values []int, t *tree) []int {
	if t != nil {
		values = appendValues(values, t.left)
		values = append(values, t.value)
		values = appendValues(values, t.right)
	}
	return values
}

func add(t *tree, value int) *tree {
	if t == nil {
		t = new(tree)
		t.value = value
		return t
	}
	if value < t.value {
		t.left = add(t.left, value)
	} else {
		t.right = add(t.right, value)
	}
	return t
}

// String returns the values of the tree in in-order traversal, formatted as
// "[v1 v2 v3]". By the BST invariant, this output is in non-decreasing
// order.
//
// A nil *tree prints as "[]" — defining the method on the pointer type lets
// us handle that case uniformly without panicking.
func (t *tree) String() string {
	if t == nil {
		return "[]"
	}
	var buf bytes.Buffer
	buf.WriteByte('[')
	first := true
	t.appendString(&buf, &first)
	buf.WriteByte(']')
	return buf.String()
}

func (t *tree) appendString(buf *bytes.Buffer, first *bool) {
	if t == nil {
		return
	}
	t.left.appendString(buf, first)
	if !*first {
		buf.WriteByte(' ')
	}
	buf.WriteString(strconv.Itoa(t.value))
	*first = false
	t.right.appendString(buf, first)
}

// (Alternative one-liner that delegates to appendValues + fmt.Sprint:
//
//	func (t *tree) String() string { return fmt.Sprint(appendValues(nil, t)) }
//
// fmt.Sprint on a []int already formats as "[1 2 3]", which is what we want.
// The explicit version above avoids the slice allocation by writing directly
// to a bytes.Buffer; for a learning exercise either is fine.)
