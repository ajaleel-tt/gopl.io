// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

package intset

import (
	"fmt"
	"testing"
	"unsafe"
)

func Example_one() {
	//!+main
	var x, y IntSet
	x.Add(1)
	x.Add(144)
	x.Add(9)
	fmt.Println(x.String()) // "{1 9 144}"

	y.Add(9)
	y.Add(42)
	fmt.Println(y.String()) // "{9 42}"

	x.UnionWith(&y)
	fmt.Println(x.String()) // "{1 9 42 144}"

	fmt.Println(x.Has(9), x.Has(123)) // "true false"
	//!-main

	// Output:
	// {1 9 144}
	// {9 42}
	// {1 9 42 144}
	// true false
}

func Example_two() {
	var x IntSet
	x.Add(1)
	x.Add(144)
	x.Add(9)
	x.Add(42)

	//!+note
	fmt.Println(&x)         // "{1 9 42 144}"
	fmt.Println(x.String()) // "{1 9 42 144}"
	fmt.Println(x)          // "{[4398046511618 0 65536]}"
	//!-note

	// Output:
	// {1 9 42 144}
	// {1 9 42 144}
	// {[4398046511618 0 65536]}
}

func TestLen(t *testing.T) {
	var s IntSet
	if got := s.Len(); got != 0 {
		t.Errorf("empty set Len() = %d, want 0", got)
	}

	s.Add(1)
	s.Add(9)
	s.Add(144)
	if got := s.Len(); got != 3 {
		t.Errorf("after 3 adds, Len() = %d, want 3", got)
	}

	// Re-adding existing elements doesn't change Len.
	s.Add(9)
	if got := s.Len(); got != 3 {
		t.Errorf("re-add same element changed Len to %d, want 3", got)
	}

	// Len decreases after Remove.
	s.Remove(9)
	if got := s.Len(); got != 2 {
		t.Errorf("after remove, Len() = %d, want 2", got)
	}
}

func TestRemove(t *testing.T) {
	var s IntSet
	s.Add(1)
	s.Add(9)
	s.Add(144)

	s.Remove(9)
	if s.Has(9) {
		t.Error("after Remove(9), Has(9) still true")
	}
	if !s.Has(1) || !s.Has(144) {
		t.Errorf("Remove(9) affected other elements: Has(1)=%v Has(144)=%v",
			s.Has(1), s.Has(144))
	}

	// Removing a non-member is a no-op.
	s.Remove(42)
	if s.Len() != 2 {
		t.Errorf("Remove of non-member changed Len to %d, want 2", s.Len())
	}

	// Remove of a value past the end of words is a no-op (no panic).
	var empty IntSet
	empty.Remove(1000)
	if empty.Len() != 0 {
		t.Errorf("Remove on empty set produced Len %d, want 0", empty.Len())
	}
}

func TestClear(t *testing.T) {
	var s IntSet
	s.Add(1)
	s.Add(100)
	s.Add(1000)

	s.Clear()
	if s.Len() != 0 {
		t.Errorf("after Clear, Len() = %d, want 0", s.Len())
	}
	if s.Has(1) || s.Has(100) || s.Has(1000) {
		t.Error("Clear did not remove all elements")
	}

	// Cleared set should still be usable.
	s.Add(7)
	if !s.Has(7) || s.Len() != 1 {
		t.Errorf("after Clear+Add(7), Has(7)=%v Len=%d", s.Has(7), s.Len())
	}

	// Clear on an empty set is a no-op.
	var empty IntSet
	empty.Clear()
	if empty.Len() != 0 {
		t.Error("Clear on empty set should still be empty")
	}
}

func TestCopy(t *testing.T) {
	var s IntSet
	s.Add(1)
	s.Add(9)
	s.Add(144)

	c := s.Copy()

	// Initial contents match.
	if c.Len() != s.Len() {
		t.Errorf("copy Len %d, original Len %d", c.Len(), s.Len())
	}
	for _, x := range []int{1, 9, 144} {
		if !c.Has(x) {
			t.Errorf("copy missing element %d", x)
		}
	}

	// Mutating the copy does not affect the original.
	c.Add(500)
	if s.Has(500) {
		t.Error("adding to copy leaked into original")
	}
	c.Remove(1)
	if !s.Has(1) {
		t.Error("removing from copy leaked into original")
	}

	// Mutating the original does not affect the copy.
	s.Add(2000)
	if c.Has(2000) {
		t.Error("adding to original leaked into copy")
	}
}

func TestAddAll(t *testing.T) {
	var s IntSet
	s.AddAll(1, 9, 144, 42)

	if s.Len() != 4 {
		t.Errorf("Len after AddAll(1,9,144,42) = %d, want 4", s.Len())
	}
	for _, x := range []int{1, 9, 42, 144} {
		if !s.Has(x) {
			t.Errorf("Has(%d) = false after AddAll", x)
		}
	}
}

func TestAddAllNoArgs(t *testing.T) {
	var s IntSet
	s.AddAll()
	if s.Len() != 0 {
		t.Errorf("AddAll() with no args produced Len %d, want 0", s.Len())
	}

	s.Add(7)
	s.AddAll()
	if s.Len() != 1 || !s.Has(7) {
		t.Errorf("AddAll() should not disturb existing elements: Len=%d Has(7)=%v",
			s.Len(), s.Has(7))
	}
}

func TestAddAllSpread(t *testing.T) {
	var s IntSet
	values := []int{2, 4, 6, 8, 10}
	s.AddAll(values...)

	if s.Len() != len(values) {
		t.Errorf("Len = %d after spread, want %d", s.Len(), len(values))
	}
	for _, v := range values {
		if !s.Has(v) {
			t.Errorf("Has(%d) = false after spread AddAll", v)
		}
	}
}

func TestAddAllDuplicates(t *testing.T) {
	var s IntSet
	s.AddAll(5, 5, 5, 10, 10)

	if s.Len() != 2 {
		t.Errorf("AddAll with duplicates produced Len %d, want 2", s.Len())
	}
	if !s.Has(5) || !s.Has(10) {
		t.Errorf("missing elements after dedup AddAll: Has(5)=%v Has(10)=%v",
			s.Has(5), s.Has(10))
	}
}

func TestAddAllCumulative(t *testing.T) {
	var s IntSet
	s.AddAll(1, 2, 3)
	s.AddAll(3, 4, 5) // overlapping
	if s.Len() != 5 {
		t.Errorf("cumulative Len = %d, want 5", s.Len())
	}
	for _, x := range []int{1, 2, 3, 4, 5} {
		if !s.Has(x) {
			t.Errorf("Has(%d) = false", x)
		}
	}
}

// makeSet is a convenience for building an IntSet from a slice of ints.
func makeSet(xs ...int) *IntSet {
	var s IntSet
	s.AddAll(xs...)
	return &s
}

// elementsOf returns the elements of s as a sorted slice for stable comparison.
func elementsOf(s *IntSet) []int {
	var out []int
	for i, word := range s.words {
		for j := 0; j < 64; j++ {
			if word&(1<<uint(j)) != 0 {
				out = append(out, 64*i+j)
			}
		}
	}
	return out
}

func equalElements(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestIntersectWith(t *testing.T) {
	cases := []struct {
		name    string
		s, u    []int
		wantSet []int
	}{
		{"overlap", []int{1, 2, 3, 4}, []int{3, 4, 5, 6}, []int{3, 4}},
		{"disjoint", []int{1, 2}, []int{3, 4}, nil},
		{"identical", []int{1, 2, 3}, []int{1, 2, 3}, []int{1, 2, 3}},
		{"s empty", nil, []int{1, 2}, nil},
		{"t empty", []int{1, 2}, nil, nil},
		{"s larger range", []int{1, 200, 300}, []int{200}, []int{200}},
		{"t larger range", []int{1, 2}, []int{1, 200, 300}, []int{1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := makeSet(tc.s...)
			u := makeSet(tc.u...)
			s.IntersectWith(u)
			got := elementsOf(s)
			if !equalElements(got, tc.wantSet) {
				t.Errorf("%v ∩ %v: got %v, want %v", tc.s, tc.u, got, tc.wantSet)
			}
			// Verify the other operand wasn't mutated.
			if !equalElements(elementsOf(u), tc.u) {
				t.Errorf("t was mutated: now %v, originally %v", elementsOf(u), tc.u)
			}
		})
	}
}

func TestDifferenceWith(t *testing.T) {
	cases := []struct {
		name    string
		s, u    []int
		wantSet []int
	}{
		{"overlap", []int{1, 2, 3, 4}, []int{3, 4, 5, 6}, []int{1, 2}},
		{"disjoint", []int{1, 2}, []int{3, 4}, []int{1, 2}},
		{"identical", []int{1, 2, 3}, []int{1, 2, 3}, nil},
		{"s empty", nil, []int{1, 2}, nil},
		{"t empty", []int{1, 2}, nil, []int{1, 2}},
		{"t larger range", []int{1, 2}, []int{1, 200}, []int{2}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := makeSet(tc.s...)
			u := makeSet(tc.u...)
			s.DifferenceWith(u)
			got := elementsOf(s)
			if !equalElements(got, tc.wantSet) {
				t.Errorf("%v - %v: got %v, want %v", tc.s, tc.u, got, tc.wantSet)
			}
			if !equalElements(elementsOf(u), tc.u) {
				t.Errorf("t was mutated: now %v, originally %v", elementsOf(u), tc.u)
			}
		})
	}
}

func TestSymmetricDifference(t *testing.T) {
	cases := []struct {
		name    string
		s, u    []int
		wantSet []int
	}{
		{"overlap", []int{1, 2, 3, 4}, []int{3, 4, 5, 6}, []int{1, 2, 5, 6}},
		{"disjoint", []int{1, 2}, []int{3, 4}, []int{1, 2, 3, 4}},
		{"identical", []int{1, 2, 3}, []int{1, 2, 3}, nil},
		{"s empty", nil, []int{1, 2}, []int{1, 2}},
		{"t empty", []int{1, 2}, nil, []int{1, 2}},
		{"t extends past s", []int{1}, []int{1, 200, 300}, []int{200, 300}},
		{"s extends past t", []int{1, 200, 300}, []int{1}, []int{200, 300}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := makeSet(tc.s...)
			u := makeSet(tc.u...)
			s.SymmetricDifference(u)
			got := elementsOf(s)
			if !equalElements(got, tc.wantSet) {
				t.Errorf("%v △ %v: got %v, want %v", tc.s, tc.u, got, tc.wantSet)
			}
			if !equalElements(elementsOf(u), tc.u) {
				t.Errorf("t was mutated: now %v, originally %v", elementsOf(u), tc.u)
			}
		})
	}
}

// TestSymmetricDifferenceIdentity confirms A △ B == (A ∪ B) - (A ∩ B)
// using the methods themselves to define one side of the equation.
func TestSymmetricDifferenceIdentity(t *testing.T) {
	a := makeSet(1, 2, 3, 4, 5, 200)
	b := makeSet(3, 4, 5, 6, 7, 300)

	// Compute (A ∪ B) - (A ∩ B) the long way.
	union := a.Copy()
	union.UnionWith(b)
	inter := a.Copy()
	inter.IntersectWith(b)
	union.DifferenceWith(inter)

	// Compute A △ B directly.
	symDiff := a.Copy()
	symDiff.SymmetricDifference(b)

	if !equalElements(elementsOf(union), elementsOf(symDiff)) {
		t.Errorf("(A∪B)-(A∩B) = %v but A△B = %v",
			elementsOf(union), elementsOf(symDiff))
	}
}

func TestUintSizeIs32Or64(t *testing.T) {
	if uintSize != 32 && uintSize != 64 {
		t.Errorf("uintSize = %d, want 32 or 64", uintSize)
	}
}

func TestUintSizeMatchesPlatform(t *testing.T) {
	// unsafe.Sizeof returns the size of a uint in bytes; convert to bits.
	want := int(unsafe.Sizeof(uint(0))) * 8
	if uintSize != want {
		t.Errorf("uintSize = %d, but unsafe.Sizeof reports uint is %d bits",
			uintSize, want)
	}
}

// TestWordBoundaryAdd exercises values that straddle a word boundary on
// either platform: uintSize-1 (last bit of word 0), uintSize (first bit of
// word 1), and uintSize+1.
func TestWordBoundaryAdd(t *testing.T) {
	var s IntSet
	values := []int{uintSize - 1, uintSize, uintSize + 1, 2*uintSize - 1, 2 * uintSize}
	for _, v := range values {
		s.Add(v)
	}
	for _, v := range values {
		if !s.Has(v) {
			t.Errorf("Has(%d) = false after Add(%d), uintSize=%d", v, v, uintSize)
		}
	}
	if s.Len() != len(values) {
		t.Errorf("Len = %d, want %d", s.Len(), len(values))
	}
}

// TestWordCountByValue verifies the internal storage grows by exactly one
// word per uintSize-block of bits. Reaches into s.words intentionally.
func TestWordCountByValue(t *testing.T) {
	cases := []struct {
		add       int
		wantWords int
	}{
		{0, 1},
		{uintSize - 1, 1},
		{uintSize, 2},
		{2*uintSize - 1, 2},
		{2 * uintSize, 3},
		{3*uintSize + 5, 4},
	}
	for _, tc := range cases {
		var s IntSet
		s.Add(tc.add)
		if got := len(s.words); got != tc.wantWords {
			t.Errorf("Add(%d) on uintSize=%d produced %d words, want %d",
				tc.add, uintSize, got, tc.wantWords)
		}
	}
}

func TestStringSpansWords(t *testing.T) {
	// Pick values such that they span across the platform's word size.
	s := makeSet(0, uintSize-1, uintSize, 2*uintSize+3)
	got := s.String()
	want := fmt.Sprintf("{0 %d %d %d}", uintSize-1, uintSize, 2*uintSize+3)
	if got != want {
		t.Errorf("String() = %q, want %q (uintSize=%d)", got, want, uintSize)
	}
}

func TestElemsEmpty(t *testing.T) {
	var s IntSet
	got := s.Elems()
	if len(got) != 0 {
		t.Errorf("empty set Elems() = %v, want empty", got)
	}
}

func TestElemsAscending(t *testing.T) {
	s := makeSet(144, 1, 42, 9)
	got := s.Elems()
	want := []int{1, 9, 42, 144}
	if !equalElements(got, want) {
		t.Errorf("Elems() = %v, want %v (ascending)", got, want)
	}
}

func TestElemsSpansMultipleWords(t *testing.T) {
	// Force elements across at least three 64-bit words.
	s := makeSet(0, 63, 64, 127, 128, 191, 200)
	got := s.Elems()
	want := []int{0, 63, 64, 127, 128, 191, 200}
	if !equalElements(got, want) {
		t.Errorf("Elems() = %v, want %v", got, want)
	}
}

func TestElemsRangeLoop(t *testing.T) {
	s := makeSet(2, 4, 6, 8, 10)
	sum := 0
	for _, x := range s.Elems() {
		sum += x
	}
	if sum != 2+4+6+8+10 {
		t.Errorf("sum via range loop = %d, want 30", sum)
	}
}

func TestElemsIndependentOfSet(t *testing.T) {
	s := makeSet(1, 2, 3)
	got := s.Elems()

	// Mutating the returned slice must not affect the set.
	got[0] = 999
	if s.Has(999) {
		t.Error("mutating Elems result leaked into set")
	}
	if !s.Has(1) {
		t.Error("Has(1) became false after mutating Elems result")
	}

	// Adding to the set must not affect a previously returned slice.
	s.Add(500)
	for _, v := range got {
		if v == 500 {
			t.Errorf("set mutation leaked into previously returned slice")
		}
	}
}

func TestElemsLenMatchesSet(t *testing.T) {
	for _, n := range []int{0, 1, 5, 64, 100} {
		var s IntSet
		for i := 0; i < n; i++ {
			s.Add(i * 7) // spread out a bit
		}
		got := s.Elems()
		if len(got) != s.Len() {
			t.Errorf("for n=%d: len(Elems)=%d, s.Len()=%d", n, len(got), s.Len())
		}
	}
}

func TestCopyOfEmpty(t *testing.T) {
	var empty IntSet
	c := empty.Copy()
	if c == nil {
		t.Fatal("Copy of empty set returned nil")
	}
	if c.Len() != 0 {
		t.Errorf("Copy of empty set has Len %d, want 0", c.Len())
	}
	// Copy of empty should be usable.
	c.Add(42)
	if !c.Has(42) || empty.Has(42) {
		t.Errorf("Copy of empty not independent: copy.Has(42)=%v empty.Has(42)=%v",
			c.Has(42), empty.Has(42))
	}
}
