package main

import (
	"math"
	"testing"
)

func TestMaxAnyEmpty(t *testing.T) {
	if got := maxAny(); got != math.MinInt {
		t.Errorf("maxAny() = %d, want math.MinInt (%d)", got, math.MinInt)
	}
}

func TestMinAnyEmpty(t *testing.T) {
	if got := minAny(); got != math.MaxInt {
		t.Errorf("minAny() = %d, want math.MaxInt (%d)", got, math.MaxInt)
	}
}

func TestMaxAnyIdentity(t *testing.T) {
	// The identity-element choice means max(maxAny(), x) == x for any x.
	for _, x := range []int{-1000, 0, 42, math.MaxInt} {
		if got := max(x, maxAny()); got != x {
			t.Errorf("max(%d, maxAny()) = %d, want %d (identity property)", x, got, x)
		}
	}
}

func TestMinAnyIdentity(t *testing.T) {
	for _, x := range []int{math.MinInt, -7, 0, 100} {
		if got := min(x, minAny()); got != x {
			t.Errorf("min(%d, minAny()) = %d, want %d (identity property)", x, got, x)
		}
	}
}

func TestMaxValues(t *testing.T) {
	cases := []struct {
		args []int
		want int
	}{
		{[]int{7}, 7},
		{[]int{1, 2, 3}, 3},
		{[]int{3, 2, 1}, 3},
		{[]int{-5, -2, -10}, -2},
		{[]int{5, 5, 5}, 5},
		{[]int{math.MinInt, math.MaxInt}, math.MaxInt},
	}
	for _, tc := range cases {
		gotAny := maxAny(tc.args...)
		gotStrict := max(tc.args[0], tc.args[1:]...)
		if gotAny != tc.want {
			t.Errorf("maxAny(%v) = %d, want %d", tc.args, gotAny, tc.want)
		}
		if gotStrict != tc.want {
			t.Errorf("max(%v) = %d, want %d", tc.args, gotStrict, tc.want)
		}
	}
}

func TestMinValues(t *testing.T) {
	cases := []struct {
		args []int
		want int
	}{
		{[]int{7}, 7},
		{[]int{1, 2, 3}, 1},
		{[]int{3, 2, 1}, 1},
		{[]int{-5, -2, -10}, -10},
		{[]int{5, 5, 5}, 5},
		{[]int{math.MinInt, math.MaxInt}, math.MinInt},
	}
	for _, tc := range cases {
		gotAny := minAny(tc.args...)
		gotStrict := min(tc.args[0], tc.args[1:]...)
		if gotAny != tc.want {
			t.Errorf("minAny(%v) = %d, want %d", tc.args, gotAny, tc.want)
		}
		if gotStrict != tc.want {
			t.Errorf("min(%v) = %d, want %d", tc.args, gotStrict, tc.want)
		}
	}
}

func TestMaxSpread(t *testing.T) {
	nums := []int{4, 1, 9, 2}
	if got := maxAny(nums...); got != 9 {
		t.Errorf("maxAny(nums...) = %d, want 9", got)
	}
}

func less(a, b int) bool { return a < b }

func absLess(a, b int) bool {
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	return a < b
}

func TestMaxByMatchesMaxWithNaturalLess(t *testing.T) {
	cases := [][]int{
		{7},
		{1, 2, 3},
		{3, 2, 1},
		{-5, -2, -10},
		{5, 5, 5},
	}
	for _, args := range cases {
		want := max(args[0], args[1:]...)
		got := maxBy(less, args[0], args[1:]...)
		if got != want {
			t.Errorf("maxBy(less, %v) = %d, want %d", args, got, want)
		}
	}
}

func TestMinByMatchesMinWithNaturalLess(t *testing.T) {
	cases := [][]int{
		{7},
		{1, 2, 3},
		{3, 2, 1},
		{-5, -2, -10},
		{5, 5, 5},
	}
	for _, args := range cases {
		want := min(args[0], args[1:]...)
		got := minBy(less, args[0], args[1:]...)
		if got != want {
			t.Errorf("minBy(less, %v) = %d, want %d", args, got, want)
		}
	}
}

func TestMaxByReversedOrdering(t *testing.T) {
	// With reversed less (greater-than), maxBy should return the smallest.
	greater := func(a, b int) bool { return a > b }
	if got := maxBy(greater, 3, 1, 4, 1, 5, 9, 2, 6); got != 1 {
		t.Errorf("maxBy(greater, ...) = %d, want 1", got)
	}
}

func TestMaxByAbs(t *testing.T) {
	// Largest by absolute value among signed ints.
	if got := maxBy(absLess, -3, 1, -4, 1, 5); got != 5 {
		t.Errorf("maxBy(absLess, ...) = %d, want 5", got)
	}
	if got := maxBy(absLess, -10, 1, 2, 3); got != -10 {
		t.Errorf("maxBy(absLess, -10, 1, 2, 3) = %d, want -10", got)
	}
}

func TestMinByAbs(t *testing.T) {
	if got := minBy(absLess, -3, -4, 5, -1, 2); got != -1 {
		t.Errorf("minBy(absLess, ...) = %d, want -1", got)
	}
}

func TestMaxByMinBySingleArg(t *testing.T) {
	// With one arg, the comparator must never be called.
	called := false
	comp := func(a, b int) bool {
		called = true
		return a < b
	}
	if got := maxBy(comp, 42); got != 42 {
		t.Errorf("maxBy(comp, 42) = %d, want 42", got)
	}
	if got := minBy(comp, 42); got != 42 {
		t.Errorf("minBy(comp, 42) = %d, want 42", got)
	}
	if called {
		t.Error("comparator should not be called with a single argument")
	}
}

func TestMaxByStableOnTies(t *testing.T) {
	// On ties (less returns false in both directions), the first equal
	// value wins — confirms we use strict less, not less-or-equal.
	values := []int{5, 5, 5}
	if got := maxBy(less, values[0], values[1:]...); got != 5 {
		t.Errorf("maxBy(less, equal values) = %d, want 5", got)
	}
}
