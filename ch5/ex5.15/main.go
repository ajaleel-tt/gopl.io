// Exercise 5.15: variadic max and min, in two flavors.
//
// maxAny / minAny accept zero arguments by returning the algebraic identity
// element of the operation: math.MinInt for max (so max(x, MinInt) == x for
// any x), math.MaxInt for min. These compose cleanly in reductions but the
// "empty result" case is a sentinel that's easy to misread.
//
// max / min make the empty case impossible by taking a required first
// argument followed by a variadic rest. The compiler enforces "at least one".
package main

import (
	"fmt"
	"math"
)

func maxAny(vals ...int) int {
	m := math.MinInt
	for _, v := range vals {
		if v > m {
			m = v
		}
	}
	return m
}

func minAny(vals ...int) int {
	m := math.MaxInt
	for _, v := range vals {
		if v < m {
			m = v
		}
	}
	return m
}

func max(first int, rest ...int) int {
	m := first
	for _, v := range rest {
		if v > m {
			m = v
		}
	}
	return m
}

func min(first int, rest ...int) int {
	m := first
	for _, v := range rest {
		if v < m {
			m = v
		}
	}
	return m
}

// maxBy returns the largest of first and rest under the ordering defined by
// comp, where comp(a, b) reports whether a is "less than" b. Pass the
// natural less for ordinary max; pass any other ordering (by absolute value,
// reversed, etc.) to customize.
func maxBy(comp func(int, int) bool, first int, rest ...int) int {
	m := first
	for _, v := range rest {
		if comp(m, v) {
			m = v
		}
	}
	return m
}

// minBy returns the smallest of first and rest under the ordering defined
// by comp, where comp(a, b) reports whether a is "less than" b.
func minBy(comp func(int, int) bool, first int, rest ...int) int {
	m := first
	for _, v := range rest {
		if comp(v, m) {
			m = v
		}
	}
	return m
}

func main() {
	fmt.Println("--- maxAny / minAny (zero args allowed) ---")
	fmt.Printf("maxAny()         = %d  (math.MinInt)\n", maxAny())
	fmt.Printf("minAny()         = %d  (math.MaxInt)\n", minAny())
	fmt.Printf("maxAny(3,1,4,1)  = %d\n", maxAny(3, 1, 4, 1))
	fmt.Printf("minAny(3,1,4,1)  = %d\n", minAny(3, 1, 4, 1))

	nums := []int{5, 9, 2, 7}
	fmt.Printf("maxAny(nums...)  = %d\n", maxAny(nums...))
	fmt.Printf("minAny(nums...)  = %d\n", minAny(nums...))

	fmt.Println("\n--- max / min (at-least-one enforced at compile time) ---")
	fmt.Printf("max(7)           = %d\n", max(7))
	fmt.Printf("max(3,1,4,1,5)   = %d\n", max(3, 1, 4, 1, 5))
	fmt.Printf("min(3,1,4,1,5)   = %d\n", min(3, 1, 4, 1, 5))
	// max() — would fail to compile: "not enough arguments in call to max"

	fmt.Println("\n--- maxBy / minBy (custom ordering) ---")
	less := func(a, b int) bool { return a < b }
	byAbs := func(a, b int) bool {
		if a < 0 {
			a = -a
		}
		if b < 0 {
			b = -b
		}
		return a < b
	}
	fmt.Printf("maxBy(less, -3,1,-4,1,5) = %d\n", maxBy(less, -3, 1, -4, 1, 5))
	fmt.Printf("minBy(less, -3,1,-4,1,5) = %d\n", minBy(less, -3, 1, -4, 1, 5))
	fmt.Printf("maxBy(byAbs,-3,1,-4,1,5) = %d  (largest |x|)\n", maxBy(byAbs, -3, 1, -4, 1, 5))
	fmt.Printf("minBy(byAbs,-3,1,-4,1,5) = %d  (smallest |x|)\n", minBy(byAbs, -3, 1, -4, 1, 5))
}
