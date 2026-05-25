// Exercise 7.10: IsPalindrome uses sort.Interface to check whether a sequence
// reads the same forward and backward — without ever calling Swap.
//
// We only need Len and Less. Equality is defined by the exercise as:
//
//   !s.Less(i, j) && !s.Less(j, i)
//
// i.e., neither element is strictly less than the other → they're equal under
// the ordering. Two-pointer walk from both ends meets in the middle.
package main

import (
	"fmt"
	"sort"
)

// IsPalindrome reports whether s reads the same forward and backward.
// Time O(n/2), space O(1). Swap is never called.
func IsPalindrome(s sort.Interface) bool {
	for i, j := 0, s.Len()-1; i < j; i, j = i+1, j-1 {
		if s.Less(i, j) || s.Less(j, i) {
			return false
		}
	}
	return true
}

func main() {
	cases := []sort.Interface{
		sort.StringSlice{"a", "b", "a"},
		sort.StringSlice{"r", "a", "c", "e", "c", "a", "r"},
		sort.StringSlice{"a", "b", "c"},
		sort.IntSlice{1, 2, 3, 2, 1},
		sort.IntSlice{1, 2, 3},
	}
	for _, c := range cases {
		fmt.Printf("%-30v → palindrome=%v\n", c, IsPalindrome(c))
	}
}
